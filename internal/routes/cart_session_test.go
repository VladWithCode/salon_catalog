package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

var (
	routeTestNow     = time.Date(2026, time.August, 6, 14, 0, 0, 0, time.UTC)
	routeTestCartID  = uuid.MustParse("01890f3a-dc00-7cc2-98c4-dc0c0c07398f")
	routeTestCartID2 = uuid.MustParse("01890f3a-dc01-7a31-a7b1-1b57a85c5412")
)

func TestCartSessionMiddlewareSuppliesSameAuthenticatedIDToCartWizardAndQuote(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	cookie := issueRouteCartCookie(t, manager)

	for _, flow := range []string{"cart", "wizard", "quote"} {
		t.Run(flow, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/"+flow, nil)
			request.AddCookie(cookie)
			recorder := httptest.NewRecorder()
			var loaderID uuid.UUID

			withCartSession(manager, func(w http.ResponseWriter, r *http.Request) {
				var ok bool
				loaderID, ok = session.CartIDFromContext(r.Context())
				if !ok {
					t.Error("authenticated cart ID missing from route context")
				}
				w.WriteHeader(http.StatusNoContent)
			})(recorder, request)

			if recorder.Code != http.StatusNoContent {
				t.Fatalf("expected existing response status, got %d", recorder.Code)
			}
			if loaderID != routeTestCartID {
				t.Fatalf("expected UUID %s, got %s", routeTestCartID, loaderID)
			}
			if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
				t.Fatalf("valid identity was renewed: %v", got)
			}
		})
	}
}

func TestCartSessionMiddlewareNeverPassesClientChosenIdentityToLoader(t *testing.T) {
	validManager := newDeterministicRouteCartManager(t, routeTestCartID)
	validCookie := issueRouteCartCookie(t, validManager)
	tamperedCookie := validCookie.Value[:len(validCookie.Value)-1] + "A"
	if strings.HasSuffix(validCookie.Value, "A") {
		tamperedCookie = validCookie.Value[:len(validCookie.Value)-1] + "B"
	}

	testCases := []struct {
		name  string
		value string
	}{
		{name: "legacy raw UUID from another cart", value: routeTestCartID.String()},
		{name: "tampered signed cookie", value: tamperedCookie},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := newDeterministicRouteCartManager(t, routeTestCartID2)
			request := httptest.NewRequest(http.MethodGet, "/carrito", nil)
			request.AddCookie(&http.Cookie{Name: session.CartCookieName, Value: testCase.value})
			recorder := httptest.NewRecorder()
			var loaderID uuid.UUID

			withCartSession(manager, func(w http.ResponseWriter, r *http.Request) {
				loaderID, _ = session.CartIDFromContext(r.Context())
			})(recorder, request)

			if loaderID != routeTestCartID2 || loaderID == routeTestCartID {
				t.Fatalf("untrusted client UUID reached loader: %s", loaderID)
			}
			if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
				t.Fatalf("expected one replacement cookie, got %d", got)
			}
		})
	}
}

func TestCartSessionMiddlewareDistinguishesTwoValidSessions(t *testing.T) {
	managerOne := newDeterministicRouteCartManager(t, routeTestCartID)
	managerTwo := newDeterministicRouteCartManager(t, routeTestCartID2)
	cookieOne := issueRouteCartCookie(t, managerOne)
	cookieTwo := issueRouteCartCookie(t, managerTwo)
	resolver := newDeterministicRouteCartManager(t, routeTestCartID)

	resolved := make([]uuid.UUID, 0, 2)
	for _, cookie := range []*http.Cookie{cookieOne, cookieTwo} {
		request := httptest.NewRequest(http.MethodGet, "/carrito", nil)
		request.AddCookie(cookie)
		recorder := httptest.NewRecorder()
		withCartSession(resolver, func(w http.ResponseWriter, r *http.Request) {
			cartID, _ := session.CartIDFromContext(r.Context())
			resolved = append(resolved, cartID)
		})(recorder, request)
	}

	if resolved[0] != routeTestCartID || resolved[1] != routeTestCartID2 || resolved[0] == resolved[1] {
		t.Fatalf("valid sessions were not kept distinct: %v", resolved)
	}
}

func TestCartSessionFirstRequestHasContextAndOneCookie(t *testing.T) {
	manager := newDeterministicRouteCartManager(t, routeTestCartID)
	recorder := httptest.NewRecorder()
	received := false

	withCartSession(manager, func(w http.ResponseWriter, r *http.Request) {
		cartID, ok := session.CartIDFromContext(r.Context())
		received = ok && cartID == routeTestCartID
		w.WriteHeader(http.StatusAccepted)
	})(recorder, httptest.NewRequest(http.MethodPut, "/carrito", strings.NewReader("")))

	if !received || recorder.Code != http.StatusAccepted {
		t.Fatalf("first request did not reach handler with context: received=%t status=%d", received, recorder.Code)
	}
	if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
		t.Fatalf("expected one cookie on first request, got %d", got)
	}
}

func TestCartWizardAndQuoteRouteMethodsRemainRegistered(t *testing.T) {
	router := newTestRouter(t)
	testCases := []struct {
		method  string
		path    string
		pattern string
	}{
		{method: http.MethodGet, path: "/carrito", pattern: "GET /carrito"},
		{method: http.MethodPut, path: "/carrito", pattern: "PUT /carrito"},
		{method: http.MethodPatch, path: "/carrito/items", pattern: "PATCH /carrito/items"},
		{method: http.MethodDelete, path: "/carrito/items", pattern: "DELETE /carrito/items"},
		{method: http.MethodDelete, path: "/carrito/items/product-id", pattern: "DELETE /carrito/items/{id}"},
		{method: http.MethodGet, path: "/solicitar-cotizacion", pattern: "GET /solicitar-cotizacion"},
		{method: http.MethodPost, path: "/solicitar-cotizacion", pattern: "POST /solicitar-cotizacion"},
		{method: http.MethodPut, path: "/cotizacion/carrito/items/product-id", pattern: "PUT /cotizacion/carrito/items/{id}"},
		{method: http.MethodDelete, path: "/cotizacion/carrito/items/product-id", pattern: "DELETE /cotizacion/carrito/items/{id}"},
		{method: http.MethodPost, path: "/wizard/wizard-id/complete", pattern: "POST /wizard/{wizard_id}/complete"},
	}

	for _, testCase := range testCases {
		request := httptest.NewRequest(testCase.method, testCase.path, nil)
		_, pattern := router.Handler(request)
		if pattern != testCase.pattern {
			t.Errorf("%s %s: expected pattern %q, got %q", testCase.method, testCase.path, testCase.pattern, pattern)
		}
	}
}

func TestHandlerWithoutCartContextReturnsControlledError(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/carrito", nil)
	if cartID, err := cartIDFromRequestContext(request); err == nil || cartID != "" {
		t.Fatalf("expected controlled missing-context error, got cartID=%q err=%v", cartID, err)
	}
}

func TestRealRouterRejectsEveryProtectedCartMutationBeforeSession(t *testing.T) {
	routes := protectedCartMutationTestRoutes()
	for _, route := range routes {
		for _, headerMode := range []string{"missing headers", "untrusted Origin"} {
			t.Run(route.pattern+"/"+headerMode, func(t *testing.T) {
				router := newTestRouter(t)
				request := httptest.NewRequest(route.method, route.path, strings.NewReader(""))
				if headerMode == "untrusted Origin" {
					request.Header.Set("Origin", "https://attacker.test")
				}
				recorder := httptest.NewRecorder()

				router.ServeHTTP(recorder, request)

				assertRouteCSRFRejected(t, recorder)
				if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
					t.Fatalf("blocked route emitted cart_id: %v", got)
				}
			})
		}
	}
}

func TestEveryProtectedCartMutationAllowsOriginAndRefererWithFakeHandler(t *testing.T) {
	for _, route := range protectedCartMutationTestRoutes() {
		for _, headerMode := range []string{"Origin", "Referer"} {
			t.Run(route.pattern+"/"+headerMode, func(t *testing.T) {
				csrfGuard := newRouteTestCSRFGuard(t)
				cartManager := newDeterministicRouteCartManager(t, routeTestCartID)
				router := NewCustomServeMux()
				handlerCalls := 0
				router.HandleFunc(route.pattern, withProtectedCartSession(csrfGuard, cartManager, func(w http.ResponseWriter, r *http.Request) {
					handlerCalls++
					if cartID, ok := session.CartIDFromContext(r.Context()); !ok || cartID != routeTestCartID {
						t.Fatalf("handler did not receive authenticated cart ID: id=%s ok=%t", cartID, ok)
					}
					w.WriteHeader(http.StatusNoContent)
				}))

				request := httptest.NewRequest(route.method, route.path, strings.NewReader(""))
				if headerMode == "Origin" {
					request.Header.Set("Origin", "http://localhost:8080")
				} else {
					request.Header.Set("Referer", "http://localhost:8080/catalogo?from=test")
				}
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, request)

				if handlerCalls != 1 || recorder.Code != http.StatusNoContent {
					t.Fatalf("trusted request did not reach handler: calls=%d status=%d", handlerCalls, recorder.Code)
				}
				if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
					t.Fatalf("first trusted request expected one cart cookie, got %d", got)
				}
			})
		}
	}
}

func TestSafeCartRelatedGETRequestsDoNotRequireOrigin(t *testing.T) {
	for _, path := range []string{"/carrito", "/solicitar-cotizacion", "/catalogo"} {
		t.Run(path, func(t *testing.T) {
			csrfGuard := newRouteTestCSRFGuard(t)
			cartManager := newDeterministicRouteCartManager(t, routeTestCartID)
			recorder := httptest.NewRecorder()
			called := false

			withProtectedCartSession(csrfGuard, cartManager, func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})(recorder, httptest.NewRequest(http.MethodGet, path, nil))

			if !called || recorder.Code != http.StatusOK {
				t.Fatalf("safe request was blocked: called=%t status=%d", called, recorder.Code)
			}
		})
	}
}

func TestHTMLRedirectAndHTMXFragmentResponsesRemainUnchangedAfterGuard(t *testing.T) {
	t.Run("HTML redirect", func(t *testing.T) {
		csrfGuard := newRouteTestCSRFGuard(t)
		cartManager := newDeterministicRouteCartManager(t, routeTestCartID)
		request := httptest.NewRequest(http.MethodPost, "/solicitar-cotizacion", nil)
		request.Header.Set("Referer", "http://localhost:8080/solicitar-cotizacion")
		recorder := httptest.NewRecorder()

		withProtectedCartSession(csrfGuard, cartManager, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "/cotizacion/confirmada")
			w.WriteHeader(http.StatusSeeOther)
		})(recorder, request)

		if recorder.Code != http.StatusSeeOther || recorder.Header().Get("Location") != "/cotizacion/confirmada" {
			t.Fatalf("guard changed redirect response: status=%d location=%q", recorder.Code, recorder.Header().Get("Location"))
		}
	})

	t.Run("HTMX fragment", func(t *testing.T) {
		csrfGuard := newRouteTestCSRFGuard(t)
		cartManager := newDeterministicRouteCartManager(t, routeTestCartID)
		request := httptest.NewRequest(http.MethodPatch, "/carrito/items", nil)
		request.Header.Set("Origin", "http://localhost:8080")
		request.Header.Set("HX-Request", "true")
		recorder := httptest.NewRecorder()

		withProtectedCartSession(csrfGuard, cartManager, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("HX-Trigger", `{"cart":"updated"}`)
			_, _ = w.Write([]byte(`<div id="cart">fragment</div>`))
		})(recorder, request)

		if recorder.Code != http.StatusOK || recorder.Body.String() != `<div id="cart">fragment</div>` || recorder.Header().Get("HX-Trigger") != `{"cart":"updated"}` {
			t.Fatalf("guard changed HTMX response: status=%d header=%q body=%q", recorder.Code, recorder.Header().Get("HX-Trigger"), recorder.Body.String())
		}
	})
}

func TestPublicContactAPIAndReadOnlyAPIsAreOutsideCartCSRFGuard(t *testing.T) {
	router := newTestRouter(t)

	contactRequest := httptest.NewRequest(http.MethodPost, "/api/contact-requests", strings.NewReader("{"))
	contactRequest.Header.Set("Content-Type", "application/json")
	contactRecorder := httptest.NewRecorder()
	router.ServeHTTP(contactRecorder, contactRequest)
	if contactRecorder.Code == http.StatusForbidden {
		t.Fatal("POST /api/contact-requests was incorrectly placed behind the cart CSRF guard")
	}
	if got := contactRecorder.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Fatalf("contact API unexpectedly emitted cart_id: %v", got)
	}

	for _, testCase := range []struct {
		path    string
		pattern string
	}{
		{path: "/api/_health", pattern: "GET /api/_health"},
		{path: "/api/socials", pattern: "GET /api/socials"},
		{path: "/api/catalog/listings", pattern: "GET /api/catalog/listings"},
		{path: "/api/catalog/categories", pattern: "GET /api/catalog/categories"},
		{path: "/api/catalog/products", pattern: "GET /api/catalog/products"},
	} {
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		_, pattern := router.Handler(request)
		if pattern != testCase.pattern {
			t.Errorf("%s registration changed: expected %q, got %q", testCase.path, testCase.pattern, pattern)
		}
	}
}

func TestUnregisteredCartMethodDoesNotExecuteMutationGuardOrHandler(t *testing.T) {
	router := newTestRouter(t)
	request := httptest.NewRequest(http.MethodPost, "/carrito", nil)
	request.Header.Set("Origin", "http://localhost:8080")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code == http.StatusForbidden || recorder.Code == http.StatusOK || recorder.Code == http.StatusNoContent {
		t.Fatalf("unregistered POST /carrito unexpectedly executed: status=%d", recorder.Code)
	}
}

func newDeterministicRouteCartManager(t *testing.T, generatedID uuid.UUID) *session.CartManager {
	t.Helper()

	manager, err := session.NewCartManager(session.Config{
		Secret: testCartCookieSecret,
		Secure: false,
		Clock:  func() time.Time { return routeTestNow },
		UUIDGenerator: func() (uuid.UUID, error) {
			return generatedID, nil
		},
	})
	if err != nil {
		t.Fatalf("create deterministic cart manager: %v", err)
	}
	return manager
}

func issueRouteCartCookie(t *testing.T, manager *session.CartManager) *http.Cookie {
	t.Helper()

	recorder := httptest.NewRecorder()
	manager.Middleware(func(http.ResponseWriter, *http.Request) {})(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one issued cookie, got %d", len(cookies))
	}
	return cookies[0]
}

type protectedCartMutationTestRoute struct {
	method  string
	path    string
	pattern string
}

func protectedCartMutationTestRoutes() []protectedCartMutationTestRoute {
	return []protectedCartMutationTestRoute{
		{method: http.MethodPut, path: "/carrito", pattern: "PUT /carrito"},
		{method: http.MethodPatch, path: "/carrito/items", pattern: "PATCH /carrito/items"},
		{method: http.MethodDelete, path: "/carrito/items", pattern: "DELETE /carrito/items"},
		{method: http.MethodDelete, path: "/carrito/items/product-id", pattern: "DELETE /carrito/items/{id}"},
		{method: http.MethodPost, path: "/wizard/wizard-id/complete", pattern: "POST /wizard/{wizard_id}/complete"},
		{method: http.MethodPost, path: "/solicitar-cotizacion", pattern: "POST /solicitar-cotizacion"},
		{method: http.MethodPut, path: "/cotizacion/carrito/items/product-id", pattern: "PUT /cotizacion/carrito/items/{id}"},
		{method: http.MethodDelete, path: "/cotizacion/carrito/items/product-id", pattern: "DELETE /cotizacion/carrito/items/{id}"},
	}
}

func newRouteTestCSRFGuard(t *testing.T) *appsecurity.CSRFGuard {
	t.Helper()
	guard, err := appsecurity.NewCSRFGuard(appsecurity.CSRFConfig{TrustedOrigins: "http://localhost:8080"})
	if err != nil {
		t.Fatalf("create route CSRF guard: %v", err)
	}
	return guard
}

func assertRouteCSRFRejected(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if recorder.Code != http.StatusForbidden || recorder.Body.String() != "Solicitud no permitida." {
		t.Fatalf("unexpected CSRF rejection: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "text/plain; charset=utf-8" || recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected CSRF headers: %v", recorder.Header())
	}
}
