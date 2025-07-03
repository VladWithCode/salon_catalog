// Package routes contains the routing logic for the application
package routes

import (
	"context"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
)

func NewRouter() http.Handler {
	router := NewCustomServeMux()

	router.HandleFunc("GET /{$}", RenderIndex)

	RegisterImagesRoutes(router)

	// Serve static files
	fs := http.FileServer(http.Dir("web/static/"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fs))

	router.NotFoundHandleFunc(render404Page)

	return router
}

func RenderIndex(w http.ResponseWriter, r *http.Request) {
	err := pages.Index().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Index err: %v\n", err)
	}
}

func render404Page(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 page not found"))
}
