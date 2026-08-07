package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

const cartAPITestProductID = "01890f3a-dc02-7cb5-a4cc-451231879f0b"

type fakeCartAPIDataLoader struct {
	cart          *db.Cart
	cartErr       error
	products      map[string]*db.CatalogProd
	productErrors map[string]error
	cartIDs       []string
	productIDs    []string
}

func (loader *fakeCartAPIDataLoader) FindCartByID(_ context.Context, cartID string) (*db.Cart, error) {
	loader.cartIDs = append(loader.cartIDs, cartID)
	return loader.cart, loader.cartErr
}

func (loader *fakeCartAPIDataLoader) FindCatalogProductDetail(productID string) (*db.CatalogProd, error) {
	loader.productIDs = append(loader.productIDs, productID)
	if err := loader.productErrors[productID]; err != nil {
		return nil, err
	}
	return loader.products[productID], nil
}

func TestGetCartAPIResponseContract(t *testing.T) {
	loader := &fakeCartAPIDataLoader{
		cart: &db.Cart{Items: []*db.CartItem{{
			ProductID: cartAPITestProductID,
			Quantity:  2,
			Name:      "stale name",
			ImageURL:  "stale.jpg",
			MaxQty:    99,
		}}},
		products: map[string]*db.CatalogProd{
			cartAPITestProductID: {
				ID:        cartAPITestProductID,
				Name:      "Mesa redonda",
				Slug:      "mesa-redonda",
				ImageURL:  "  upload_2026-01-01T12:00:00_0.jpg  ",
				Available: true,
				Quantity:  10,
			},
		},
	}
	recorder := serveCartAPIHandler(t, loader)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	assertCartAPIHeaders(t, recorder)

	want := `{"cart":{"items":[{"product_id":"01890f3a-dc02-7cb5-a4cc-451231879f0b","name":"Mesa redonda","slug":"mesa-redonda","image_filename":"upload_2026-01-01T12:00:00_0.jpg","quantity":2,"max_quantity":10,"available":true}],"total_items":1}}` + "\n"
	if recorder.Body.String() != want {
		t.Fatalf("unexpected response body:\nwant %s\ngot  %s", want, recorder.Body.String())
	}
	if len(loader.cartIDs) != 1 || loader.cartIDs[0] != routeTestCartID.String() {
		t.Fatalf("loader did not receive context identity: %v", loader.cartIDs)
	}
	if len(loader.productIDs) != 1 || loader.productIDs[0] != cartAPITestProductID {
		t.Fatalf("unexpected product lookup: %v", loader.productIDs)
	}
}

func TestGetCartAPITotalItemsCountsLinesNotUnits(t *testing.T) {
	secondProductID := "01890f3a-dc03-7ec7-b717-ccb03445e224"
	loader := &fakeCartAPIDataLoader{
		cart: &db.Cart{Items: []*db.CartItem{
			{ProductID: cartAPITestProductID, Quantity: 3},
			{ProductID: secondProductID, Quantity: 4},
		}},
		products: map[string]*db.CatalogProd{
			cartAPITestProductID: {ID: cartAPITestProductID, Name: "Mesa", Slug: "mesa", Available: true, Quantity: 5},
			secondProductID:      {ID: secondProductID, Name: "Silla", Slug: "silla", Available: true, Quantity: 8},
		},
	}
	recorder := serveCartAPIHandler(t, loader)

	var response cartAPIResponse
	decodeCartAPIResponse(t, recorder, &response)
	if response.Cart.TotalItems != 2 || len(response.Cart.Items) != 2 {
		t.Fatalf("expected two distinct lines, got total=%d items=%d", response.Cart.TotalItems, len(response.Cart.Items))
	}
}

func TestGetCartAPIRevalidatesCurrentCatalogState(t *testing.T) {
	loader := &fakeCartAPIDataLoader{
		cart: &db.Cart{Items: []*db.CartItem{{ProductID: cartAPITestProductID, Quantity: 2, MaxQty: 10}}},
		products: map[string]*db.CatalogProd{
			cartAPITestProductID: {ID: cartAPITestProductID, Name: "Mesa", Slug: "mesa", Available: true, Quantity: 0},
		},
	}
	recorder := serveCartAPIHandler(t, loader)

	var response cartAPIResponse
	decodeCartAPIResponse(t, recorder, &response)
	item := response.Cart.Items[0]
	if item.Available || item.MaxQuantity != 0 || item.Quantity != 2 {
		t.Fatalf("catalog state was not revalidated: %+v", item)
	}
}

func TestGetCartAPIMissingCartAndEmptyCartReturnCanonicalEmptyState(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		loader *fakeCartAPIDataLoader
	}{
		{name: "missing cart", loader: &fakeCartAPIDataLoader{cartErr: db.ErrCartNotFound}},
		{name: "empty cart", loader: &fakeCartAPIDataLoader{cart: &db.Cart{Items: make([]*db.CartItem, 0)}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serveCartAPIHandler(t, testCase.loader)
			if recorder.Code != http.StatusOK || recorder.Body.String() != `{"cart":{"items":[],"total_items":0}}`+"\n" {
				t.Fatalf("unexpected empty response: status=%d body=%q", recorder.Code, recorder.Body.String())
			}
			if len(testCase.loader.productIDs) != 0 {
				t.Fatalf("empty cart triggered product reads: %v", testCase.loader.productIDs)
			}
		})
	}
}

func TestGetCartAPIFailuresAreControlledAndDoNotLeakDetails(t *testing.T) {
	secretError := errors.New("dial postgres://internal-user:secret@db.internal/private")
	for _, testCase := range []struct {
		name        string
		loader      *fakeCartAPIDataLoader
		withContext bool
		wantStatus  int
	}{
		{
			name:        "cart database failure",
			loader:      &fakeCartAPIDataLoader{cartErr: secretError},
			withContext: true,
			wantStatus:  http.StatusServiceUnavailable,
		},
		{
			name: "product database failure",
			loader: &fakeCartAPIDataLoader{
				cart:          &db.Cart{Items: []*db.CartItem{{ProductID: cartAPITestProductID}}},
				productErrors: map[string]error{cartAPITestProductID: secretError},
			},
			withContext: true,
			wantStatus:  http.StatusServiceUnavailable,
		},
		{
			name: "missing product",
			loader: &fakeCartAPIDataLoader{
				cart:     &db.Cart{Items: []*db.CartItem{{ProductID: cartAPITestProductID}}},
				products: map[string]*db.CatalogProd{},
			},
			withContext: true,
			wantStatus:  http.StatusServiceUnavailable,
		},
		{
			name:        "missing session context",
			loader:      &fakeCartAPIDataLoader{},
			withContext: false,
			wantStatus:  http.StatusInternalServerError,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/cart", nil)
			recorder := httptest.NewRecorder()
			handler := getCartAPIHandler(testCase.loader)
			if testCase.withContext {
				manager := newDeterministicRouteCartManager(t, routeTestCartID)
				request.AddCookie(issueRouteCartCookie(t, manager))
				handler = withCartSession(manager, handler)
			}
			handler(recorder, request)

			if recorder.Code != testCase.wantStatus || recorder.Body.String() != `{"error":"cart_unavailable"}`+"\n" {
				t.Fatalf("unexpected controlled error: status=%d body=%q", recorder.Code, recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "postgres") || strings.Contains(recorder.Body.String(), "secret") || strings.Contains(recorder.Body.String(), "internal") {
				t.Fatalf("internal detail leaked: %q", recorder.Body.String())
			}
			if !testCase.withContext && len(testCase.loader.cartIDs) != 0 {
				t.Fatalf("loader called without authenticated context: %v", testCase.loader.cartIDs)
			}
		})
	}
}

func TestGetCartAPIExposesOnlyCanonicalFields(t *testing.T) {
	loader := &fakeCartAPIDataLoader{
		cart: &db.Cart{
			ID:            routeTestCartID.String(),
			CustomerName:  "private name",
			CustomerEmail: "private@example.test",
			Items: []*db.CartItem{{
				ProductID: cartAPITestProductID,
				Quantity:  1,
				Category:  "private category",
				Source:    "private source",
			}},
		},
		products: map[string]*db.CatalogProd{
			cartAPITestProductID: {
				ID:              cartAPITestProductID,
				Name:            "Mesa",
				Slug:            "mesa",
				Description:     "private description",
				LongDescription: "private long description",
				CategoryName:    "private category",
				Available:       true,
				Quantity:        4,
			},
		},
	}
	recorder := serveCartAPIHandler(t, loader)

	var raw map[string]any
	decodeCartAPIResponse(t, recorder, &raw)
	assertExactJSONKeys(t, raw, "cart")
	cart := raw["cart"].(map[string]any)
	assertExactJSONKeys(t, cart, "items", "total_items")
	item := cart["items"].([]any)[0].(map[string]any)
	assertExactJSONKeys(t, item, "product_id", "name", "slug", "image_filename", "quantity", "max_quantity", "available")

	body := recorder.Body.String()
	for _, forbidden := range []string{routeTestCartID.String(), "private name", "private@example.test", "private category", "private source", "private description"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("forbidden value %q leaked in %s", forbidden, body)
		}
	}
}

func TestCartAPIRouteUsesSignedSessionIdentityOnly(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	loader := &fakeCartAPIDataLoader{cartErr: db.ErrCartNotFound}
	router := NewCustomServeMux()
	registerCartAPIRoutes(router, manager, loader)

	request := httptest.NewRequest(http.MethodGet, "/api/cart?cart_id="+routeTestCartID2.String(), strings.NewReader(`{"cart_id":"`+routeTestCartID2.String()+`"}`))
	request.Header.Set("X-Cart-ID", routeTestCartID2.String())
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || len(loader.cartIDs) != 1 || loader.cartIDs[0] != routeTestCartID.String() {
		t.Fatalf("client-selected identity affected lookup: status=%d ids=%v", recorder.Code, loader.cartIDs)
	}
	if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
		t.Fatalf("first request expected one signed cookie, got %d", got)
	}
	if strings.Contains(recorder.Body.String(), routeTestCartID.String()) || strings.Contains(recorder.Body.String(), routeTestCartID2.String()) {
		t.Fatalf("cart identity leaked in response: %s", recorder.Body.String())
	}
}

func TestCartAPIRouteKeepsValidCookieWithoutRenewal(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID2)
	validCookie := issueRouteCartCookie(t, newDeterministicRouteCartManager(t, routeTestCartID))
	loader := &fakeCartAPIDataLoader{cartErr: db.ErrCartNotFound}
	router := NewCustomServeMux()
	registerCartAPIRoutes(router, manager, loader)

	request := httptest.NewRequest(http.MethodGet, "/api/cart", nil)
	request.AddCookie(validCookie)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || len(loader.cartIDs) != 1 || loader.cartIDs[0] != routeTestCartID.String() {
		t.Fatalf("valid session identity changed: status=%d ids=%v", recorder.Code, loader.cartIDs)
	}
	if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Fatalf("valid cookie was renewed: %v", got)
	}
}

func TestCartAPIRouteReplacesUntrustedCookieBeforeLookup(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID2)
	loader := &fakeCartAPIDataLoader{cartErr: db.ErrCartNotFound}
	router := NewCustomServeMux()
	registerCartAPIRoutes(router, manager, loader)

	request := httptest.NewRequest(http.MethodGet, "/api/cart", nil)
	request.AddCookie(&http.Cookie{Name: session.CartCookieName, Value: routeTestCartID.String()})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || len(loader.cartIDs) != 1 || loader.cartIDs[0] != routeTestCartID2.String() {
		t.Fatalf("untrusted cookie reached lookup: status=%d ids=%v", recorder.Code, loader.cartIDs)
	}
	if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
		t.Fatalf("invalid cookie expected one replacement, got %d", got)
	}
}

func TestCartAPIRouteRegistersOnlyGETWithoutCSRF(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	loader := &fakeCartAPIDataLoader{cartErr: db.ErrCartNotFound}
	router := NewCustomServeMux()
	registerCartAPIRoutes(router, manager, loader)

	getRequest := httptest.NewRequest(http.MethodGet, "/api/cart", nil)
	_, pattern := router.Handler(getRequest)
	if pattern != "GET /api/cart" {
		t.Fatalf("expected exact GET pattern, got %q", pattern)
	}
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("GET without Origin or Referer was blocked: %d", getRecorder.Code)
	}

	for _, testCase := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/cart"},
		{method: http.MethodPut, path: "/api/cart"},
		{method: http.MethodPatch, path: "/api/cart"},
		{method: http.MethodDelete, path: "/api/cart"},
		{method: http.MethodGet, path: "/api/cart/items"},
		{method: http.MethodPost, path: "/api/cart/items"},
	} {
		t.Run(testCase.method+" "+testCase.path, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			_, gotPattern := router.Handler(request)
			if gotPattern != "" {
				t.Fatalf("unexpected registered pattern %q", gotPattern)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
				t.Fatalf("unregistered route ran session middleware: %v", got)
			}
		})
	}
	if len(loader.cartIDs) != 1 {
		t.Fatalf("unregistered routes reached loader: %v", loader.cartIDs)
	}
}

func TestRealRouterRegistersCartAPIWithoutChangingExistingReadAPIs(t *testing.T) {
	router := newTestRouter(t)
	for _, testCase := range []struct {
		path    string
		pattern string
	}{
		{path: "/api/cart", pattern: "GET /api/cart"},
		{path: "/api/_health", pattern: "GET /api/_health"},
		{path: "/api/socials", pattern: "GET /api/socials"},
		{path: "/api/catalog/listings", pattern: "GET /api/catalog/listings"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		_, pattern := router.Handler(request)
		if pattern != testCase.pattern {
			t.Errorf("%s: expected %q, got %q", testCase.path, testCase.pattern, pattern)
		}
	}
}

func serveCartAPIHandler(t *testing.T, loader cartAPIDataLoader) *httptest.ResponseRecorder {
	t.Helper()
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	request := httptest.NewRequest(http.MethodGet, "/api/cart", nil)
	request.AddCookie(issueRouteCartCookie(t, manager))
	recorder := httptest.NewRecorder()
	withCartSession(manager, getCartAPIHandler(loader))(recorder, request)
	return recorder
}

func assertCartAPIHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		recorder.Header().Get("Cache-Control") != "no-store" ||
		recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected headers: %v", recorder.Header())
	}
}

func decodeCartAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
}

func assertExactJSONKeys(t *testing.T, object map[string]any, expected ...string) {
	t.Helper()
	if len(object) != len(expected) {
		t.Fatalf("expected keys %v, got %v", expected, object)
	}
	for _, key := range expected {
		if _, ok := object[key]; !ok {
			t.Fatalf("missing key %q in %v", key, object)
		}
	}
}
