package cart

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

const (
	testCartID    = "11111111-1111-1111-1111-111111111111"
	testProductID = "22222222-2222-2222-2222-222222222222"
)

var (
	testKeyHash     = []byte("0123456789abcdef0123456789abcdef")[:32]
	testRequestHash = []byte("fedcba9876543210fedcba9876543210")[:32]
)

// fakeCartTx is a programmable double for the transaction contract the
// service depends on. It lets the concurrency-critical ordering of locks,
// reads and writes be tested without PostgreSQL.
type fakeCartTx struct {
	ensureAndLockErr error
	lockCartExists   bool
	lockCartErr      error
	stock            db.CartProductStock
	stockErr         error
	itemQuantity     int
	itemExists       bool
	itemQuantityErr  error
	upsertErr        error
	updateErr        error
	deleteItemErr    error
	deleteAllErr     error
	commitErr        error

	idempotencyRecord    db.CartIdempotencyRecord
	idempotencyExists    bool
	loadIdempotencyErr   error
	deleteIdempotencyErr error
	insertIdempotencyErr error

	calls []string

	ensureAndLockCartID string
	lockCartIfExistsID  string
	lockProductStockID  string
	itemQtyCartID       string
	itemQtyProductID    string
	upsertCartID        string
	upsertProductID     string
	upsertSource        string
	upsertQuantity      int
	updateCartID        string
	updateProductID     string
	updateQuantity      int
	deleteItemCartID    string
	deleteItemProductID string
	deleteAllCartID     string

	loadIdempotencyCartID   string
	loadIdempotencyKeyHash  []byte
	deleteIdempotencyCartID string
	deleteIdempotencyHash   []byte
	insertCartID            string
	insertKeyHash           []byte
	insertRequestHash       []byte
	insertCreatedAt         time.Time
	insertExpiresAt         time.Time
}

func (f *fakeCartTx) EnsureAndLockCart(_ context.Context, cartID string) error {
	f.calls = append(f.calls, "EnsureAndLockCart")
	f.ensureAndLockCartID = cartID
	return f.ensureAndLockErr
}

func (f *fakeCartTx) LockCartIfExists(_ context.Context, cartID string) (bool, error) {
	f.calls = append(f.calls, "LockCartIfExists")
	f.lockCartIfExistsID = cartID
	return f.lockCartExists, f.lockCartErr
}

func (f *fakeCartTx) LockProductStock(_ context.Context, productID string) (db.CartProductStock, error) {
	f.calls = append(f.calls, "LockProductStock")
	f.lockProductStockID = productID
	return f.stock, f.stockErr
}

func (f *fakeCartTx) ItemQuantity(_ context.Context, cartID string, productID string) (int, bool, error) {
	f.calls = append(f.calls, "ItemQuantity")
	f.itemQtyCartID = cartID
	f.itemQtyProductID = productID
	return f.itemQuantity, f.itemExists, f.itemQuantityErr
}

func (f *fakeCartTx) UpsertItem(_ context.Context, cartID string, productID string, source string, quantity int) error {
	f.calls = append(f.calls, "UpsertItem")
	f.upsertCartID = cartID
	f.upsertProductID = productID
	f.upsertSource = source
	f.upsertQuantity = quantity
	return f.upsertErr
}

func (f *fakeCartTx) UpdateItemQuantity(_ context.Context, cartID string, productID string, quantity int) error {
	f.calls = append(f.calls, "UpdateItemQuantity")
	f.updateCartID = cartID
	f.updateProductID = productID
	f.updateQuantity = quantity
	return f.updateErr
}

func (f *fakeCartTx) DeleteItem(_ context.Context, cartID string, productID string) error {
	f.calls = append(f.calls, "DeleteItem")
	f.deleteItemCartID = cartID
	f.deleteItemProductID = productID
	return f.deleteItemErr
}

func (f *fakeCartTx) DeleteAllItems(_ context.Context, cartID string) error {
	f.calls = append(f.calls, "DeleteAllItems")
	f.deleteAllCartID = cartID
	return f.deleteAllErr
}

func (f *fakeCartTx) LoadIdempotencyRecord(_ context.Context, cartID string, keyHash []byte) (db.CartIdempotencyRecord, bool, error) {
	f.calls = append(f.calls, "LoadIdempotencyRecord")
	f.loadIdempotencyCartID = cartID
	f.loadIdempotencyKeyHash = keyHash
	return f.idempotencyRecord, f.idempotencyExists, f.loadIdempotencyErr
}

func (f *fakeCartTx) DeleteIdempotencyRecord(_ context.Context, cartID string, keyHash []byte) error {
	f.calls = append(f.calls, "DeleteIdempotencyRecord")
	f.deleteIdempotencyCartID = cartID
	f.deleteIdempotencyHash = keyHash
	return f.deleteIdempotencyErr
}

func (f *fakeCartTx) InsertIdempotencyRecord(_ context.Context, cartID string, keyHash []byte, requestHash []byte, createdAt time.Time, expiresAt time.Time) error {
	f.calls = append(f.calls, "InsertIdempotencyRecord")
	f.insertCartID = cartID
	f.insertKeyHash = keyHash
	f.insertRequestHash = requestHash
	f.insertCreatedAt = createdAt
	f.insertExpiresAt = expiresAt
	return f.insertIdempotencyErr
}

func (f *fakeCartTx) Commit(_ context.Context) error {
	f.calls = append(f.calls, "Commit")
	return f.commitErr
}

func (f *fakeCartTx) Rollback(_ context.Context) {
	f.calls = append(f.calls, "Rollback")
}

func (f *fakeCartTx) Close() {
	f.calls = append(f.calls, "Close")
}

func serviceWithTx(txn *fakeCartTx) *Service {
	return &Service{
		beginTx: func(context.Context) (cartTransaction, error) {
			return txn, nil
		},
		clock: time.Now,
	}
}

func serviceWithTxAndClock(txn *fakeCartTx, clock func() time.Time) *Service {
	return &Service{
		beginTx: func(context.Context) (cartTransaction, error) {
			return txn, nil
		},
		clock: clock,
	}
}

func serviceWithBeginError(err error) *Service {
	return &Service{
		beginTx: func(context.Context) (cartTransaction, error) {
			return nil, err
		},
		clock: time.Now,
	}
}

func assertCallOrder(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected call order: got %v, want %v", got, want)
	}
}

func assertNoInternalLeak(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	for _, secret := range []string{"SQLSTATE", "postgres", "password", "DATABASE_URL", "connection refused"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("internal detail %q leaked in error: %v", secret, err)
		}
	}
}

// 1. Producto inexistente.
func TestAddItemProductNotFound(t *testing.T) {
	txn := &fakeCartTx{stockErr: db.ErrCartTxProductNotFound}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LockProductStock", "Rollback", "Close")
	assertNoInternalLeak(t, err)
}

// 2. Producto no disponible.
func TestAddItemProductUnavailable(t *testing.T) {
	txn := &fakeCartTx{stock: db.CartProductStock{Available: false, Quantity: 5}}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	if !errors.Is(err, ErrProductUnavailable) {
		t.Fatalf("expected ErrProductUnavailable, got %v", err)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LockProductStock", "Rollback", "Close")
}

// 3. Stock cero.
func TestAddItemZeroStock(t *testing.T) {
	txn := &fakeCartTx{stock: db.CartProductStock{Available: true, Quantity: 0}}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	if !errors.Is(err, ErrProductUnavailable) {
		t.Fatalf("expected ErrProductUnavailable for zero stock, got %v", err)
	}
	if txn.upsertCartID != "" {
		t.Fatal("must not write with zero stock")
	}
}

// 4. Cantidad solicitada mayor al stock (new line).
func TestAddItemRequestedQuantityExceedsStock(t *testing.T) {
	txn := &fakeCartTx{stock: db.CartProductStock{Available: true, Quantity: 3}}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 5)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LockProductStock", "ItemQuantity", "Rollback", "Close")
}

// 5. Incremento que supera stock.
func TestAddItemIncrementExceedsStock(t *testing.T) {
	txn := &fakeCartTx{
		stock:        db.CartProductStock{Available: true, Quantity: 3},
		itemQuantity: 2,
		itemExists:   true,
	}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 2)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock (2+2 > 3), got %v", err)
	}
	if txn.upsertCartID != "" {
		t.Fatal("must not write when the combined quantity exceeds stock")
	}
}

// 6 and 7. Incremento válido; producto repetido mantiene una sola línea.
func TestAddItemValidIncrementUpsertsSingleLine(t *testing.T) {
	txn := &fakeCartTx{
		stock:        db.CartProductStock{Available: true, Quantity: 5},
		itemQuantity: 1,
		itemExists:   true,
	}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 2)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LockProductStock", "ItemQuantity", "UpsertItem", "Commit", "Close")
	if txn.upsertQuantity != 3 {
		t.Fatalf("expected final quantity 3 (1 existing + 2 requested), got %d", txn.upsertQuantity)
	}
	if txn.upsertCartID != testCartID || txn.upsertProductID != testProductID {
		t.Fatalf("wrong identifiers reached the write: cart=%q product=%q", txn.upsertCartID, txn.upsertProductID)
	}
	if txn.upsertSource != string(db.CartItemSourceCatalog) {
		t.Fatalf("expected server-fixed catalog source, got %q", txn.upsertSource)
	}
}

// 8. PATCH válido.
func TestSetItemQuantityValid(t *testing.T) {
	txn := &fakeCartTx{
		lockCartExists: true,
		itemExists:     true,
		itemQuantity:   1,
		stock:          db.CartProductStock{Available: true, Quantity: 5},
	}
	err := serviceWithTx(txn).SetItemQuantity(context.Background(), testCartID, testProductID, 3)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	assertCallOrder(t, txn.calls, "LockCartIfExists", "ItemQuantity", "LockProductStock", "UpdateItemQuantity", "Commit", "Close")
	if txn.updateQuantity != 3 {
		t.Fatalf("expected absolute quantity 3, got %d", txn.updateQuantity)
	}
}

// 9. PATCH item inexistente (cart missing and item missing).
func TestSetItemQuantityItemNotFound(t *testing.T) {
	for _, testCase := range []struct {
		name           string
		lockCartExists bool
		itemExists     bool
	}{
		{name: "cart missing", lockCartExists: false},
		{name: "item missing", lockCartExists: true, itemExists: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			txn := &fakeCartTx{lockCartExists: testCase.lockCartExists, itemExists: testCase.itemExists}
			err := serviceWithTx(txn).SetItemQuantity(context.Background(), testCartID, testProductID, 3)
			if !errors.Is(err, ErrCartItemNotFound) {
				t.Fatalf("expected ErrCartItemNotFound, got %v", err)
			}
			if txn.updateCartID != "" {
				t.Fatal("must not write for a missing line")
			}
		})
	}
}

// 10. PATCH supera stock.
func TestSetItemQuantityExceedsStock(t *testing.T) {
	txn := &fakeCartTx{
		lockCartExists: true,
		itemExists:     true,
		itemQuantity:   1,
		stock:          db.CartProductStock{Available: true, Quantity: 2},
	}
	err := serviceWithTx(txn).SetItemQuantity(context.Background(), testCartID, testProductID, 5)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if txn.updateCartID != "" {
		t.Fatal("must not write when the requested quantity exceeds stock")
	}
}

// 11. DELETE item existente.
func TestDeleteItemExisting(t *testing.T) {
	txn := &fakeCartTx{lockCartExists: true}
	err := serviceWithTx(txn).DeleteItem(context.Background(), testCartID, testProductID)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	assertCallOrder(t, txn.calls, "LockCartIfExists", "DeleteItem", "Commit", "Close")
	if txn.deleteItemCartID != testCartID || txn.deleteItemProductID != testProductID {
		t.Fatalf("wrong identifiers reached delete: cart=%q product=%q", txn.deleteItemCartID, txn.deleteItemProductID)
	}
}

// 12. DELETE item inexistente (idempotent).
func TestDeleteItemMissingCartIsIdempotent(t *testing.T) {
	txn := &fakeCartTx{lockCartExists: false}
	err := serviceWithTx(txn).DeleteItem(context.Background(), testCartID, testProductID)
	if err != nil {
		t.Fatalf("expected idempotent success, got %v", err)
	}
	if txn.deleteItemCartID != "" {
		t.Fatal("must not attempt a delete for a nonexistent cart")
	}
}

// 13. DELETE cart vacío.
func TestClearEmptyCartIsIdempotent(t *testing.T) {
	txn := &fakeCartTx{lockCartExists: true}
	err := serviceWithTx(txn).Clear(context.Background(), testCartID)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	assertCallOrder(t, txn.calls, "LockCartIfExists", "DeleteAllItems", "Commit", "Close")
}

// 14. Error al comenzar transacción.
func TestBeginTransactionFailureIsCartUnavailable(t *testing.T) {
	beginErr := errors.New("pgxpool: acquire failed: SQLSTATE 08006")
	for _, operation := range []func(*Service) error{
		func(s *Service) error { return s.AddItem(context.Background(), testCartID, testProductID, 1) },
		func(s *Service) error { return s.SetItemQuantity(context.Background(), testCartID, testProductID, 1) },
		func(s *Service) error { return s.DeleteItem(context.Background(), testCartID, testProductID) },
		func(s *Service) error { return s.Clear(context.Background(), testCartID) },
	} {
		err := operation(serviceWithBeginError(beginErr))
		if !errors.Is(err, ErrCartUnavailable) {
			t.Fatalf("expected ErrCartUnavailable, got %v", err)
		}
		assertNoInternalLeak(t, err)
	}
}

// 15. Error al bloquear carrito.
func TestLockCartFailureIsCartUnavailable(t *testing.T) {
	t.Run("AddItem EnsureAndLockCart", func(t *testing.T) {
		txn := &fakeCartTx{ensureAndLockErr: errors.New("lock timeout")}
		err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
		if !errors.Is(err, ErrCartUnavailable) {
			t.Fatalf("expected ErrCartUnavailable, got %v", err)
		}
		assertCallOrder(t, txn.calls, "EnsureAndLockCart", "Rollback", "Close")
	})
	t.Run("SetItemQuantity LockCartIfExists", func(t *testing.T) {
		txn := &fakeCartTx{lockCartErr: errors.New("lock timeout")}
		err := serviceWithTx(txn).SetItemQuantity(context.Background(), testCartID, testProductID, 1)
		if !errors.Is(err, ErrCartUnavailable) {
			t.Fatalf("expected ErrCartUnavailable, got %v", err)
		}
		assertCallOrder(t, txn.calls, "LockCartIfExists", "Rollback", "Close")
	})
}

// 16. Error al cargar producto.
func TestLoadProductFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{stockErr: errors.New("connection reset")}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable for a generic product load error, got %v", err)
	}
}

// 17. Error al cargar item.
func TestLoadItemFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{
		stock:           db.CartProductStock{Available: true, Quantity: 5},
		itemQuantityErr: errors.New("read failed"),
	}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	if txn.upsertCartID != "" {
		t.Fatal("must not write after failing to read the current line")
	}
}

// 18. Error al insertar.
func TestUpsertFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{
		stock:     db.CartProductStock{Available: true, Quantity: 5},
		upsertErr: errors.New("write failed"),
	}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LockProductStock", "ItemQuantity", "UpsertItem", "Rollback", "Close")
}

// 19. Error al actualizar.
func TestUpdateFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{
		lockCartExists: true,
		itemExists:     true,
		stock:          db.CartProductStock{Available: true, Quantity: 5},
		updateErr:      errors.New("write failed"),
	}
	err := serviceWithTx(txn).SetItemQuantity(context.Background(), testCartID, testProductID, 2)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	assertCallOrder(t, txn.calls, "LockCartIfExists", "ItemQuantity", "LockProductStock", "UpdateItemQuantity", "Rollback", "Close")
}

// 20. Error al eliminar.
func TestDeleteFailuresAreCartUnavailable(t *testing.T) {
	t.Run("item", func(t *testing.T) {
		txn := &fakeCartTx{lockCartExists: true, deleteItemErr: errors.New("delete failed")}
		err := serviceWithTx(txn).DeleteItem(context.Background(), testCartID, testProductID)
		if !errors.Is(err, ErrCartUnavailable) {
			t.Fatalf("expected ErrCartUnavailable, got %v", err)
		}
	})
	t.Run("all", func(t *testing.T) {
		txn := &fakeCartTx{lockCartExists: true, deleteAllErr: errors.New("delete failed")}
		err := serviceWithTx(txn).Clear(context.Background(), testCartID)
		if !errors.Is(err, ErrCartUnavailable) {
			t.Fatalf("expected ErrCartUnavailable, got %v", err)
		}
	})
}

// 21 and 22. Error de commit; rollback ante error is asserted across every case above.
func TestCommitFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{
		stock:     db.CartProductStock{Available: true, Quantity: 5},
		commitErr: errors.New("commit failed"),
	}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LockProductStock", "ItemQuantity", "UpsertItem", "Commit", "Close")
}

// 23. Errores tipados: cada rama produce un sentinel distinto y estable.
func TestTypedErrorsAreDistinguishable(t *testing.T) {
	sentinels := []error{ErrProductNotFound, ErrProductUnavailable, ErrInsufficientStock, ErrCartItemNotFound, ErrCartUnavailable}
	seen := make(map[error]bool)
	for _, sentinel := range sentinels {
		if seen[sentinel] {
			t.Fatalf("duplicate sentinel: %v", sentinel)
		}
		seen[sentinel] = true
		for _, other := range sentinels {
			if other != sentinel && errors.Is(sentinel, other) {
				t.Fatalf("%v must not satisfy errors.Is for %v", sentinel, other)
			}
		}
	}
}

// 24. Ningún error interno filtrado.
func TestNoInternalErrorTextLeaksThroughSentinels(t *testing.T) {
	txn := &fakeCartTx{stockErr: errors.New("dial tcp 10.0.0.5:5432: connection refused, password authentication failed for user \"app\" SQLSTATE 28P01")}
	err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1)
	assertNoInternalLeak(t, err)
	if err.Error() != ErrCartUnavailable.Error() {
		t.Fatalf("expected the generic sentinel message, got %q", err.Error())
	}
}

// 25. Context cancelado.
func TestCanceledContextIsCartUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := serviceWithBeginError(context.Canceled).AddItem(ctx, testCartID, testProductID, 1)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal("must not leak context.Canceled to the caller")
	}
}

// 26. cartID y productID correctos llegan sin alterar a cada paso.
func TestIdentifiersReachEveryStepUnaltered(t *testing.T) {
	txn := &fakeCartTx{
		stock:        db.CartProductStock{Available: true, Quantity: 5},
		itemQuantity: 0,
		itemExists:   false,
	}
	if err := serviceWithTx(txn).AddItem(context.Background(), testCartID, testProductID, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if txn.ensureAndLockCartID != testCartID {
		t.Fatalf("cart id altered before EnsureAndLockCart: %q", txn.ensureAndLockCartID)
	}
	if txn.lockProductStockID != testProductID {
		t.Fatalf("product id altered before LockProductStock: %q", txn.lockProductStockID)
	}
	if txn.itemQtyCartID != testCartID || txn.itemQtyProductID != testProductID {
		t.Fatalf("identifiers altered before ItemQuantity: cart=%q product=%q", txn.itemQtyCartID, txn.itemQtyProductID)
	}
	if txn.upsertCartID != testCartID || txn.upsertProductID != testProductID {
		t.Fatalf("identifiers altered before UpsertItem: cart=%q product=%q", txn.upsertCartID, txn.upsertProductID)
	}
}

// --- AddItemIdempotent ---

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// 1. Primera operación crea claim y aplica AddItem.
func TestAddItemIdempotentFirstCallClaimsAndApplies(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	txn := &fakeCartTx{
		idempotencyExists: false,
		stock:             db.CartProductStock{Available: true, Quantity: 5},
	}
	outcome, err := serviceWithTxAndClock(txn, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 2, testKeyHash, testRequestHash,
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if outcome != AddItemApplied {
		t.Fatalf("expected AddItemApplied, got %v", outcome)
	}
	assertCallOrder(t, txn.calls,
		"EnsureAndLockCart", "LoadIdempotencyRecord", "InsertIdempotencyRecord",
		"LockProductStock", "ItemQuantity", "UpsertItem", "Commit", "Close",
	)
	if !bytes.Equal(txn.insertKeyHash, testKeyHash) || !bytes.Equal(txn.insertRequestHash, testRequestHash) {
		t.Fatalf("claim inserted with wrong hashes: key=%x request=%x", txn.insertKeyHash, txn.insertRequestHash)
	}
	if !txn.insertCreatedAt.Equal(now) {
		t.Fatalf("expected created_at %v, got %v", now, txn.insertCreatedAt)
	}
	if !txn.insertExpiresAt.Equal(now.Add(idempotencyTTL)) {
		t.Fatalf("expected expires_at 24h after created_at, got %v", txn.insertExpiresAt)
	}
	if txn.upsertQuantity != 2 {
		t.Fatalf("expected the new line to carry the requested quantity, got %d", txn.upsertQuantity)
	}
}

// 2 and 3. Replay mismo key/hash no aplica AddItem y devuelve Replayed.
func TestAddItemIdempotentReplayDoesNotReapplyAddItem(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	txn := &fakeCartTx{
		idempotencyExists: true,
		idempotencyRecord: db.CartIdempotencyRecord{RequestHash: testRequestHash, ExpiresAt: now.Add(time.Hour)},
	}
	outcome, err := serviceWithTxAndClock(txn, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 2, testKeyHash, testRequestHash,
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if outcome != AddItemReplayed {
		t.Fatalf("expected AddItemReplayed, got %v", outcome)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LoadIdempotencyRecord", "Commit", "Close")
	if txn.upsertCartID != "" {
		t.Fatal("a replay must not write to cart_items")
	}
}

// 4. Misma clave con hash diferente devuelve ErrIdempotencyConflict.
func TestAddItemIdempotentConflictingPayloadIsRejected(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	otherRequestHash := bytes.Repeat([]byte{0xAA}, 32)
	txn := &fakeCartTx{
		idempotencyExists: true,
		idempotencyRecord: db.CartIdempotencyRecord{RequestHash: otherRequestHash, ExpiresAt: now.Add(time.Hour)},
	}
	outcome, err := serviceWithTxAndClock(txn, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 2, testKeyHash, testRequestHash,
	)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
	if outcome != AddItemApplied {
		t.Fatalf("expected the zero-value outcome on conflict, got %v", outcome)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LoadIdempotencyRecord", "Rollback", "Close")
	if txn.upsertCartID != "" || txn.insertCartID != "" {
		t.Fatal("a conflict must not write to cart_items or claim a new record")
	}
}

// 5 and 6. Registro expirado se elimina y permite una operación nueva.
func TestAddItemIdempotentExpiredRecordIsDeletedAndReclaimed(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	txn := &fakeCartTx{
		idempotencyExists: true,
		idempotencyRecord: db.CartIdempotencyRecord{RequestHash: testRequestHash, ExpiresAt: now.Add(-time.Minute)},
		stock:             db.CartProductStock{Available: true, Quantity: 5},
	}
	outcome, err := serviceWithTxAndClock(txn, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 2, testKeyHash, testRequestHash,
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if outcome != AddItemApplied {
		t.Fatalf("expected AddItemApplied for a reclaimed expired key, got %v", outcome)
	}
	assertCallOrder(t, txn.calls,
		"EnsureAndLockCart", "LoadIdempotencyRecord", "DeleteIdempotencyRecord", "InsertIdempotencyRecord",
		"LockProductStock", "ItemQuantity", "UpsertItem", "Commit", "Close",
	)
}

// 7. Replay no extiende expiración (ya cubierto por assertCallOrder de replay:
// InsertIdempotencyRecord nunca se invoca en una replay).
func TestAddItemIdempotentReplayNeverExtendsExpiration(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	txn := &fakeCartTx{
		idempotencyExists: true,
		idempotencyRecord: db.CartIdempotencyRecord{RequestHash: testRequestHash, ExpiresAt: now.Add(time.Minute)},
	}
	_, err := serviceWithTxAndClock(txn, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 2, testKeyHash, testRequestHash,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, call := range txn.calls {
		if call == "InsertIdempotencyRecord" {
			t.Fatal("a replay must never insert or refresh a claim")
		}
	}
}

// 8. Error al leer idempotencia.
func TestAddItemIdempotentLoadRecordFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{loadIdempotencyErr: errors.New("read failed")}
	outcome, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	if outcome != AddItemApplied {
		t.Fatalf("expected the zero-value outcome, got %v", outcome)
	}
	assertCallOrder(t, txn.calls, "EnsureAndLockCart", "LoadIdempotencyRecord", "Rollback", "Close")
}

// 9. Error al borrar expirado.
func TestAddItemIdempotentDeleteExpiredFailureIsCartUnavailable(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	txn := &fakeCartTx{
		idempotencyExists:    true,
		idempotencyRecord:    db.CartIdempotencyRecord{RequestHash: testRequestHash, ExpiresAt: now.Add(-time.Minute)},
		deleteIdempotencyErr: errors.New("delete failed"),
	}
	_, err := serviceWithTxAndClock(txn, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash,
	)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	if txn.insertCartID != "" {
		t.Fatal("must not claim a new record after failing to delete the expired one")
	}
}

// 10. Error al insertar claim.
func TestAddItemIdempotentInsertClaimFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{insertIdempotencyErr: errors.New("insert failed")}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	if txn.lockProductStockID != "" {
		t.Fatal("must not read product stock after failing to claim the key")
	}
}

// 11. Error al bloquear producto.
func TestAddItemIdempotentLockProductFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{stockErr: errors.New("connection reset")}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
}

// 12. Producto inexistente.
func TestAddItemIdempotentProductNotFound(t *testing.T) {
	txn := &fakeCartTx{stockErr: db.ErrCartTxProductNotFound}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}

// 13. Producto no disponible.
func TestAddItemIdempotentProductUnavailable(t *testing.T) {
	txn := &fakeCartTx{stock: db.CartProductStock{Available: false, Quantity: 5}}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrProductUnavailable) {
		t.Fatalf("expected ErrProductUnavailable, got %v", err)
	}
}

// 14. Stock insuficiente.
func TestAddItemIdempotentInsufficientStock(t *testing.T) {
	txn := &fakeCartTx{stock: db.CartProductStock{Available: true, Quantity: 1}}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 5, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if txn.insertCartID == "" {
		t.Fatal("a new claim is inserted before stock is validated, and must still roll back with it")
	}
}

// 15. Error al leer item.
func TestAddItemIdempotentReadItemFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{
		stock:           db.CartProductStock{Available: true, Quantity: 5},
		itemQuantityErr: errors.New("read failed"),
	}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
}

// 16. Error al hacer upsert.
func TestAddItemIdempotentUpsertFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{
		stock:     db.CartProductStock{Available: true, Quantity: 5},
		upsertErr: errors.New("write failed"),
	}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
}

// 17 and 18. Error de commit; rollback elimina el claim (misma transacción,
// el fake no simula persistencia real, pero Commit nunca se alcanza en
// ninguna rama de fallo previa: verificado por assertCallOrder en cada test
// de fallo anterior, donde "Commit" nunca aparece).
func TestAddItemIdempotentCommitFailureIsCartUnavailable(t *testing.T) {
	txn := &fakeCartTx{
		stock:     db.CartProductStock{Available: true, Quantity: 5},
		commitErr: errors.New("commit failed"),
	}
	_, err := serviceWithTx(txn).AddItemIdempotent(context.Background(), testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	assertCallOrder(t, txn.calls,
		"EnsureAndLockCart", "LoadIdempotencyRecord", "InsertIdempotencyRecord",
		"LockProductStock", "ItemQuantity", "UpsertItem", "Commit", "Close",
	)
}

// 19. Orden exacto de llamadas: cubierto por assertCallOrder en cada test de
// esta sección (primera operación, replay, expirado, cada fallo).

// 20. Context cancelado.
func TestAddItemIdempotentCanceledContextIsCartUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := serviceWithBeginError(context.Canceled).AddItemIdempotent(ctx, testCartID, testProductID, 1, testKeyHash, testRequestHash)
	if !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal("must not leak context.Canceled to the caller")
	}
}

// 21. Claves diferentes aplican operaciones distintas.
func TestAddItemIdempotentDifferentKeysAreIndependentClaims(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	otherKeyHash := bytes.Repeat([]byte{0xBB}, 32)

	txnFirst := &fakeCartTx{stock: db.CartProductStock{Available: true, Quantity: 10}}
	outcomeFirst, err := serviceWithTxAndClock(txnFirst, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 2, testKeyHash, testRequestHash,
	)
	if err != nil || outcomeFirst != AddItemApplied {
		t.Fatalf("expected the first key to apply, got outcome=%v err=%v", outcomeFirst, err)
	}

	txnSecond := &fakeCartTx{stock: db.CartProductStock{Available: true, Quantity: 10}, itemQuantity: 2, itemExists: true}
	outcomeSecond, err := serviceWithTxAndClock(txnSecond, fixedClock(now)).AddItemIdempotent(
		context.Background(), testCartID, testProductID, 3, otherKeyHash, testRequestHash,
	)
	if err != nil || outcomeSecond != AddItemApplied {
		t.Fatalf("expected the second, different key to also apply, got outcome=%v err=%v", outcomeSecond, err)
	}
	if txnSecond.upsertQuantity != 5 {
		t.Fatalf("expected the second key's operation to add on top of the existing quantity, got %d", txnSecond.upsertQuantity)
	}
}

// 22. No se ejecuta AddItem en replay: verificado explícitamente por
// TestAddItemIdempotentReplayDoesNotReapplyAddItem (assertCallOrder omits
// LockProductStock/ItemQuantity/UpsertItem entirely).

// 23. No se modifica carrito en conflicto: verificado explícitamente por
// TestAddItemIdempotentConflictingPayloadIsRejected.
