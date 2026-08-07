package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCategoriesMutationsRequireAuth closes the finding from 08-final-readiness.md:
// POST/PUT/DELETE /api/categories were registered without auth.ValidateAuth.
// auth.ValidateAuth rejects (302 to /iniciar-sesion) before ever calling the
// wrapped handler, so this is safe to assert without a live database — a
// request that reaches CreateCategory/UpdateCategory/DeleteCategory without
// a valid auth_token would panic on the nil DB pool, exactly like every
// other DB-backed handler in this package when unauthenticated.
func TestCategoriesMutationsRequireAuth(t *testing.T) {
	router := NewCustomServeMux()
	RegisterCategoriesRoutes(router)

	for _, testCase := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "POST without auth rejected", method: http.MethodPost, path: "/api/categories"},
		{name: "PUT without auth rejected", method: http.MethodPut, path: "/api/categories/11111111-1111-1111-1111-111111111111"},
		{name: "DELETE without auth rejected", method: http.MethodDelete, path: "/api/categories/11111111-1111-1111-1111-111111111111"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code == http.StatusOK {
				t.Fatalf("expected an unauthenticated %s to be rejected, got 200", testCase.method)
			}
			if recorder.Code != http.StatusFound {
				t.Fatalf("expected the standard unauthenticated redirect (302), got %d", recorder.Code)
			}
			if location := recorder.Header().Get("Location"); location != "/iniciar-sesion" {
				t.Fatalf("expected redirect to /iniciar-sesion, got %q", location)
			}
		})
	}
}

// TestCategoriesInvalidAuthCookieRejected confirms a malformed/garbage
// auth_token cookie is treated the same as no cookie at all — never reaches
// the handler.
func TestCategoriesInvalidAuthCookieRejected(t *testing.T) {
	router := NewCustomServeMux()
	RegisterCategoriesRoutes(router)

	request := httptest.NewRequest(http.MethodPost, "/api/categories", nil)
	request.AddCookie(&http.Cookie{Name: "auth_token", Value: "not-a-real-jwt"})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("expected an invalid token to be rejected with 302, got %d", recorder.Code)
	}
}

// TestCategoriesGetRemainsPublic confirms the read-only listing was not
// swept into the auth fix — only the three mutations were unauthenticated,
// GET was never the finding.
func TestCategoriesGetRemainsPublic(t *testing.T) {
	router := NewCustomServeMux()
	RegisterCategoriesRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	_, pattern := router.Handler(request)
	if pattern != "GET /api/categories" {
		t.Fatalf("expected GET /api/categories to still be registered, got pattern %q", pattern)
	}
}

// TestCategoriesUnregisteredMethodsDoNotMatch confirms no stray pattern
// collision was introduced by the auth wrapping.
func TestCategoriesUnregisteredMethodsDoNotMatch(t *testing.T) {
	router := NewCustomServeMux()
	RegisterCategoriesRoutes(router)

	request := httptest.NewRequest(http.MethodPatch, "/api/categories", nil)
	_, pattern := router.Handler(request)
	if pattern != "" {
		t.Fatalf("expected PATCH /api/categories to match nothing, got pattern %q", pattern)
	}
}

// TestQuotesMutationsRequireAuth closes the same class of finding in
// internal/routes/quotes.go: POST/PUT /api/quotes had no consumer anywhere
// in the repo and no auth gate.
func TestQuotesMutationsRequireAuth(t *testing.T) {
	router := NewCustomServeMux()
	RegisterQuotesRoutes(router)

	for _, testCase := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "POST without auth rejected", method: http.MethodPost, path: "/api/quotes"},
		{name: "PUT without auth rejected", method: http.MethodPut, path: "/api/quotes/11111111-1111-1111-1111-111111111111"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusFound {
				t.Fatalf("expected the standard unauthenticated redirect (302), got %d", recorder.Code)
			}
		})
	}
}
