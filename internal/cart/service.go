// Package cart provides the single, transaction-scoped implementation of
// cart mutations (add, set quantity, remove, clear). It exists to close the
// lost-update and overselling race documented in 05-cart-integration.md as
// CART-07/CART-11: previously a product was validated and the cart saved as
// separate database round trips on separate connections, none of them
// serialized against a concurrent mutation of the same cart.
//
// Every write here happens inside one transaction that locks the cart row
// first (serializing all mutations of that cart) and the product row second
// (reading stock consistently). Adding to the cart never reserves or
// decrements stock: it only guarantees the cart's own quantity never exceeds
// the stock observed during that same transaction.
//
// This package does not implement idempotency keys. A client that retries
// the same POST after a timeout can still add twice; that risk is documented
// and deferred to a later phase, not solved here.
package cart

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

// idempotencyTTL is the fixed lifetime of a claimed idempotency key. A
// replay never extends it; an expired claim is deleted and its key can be
// reused as a brand new operation.
const idempotencyTTL = 24 * time.Hour

// AddItemOutcome reports whether AddItemIdempotent actually mutated the cart
// or matched an existing, still-valid claim for the same key and payload.
type AddItemOutcome int

const (
	// AddItemApplied means the mutation ran: either a new claim was created
	// and the line written, or the caller supplied a fresh idempotency key.
	AddItemApplied AddItemOutcome = iota
	// AddItemReplayed means an unexpired claim already matched this exact
	// key and payload; AddItem was not executed again.
	AddItemReplayed
)

var (
	// ErrProductNotFound means the product row does not exist.
	ErrProductNotFound = errors.New("cart: product not found")
	// ErrProductUnavailable means the product exists but is marked
	// unavailable or has zero stock.
	ErrProductUnavailable = errors.New("cart: product unavailable")
	// ErrInsufficientStock means the requested final quantity would exceed
	// the stock observed inside the transaction.
	ErrInsufficientStock = errors.New("cart: insufficient stock")
	// ErrCartItemNotFound means the cart has no line for that product.
	ErrCartItemNotFound = errors.New("cart: item not found")
	// ErrCartUnavailable is the catch-all for any database failure. It never
	// wraps the underlying error so callers cannot leak driver details.
	ErrCartUnavailable = errors.New("cart: unavailable")
	// ErrIdempotencyConflict means the same idempotency key was already
	// claimed by this cart for a different product or quantity.
	ErrIdempotencyConflict = errors.New("cart: idempotency key reused with a different request")
)

// cartTransaction is the minimal transaction contract the service needs. It
// is satisfied by *db.CartTx in production and by a fake in tests, so the
// concurrency-critical ordering of locks and reads can be unit tested
// without PostgreSQL.
type cartTransaction interface {
	EnsureAndLockCart(ctx context.Context, cartID string) error
	LockCartIfExists(ctx context.Context, cartID string) (bool, error)
	LockProductStock(ctx context.Context, productID string) (db.CartProductStock, error)
	ItemQuantity(ctx context.Context, cartID string, productID string) (int, bool, error)
	UpsertItem(ctx context.Context, cartID string, productID string, source string, quantity int) error
	UpdateItemQuantity(ctx context.Context, cartID string, productID string, quantity int) error
	DeleteItem(ctx context.Context, cartID string, productID string) error
	DeleteAllItems(ctx context.Context, cartID string) error
	LoadIdempotencyRecord(ctx context.Context, cartID string, keyHash []byte) (db.CartIdempotencyRecord, bool, error)
	DeleteIdempotencyRecord(ctx context.Context, cartID string, keyHash []byte) error
	InsertIdempotencyRecord(ctx context.Context, cartID string, keyHash []byte, requestHash []byte, createdAt time.Time, expiresAt time.Time) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context)
	Close()
}

type transactionFactory func(context.Context) (cartTransaction, error)

// Service implements the atomic cart mutations. The zero value is not
// usable; construct it with NewService.
type Service struct {
	beginTx transactionFactory
	clock   func() time.Time
}

// NewService builds a Service backed by real PostgreSQL transactions on the
// package's own production connection pool. This is the only constructor
// production code calls.
func NewService() *Service {
	return newServiceWithTxFactory(beginDatabaseCartTx)
}

// newServiceWithTxFactory builds a Service around an explicit transaction
// factory. It is unexported: only code inside this package — production's
// NewService above, and this package's own _test.go files — can supply a
// factory. A same-package integration test can bind one to a throwaway
// *pgxpool.Pool via db.BeginCartTxWithPool without any package-level
// mutable state, and two Services built this way never share state.
func newServiceWithTxFactory(factory transactionFactory) *Service {
	return &Service{beginTx: factory, clock: time.Now}
}

func beginDatabaseCartTx(ctx context.Context) (cartTransaction, error) {
	return db.BeginCartTx(ctx)
}

// AddItem adds quantity units of productID to cartID, on top of whatever
// quantity is already in the cart for that product. The final quantity must
// not exceed the stock observed inside the transaction. The source of the
// written line is always fixed to the catalog: this operation is never
// reached from the wizard.
func (s *Service) AddItem(ctx context.Context, cartID string, productID string, quantity int) error {
	txn, err := s.beginTx(ctx)
	if err != nil {
		return ErrCartUnavailable
	}
	defer txn.Close()

	if err := txn.EnsureAndLockCart(ctx, cartID); err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}

	stock, err := txn.LockProductStock(ctx, productID)
	if err != nil {
		txn.Rollback(ctx)
		return translateProductLoadError(err)
	}
	if !stock.Available || stock.Quantity <= 0 {
		txn.Rollback(ctx)
		return ErrProductUnavailable
	}

	currentQuantity, _, err := txn.ItemQuantity(ctx, cartID, productID)
	if err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}

	finalQuantity := currentQuantity + quantity
	if finalQuantity > stock.Quantity {
		txn.Rollback(ctx)
		return ErrInsufficientStock
	}

	if err := txn.UpsertItem(ctx, cartID, productID, string(db.CartItemSourceCatalog), finalQuantity); err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}

	if err := txn.Commit(ctx); err != nil {
		return ErrCartUnavailable
	}
	return nil
}

// AddItemIdempotent behaves like AddItem, but a repeated call carrying the
// same keyHash and requestHash within idempotencyTTL is a replay: it does
// not run AddItem's logic again and reports AddItemReplayed instead of
// mutating the cart a second time. The same keyHash reused with a different
// requestHash (a different product or quantity) never mutates the cart and
// returns ErrIdempotencyConflict.
//
// The claim and the mutation it guards always share one transaction: if the
// mutation fails for any reason, the whole transaction rolls back and the
// claim never persists, so the client can retry with the same key as if the
// first attempt never happened. Only a genuinely new or expired-then-cleared
// claim reaches the stock validation and write below; a valid replay never
// does.
func (s *Service) AddItemIdempotent(
	ctx context.Context,
	cartID string,
	productID string,
	quantity int,
	keyHash []byte,
	requestHash []byte,
) (AddItemOutcome, error) {
	txn, err := s.beginTx(ctx)
	if err != nil {
		return AddItemApplied, ErrCartUnavailable
	}
	defer txn.Close()

	if err := txn.EnsureAndLockCart(ctx, cartID); err != nil {
		txn.Rollback(ctx)
		return AddItemApplied, ErrCartUnavailable
	}

	now := s.clock().UTC()

	record, exists, err := txn.LoadIdempotencyRecord(ctx, cartID, keyHash)
	if err != nil {
		txn.Rollback(ctx)
		return AddItemApplied, ErrCartUnavailable
	}

	if exists {
		if now.Before(record.ExpiresAt) {
			if !bytes.Equal(record.RequestHash, requestHash) {
				txn.Rollback(ctx)
				return AddItemApplied, ErrIdempotencyConflict
			}

			// Same key, same payload, still valid: this is a replay. Do not
			// run AddItem again; just close out the transaction that holds
			// the locks acquired above.
			if err := txn.Commit(ctx); err != nil {
				return AddItemApplied, ErrCartUnavailable
			}
			return AddItemReplayed, nil
		}

		// The claim expired. Delete it so the key can be claimed again as a
		// new operation, then fall through to claim and apply below.
		if err := txn.DeleteIdempotencyRecord(ctx, cartID, keyHash); err != nil {
			txn.Rollback(ctx)
			return AddItemApplied, ErrCartUnavailable
		}
	}

	expiresAt := now.Add(idempotencyTTL)
	if err := txn.InsertIdempotencyRecord(ctx, cartID, keyHash, requestHash, now, expiresAt); err != nil {
		txn.Rollback(ctx)
		return AddItemApplied, ErrCartUnavailable
	}

	stock, err := txn.LockProductStock(ctx, productID)
	if err != nil {
		txn.Rollback(ctx)
		return AddItemApplied, translateProductLoadError(err)
	}
	if !stock.Available || stock.Quantity <= 0 {
		txn.Rollback(ctx)
		return AddItemApplied, ErrProductUnavailable
	}

	currentQuantity, _, err := txn.ItemQuantity(ctx, cartID, productID)
	if err != nil {
		txn.Rollback(ctx)
		return AddItemApplied, ErrCartUnavailable
	}

	finalQuantity := currentQuantity + quantity
	if finalQuantity > stock.Quantity {
		txn.Rollback(ctx)
		return AddItemApplied, ErrInsufficientStock
	}

	if err := txn.UpsertItem(ctx, cartID, productID, string(db.CartItemSourceCatalog), finalQuantity); err != nil {
		txn.Rollback(ctx)
		return AddItemApplied, ErrCartUnavailable
	}

	if err := txn.Commit(ctx); err != nil {
		return AddItemApplied, ErrCartUnavailable
	}
	return AddItemApplied, nil
}

// SetItemQuantity fixes the absolute quantity of an existing cart line. It
// never creates a line: a missing line is ErrCartItemNotFound. The line's
// source is left untouched.
//
// Two concurrent PATCH requests for the same line are serialized by the cart
// lock and apply in last-write-wins order once serialized: the transaction
// that commits last determines the final quantity. Neither request can
// create a duplicate line or leave a partial write, because both operate on
// the same (cart_id, product_id) primary key inside a single UPDATE.
func (s *Service) SetItemQuantity(ctx context.Context, cartID string, productID string, quantity int) error {
	txn, err := s.beginTx(ctx)
	if err != nil {
		return ErrCartUnavailable
	}
	defer txn.Close()

	exists, err := txn.LockCartIfExists(ctx, cartID)
	if err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}
	if !exists {
		txn.Rollback(ctx)
		return ErrCartItemNotFound
	}

	_, itemExists, err := txn.ItemQuantity(ctx, cartID, productID)
	if err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}
	if !itemExists {
		txn.Rollback(ctx)
		return ErrCartItemNotFound
	}

	stock, err := txn.LockProductStock(ctx, productID)
	if err != nil {
		txn.Rollback(ctx)
		if errors.Is(err, db.ErrCartTxProductNotFound) {
			// The product row vanished while a line for it still exists in
			// the cart. products.id cascades into cart_items on delete, so
			// this should not happen in practice; treat it the same as any
			// other product that can no longer be sold rather than invent a
			// new error code.
			return ErrProductUnavailable
		}
		return ErrCartUnavailable
	}
	if !stock.Available || stock.Quantity <= 0 {
		txn.Rollback(ctx)
		return ErrProductUnavailable
	}
	if quantity > stock.Quantity {
		txn.Rollback(ctx)
		return ErrInsufficientStock
	}

	if err := txn.UpdateItemQuantity(ctx, cartID, productID, quantity); err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}

	if err := txn.Commit(ctx); err != nil {
		return ErrCartUnavailable
	}
	return nil
}

// DeleteItem removes a single line from a cart. It is idempotent: deleting a
// line that does not exist, or a cart that does not exist, succeeds without
// error.
func (s *Service) DeleteItem(ctx context.Context, cartID string, productID string) error {
	txn, err := s.beginTx(ctx)
	if err != nil {
		return ErrCartUnavailable
	}
	defer txn.Close()

	exists, err := txn.LockCartIfExists(ctx, cartID)
	if err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}
	if !exists {
		txn.Rollback(ctx)
		return nil
	}

	if err := txn.DeleteItem(ctx, cartID, productID); err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}

	if err := txn.Commit(ctx); err != nil {
		return ErrCartUnavailable
	}
	return nil
}

// Clear removes every line from a cart. It is idempotent and never deletes
// the carts row itself.
func (s *Service) Clear(ctx context.Context, cartID string) error {
	txn, err := s.beginTx(ctx)
	if err != nil {
		return ErrCartUnavailable
	}
	defer txn.Close()

	exists, err := txn.LockCartIfExists(ctx, cartID)
	if err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}
	if !exists {
		txn.Rollback(ctx)
		return nil
	}

	if err := txn.DeleteAllItems(ctx, cartID); err != nil {
		txn.Rollback(ctx)
		return ErrCartUnavailable
	}

	if err := txn.Commit(ctx); err != nil {
		return ErrCartUnavailable
	}
	return nil
}

func translateProductLoadError(err error) error {
	if errors.Is(err, db.ErrCartTxProductNotFound) {
		return ErrProductNotFound
	}
	return ErrCartUnavailable
}
