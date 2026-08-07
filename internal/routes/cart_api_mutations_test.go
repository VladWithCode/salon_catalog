package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vladwithcode/salon_catalog/internal/cart"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

// cartAPITestIdempotencyKey is a valid Idempotency-Key value (18 characters,
// within the required 16-128 range) reused by tests that do not exercise the
// header's own validation.
const cartAPITestIdempotencyKey = "test-idem-key-001"

// fakeCartMutationOperations stands in for internal/cart.Service so the
// error-code mapping performed by cartMutationServiceAdapter can be tested
// without a database. The deep transactional unit tests (locking order,
// stock math, rollback behavior) live in internal/cart/service_test.go,
// next to the code they exercise.
type fakeCartMutationOperations struct {
	err     error
	outcome cart.AddItemOutcome
	calls   []cartAPIMutationCall
}

func (ops *fakeCartMutationOperations) AddItem(_ context.Context, cartID string, productID string, quantity int) error {
	ops.calls = append(ops.calls, cartAPIMutationCall{operation: "add", cartID: cartID, productID: productID, quantity: quantity})
	return ops.err
}

func (ops *fakeCartMutationOperations) AddItemIdempotent(_ context.Context, cartID string, productID string, quantity int, _ []byte, _ []byte) (cart.AddItemOutcome, error) {
	ops.calls = append(ops.calls, cartAPIMutationCall{operation: "add_idempotent", cartID: cartID, productID: productID, quantity: quantity})
	return ops.outcome, ops.err
}

func (ops *fakeCartMutationOperations) SetItemQuantity(_ context.Context, cartID string, productID string, quantity int) error {
	ops.calls = append(ops.calls, cartAPIMutationCall{operation: "set", cartID: cartID, productID: productID, quantity: quantity})
	return ops.err
}

func (ops *fakeCartMutationOperations) DeleteItem(_ context.Context, cartID string, productID string) error {
	ops.calls = append(ops.calls, cartAPIMutationCall{operation: "delete", cartID: cartID, productID: productID})
	return ops.err
}

func (ops *fakeCartMutationOperations) Clear(_ context.Context, cartID string) error {
	ops.calls = append(ops.calls, cartAPIMutationCall{operation: "clear", cartID: cartID})
	return ops.err
}

// TestCartMutationServiceAdapterTranslatesTypedErrors confirms the adapter
// maps every internal/cart sentinel to the exact errCartAPI* sentinel that
// writeCartAPIMutationError already turns into the stable public HTTP
// contract (404/409/503), and that unrecognized errors degrade safely to
// cart_unavailable without leaking their text.
func TestCartMutationServiceAdapterTranslatesTypedErrors(t *testing.T) {
	secretErr := errors.New("dial tcp 10.0.0.5:5432: password authentication failed")
	for _, testCase := range []struct {
		name    string
		err     error
		wantErr error
	}{
		{name: "nil", err: nil, wantErr: nil},
		{name: "product not found", err: cart.ErrProductNotFound, wantErr: errCartAPIProductNotFound},
		{name: "product unavailable", err: cart.ErrProductUnavailable, wantErr: errCartAPIProductUnavailable},
		{name: "insufficient stock", err: cart.ErrInsufficientStock, wantErr: errCartAPIInsufficientStock},
		{name: "item not found", err: cart.ErrCartItemNotFound, wantErr: errCartAPIItemNotFound},
		{name: "cart unavailable", err: cart.ErrCartUnavailable, wantErr: errCartAPIUnavailable},
		{name: "idempotency conflict", err: cart.ErrIdempotencyConflict, wantErr: errCartAPIIdempotencyConflict},
		{name: "unrecognized error", err: secretErr, wantErr: errCartAPIUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ops := &fakeCartMutationOperations{err: testCase.err}
			adapter := newCartAPIMutationService(ops)

			for _, invoke := range []func() error{
				func() error {
					return adapter.AddItem(context.Background(), routeTestCartID.String(), cartAPITestProductID, 1)
				},
				func() error {
					_, err := adapter.AddItemIdempotent(context.Background(), routeTestCartID.String(), cartAPITestProductID, 1, []byte("key-hash"), []byte("request-hash"))
					return err
				},
				func() error {
					return adapter.SetItemQuantity(context.Background(), routeTestCartID.String(), cartAPITestProductID, 1)
				},
				func() error {
					return adapter.DeleteItem(context.Background(), routeTestCartID.String(), cartAPITestProductID)
				},
				func() error { return adapter.Clear(context.Background(), routeTestCartID.String()) },
			} {
				got := invoke()
				if !errors.Is(got, testCase.wantErr) && got != testCase.wantErr {
					t.Fatalf("expected %v, got %v", testCase.wantErr, got)
				}
				if got != nil && strings.Contains(got.Error(), "password") {
					t.Fatalf("adapter leaked internal error text: %v", got)
				}
			}
			if len(ops.calls) != 5 {
				t.Fatalf("expected the adapter to call through to every operation, got %d calls", len(ops.calls))
			}
		})
	}
}

func TestCartMutationServiceAdapterPassesIdentifiersUnaltered(t *testing.T) {
	ops := &fakeCartMutationOperations{}
	adapter := newCartAPIMutationService(ops)
	if err := adapter.AddItem(context.Background(), routeTestCartID.String(), cartAPITestProductID, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ops.calls) != 1 {
		t.Fatalf("expected exactly one call, got %d", len(ops.calls))
	}
	call := ops.calls[0]
	if call.cartID != routeTestCartID.String() || call.productID != cartAPITestProductID || call.quantity != 3 {
		t.Fatalf("adapter altered call arguments: %+v", call)
	}
}

type cartAPIMutationCall struct {
	operation string
	cartID    string
	productID string
	quantity  int
}

type fakeCartAPIMutationService struct {
	calls     []cartAPIMutationCall
	errors    map[string]error
	outcome   cart.AddItemOutcome
	afterCall func(cartAPIMutationCall)
}

func (service *fakeCartAPIMutationService) record(call cartAPIMutationCall) error {
	service.calls = append(service.calls, call)
	if service.afterCall != nil {
		service.afterCall(call)
	}
	return service.errors[call.operation]
}

func (service *fakeCartAPIMutationService) AddItem(_ context.Context, cartID string, productID string, quantity int) error {
	return service.record(cartAPIMutationCall{operation: "add", cartID: cartID, productID: productID, quantity: quantity})
}

func (service *fakeCartAPIMutationService) AddItemIdempotent(_ context.Context, cartID string, productID string, quantity int, _ []byte, _ []byte) (cart.AddItemOutcome, error) {
	err := service.record(cartAPIMutationCall{operation: "add", cartID: cartID, productID: productID, quantity: quantity})
	return service.outcome, err
}

func (service *fakeCartAPIMutationService) SetItemQuantity(_ context.Context, cartID string, productID string, quantity int) error {
	return service.record(cartAPIMutationCall{operation: "set", cartID: cartID, productID: productID, quantity: quantity})
}

func (service *fakeCartAPIMutationService) DeleteItem(_ context.Context, cartID string, productID string) error {
	return service.record(cartAPIMutationCall{operation: "delete", cartID: cartID, productID: productID})
}

func (service *fakeCartAPIMutationService) Clear(_ context.Context, cartID string) error {
	return service.record(cartAPIMutationCall{operation: "clear", cartID: cartID})
}

func TestPostCartAPIItemStrictParsing(t *testing.T) {
	validBody := `{"product_id":"` + cartAPITestProductID + `","quantity":1}`
	for _, testCase := range []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantError   string
		wantCalls   int
	}{
		{name: "application json", contentType: "application/json", body: validBody, wantStatus: 200, wantCalls: 1},
		{name: "json charset", contentType: "application/json; charset=utf-8", body: validBody, wantStatus: 200, wantCalls: 1},
		{name: "missing content type", body: validBody, wantStatus: 415, wantError: "unsupported_media_type"},
		{name: "text plain", contentType: "text/plain", body: validBody, wantStatus: 415, wantError: "unsupported_media_type"},
		{name: "form encoded", contentType: "application/x-www-form-urlencoded", body: "product_id=x", wantStatus: 415, wantError: "unsupported_media_type"},
		{name: "empty body", contentType: "application/json", wantStatus: 400, wantError: "invalid_request"},
		{name: "null root", contentType: "application/json", body: "null", wantStatus: 400, wantError: "invalid_request"},
		{name: "array root", contentType: "application/json", body: "[]", wantStatus: 400, wantError: "invalid_request"},
		{name: "malformed", contentType: "application/json", body: "{", wantStatus: 400, wantError: "invalid_request"},
		{name: "double json", contentType: "application/json", body: validBody + validBody, wantStatus: 400, wantError: "invalid_request"},
		{name: "trailing data", contentType: "application/json", body: validBody + "x", wantStatus: 400, wantError: "invalid_request"},
		{name: "unknown field", contentType: "application/json", body: `{"product_id":"` + cartAPITestProductID + `","quantity":1,"cart_id":"x"}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "body too large", contentType: "application/json", body: `{"product_id":"` + strings.Repeat("a", int(maxCartAPIRequestBytes)) + `","quantity":1}`, wantStatus: 413, wantError: "request_too_large"},
		{name: "missing product id", contentType: "application/json", body: `{"quantity":1}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "empty product id", contentType: "application/json", body: `{"product_id":" ","quantity":1}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "invalid uuid", contentType: "application/json", body: `{"product_id":"not-a-uuid","quantity":1}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "nil uuid", contentType: "application/json", body: `{"product_id":"00000000-0000-0000-0000-000000000000","quantity":1}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "missing quantity", contentType: "application/json", body: `{"product_id":"` + cartAPITestProductID + `"}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "zero quantity", contentType: "application/json", body: `{"product_id":"` + cartAPITestProductID + `","quantity":0}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "negative quantity", contentType: "application/json", body: `{"product_id":"` + cartAPITestProductID + `","quantity":-1}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "decimal quantity", contentType: "application/json", body: `{"product_id":"` + cartAPITestProductID + `","quantity":1.5}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "string quantity", contentType: "application/json", body: `{"product_id":"` + cartAPITestProductID + `","quantity":"1"}`, wantStatus: 400, wantError: "invalid_request"},
		{name: "quantity overflow", contentType: "application/json", body: `{"product_id":"` + cartAPITestProductID + `","quantity":999999999999999999999999}`, wantStatus: 400, wantError: "invalid_request"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutations := &fakeCartAPIMutationService{}
			loader := canonicalEmptyCartAPILoader()
			request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(testCase.body))
			if testCase.contentType != "" {
				request.Header.Set("Content-Type", testCase.contentType)
			}
			request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
			recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(loader, mutations)), request)
			assertCartAPIMutationHTTPResult(t, recorder, testCase.wantStatus, testCase.wantError)
			if len(mutations.calls) != testCase.wantCalls {
				t.Fatalf("expected %d operations, got %d", testCase.wantCalls, len(mutations.calls))
			}
		})
	}
}

func TestPostCartAPIItemTrimsProductIDAndUsesContextCart(t *testing.T) {
	mutations := &fakeCartAPIMutationService{}
	request := httptest.NewRequest(http.MethodPost, "/api/cart/items?cart_id="+routeTestCartID2.String(), strings.NewReader(`{"product_id":"  `+cartAPITestProductID+`  ","quantity":2}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Cart-ID", routeTestCartID2.String())
	request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
	recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(canonicalEmptyCartAPILoader(), mutations)), request)
	if recorder.Code != http.StatusOK || len(mutations.calls) != 1 {
		t.Fatalf("unexpected result: status=%d calls=%v", recorder.Code, mutations.calls)
	}
	call := mutations.calls[0]
	if call.cartID != routeTestCartID.String() || call.productID != cartAPITestProductID || call.quantity != 2 {
		t.Fatalf("untrusted identity or unnormalized input reached operation: %+v", call)
	}
}

func TestPatchCartAPIItemStrictParsing(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		productID  string
		body       string
		wantStatus int
		wantCalls  int
	}{
		{name: "valid absolute quantity", productID: cartAPITestProductID, body: `{"quantity":2}`, wantStatus: 200, wantCalls: 1},
		{name: "invalid path uuid", productID: "bad", body: `{"quantity":2}`, wantStatus: 400},
		{name: "nil path uuid", productID: "00000000-0000-0000-0000-000000000000", body: `{"quantity":2}`, wantStatus: 400},
		{name: "product id in body", productID: cartAPITestProductID, body: `{"product_id":"` + cartAPITestProductID + `","quantity":2}`, wantStatus: 400},
		{name: "missing quantity", productID: cartAPITestProductID, body: `{}`, wantStatus: 400},
		{name: "zero quantity", productID: cartAPITestProductID, body: `{"quantity":0}`, wantStatus: 400},
		{name: "negative quantity", productID: cartAPITestProductID, body: `{"quantity":-1}`, wantStatus: 400},
		{name: "decimal quantity", productID: cartAPITestProductID, body: `{"quantity":1.5}`, wantStatus: 400},
		{name: "string quantity", productID: cartAPITestProductID, body: `{"quantity":"2"}`, wantStatus: 400},
		{name: "overflow quantity", productID: cartAPITestProductID, body: `{"quantity":999999999999999999999999}`, wantStatus: 400},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutations := &fakeCartAPIMutationService{}
			request := httptest.NewRequest(http.MethodPatch, "/api/cart/items/"+testCase.productID, strings.NewReader(testCase.body))
			request.SetPathValue("product_id", testCase.productID)
			request.Header.Set("Content-Type", "application/json")
			recorder := serveCartAPIMutationHandler(t, patchCartAPIItemHandler(canonicalEmptyCartAPILoader(), mutations), request)
			assertCartAPIMutationHTTPResult(t, recorder, testCase.wantStatus, map[int]string{200: "", 400: "invalid_request"}[testCase.wantStatus])
			if len(mutations.calls) != testCase.wantCalls {
				t.Fatalf("expected %d operations, got %d", testCase.wantCalls, len(mutations.calls))
			}
			if testCase.wantCalls == 1 && mutations.calls[0].quantity != 2 {
				t.Fatalf("PATCH did not use absolute quantity: %+v", mutations.calls[0])
			}
		})
	}
}

func TestCartAPIMutationPublicErrorsAreStable(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "product missing", err: errCartAPIProductNotFound, wantStatus: 404, wantCode: "product_not_found"},
		{name: "product unavailable", err: errCartAPIProductUnavailable, wantStatus: 409, wantCode: "product_unavailable"},
		{name: "stock", err: errCartAPIInsufficientStock, wantStatus: 409, wantCode: "insufficient_stock"},
		{name: "item missing", err: errCartAPIItemNotFound, wantStatus: 404, wantCode: "cart_item_not_found"},
		{name: "database", err: errors.New("postgres://user:secret@internal"), wantStatus: 503, wantCode: "cart_unavailable"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutations := &fakeCartAPIMutationService{errors: map[string]error{"add": testCase.err}}
			request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":1}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
			recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(canonicalEmptyCartAPILoader(), mutations)), request)
			assertCartAPIMutationHTTPResult(t, recorder, testCase.wantStatus, testCase.wantCode)
			if strings.Contains(recorder.Body.String(), "postgres") || strings.Contains(recorder.Body.String(), "secret") || strings.Contains(recorder.Body.String(), "internal") {
				t.Fatalf("internal error leaked: %s", recorder.Body.String())
			}
		})
	}
}

func TestCartAPIMutationResponseReloadsPersistedCanonicalState(t *testing.T) {
	loader := &fakeCartAPIDataLoader{
		cart: &db.Cart{Items: []*db.CartItem{{ProductID: cartAPITestProductID, Quantity: 4}}},
		products: map[string]*db.CatalogProd{
			cartAPITestProductID: {ID: cartAPITestProductID, Name: "Mesa persistida", Slug: "mesa-persistida", ImageURL: " mesa.jpg ", Available: true, Quantity: 8},
		},
	}
	mutations := &fakeCartAPIMutationService{}
	request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":2}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
	recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(loader, mutations)), request)

	want := `{"cart":{"items":[{"product_id":"` + cartAPITestProductID + `","name":"Mesa persistida","slug":"mesa-persistida","image_filename":"mesa.jpg","quantity":4,"max_quantity":8,"available":true}],"total_items":1}}` + "\n"
	if recorder.Code != http.StatusOK || recorder.Body.String() != want || len(mutations.calls) != 1 {
		t.Fatalf("response was not reloaded persisted state: status=%d body=%q calls=%d", recorder.Code, recorder.Body.String(), len(mutations.calls))
	}
	if len(loader.cartIDs) != 1 || len(loader.productIDs) != 1 {
		t.Fatalf("canonical loader was not used after mutation: cart=%v product=%v", loader.cartIDs, loader.productIDs)
	}
}

func TestCartAPIMutationReloadFailureDoesNotRepeatWrite(t *testing.T) {
	mutations := &fakeCartAPIMutationService{}
	loader := &fakeCartAPIDataLoader{cartErr: errors.New("reload failed")}
	request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
	recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(loader, mutations)), request)
	assertCartAPIMutationHTTPResult(t, recorder, http.StatusServiceUnavailable, "cart_unavailable")
	if len(mutations.calls) != 1 {
		t.Fatalf("write was repeated after reload failure: %v", mutations.calls)
	}
}

func TestDeleteCartAPIHandlersAreIdempotentAndIgnoreClientCartID(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		path      string
		handler   func(cartAPIDataLoader, cartAPIMutationService) http.HandlerFunc
		operation string
	}{
		{name: "delete item", path: "/api/cart/items/" + cartAPITestProductID, operation: "delete", handler: func(loader cartAPIDataLoader, mutations cartAPIMutationService) http.HandlerFunc {
			return deleteCartAPIItemHandler(loader, mutations)
		}},
		{name: "clear cart", path: "/api/cart", operation: "clear", handler: func(loader cartAPIDataLoader, mutations cartAPIMutationService) http.HandlerFunc {
			return deleteCartAPIHandler(loader, mutations)
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mutations := &fakeCartAPIMutationService{}
			request := httptest.NewRequest(http.MethodDelete, testCase.path+"?cart_id="+routeTestCartID2.String(), strings.NewReader(`{"cart_id":"`+routeTestCartID2.String()+`"}`))
			request.Header.Set("X-Cart-ID", routeTestCartID2.String())
			if testCase.operation == "delete" {
				request.SetPathValue("product_id", cartAPITestProductID)
			}
			recorder := serveCartAPIMutationHandler(t, testCase.handler(canonicalEmptyCartAPILoader(), mutations), request)
			if recorder.Code != http.StatusOK || len(mutations.calls) != 1 || mutations.calls[0].cartID != routeTestCartID.String() {
				t.Fatalf("unexpected scoped delete: status=%d calls=%+v", recorder.Code, mutations.calls)
			}
		})
	}
}

func TestDeleteCartAPIItemRejectsInvalidUUIDBeforeOperation(t *testing.T) {
	mutations := &fakeCartAPIMutationService{}
	request := httptest.NewRequest(http.MethodDelete, "/api/cart/items/bad", nil)
	request.SetPathValue("product_id", "bad")
	recorder := serveCartAPIMutationHandler(t, deleteCartAPIItemHandler(canonicalEmptyCartAPILoader(), mutations), request)
	assertCartAPIMutationHTTPResult(t, recorder, http.StatusBadRequest, "invalid_request")
	if len(mutations.calls) != 0 {
		t.Fatalf("invalid UUID executed operation: %v", mutations.calls)
	}
}

func TestCartAPIMutationRoutesRequireCSRFFirst(t *testing.T) {
	for _, testCase := range cartAPIMutationTestRoutes() {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			manager := newDeterministicRouteCartManager(t, routeTestCartID)
			mutations := &fakeCartAPIMutationService{}
			router := NewCustomServeMux()
			registerCartAPIMutationRoutes(router, manager, newRouteTestCSRFGuard(t), canonicalEmptyCartAPILoader(), mutations)
			request := cartAPIMutationRouteRequest(testCase)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			assertRouteCSRFRejected(t, recorder)
			if len(recorder.Header().Values("Set-Cookie")) != 0 || len(mutations.calls) != 0 {
				t.Fatalf("CSRF rejection ran session or operation: cookies=%v calls=%v", recorder.Header().Values("Set-Cookie"), mutations.calls)
			}
		})
	}
}

func TestCartAPIMutationRoutesAllowTrustedOriginAndReferer(t *testing.T) {
	for _, header := range []string{"Origin", "Referer"} {
		for _, testCase := range cartAPIMutationTestRoutes() {
			t.Run(header+" "+testCase.method+" "+testCase.path, func(t *testing.T) {
				manager := newDeterministicRouteCartManager(t, routeTestCartID)
				mutations := &fakeCartAPIMutationService{}
				router := NewCustomServeMux()
				registerCartAPIMutationRoutes(router, manager, newRouteTestCSRFGuard(t), canonicalEmptyCartAPILoader(), mutations)
				request := cartAPIMutationRouteRequest(testCase)
				if header == "Origin" {
					request.Header.Set("Origin", "http://localhost:8080")
				} else {
					request.Header.Set("Referer", "http://localhost:8080/catalogo?test=1")
				}
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)

				if recorder.Code != http.StatusOK || len(mutations.calls) != 1 || mutations.calls[0].cartID != routeTestCartID.String() {
					t.Fatalf("trusted request failed: status=%d calls=%+v body=%s", recorder.Code, mutations.calls, recorder.Body.String())
				}
				if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
					t.Fatalf("first trusted request expected one signed cookie, got %d", got)
				}
			})
		}
	}
}

func TestCartAPIMutationRouteKeepsValidCookieAndReplacesTamperedCookie(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		cookie      *http.Cookie
		wantID      string
		wantCookies int
	}{
		{
			name:        "valid cookie",
			cookie:      issueRouteCartCookie(t, newDeterministicRouteCartManager(t, routeTestCartID)),
			wantID:      routeTestCartID.String(),
			wantCookies: 0,
		},
		{
			name:        "tampered cookie",
			cookie:      &http.Cookie{Name: session.CartCookieName, Value: routeTestCartID.String()},
			wantID:      routeTestCartID2.String(),
			wantCookies: 1,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			generated := routeTestCartID2
			manager := newDeterministicRouteCartManager(t, generated)
			mutations := &fakeCartAPIMutationService{}
			router := NewCustomServeMux()
			registerCartAPIMutationRoutes(router, manager, newRouteTestCSRFGuard(t), canonicalEmptyCartAPILoader(), mutations)
			request := httptest.NewRequest(http.MethodDelete, "/api/cart", nil)
			request.Header.Set("Origin", "http://localhost:8080")
			request.AddCookie(testCase.cookie)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK || len(mutations.calls) != 1 || mutations.calls[0].cartID != testCase.wantID {
				t.Fatalf("unexpected session resolution: status=%d calls=%+v", recorder.Code, mutations.calls)
			}
			if got := len(recorder.Header().Values("Set-Cookie")); got != testCase.wantCookies {
				t.Fatalf("expected %d cookies, got %d", testCase.wantCookies, got)
			}
		})
	}
}

func TestCartAPIMutationOriginIsAuthoritative(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	mutations := &fakeCartAPIMutationService{}
	router := NewCustomServeMux()
	registerCartAPIMutationRoutes(router, manager, newRouteTestCSRFGuard(t), canonicalEmptyCartAPILoader(), mutations)
	request := httptest.NewRequest(http.MethodDelete, "/api/cart", nil)
	request.Header.Set("Origin", "https://attacker.test")
	request.Header.Set("Referer", "http://localhost:8080/catalogo")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	assertRouteCSRFRejected(t, recorder)
	if len(mutations.calls) != 0 || len(recorder.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("untrusted Origin fell back to trusted Referer")
	}
}

func TestCartAPIMutationRoutePatternsAndRegressions(t *testing.T) {
	router := newTestRouter(t)
	for _, testCase := range []struct {
		method  string
		path    string
		pattern string
	}{
		{method: http.MethodGet, path: "/api/cart", pattern: "GET /api/cart"},
		{method: http.MethodPost, path: "/api/cart/items", pattern: "POST /api/cart/items"},
		{method: http.MethodPatch, path: "/api/cart/items/" + cartAPITestProductID, pattern: "PATCH /api/cart/items/{product_id}"},
		{method: http.MethodDelete, path: "/api/cart/items/" + cartAPITestProductID, pattern: "DELETE /api/cart/items/{product_id}"},
		{method: http.MethodDelete, path: "/api/cart", pattern: "DELETE /api/cart"},
		{method: http.MethodPost, path: "/api/contact-requests", pattern: "POST /api/contact-requests"},
		{method: http.MethodGet, path: "/api/catalog/products", pattern: "GET /api/catalog/products"},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		_, pattern := router.Handler(request)
		if pattern != testCase.pattern {
			t.Errorf("%s %s: expected %q, got %q", testCase.method, testCase.path, testCase.pattern, pattern)
		}
	}
	for _, testCase := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPut, path: "/api/cart"},
		{method: http.MethodPost, path: "/api/cart"},
		{method: http.MethodPut, path: "/api/cart/items"},
		{method: http.MethodPost, path: "/api/cart/items/" + cartAPITestProductID},
	} {
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		_, pattern := router.Handler(request)
		if pattern != "" {
			t.Errorf("unexpected route registered for %s %s: %q", testCase.method, testCase.path, pattern)
		}
	}
}

func canonicalEmptyCartAPILoader() *fakeCartAPIDataLoader {
	return &fakeCartAPIDataLoader{cartErr: db.ErrCartNotFound}
}

func serveCartAPIMutationHandler(t *testing.T, handler http.HandlerFunc, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	request.AddCookie(issueRouteCartCookie(t, manager))
	recorder := httptest.NewRecorder()
	withCartSession(manager, handler)(recorder, request)
	return recorder
}

func assertCartAPIMutationHTTPResult(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantError string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("expected status %d, got %d: %s", wantStatus, recorder.Code, recorder.Body.String())
	}
	assertCartAPIHeaders(t, recorder)
	if wantError != "" {
		wantBody := `{"error":"` + wantError + `"}` + "\n"
		if recorder.Body.String() != wantBody {
			t.Fatalf("expected body %q, got %q", wantBody, recorder.Body.String())
		}
	}
}

type cartAPIMutationRouteCase struct {
	method string
	path   string
	body   string
}

func cartAPIMutationTestRoutes() []cartAPIMutationRouteCase {
	return []cartAPIMutationRouteCase{
		{method: http.MethodPost, path: "/api/cart/items", body: `{"product_id":"` + cartAPITestProductID + `","quantity":1}`},
		{method: http.MethodPatch, path: "/api/cart/items/" + cartAPITestProductID, body: `{"quantity":2}`},
		{method: http.MethodDelete, path: "/api/cart/items/" + cartAPITestProductID},
		{method: http.MethodDelete, path: "/api/cart"},
	}
}

func cartAPIMutationRouteRequest(testCase cartAPIMutationRouteCase) *http.Request {
	request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
	if testCase.method == http.MethodPost || testCase.method == http.MethodPatch {
		request.Header.Set("Content-Type", "application/json")
	}
	if testCase.method == http.MethodPost {
		request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
	}
	return request
}

// --- Idempotency-Key header validation (section 18) ---

func TestValidateCartAPIIdempotencyKeyHeader(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		setup   func(*http.Request)
		wantErr error
		wantKey string
	}{
		{name: "valid header", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey) }, wantKey: cartAPITestIdempotencyKey},
		{name: "absent", setup: func(r *http.Request) {}, wantErr: errCartAPIIdempotencyKeyRequired},
		{name: "empty value", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "") }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "too short (15 chars)", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "abcdefghijklmno") }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "exactly 16 chars", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "abcdefghijklmnop") }, wantKey: "abcdefghijklmnop"},
		{name: "exactly 128 chars", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", strings.Repeat("a", 128)) }, wantKey: strings.Repeat("a", 128)},
		{name: "129 chars", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", strings.Repeat("a", 129)) }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "exterior leading space", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", " "+cartAPITestIdempotencyKey) }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "exterior trailing space", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey+" ") }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "interior space", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "test idem key 001") }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "unicode", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "clave-idempotente-ñ01") }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "NUL byte", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "abcdefgh\x00ijklmnop") }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "comma", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "abcdefgh,ijklmnop123") }, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "multiple headers", setup: func(r *http.Request) {
			r.Header.Add("Idempotency-Key", cartAPITestIdempotencyKey)
			r.Header.Add("Idempotency-Key", cartAPITestIdempotencyKey)
		}, wantErr: errCartAPIInvalidIdempotencyKey},
		{name: "allowed charset: dot dash underscore colon", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "abc.def-ghi_jkl:001") }, wantKey: "abc.def-ghi_jkl:001"},
		{name: "disallowed character (slash)", setup: func(r *http.Request) { r.Header.Set("Idempotency-Key", "abcdefgh/ijklmnop123") }, wantErr: errCartAPIInvalidIdempotencyKey},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/cart/items", nil)
			testCase.setup(request)
			key, err := validateCartAPIIdempotencyKeyHeader(request)
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("expected %v, got %v", testCase.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if key != testCase.wantKey {
				t.Fatalf("expected key %q, got %q (value was not corrected, only accepted or rejected)", testCase.wantKey, key)
			}
		})
	}
}

// requireCartAPIIdempotencyKey must reject before session, before Set-Cookie,
// before the handler, before any product read or transaction.
func TestRequireCartAPIIdempotencyKeyRejectsBeforeSessionAndHandler(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		header  string
		wantErr string
	}{
		{name: "missing", header: "", wantErr: "idempotency_key_required"},
		{name: "malformed", header: "too-short", wantErr: "invalid_idempotency_key"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			manager := newDeterministicRouteCartManager(t, routeTestCartID)
			handlerCalled := false
			next := func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
			}

			request := httptest.NewRequest(http.MethodPost, "/api/cart/items", nil)
			if testCase.header != "" {
				request.Header.Set("Idempotency-Key", testCase.header)
			}
			recorder := httptest.NewRecorder()
			requireCartAPIIdempotencyKey(withCartSession(manager, next))(recorder, request)

			assertCartAPIMutationHTTPResult(t, recorder, http.StatusBadRequest, testCase.wantErr)
			if handlerCalled {
				t.Fatal("the handler must not run when the idempotency key is missing or invalid")
			}
			if len(recorder.Header().Values("Set-Cookie")) != 0 {
				t.Fatal("no cart session cookie may be issued when the idempotency key is rejected")
			}
		})
	}
}

func TestCartAPIMutationCSRFRunsBeforeIdempotencyKeyGuard(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	mutations := &fakeCartAPIMutationService{}
	router := NewCustomServeMux()
	registerCartAPIMutationRoutes(router, manager, newRouteTestCSRFGuard(t), canonicalEmptyCartAPILoader(), mutations)

	// No Origin/Referer and no Idempotency-Key: if the CSRF guard runs
	// first, as required, the response is the generic CSRF rejection, not
	// idempotency_key_required.
	request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":1}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assertRouteCSRFRejected(t, recorder)
	if strings.Contains(recorder.Body.String(), "idempotency") {
		t.Fatalf("expected the CSRF rejection, not an idempotency error: %s", recorder.Body.String())
	}
	if len(mutations.calls) != 0 {
		t.Fatal("CSRF rejection must not reach the mutation")
	}
}

func TestCartAPIMutationInvalidIdempotencyKeyRejectedBeforeSessionInFullRouter(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	mutations := &fakeCartAPIMutationService{}
	router := NewCustomServeMux()
	registerCartAPIMutationRoutes(router, manager, newRouteTestCSRFGuard(t), canonicalEmptyCartAPILoader(), mutations)

	request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:8080")
	// Trusted origin, so CSRF passes; no Idempotency-Key header at all.
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assertCartAPIMutationHTTPResult(t, recorder, http.StatusBadRequest, "idempotency_key_required")
	if len(recorder.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("a missing idempotency key must not trigger cart session creation")
	}
	if len(mutations.calls) != 0 {
		t.Fatal("a missing idempotency key must not reach the mutation")
	}
}

// PATCH and DELETE must not require the header. Each request below carries
// an invalid path UUID so the handler rejects it with invalid_request before
// ever reaching the database — the point is only to confirm the response is
// never idempotency_key_required, without needing a live PostgreSQL
// connection. GET /api/cart is registered by a wholly separate function
// (registerCartAPIRoutes in cart_api.go, untouched by this phase) and is not
// exercised here.
func TestOnlyPostCartItemsRequiresIdempotencyKey(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	mutations := &fakeCartAPIMutationService{}
	router := NewCustomServeMux()
	registerCartAPIMutationRoutes(router, manager, newRouteTestCSRFGuard(t), canonicalEmptyCartAPILoader(), mutations)

	for _, testCase := range []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPatch, path: "/api/cart/items/not-a-uuid", body: `{"quantity":1}`},
		{method: http.MethodDelete, path: "/api/cart/items/not-a-uuid"},
		{method: http.MethodDelete, path: "/api/cart"},
	} {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
			if testCase.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			request.Header.Set("Origin", "http://localhost:8080")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if strings.Contains(recorder.Body.String(), "idempotency") {
				t.Fatalf("%s %s unexpectedly required an idempotency key: %s", testCase.method, testCase.path, recorder.Body.String())
			}
		})
	}
}

// --- Hashing (section 19) ---

func TestHashCartAPIIdempotencyKey(t *testing.T) {
	first := hashCartAPIIdempotencyKey("key-one-0123456789")
	same := hashCartAPIIdempotencyKey("key-one-0123456789")
	different := hashCartAPIIdempotencyKey("key-two-0123456789")

	if len(first) != 32 {
		t.Fatalf("expected a 32-byte digest, got %d bytes", len(first))
	}
	if !bytesEqual(first, same) {
		t.Fatal("the same key must hash to the same digest")
	}
	if bytesEqual(first, different) {
		t.Fatal("a different key must hash to a different digest")
	}
	if strings.Contains(string(first), "key-one") {
		t.Fatal("the raw key must not appear inside its own hash")
	}
}

func TestHashCartAPIAddItemRequest(t *testing.T) {
	base := hashCartAPIAddItemRequest(cartAPITestProductID, 2)
	sameAgain := hashCartAPIAddItemRequest(cartAPITestProductID, 2)
	differentProduct := hashCartAPIAddItemRequest(routeTestCartID2.String(), 2)
	differentQuantity := hashCartAPIAddItemRequest(cartAPITestProductID, 3)

	if len(base) != 32 {
		t.Fatalf("expected a 32-byte digest, got %d bytes", len(base))
	}
	if !bytesEqual(base, sameAgain) {
		t.Fatal("the same product and quantity must hash to the same digest")
	}
	if bytesEqual(base, differentProduct) {
		t.Fatal("a different product must change the hash")
	}
	if bytesEqual(base, differentQuantity) {
		t.Fatal("a different quantity must change the hash")
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- Route-level idempotency integration (section 21) ---

func TestPostCartAPIItemNewKeyApplies(t *testing.T) {
	mutations := &fakeCartAPIMutationService{outcome: cart.AddItemApplied}
	request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
	recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(canonicalEmptyCartAPILoader(), mutations)), request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Idempotency-Replayed"); got != "" {
		t.Fatalf("a freshly applied operation must not carry Idempotency-Replayed, got %q", got)
	}
}

func TestPostCartAPIItemReplaySetsHeaderAndDoesNotErrorTwice(t *testing.T) {
	mutations := &fakeCartAPIMutationService{outcome: cart.AddItemReplayed}
	request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
	recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(canonicalEmptyCartAPILoader(), mutations)), request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("a replay must still respond 200 with the canonical cart, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Idempotency-Replayed"); got != "true" {
		t.Fatalf("expected Idempotency-Replayed: true, got %q", got)
	}
	for _, forbidden := range []string{"replayed", "idempotency_key", "operation_id", "duplicate"} {
		if strings.Contains(recorder.Body.String(), forbidden) {
			t.Fatalf("response body must not add new JSON fields, found %q: %s", forbidden, recorder.Body.String())
		}
	}
}

func TestPostCartAPIItemConflictReturns409(t *testing.T) {
	mutations := &fakeCartAPIMutationService{errors: map[string]error{"add": errCartAPIIdempotencyConflict}}
	request := httptest.NewRequest(http.MethodPost, "/api/cart/items", strings.NewReader(`{"product_id":"`+cartAPITestProductID+`","quantity":1}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", cartAPITestIdempotencyKey)
	recorder := serveCartAPIMutationHandler(t, requireCartAPIIdempotencyKey(postCartAPIItemHandler(canonicalEmptyCartAPILoader(), mutations)), request)

	assertCartAPIMutationHTTPResult(t, recorder, http.StatusConflict, "idempotency_conflict")
}
