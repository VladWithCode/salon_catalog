package routes

import (
	"errors"
	"net/http"

	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

var errCartSessionUnavailable = errors.New("cart session unavailable")

func cartIDFromRequestContext(r *http.Request) (string, error) {
	cartID, ok := session.CartIDFromContext(r.Context())
	if !ok {
		return "", errCartSessionUnavailable
	}
	return cartID.String(), nil
}

func withCartSession(cartSessions *session.CartManager, next http.HandlerFunc) http.HandlerFunc {
	return cartSessions.Middleware(next)
}

func withProtectedCartSession(csrfGuard *appsecurity.CSRFGuard, cartSessions *session.CartManager, next http.HandlerFunc) http.HandlerFunc {
	return csrfGuard.Protect(withCartSession(cartSessions, next))
}
