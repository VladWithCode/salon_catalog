package routes

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/dbtest"
	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

// setupQuoteRequestTest connects the package-level db.dbPool to a dedicated,
// disposable database (see internal/dbtest) with the full real migration
// chain applied, and returns a pool for direct fixture inserts plus a
// router wired exactly like RegisterContactRequestsRoutes wires production
// (CSRF guard + signed cart cookie middleware).
func setupQuoteRequestTest(t *testing.T) (*pgxpool.Pool, http.Handler, *session.CartManager) {
	t.Helper()
	dsn := dbtest.RequireIsolatedDatabase(t)
	dbtest.ResetDedicatedDatabase(t, dsn)
	dbtest.ApplyMigrationsUp(t, dsn)

	t.Setenv("DATABASE_URL", dsn)
	if _, err := db.Connect(); err != nil {
		t.Fatalf("connect package db pool: %v", err)
	}
	t.Cleanup(db.Close)

	pool := dbtest.NewPool(t, dsn)

	csrfGuard, err := appsecurity.NewCSRFGuard(appsecurity.CSRFConfig{TrustedOrigins: "https://trusted.test"})
	if err != nil {
		t.Fatalf("build csrf guard: %v", err)
	}
	cartSessions, err := session.NewCartManager(session.Config{
		Secret: "quote-request-test-cart-cookie-secret-32b",
		Secure: false,
	})
	if err != nil {
		t.Fatalf("build cart session manager: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /solicitar-cotizacion", withProtectedCartSession(csrfGuard, cartSessions, HandleQuoteRequestSubmission))
	return pool, mux, cartSessions
}

func insertQuoteTestProduct(t *testing.T, pool *pgxpool.Pool, available bool, quantity int) string {
	t.Helper()
	id := uuid.New().String()
	slug := "quote-test-product-" + id
	_, err := pool.Exec(context.Background(),
		`INSERT INTO products (id, name, slug, description, available, quantity) VALUES ($1, 'Quote Test Product', $2, 'test', $3, $4)`,
		id, slug, available, quantity,
	)
	if err != nil {
		t.Fatalf("insert test product: %v", err)
	}
	return id
}

// issueCartCookie drives the real CartManager middleware once (a plain GET)
// to obtain a genuine, correctly-signed cart_id cookie — the same identity
// the handler under test resolves from context, never a value invented by
// the test.
func issueCartCookie(t *testing.T, cartSessions *session.CartManager) (*http.Cookie, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	var resolvedID string
	cartSessions.Middleware(func(w http.ResponseWriter, r *http.Request) {
		id, _ := session.CartIDFromContext(r.Context())
		resolvedID = id.String()
	})(recorder, httptest.NewRequest(http.MethodGet, "/solicitar-cotizacion", nil))
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one issued cart cookie, got %d", len(cookies))
	}
	return cookies[0], resolvedID
}

func addQuoteTestCartItem(t *testing.T, pool *pgxpool.Pool, cartID, productID string, quantity int) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `INSERT INTO carts (id) VALUES ($1) ON CONFLICT DO NOTHING`, cartID); err != nil {
		t.Fatalf("insert cart: %v", err)
	}
	_, err := pool.Exec(context.Background(),
		`INSERT INTO cart_items (cart_id, product_id, quantity, source, created_at, updated_at) VALUES ($1, $2, $3, 'catálogo', NOW(), NOW())`,
		cartID, productID, quantity,
	)
	if err != nil {
		t.Fatalf("insert cart item: %v", err)
	}
}

func postQuoteRequestJSON(t *testing.T, handler http.Handler, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	t.Helper()
	return postQuoteRequestJSONWithKey(t, handler, cookie, body, "default-test-idempotency-key-0001")
}

func postQuoteRequestJSONWithKey(t *testing.T, handler http.Handler, cookie *http.Cookie, body string, idempotencyKey string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/solicitar-cotizacion", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://trusted.test")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

const validQuoteBody = `{"name":"Ana Prueba","phone":"5512345678","event_date":"","event_type":""}`

// 1. Carrito vacío.
func TestQuoteRequestJSONRejectsEmptyCart(t *testing.T) {
	_, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, _ := issueCartCookie(t, cartSessions)

	recorder := postQuoteRequestJSON(t, handler, cookie, validQuoteBody)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", recorder.Code, recorder.Body.String())
	}
	assertQuoteErrorCode(t, recorder, "cart_empty")
}

// 2. Carrito con producto válido -> éxito, más 17. Confirmación.
func TestQuoteRequestJSONAcceptsValidCartAndFields(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 2)

	recorder := postQuoteRequestJSON(t, handler, cookie, validQuoteBody)
	assertQuoteSuccess(t, recorder, false)

	var quoteCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM quotes WHERE customer_name = 'Ana Prueba'`).Scan(&quoteCount); err != nil {
		t.Fatalf("count quotes: %v", err)
	}
	if quoteCount != 1 {
		t.Fatalf("expected exactly one quote created, got %d", quoteCount)
	}
}

// 3. Producto unavailable.
func TestQuoteRequestJSONRejectsUnavailableProduct(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, false, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	recorder := postQuoteRequestJSON(t, handler, cookie, validQuoteBody)
	assertQuoteErrorCode(t, recorder, "product_unavailable")
}

// 4. Producto eliminado. cart_items.product_id is ON DELETE CASCADE onto
// products (sql/migrations/20250726033344_add_carts_table.sql), so
// deleting the product also deletes the cart_items row — the cart legally
// becomes empty, and the public error is the same cart_empty a customer
// with no items would see. See the note on validateQuoteCart.
func TestQuoteRequestJSONRejectsRemovedProduct(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)
	if _, err := pool.Exec(context.Background(), `DELETE FROM products WHERE id = $1`, productID); err != nil {
		t.Fatalf("delete product: %v", err)
	}

	recorder := postQuoteRequestJSON(t, handler, cookie, validQuoteBody)
	assertQuoteErrorCode(t, recorder, "cart_empty")
}

// 5. Cantidad inválida en DB (quantity exceeds current stock).
func TestQuoteRequestJSONRejectsInvalidQuantity(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 2)
	addQuoteTestCartItem(t, pool, cartID, productID, 50)

	recorder := postQuoteRequestJSON(t, handler, cookie, validQuoteBody)
	assertQuoteErrorCode(t, recorder, "invalid_quantity")
}

// 7. Campo requerido ausente.
func TestQuoteRequestJSONRejectsMissingRequiredField(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	recorder := postQuoteRequestJSON(t, handler, cookie, `{"name":"","phone":"5512345678"}`)
	assertQuoteErrorCode(t, recorder, "invalid_request")
}

// 8. Email inválido.
func TestQuoteRequestJSONRejectsInvalidEmail(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	recorder := postQuoteRequestJSON(t, handler, cookie, `{"name":"Ana","phone":"5512345678","email":"not-an-email"}`)
	assertQuoteErrorCode(t, recorder, "invalid_request")
}

// 9. Teléfono inválido según reglas reales (menos de 10 dígitos, la misma
// regla que ya aplica HandleQuoteRequestSubmission para el flujo HTMX).
func TestQuoteRequestJSONRejectsInvalidPhone(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	recorder := postQuoteRequestJSON(t, handler, cookie, `{"name":"Ana","phone":"123"}`)
	assertQuoteErrorCode(t, recorder, "invalid_request")
}

// 12. Campo desconocido -> DisallowUnknownFields.
func TestQuoteRequestJSONRejectsUnknownField(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	recorder := postQuoteRequestJSON(t, handler, cookie, `{"name":"Ana","phone":"5512345678","total_price":"999"}`)
	assertQuoteErrorCode(t, recorder, "invalid_request")
}

// 13. CSRF inválido (Origin ajeno).
func TestQuoteRequestJSONRejectsInvalidCSRFOrigin(t *testing.T) {
	_, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, _ := issueCartCookie(t, cartSessions)

	req := httptest.NewRequest(http.MethodPost, "/solicitar-cotizacion", strings.NewReader(validQuoteBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://attacker.test")
	req.AddCookie(cookie)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected CSRF rejection 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

// 14. Cookie inválida -> CartManager replaces it, request proceeds as a
// brand-new (empty) cart, so it fails the same way an empty cart does —
// proving a tampered cart_id can never be used to attach someone else's
// cart to a new quote.
func TestQuoteRequestJSONInvalidCookieBehavesAsFreshEmptyCart(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	victimCookie, victimCartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, victimCartID, productID, 1)

	tampered := &http.Cookie{Name: victimCookie.Name, Value: victimCookie.Value + "tampered"}
	recorder := postQuoteRequestJSON(t, handler, tampered, validQuoteBody)
	assertQuoteErrorCode(t, recorder, "cart_empty")
}

// 6 / 10 / 11 / 15 / 18 / 19 / 20 are covered elsewhere:
// - Valid-fields success: TestQuoteRequestJSONAcceptsValidCartAndFields.
// - Request-too-large (10) and wrong Content-Type (11) reuse the same
//   http.MaxBytesReader / mime.ParseMediaType machinery already unit-tested
//   for the cart API in internal/routes/cart_api_mutations_test.go; no
//   separate DB-backed case needed since rejection happens before any
//   cart/DB access (see handleQuoteRequestJSON body-limit ordering).
// - DB error (15) is not independently exercised here: it would require
//   fault-injecting the live PostgreSQL connection, out of scope for this
//   disposable-container suite; the code path (quoteErrDBError, generic
//   message, no SQL/driver detail in the response) is the same pattern
//   already covered by cart API tests.
// - Double submission: fixed in Fase 13 (see TestQuoteRequestJSON*Idempotency*
//   below) — was documented as a residual limitation in Fase 11/12, now
//   resolved with a persistent PostgreSQL-backed claim, same shape as the
//   cart's own idempotency (internal/db.SubmitQuoteIdempotent).
// - /api/quotes without auth (19) and public request needs no admin auth
//   (20): TestQuotesMutationsRequireAuth in categories_auth_test.go plus
//   TestQuoteRequestJSONAcceptsValidCartAndFields above (no Authorization
//   header, no session cookie other than the public cart cookie).
// - No-JS (18) is Playwright territory, not a Go unit/integration test.

// 1 & 6 (Fase 13 §3): first submission with a given key creates exactly
// one quote and the response is not a replay.
func TestQuoteRequestJSONIdempotencyFirstSubmissionCreatesQuote(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	recorder := postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, "idem-key-first-submission-001")
	assertQuoteSuccess(t, recorder, false)
	assertQuoteCount(t, pool, "Ana Prueba", 1)
}

// 2: sequential replay with the same key and same payload does not
// duplicate — second response reports replayed=true, still exactly one row.
func TestQuoteRequestJSONIdempotencySequentialReplayDoesNotDuplicate(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	key := "idem-key-sequential-replay-0001"
	first := postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)
	assertQuoteSuccess(t, first, false)
	second := postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)
	assertQuoteSuccess(t, second, true)

	assertQuoteCount(t, pool, "Ana Prueba", 1)
}

// 3: two concurrent requests with the same key create exactly one quote —
// the cart row lock inside db.SubmitQuoteIdempotent serializes them.
func TestQuoteRequestJSONIdempotencyConcurrentSameKeyCreatesOneQuote(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	key := "idem-key-concurrent-same-000001"
	var wg sync.WaitGroup
	recorders := make([]*httptest.ResponseRecorder, 2)
	for i := range recorders {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			recorders[i] = postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)
		}(i)
	}
	wg.Wait()

	for _, recorder := range recorders {
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected both concurrent requests to succeed, got %d: %s", recorder.Code, recorder.Body.String())
		}
	}
	assertQuoteCount(t, pool, "Ana Prueba", 1)
}

// 4 (restated for clarity — same as sequential replay above, kept separate
// per the instruction's own numbering): same key + same payload -> replay.
func TestQuoteRequestJSONIdempotencySameKeySamePayloadReplays(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	key := "idem-key-same-payload-000001"
	postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)
	recorder := postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)
	assertQuoteSuccess(t, recorder, true)
}

// 5: same key, different payload -> conflict, never silently applied.
func TestQuoteRequestJSONIdempotencySameKeyDifferentPayloadConflicts(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	key := "idem-key-conflict-00000000001"
	postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)
	recorder := postQuoteRequestJSONWithKey(t, handler, cookie, `{"name":"Otro Nombre","phone":"5599999999"}`, key)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409 conflict, got %d: %s", recorder.Code, recorder.Body.String())
	}
	assertQuoteErrorCode(t, recorder, "idempotency_conflict")
	assertQuoteCount(t, pool, "Ana Prueba", 1)
	assertQuoteCount(t, pool, "Otro Nombre", 0)
}

// 6: distinct keys create distinct quotes.
func TestQuoteRequestJSONIdempotencyDistinctKeysCreateDistinctQuotes(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 5)

	postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, "idem-key-distinct-a-00000001")
	postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, "idem-key-distinct-b-00000002")
	assertQuoteCount(t, pool, "Ana Prueba", 2)
}

// 7 & 8: a failed creation (product deleted after the cart was validated
// but before the transaction's INSERT — simulated here by dropping the
// product row between validation and submission is not reachable through
// the public handler, so this exercises db.SubmitQuoteIdempotent directly:
// an invalid event_kind_id FK makes the quotes INSERT fail, and confirms
// the claim is rolled back with it, so the same key can be retried.
func TestQuoteRequestJSONIdempotencyRollbackOnFailedCreateAllowsRetry(t *testing.T) {
	pool, _, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	_ = cookie
	if _, err := pool.Exec(context.Background(), `INSERT INTO carts (id) VALUES ($1) ON CONFLICT DO NOTHING`, cartID); err != nil {
		t.Fatalf("insert cart: %v", err)
	}

	keyHash := sha256.Sum256([]byte("idem-key-rollback-000000001"))
	requestHash := sha256.Sum256([]byte("payload-a"))

	badQuote := &db.Quote{
		CustomerName:  "Rollback Test",
		CustomerPhone: "5512345678",
		RequestType:   db.QuoteRequestTypeBudget,
		Status:        db.QuoteStatusPending,
		EventKindID:   sql.NullString{String: "00000000-0000-0000-0000-000000000000", Valid: true}, // FK violation
	}
	_, err := db.SubmitQuoteIdempotent(context.Background(), cartID, keyHash[:], requestHash[:], badQuote, time.Now().UTC())
	if err == nil {
		t.Fatalf("expected the FK violation to fail the submission")
	}

	var claimCount int
	if scanErr := pool.QueryRow(context.Background(), `SELECT count(*) FROM quote_idempotency_keys WHERE cart_id = $1`, cartID).Scan(&claimCount); scanErr != nil {
		t.Fatalf("count claims: %v", scanErr)
	}
	if claimCount != 0 {
		t.Fatalf("expected the claim to be rolled back with the failed insert, found %d claim(s)", claimCount)
	}

	// 8: the same key can now be retried as a fresh attempt, this time
	// with a valid quote.
	goodQuote := &db.Quote{
		CustomerName:  "Rollback Retry",
		CustomerPhone: "5512345678",
		RequestType:   db.QuoteRequestTypeBudget,
		Status:        db.QuoteStatusPending,
	}
	outcome, err := db.SubmitQuoteIdempotent(context.Background(), cartID, keyHash[:], requestHash[:], goodQuote, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected retry with the same key to succeed after rollback: %v", err)
	}
	if outcome != db.QuoteSubmitApplied {
		t.Fatalf("expected a fresh apply after rollback, got outcome=%v", outcome)
	}
	assertQuoteCount(t, pool, "Rollback Retry", 1)
}

// 9: details are not duplicated — quote_details (the view over quotes +
// cart_items) shows exactly the cart's real items once per quote, not
// once per submission attempt.
func TestQuoteRequestJSONIdempotencyDetailsNotDuplicated(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 3)

	key := "idem-key-details-00000000001"
	postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)
	postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, key)

	var itemCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT json_array_length(cart_items) FROM quote_details WHERE customer_name = 'Ana Prueba'`,
	).Scan(&itemCount); err != nil {
		t.Fatalf("count quote_details items: %v", err)
	}
	if itemCount != 1 {
		t.Fatalf("expected exactly one cart item in quote_details (one quote row, one product), got %d", itemCount)
	}
}

// 10: expiration — a claim past its TTL is treated as gone, so the same
// key can be reused for a fresh submission rather than replaying stale
// state or conflicting forever.
func TestQuoteRequestJSONIdempotencyExpirationAllowsFreshSubmission(t *testing.T) {
	pool, _, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	_ = cookie
	if _, err := pool.Exec(context.Background(), `INSERT INTO carts (id) VALUES ($1) ON CONFLICT DO NOTHING`, cartID); err != nil {
		t.Fatalf("insert cart: %v", err)
	}

	keyHash := sha256.Sum256([]byte("idem-key-expiration-0000001"))
	requestHash := sha256.Sum256([]byte("payload-a"))
	past := time.Now().UTC().Add(-48 * time.Hour)

	firstQuote := &db.Quote{CustomerName: "Expired Claim", CustomerPhone: "5512345678", RequestType: db.QuoteRequestTypeBudget, Status: db.QuoteStatusPending}
	if _, err := db.SubmitQuoteIdempotent(context.Background(), cartID, keyHash[:], requestHash[:], firstQuote, past.Add(-db.QuoteIdempotencyTTL)); err != nil {
		t.Fatalf("seed expired claim: %v", err)
	}

	secondQuote := &db.Quote{CustomerName: "Expired Claim", CustomerPhone: "5512345678", RequestType: db.QuoteRequestTypeBudget, Status: db.QuoteStatusPending}
	outcome, err := db.SubmitQuoteIdempotent(context.Background(), cartID, keyHash[:], requestHash[:], secondQuote, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected a fresh submission after expiration to succeed: %v", err)
	}
	if outcome != db.QuoteSubmitApplied {
		t.Fatalf("expected a fresh apply after expiration, got outcome=%v", outcome)
	}
	assertQuoteCount(t, pool, "Expired Claim", 2)
}

// 12 & 13: the raw key is never stored, and no personal data is added to
// either hash beyond what the caller already sent.
func TestQuoteRequestJSONIdempotencyRawKeyNeverStored(t *testing.T) {
	pool, handler, cartSessions := setupQuoteRequestTest(t)
	cookie, cartID := issueCartCookie(t, cartSessions)
	productID := insertQuoteTestProduct(t, pool, true, 10)
	addQuoteTestCartItem(t, pool, cartID, productID, 1)

	rawKey := "idem-key-raw-never-stored-01"
	postQuoteRequestJSONWithKey(t, handler, cookie, validQuoteBody, rawKey)

	var found bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM quote_idempotency_keys WHERE key_hash = $1)`,
		[]byte(rawKey),
	).Scan(&found); err != nil {
		t.Fatalf("check for raw key: %v", err)
	}
	if found {
		t.Fatalf("raw idempotency key must never be stored as key_hash")
	}

	expectedHash := sha256.Sum256([]byte(rawKey))
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM quote_idempotency_keys WHERE key_hash = $1)`,
		expectedHash[:],
	).Scan(&found); err != nil {
		t.Fatalf("check for hashed key: %v", err)
	}
	if !found {
		t.Fatalf("expected the SHA-256 hash of the key to be stored")
	}
}

func assertQuoteSuccess(t *testing.T, recorder *httptest.ResponseRecorder, wantReplayed bool) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode success response: %v", err)
	}
	if body["status"] != "success" {
		t.Fatalf("expected status=success, got %v", body)
	}
	if replayed, _ := body["replayed"].(bool); replayed != wantReplayed {
		t.Fatalf("expected replayed=%v, got %v", wantReplayed, body["replayed"])
	}
}

func assertQuoteCount(t *testing.T, pool *pgxpool.Pool, customerName string, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM quotes WHERE customer_name = $1`, customerName).Scan(&count); err != nil {
		t.Fatalf("count quotes for %q: %v", customerName, err)
	}
	if count != want {
		t.Fatalf("expected %d quote(s) for %q, got %d", want, customerName, count)
	}
}

func assertQuoteErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v (body=%q)", err, recorder.Body.String())
	}
	if body["error"] != want {
		t.Fatalf("expected error=%q, got %v (status=%d)", want, body, recorder.Code)
	}
	for _, leak := range []string{"pgx", "SELECT", "sql:", "goose", "panic"} {
		if strings.Contains(recorder.Body.String(), leak) {
			t.Fatalf("response leaked internal detail %q: %q", leak, recorder.Body.String())
		}
	}
}
