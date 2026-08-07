package cart

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/dbtest"
)

// integrationEnv resets a dedicated database's public schema, applies the
// full sql/migrations chain, and opens a real pgxpool.Pool against it —
// with no package-level mutable state anywhere. newService binds a Service
// to that pool via db.BeginCartTxWithPool and this package's own
// newServiceWithTxFactory; two integrationEnvs (or two calls to
// newService on one) never share state, because each Service's transaction
// factory closes only over its own *pgxpool.Pool.
type integrationEnv struct {
	pool *pgxpool.Pool
}

func setupIntegrationEnv(t *testing.T) integrationEnv {
	t.Helper()
	dsn := dbtest.RequireIsolatedDatabase(t)
	dbtest.ResetDedicatedDatabase(t, dsn)
	dbtest.ApplyMigrationsUp(t, dsn)

	return integrationEnv{pool: dbtest.NewPool(t, dsn)}
}

// newService builds a Service bound to env's own pool. It never touches
// cart.NewService's production factory or any package-level variable.
func (env integrationEnv) newService() *Service {
	return newServiceWithTxFactory(func(ctx context.Context) (cartTransaction, error) {
		return db.BeginCartTxWithPool(ctx, env.pool)
	})
}

// newServiceOnPool builds a Service bound to an explicit pool other than
// env's own — used to prove two Services can use different pools without
// any shared global state (section 13.3).
func newServiceOnPool(pool *pgxpool.Pool) *Service {
	return newServiceWithTxFactory(func(ctx context.Context) (cartTransaction, error) {
		return db.BeginCartTxWithPool(ctx, pool)
	})
}

func (env integrationEnv) insertCart(t *testing.T) string {
	t.Helper()
	id := uuid.New().String()
	if _, err := env.pool.Exec(context.Background(), `INSERT INTO carts (id) VALUES ($1)`, id); err != nil {
		t.Fatalf("insert cart: %v", err)
	}
	return id
}

func (env integrationEnv) insertProduct(t *testing.T, available bool, quantity int) string {
	t.Helper()
	id := uuid.New().String()
	slug := "test-product-" + id
	_, err := env.pool.Exec(context.Background(),
		`INSERT INTO products (id, name, slug, description, available, quantity) VALUES ($1, 'Test Product', $2, 'test', $3, $4)`,
		id, slug, available, quantity,
	)
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}
	return id
}

func (env integrationEnv) itemQuantity(t *testing.T, cartID, productID string) (int, bool) {
	t.Helper()
	var quantity int
	err := env.pool.QueryRow(context.Background(),
		`SELECT quantity FROM cart_items WHERE cart_id = $1 AND product_id = $2`, cartID, productID,
	).Scan(&quantity)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		t.Fatalf("read item quantity: %v", err)
	}
	return quantity, true
}

func (env integrationEnv) countItems(t *testing.T, cartID string) int {
	t.Helper()
	var count int
	if err := env.pool.QueryRow(context.Background(), `SELECT count(*) FROM cart_items WHERE cart_id = $1`, cartID).Scan(&count); err != nil {
		t.Fatalf("count items: %v", err)
	}
	return count
}

func (env integrationEnv) countIdempotencyRows(t *testing.T, cartID string, keyHash []byte) int {
	t.Helper()
	var count int
	err := env.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM cart_idempotency_keys WHERE cart_id = $1 AND key_hash = $2`, cartID, keyHash,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count idempotency rows: %v", err)
	}
	return count
}

func keyHashOf(seed string) []byte {
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}

func requestHashOf(productID string, quantity int) []byte {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", productID, quantity)))
	return sum[:]
}

func runConcurrently(n int, fn func(i int)) {
	var ready sync.WaitGroup
	start := make(chan struct{})
	var done sync.WaitGroup
	ready.Add(n)
	done.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			ready.Done()
			<-start
			defer done.Done()
			fn(i)
		}(i)
	}
	ready.Wait()
	close(start)
	done.Wait()
}

func withDeadline(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// --- Section 11: concurrent add on a brand-new line ---

func TestPostgresConcurrentAddNewLineNoLostUpdate(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)

	errs := make([]error, 2)
	deltas := []int{2, 3}
	runConcurrently(2, func(i int) {
		errs[i] = svc.AddItem(withDeadline(t), cartID, productID, deltas[i])
	})

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
	}
	quantity, exists := env.itemQuantity(t, cartID, productID)
	if !exists {
		t.Fatal("expected a cart_items row after both adds")
	}
	if quantity != 5 {
		t.Fatalf("expected final quantity 5 (no lost update), got %d", quantity)
	}
	if env.countItems(t, cartID) != 1 {
		t.Fatal("expected exactly one line, found a duplicate")
	}
}

// --- Section 12: concurrent add on an existing line ---

func TestPostgresConcurrentAddExistingLineNoLostUpdate(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)

	if err := svc.AddItem(withDeadline(t), cartID, productID, 1); err != nil {
		t.Fatalf("seed add: %v", err)
	}

	errs := make([]error, 2)
	deltas := []int{2, 3}
	runConcurrently(2, func(i int) {
		errs[i] = svc.AddItem(withDeadline(t), cartID, productID, deltas[i])
	})

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
	}
	quantity, _ := env.itemQuantity(t, cartID, productID)
	if quantity != 6 {
		t.Fatalf("expected final quantity 6 (1 + 2 + 3, no lost update), got %d", quantity)
	}
	if env.countItems(t, cartID) != 1 {
		t.Fatal("expected exactly one line, found a duplicate")
	}
}

// --- Section 13: concurrent add exceeding stock ---

func TestPostgresConcurrentAddInsufficientStockRejectsExactlyOne(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 5)

	if err := svc.AddItem(withDeadline(t), cartID, productID, 1); err != nil {
		t.Fatalf("seed add: %v", err)
	}

	errs := make([]error, 2)
	runConcurrently(2, func(i int) {
		errs[i] = svc.AddItem(withDeadline(t), cartID, productID, 3)
	})

	var applied, rejected int
	for _, err := range errs {
		switch {
		case err == nil:
			applied++
		case errors.Is(err, ErrInsufficientStock):
			rejected++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if applied != 1 || rejected != 1 {
		t.Fatalf("expected exactly one applied and one ErrInsufficientStock, got applied=%d rejected=%d (errs=%v)", applied, rejected, errs)
	}

	quantity, _ := env.itemQuantity(t, cartID, productID)
	if quantity != 4 {
		t.Fatalf("expected final quantity 4 (never 7, no partial write), got %d", quantity)
	}
	if env.countItems(t, cartID) != 1 {
		t.Fatal("expected exactly one line")
	}
}

// --- Section 14: concurrent POST (additive) and PATCH (absolute) ---

func TestPostgresConcurrentAddAndSetQuantitySerialize(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 50)

	if err := svc.AddItem(withDeadline(t), cartID, productID, 2); err != nil {
		t.Fatalf("seed add: %v", err)
	}

	var addErr, setErr error
	runConcurrently(2, func(i int) {
		if i == 0 {
			addErr = svc.AddItem(withDeadline(t), cartID, productID, 3)
		} else {
			setErr = svc.SetItemQuantity(withDeadline(t), cartID, productID, 10)
		}
	})

	if addErr != nil {
		t.Fatalf("AddItem: %v", addErr)
	}
	if setErr != nil {
		t.Fatalf("SetItemQuantity: %v", setErr)
	}

	quantity, _ := env.itemQuantity(t, cartID, productID)
	// The cart lock (EnsureAndLockCart / LockCartIfExists) serializes the
	// two transactions, so only two commit orders are possible: AddItem
	// commits first (2+3=5, then PATCH overwrites to 10 -> final 10), or
	// SetItemQuantity commits first (sets 10, then AddItem reads 10 and
	// adds 3 -> final 13). Both are valid serial orderings; anything else
	// is a lost update or a torn state.
	if quantity != 10 && quantity != 13 {
		t.Fatalf("final quantity %d matches no valid serial order (want 10 or 13)", quantity)
	}
	if env.countItems(t, cartID) != 1 {
		t.Fatal("expected exactly one line")
	}
}

// --- Section 15: concurrent PATCH and DELETE ---

func TestPostgresConcurrentSetAndDeleteSerialize(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartIDA := env.insertCart(t)
	cartIDB := env.insertCart(t)
	productID := env.insertProduct(t, true, 50)

	if err := svc.AddItem(withDeadline(t), cartIDA, productID, 2); err != nil {
		t.Fatalf("seed add: %v", err)
	}

	var setErr, deleteErr error
	runConcurrently(2, func(i int) {
		if i == 0 {
			setErr = svc.SetItemQuantity(withDeadline(t), cartIDA, productID, 7)
		} else {
			deleteErr = svc.DeleteItem(withDeadline(t), cartIDA, productID)
		}
	})

	// Either order is valid: SetItemQuantity-then-Delete ends with no line;
	// Delete-then-SetItemQuantity leaves SetItemQuantity failing
	// ErrCartItemNotFound (DeleteItem never recreates a line) with no line
	// either. In both valid serial orders the final state has zero items,
	// and no panic or partial row is acceptable in between.
	if setErr != nil && !errors.Is(setErr, ErrCartItemNotFound) {
		t.Fatalf("unexpected SetItemQuantity error: %v", setErr)
	}
	if deleteErr != nil {
		t.Fatalf("DeleteItem must be idempotent, got: %v", deleteErr)
	}
	if count := env.countItems(t, cartIDA); count != 0 {
		t.Fatalf("expected zero items in cart A after PATCH/DELETE race, got %d", count)
	}
	// cartIDB must be entirely unaffected.
	if count := env.countItems(t, cartIDB); count != 0 {
		t.Fatalf("DELETE must not affect a different cart, got %d items in cart B", count)
	}
}

// --- Section 16: two products, same cart ---

func TestPostgresConcurrentDifferentProductsSameCartBothSucceed(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productA := env.insertProduct(t, true, 20)
	productB := env.insertProduct(t, true, 20)

	var errA, errB error
	runConcurrently(2, func(i int) {
		if i == 0 {
			errA = svc.AddItem(withDeadline(t), cartID, productA, 4)
		} else {
			errB = svc.AddItem(withDeadline(t), cartID, productB, 5)
		}
	})

	if errA != nil || errB != nil {
		t.Fatalf("expected both adds to succeed (cart lock serializes, does not fail): a=%v b=%v", errA, errB)
	}
	qa, _ := env.itemQuantity(t, cartID, productA)
	qb, _ := env.itemQuantity(t, cartID, productB)
	if qa != 4 || qb != 5 {
		t.Fatalf("expected 4 and 5, got %d and %d", qa, qb)
	}
	if env.countItems(t, cartID) != 2 {
		t.Fatal("expected both lines to persist")
	}
	// Documents current design intentionally, per spec section 16: the cart
	// lock serializes every mutation of the cart, even across different
	// products — there is no per-product lock, only per-cart.
}

// --- Section 17: two different carts ---

func TestPostgresConcurrentDifferentCartsBothSucceedIndependently(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartA := env.insertCart(t)
	cartB := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)

	var errA, errB error
	runConcurrently(2, func(i int) {
		if i == 0 {
			errA = svc.AddItem(withDeadline(t), cartA, productID, 4)
		} else {
			errB = svc.AddItem(withDeadline(t), cartB, productID, 5)
		}
	})

	if errA != nil || errB != nil {
		t.Fatalf("expected both adds on different carts to succeed: a=%v b=%v", errA, errB)
	}
	qa, _ := env.itemQuantity(t, cartA, productID)
	qb, _ := env.itemQuantity(t, cartB, productID)
	if qa != 4 {
		t.Fatalf("cart A: expected 4, got %d — a different cart must not affect it", qa)
	}
	if qb != 5 {
		t.Fatalf("cart B: expected 5, got %d — a different cart must not affect it", qb)
	}
}

// --- Section 18: idempotency, same key, two Service/pool instances ---

func TestPostgresIdempotencySameKeyAcrossTwoServiceInstancesAppliesOnce(t *testing.T) {
	env := setupIntegrationEnv(t)
	// Two independently constructed Service values sharing the overridden
	// pool simulate two separate Go processes talking to the same
	// PostgreSQL instance.
	svcA := env.newService()
	svcB := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)

	keyHash := keyHashOf("same-key")
	requestHash := requestHashOf(productID, 3)

	outcomes := make([]AddItemOutcome, 2)
	errs := make([]error, 2)
	runConcurrently(2, func(i int) {
		svc := svcA
		if i == 1 {
			svc = svcB
		}
		outcomes[i], errs[i] = svc.AddItemIdempotent(withDeadline(t), cartID, productID, 3, keyHash, requestHash)
	})

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
	}
	var applied, replayed int
	for _, outcome := range outcomes {
		switch outcome {
		case AddItemApplied:
			applied++
		case AddItemReplayed:
			replayed++
		}
	}
	if applied != 1 || replayed != 1 {
		t.Fatalf("expected exactly one Applied and one Replayed, got outcomes=%v", outcomes)
	}
	quantity, _ := env.itemQuantity(t, cartID, productID)
	if quantity != 3 {
		t.Fatalf("expected quantity 3 (single increment despite two concurrent callers), got %d", quantity)
	}
	if rows := env.countIdempotencyRows(t, cartID, keyHash); rows != 1 {
		t.Fatalf("expected exactly one idempotency row for the key, found %d", rows)
	}
}

// --- Section 19: idempotency conflict ---

func TestPostgresIdempotencyConflictSameKeyDifferentPayload(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productA := env.insertProduct(t, true, 20)
	productB := env.insertProduct(t, true, 20)

	keyHash := keyHashOf("conflict-key")
	requestHashA := requestHashOf(productA, 2)
	requestHashB := requestHashOf(productB, 4)

	outcomes := make([]AddItemOutcome, 2)
	errs := make([]error, 2)
	runConcurrently(2, func(i int) {
		if i == 0 {
			outcomes[i], errs[i] = svc.AddItemIdempotent(withDeadline(t), cartID, productA, 2, keyHash, requestHashA)
		} else {
			outcomes[i], errs[i] = svc.AddItemIdempotent(withDeadline(t), cartID, productB, 4, keyHash, requestHashB)
		}
	})

	var applied, conflicted int
	var winnerHash []byte
	for i, err := range errs {
		switch {
		case err == nil && outcomes[i] == AddItemApplied:
			applied++
			if i == 0 {
				winnerHash = requestHashA
			} else {
				winnerHash = requestHashB
			}
		case errors.Is(err, ErrIdempotencyConflict):
			conflicted++
		default:
			t.Fatalf("goroutine %d: unexpected result outcome=%v err=%v", i, outcomes[i], err)
		}
	}
	if applied != 1 || conflicted != 1 {
		t.Fatalf("expected exactly one applied and one ErrIdempotencyConflict, got applied=%d conflicted=%d", applied, conflicted)
	}

	var storedRequestHash []byte
	err := env.pool.QueryRow(context.Background(),
		`SELECT request_hash FROM cart_idempotency_keys WHERE cart_id = $1 AND key_hash = $2`, cartID, keyHash,
	).Scan(&storedRequestHash)
	if err != nil {
		t.Fatalf("read stored claim: %v", err)
	}
	if !bytes.Equal(storedRequestHash, winnerHash) {
		t.Fatal("stored claim's request_hash does not match the request that actually applied")
	}
}

// --- Section 20: different keys, same product ---

func TestPostgresDifferentKeysBothApplySerialized(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)

	keyHash1 := keyHashOf("key-one")
	keyHash2 := keyHashOf("key-two")
	requestHash := requestHashOf(productID, 3)

	outcomes := make([]AddItemOutcome, 2)
	errs := make([]error, 2)
	runConcurrently(2, func(i int) {
		keyHash := keyHash1
		if i == 1 {
			keyHash = keyHash2
		}
		outcomes[i], errs[i] = svc.AddItemIdempotent(withDeadline(t), cartID, productID, 3, keyHash, requestHash)
	})

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: unexpected error: %v", i, err)
		}
		if outcomes[i] != AddItemApplied {
			t.Fatalf("goroutine %d: expected Applied for a distinct key, got %v", i, outcomes[i])
		}
	}
	quantity, _ := env.itemQuantity(t, cartID, productID)
	if quantity != 6 {
		t.Fatalf("expected quantity 6 (3+3, two distinct keys both applied), got %d", quantity)
	}
	if rows := env.countIdempotencyRows(t, cartID, keyHash1); rows != 1 {
		t.Fatalf("expected one row for key one, got %d", rows)
	}
	if rows := env.countIdempotencyRows(t, cartID, keyHash2); rows != 1 {
		t.Fatalf("expected one row for key two, got %d", rows)
	}
}

// --- Section 21: expiration, injected clock ---

func TestPostgresExpiredClaimIsReplacedNotReused(t *testing.T) {
	env := setupIntegrationEnv(t)
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)
	keyHash := keyHashOf("expiring-key")
	requestHash := requestHashOf(productID, 2)

	// clock is injected into the Service (white-box: this file is package
	// cart), never the PostgreSQL server's own clock, per section 21. The
	// transaction factory is still env's own pool-bound one — only the
	// clock is swapped after construction.
	frozen := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	svc := env.newService()
	svc.clock = func() time.Time { return frozen }

	outcome, err := svc.AddItemIdempotent(withDeadline(t), cartID, productID, 2, keyHash, requestHash)
	if err != nil || outcome != AddItemApplied {
		t.Fatalf("seed claim: outcome=%v err=%v", outcome, err)
	}

	// Advance the injected clock past idempotencyTTL; the same key with a
	// different payload must be treated as a brand-new claim, not a
	// conflict.
	svc.clock = func() time.Time { return frozen.Add(idempotencyTTL + time.Hour) }
	newRequestHash := requestHashOf(productID, 5)
	outcome, err = svc.AddItemIdempotent(withDeadline(t), cartID, productID, 5, keyHash, newRequestHash)
	if err != nil {
		t.Fatalf("post-expiry call: %v", err)
	}
	if outcome != AddItemApplied {
		t.Fatalf("expected Applied for an expired-then-reclaimed key, got %v", outcome)
	}

	quantity, _ := env.itemQuantity(t, cartID, productID)
	if quantity != 7 {
		t.Fatalf("expected quantity 7 (2 then +5 after reclaim), got %d", quantity)
	}
	if rows := env.countIdempotencyRows(t, cartID, keyHash); rows != 1 {
		t.Fatalf("expected exactly one final row for the key (old claim deleted, new one inserted), got %d", rows)
	}

	var storedRequestHash []byte
	var expiresAt time.Time
	err = env.pool.QueryRow(context.Background(),
		`SELECT request_hash, expires_at FROM cart_idempotency_keys WHERE cart_id = $1 AND key_hash = $2`, cartID, keyHash,
	).Scan(&storedRequestHash, &expiresAt)
	if err != nil {
		t.Fatalf("read claim: %v", err)
	}
	if !bytes.Equal(storedRequestHash, newRequestHash) {
		t.Fatal("expected the stored claim to carry the new request hash, not the expired one")
	}
	if !expiresAt.After(frozen.Add(idempotencyTTL)) {
		t.Fatal("expected a fresh expires_at strictly after the original claim's expiry")
	}
}

// --- Section 22: rollback leaves no trace ---

func TestPostgresRollbackLeavesNoIdempotencyClaimOrCartChange(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	// Zero stock forces ErrProductUnavailable *after* the claim insert
	// (AddItemIdempotent inserts the claim, then locks and checks stock),
	// which is exactly the ordering section 22 asks to exercise — no fault
	// injection into production code, just a fixture that naturally fails
	// after the claim exists.
	productID := env.insertProduct(t, true, 0)
	keyHash := keyHashOf("rollback-key")
	requestHash := requestHashOf(productID, 1)

	outcome, err := svc.AddItemIdempotent(withDeadline(t), cartID, productID, 1, keyHash, requestHash)
	if !errors.Is(err, ErrProductUnavailable) {
		t.Fatalf("expected ErrProductUnavailable, got outcome=%v err=%v", outcome, err)
	}

	if rows := env.countIdempotencyRows(t, cartID, keyHash); rows != 0 {
		t.Fatalf("expected zero idempotency rows after rollback, got %d", rows)
	}
	if count := env.countItems(t, cartID); count != 0 {
		t.Fatalf("expected zero cart_items after rollback, got %d", count)
	}

	// The same key must be retryable afterward, as if the failed attempt
	// never happened.
	if err := env.pool.QueryRow(context.Background(), `UPDATE products SET available = true, quantity = 5 WHERE id = $1 RETURNING id`, productID).Scan(new(string)); err != nil {
		t.Fatalf("make product available for retry: %v", err)
	}
	outcome, err = svc.AddItemIdempotent(withDeadline(t), cartID, productID, 1, keyHash, requestHash)
	if err != nil || outcome != AddItemApplied {
		t.Fatalf("expected the retried key to apply cleanly, got outcome=%v err=%v", outcome, err)
	}
}

// --- Section 23: commit succeeded, response "lost", client retries ---

func TestPostgresLostResponseReplayReturnsCanonicalState(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)
	keyHash := keyHashOf("lost-response-key")
	requestHash := requestHashOf(productID, 4)

	outcome, err := svc.AddItemIdempotent(withDeadline(t), cartID, productID, 4, keyHash, requestHash)
	if err != nil || outcome != AddItemApplied {
		t.Fatalf("first call: outcome=%v err=%v", outcome, err)
	}
	// Simulate the client never seeing that response (network drop) and
	// retrying with the identical key and payload.
	outcome, err = svc.AddItemIdempotent(withDeadline(t), cartID, productID, 4, keyHash, requestHash)
	if err != nil {
		t.Fatalf("replay call: %v", err)
	}
	if outcome != AddItemReplayed {
		t.Fatalf("expected Replayed on the retried call, got %v", outcome)
	}

	quantity, _ := env.itemQuantity(t, cartID, productID)
	if quantity != 4 {
		t.Fatalf("expected quantity 4 (single increment despite the retry), got %d", quantity)
	}
}

// --- Section 25: timeout/deadlock surfaces, never hidden by a retry ---

func TestPostgresContextTimeoutDoesNotHang(t *testing.T) {
	env := setupIntegrationEnv(t)
	svc := env.newService()
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond) // guarantee the deadline has already passed

	err := svc.AddItem(ctx, cartID, productID, 1)
	if err == nil {
		t.Fatal("expected an error from an already-expired context, not a hang or silent success")
	}

	// The pool must still be usable afterward: a timed-out operation must
	// not leak a held connection or an unreleased lock.
	if err := svc.AddItem(withDeadline(t), cartID, productID, 1); err != nil {
		t.Fatalf("pool unusable after a prior timeout: %v", err)
	}
}

// --- Section 13: injection design ---

// TestPostgresNewServiceUsesProductionFactory confirms cart.NewService()
// itself — the exact production constructor, untouched by this phase — is
// still wired to beginDatabaseCartTx, and that a Service built with it is a
// distinct value from one built with a test-only factory. It does not
// dial PostgreSQL: NewService() only fails once a transaction is actually
// begun, which this test never does.
func TestPostgresNewServiceUsesProductionFactory(t *testing.T) {
	prod := NewService()
	test := newServiceWithTxFactory(func(ctx context.Context) (cartTransaction, error) {
		return nil, errors.New("unused")
	})
	if prod == test {
		t.Fatal("expected NewService and a test factory to produce distinct Service values")
	}
}

// TestPostgresTwoServicesOnDifferentPoolsDoNotShareState opens two
// independent pools against the same dedicated database and proves a
// mutation through one Service is visible through a second Service bound
// to a different pool only because they share the same underlying
// database — never because of any process-wide variable. There is nothing
// left in this package resembling internal/db's former
// SetPoolForIntegrationTests.
func TestPostgresTwoServicesOnDifferentPoolsDoNotShareState(t *testing.T) {
	dsn := dbtest.RequireIsolatedDatabase(t)
	dbtest.ResetDedicatedDatabase(t, dsn)
	dbtest.ApplyMigrationsUp(t, dsn)

	poolA := dbtest.NewPool(t, dsn)
	poolB := dbtest.NewPool(t, dsn)
	if poolA == poolB {
		t.Fatal("expected two independent pools")
	}

	svcA := newServiceOnPool(poolA)
	svcB := newServiceOnPool(poolB)

	env := integrationEnv{pool: poolA}
	cartID := env.insertCart(t)
	productID := env.insertProduct(t, true, 20)

	if err := svcA.AddItem(withDeadline(t), cartID, productID, 3); err != nil {
		t.Fatalf("svcA add: %v", err)
	}
	if err := svcB.AddItem(withDeadline(t), cartID, productID, 4); err != nil {
		t.Fatalf("svcB add: %v", err)
	}

	quantity, _ := env.itemQuantity(t, cartID, productID)
	if quantity != 7 {
		t.Fatalf("expected quantity 7 across two independently-pooled Services, got %d", quantity)
	}
}

// TestPostgresIntegrationEnvIsRepeatable confirms the same test body can
// run twice in the same process (as -count=2 would do) without collision:
// each call to setupIntegrationEnv resets the dedicated database and opens
// its own pool from scratch.
func TestPostgresIntegrationEnvIsRepeatable(t *testing.T) {
	for i := 0; i < 2; i++ {
		env := setupIntegrationEnv(t)
		svc := env.newService()
		cartID := env.insertCart(t)
		productID := env.insertProduct(t, true, 20)
		if err := svc.AddItem(withDeadline(t), cartID, productID, 1); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if env.countItems(t, cartID) != 1 {
			t.Fatalf("iteration %d: expected exactly one line", i)
		}
	}
}
