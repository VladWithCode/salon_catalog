package routes

import (
	"context"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
)

func RegisterDashboardRoutes(router *customServeMux) {
	router.HandleFunc("GET /panel", auth.ValidateAuth(RenderDashboard))
}

func RenderDashboard(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := dashboard.Dashboard().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Dashboard err: %v\n", err)
	}
}
