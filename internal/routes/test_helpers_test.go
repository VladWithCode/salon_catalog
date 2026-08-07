package routes

import (
	"testing"

	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

const testCartCookieSecret = "test-only-cart-cookie-secret-32-bytes"

func newTestCartManager(t *testing.T) *session.CartManager {
	t.Helper()

	manager, err := session.NewCartManager(session.Config{
		Secret: testCartCookieSecret,
		Secure: false,
	})
	if err != nil {
		t.Fatalf("create cart manager: %v", err)
	}
	return manager
}

func newTestRouter(t *testing.T) *customServeMux {
	t.Helper()

	csrfGuard, err := appsecurity.NewCSRFGuard(appsecurity.CSRFConfig{TrustedOrigins: "http://localhost:8080"})
	if err != nil {
		t.Fatalf("create CSRF guard: %v", err)
	}
	router, ok := NewRouter(newTestCartManager(t), csrfGuard).(*customServeMux)
	if !ok {
		t.Fatal("expected NewRouter to return a customServeMux")
	}
	return router
}
