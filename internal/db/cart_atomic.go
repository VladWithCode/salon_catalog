package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrCartTxProductNotFound is returned by CartTx.LockProductStock when the
// product row does not exist.
var ErrCartTxProductNotFound = errors.New("cart tx: product not found")

// CartProductStock is the minimal, server-trusted stock snapshot read and
// row-locked inside a cart mutation transaction. It is never derived from
// client input.
type CartProductStock struct {
	Available bool
	Quantity  int
}

// CartTx scopes a single cart mutation to one transaction on one pooled
// connection, so validation and writes for that mutation always happen
// atomically: no SELECT or UPDATE outside the transaction, no separate
// connections per step.
type CartTx struct {
	conn *pgxpool.Conn
	tx   pgx.Tx
	done bool
}

// BeginCartTx acquires a pooled connection from the package's own
// production pool and starts a transaction for a single cart mutation.
// Callers must call Commit or Rollback exactly once, then Close to release
// the connection.
func BeginCartTx(ctx context.Context) (*CartTx, error) {
	conn, err := GetConnWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return beginCartTxOnConn(ctx, conn)
}

// BeginCartTxWithPool starts a cart mutation transaction on a
// caller-supplied pool instead of the package's own production pool. It
// exists so a caller can point internal/cart.Service at an explicit
// *pgxpool.Pool — a throwaway integration-test database, for instance —
// without any package-level mutable state. It opens exactly one connection
// and one transaction per call, the same as BeginCartTx.
func BeginCartTxWithPool(ctx context.Context, pool *pgxpool.Pool) (*CartTx, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return beginCartTxOnConn(ctx, conn)
}

func beginCartTxOnConn(ctx context.Context, conn *pgxpool.Conn) (*CartTx, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		conn.Release()
		return nil, err
	}
	return &CartTx{conn: conn, tx: tx}, nil
}

// Commit commits the transaction.
func (t *CartTx) Commit(ctx context.Context) error {
	if t.done {
		return nil
	}
	t.done = true
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction if it has not already been committed or
// rolled back. The error is discarded: the caller already holds the error
// that triggered the rollback, and a rollback failure on an already-broken
// connection carries nothing worth surfacing to the client.
func (t *CartTx) Rollback(ctx context.Context) {
	if t.done {
		return
	}
	t.done = true
	_ = t.tx.Rollback(ctx)
}

// Close releases the pooled connection. Call it via defer immediately after
// BeginCartTx succeeds; it is safe to call after Commit or Rollback.
func (t *CartTx) Close() {
	t.conn.Release()
}

// EnsureAndLockCart guarantees a carts row exists for cartID and locks it for
// the rest of the transaction. Every other transaction mutating the same
// cart blocks on this lock until this one commits or rolls back, which is
// what serializes concurrent mutations of the same cart.
func (t *CartTx) EnsureAndLockCart(ctx context.Context, cartID string) error {
	if _, err := t.tx.Exec(ctx, `INSERT INTO carts (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`, cartID); err != nil {
		return err
	}

	var locked string
	return t.tx.QueryRow(ctx, `SELECT id FROM carts WHERE id = $1 FOR UPDATE`, cartID).Scan(&locked)
}

// LockCartIfExists locks the carts row for cartID if it exists, without
// creating one, and reports whether the row existed.
func (t *CartTx) LockCartIfExists(ctx context.Context, cartID string) (bool, error) {
	var locked string
	err := t.tx.QueryRow(ctx, `SELECT id FROM carts WHERE id = $1 FOR UPDATE`, cartID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// LockProductStock reads and row-locks the current availability and stock of
// a product directly from products, the source of truth for stock. It reads
// the base table rather than the catalog_products view because that view
// joins categories and images and cannot be locked safely with FOR UPDATE.
func (t *CartTx) LockProductStock(ctx context.Context, productID string) (CartProductStock, error) {
	var stock CartProductStock
	err := t.tx.QueryRow(
		ctx,
		`SELECT available, quantity FROM products WHERE id = $1 FOR UPDATE`,
		productID,
	).Scan(&stock.Available, &stock.Quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return CartProductStock{}, ErrCartTxProductNotFound
	}
	if err != nil {
		return CartProductStock{}, err
	}
	return stock, nil
}

// ItemQuantity reads the current quantity of a cart line, if any. Reading it
// without its own row lock is safe: the cart-level lock already acquired by
// EnsureAndLockCart or LockCartIfExists serializes every writer of this
// cart's lines for the lifetime of the transaction.
func (t *CartTx) ItemQuantity(ctx context.Context, cartID string, productID string) (int, bool, error) {
	var quantity int
	err := t.tx.QueryRow(
		ctx,
		`SELECT quantity FROM cart_items WHERE cart_id = $1 AND product_id = $2`,
		cartID, productID,
	).Scan(&quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return quantity, true, nil
}

// UpsertItem inserts a new cart line or overwrites an existing one with the
// given quantity and source, relying on the (cart_id, product_id) primary
// key to guarantee a single line per product. It is used by AddItem, where
// the source is always server-fixed rather than client-controlled.
func (t *CartTx) UpsertItem(ctx context.Context, cartID string, productID string, source string, quantity int) error {
	_, err := t.tx.Exec(
		ctx,
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (cart_id, product_id) DO UPDATE SET
		   quantity = EXCLUDED.quantity,
		   source = EXCLUDED.source,
		   updated_at = NOW()`,
		cartID, productID, quantity, source,
	)
	return err
}

// UpdateItemQuantity sets the absolute quantity of an existing cart line
// without touching its source. It is used by SetItemQuantity (PATCH), which
// represents an absolute quantity and must not reassign the line's origin.
func (t *CartTx) UpdateItemQuantity(ctx context.Context, cartID string, productID string, quantity int) error {
	_, err := t.tx.Exec(
		ctx,
		`UPDATE cart_items SET quantity = $3, updated_at = NOW() WHERE cart_id = $1 AND product_id = $2`,
		cartID, productID, quantity,
	)
	return err
}

// DeleteItem removes a single cart line scoped to (cart_id, product_id). It
// is idempotent: deleting a line that does not exist affects zero rows and
// returns no error.
func (t *CartTx) DeleteItem(ctx context.Context, cartID string, productID string) error {
	_, err := t.tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1 AND product_id = $2`, cartID, productID)
	return err
}

// DeleteAllItems removes every line of a cart, scoped to cart_id. It does not
// delete the carts row itself: a submitted quote may still reference it.
func (t *CartTx) DeleteAllItems(ctx context.Context, cartID string) error {
	_, err := t.tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	return err
}

// CartIdempotencyRecord is the minimal state read back for a stored
// idempotency claim: the hash of the request it was first associated with,
// and when that claim stops being valid. It never carries the raw
// Idempotency-Key header value; only its caller-supplied SHA-256 digest is
// ever written to or read from the database.
type CartIdempotencyRecord struct {
	RequestHash []byte
	ExpiresAt   time.Time
}

// LoadIdempotencyRecord reads and row-locks a stored idempotency claim for
// (cartID, keyHash), if one exists, and reports whether it was found. The
// row lock is defense in depth: the cart-level lock already acquired by
// EnsureAndLockCart before this is called already serializes every writer
// of this cart, including its idempotency claims.
func (t *CartTx) LoadIdempotencyRecord(ctx context.Context, cartID string, keyHash []byte) (CartIdempotencyRecord, bool, error) {
	var record CartIdempotencyRecord
	err := t.tx.QueryRow(
		ctx,
		`SELECT request_hash, expires_at FROM cart_idempotency_keys WHERE cart_id = $1 AND key_hash = $2 FOR UPDATE`,
		cartID, keyHash,
	).Scan(&record.RequestHash, &record.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CartIdempotencyRecord{}, false, nil
	}
	if err != nil {
		return CartIdempotencyRecord{}, false, err
	}
	return record, true, nil
}

// DeleteIdempotencyRecord removes an expired claim, scoped to (cart_id,
// key_hash), so that key can be claimed again as a new operation.
func (t *CartTx) DeleteIdempotencyRecord(ctx context.Context, cartID string, keyHash []byte) error {
	_, err := t.tx.Exec(ctx, `DELETE FROM cart_idempotency_keys WHERE cart_id = $1 AND key_hash = $2`, cartID, keyHash)
	return err
}

// InsertIdempotencyRecord claims a new idempotency key for cartID. It must
// run inside the same transaction as the mutation it guards: if that
// transaction rolls back, the claim never persists and the client may retry
// with the same key as if the first attempt never happened.
func (t *CartTx) InsertIdempotencyRecord(ctx context.Context, cartID string, keyHash []byte, requestHash []byte, createdAt time.Time, expiresAt time.Time) error {
	_, err := t.tx.Exec(
		ctx,
		`INSERT INTO cart_idempotency_keys (cart_id, key_hash, request_hash, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		cartID, keyHash, requestHash, createdAt, expiresAt,
	)
	return err
}
