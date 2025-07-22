package routes

import "net/http"

func RegisterDashboardRoutes(router *customServeMux) {
	router.HandleFunc("GET /dash", RenderDashboard)
}

func RenderDashboard(w http.ResponseWriter, r *http.Request) {
}
