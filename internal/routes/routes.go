// Package routes contains the routing logic for the application
package routes

import (
	"context"
	"net/http"

)

func NewRouter() http.Handler {
	router := NewCustomServeMux()

	router.HandleFunc("GET /{$}", RenderIndex)

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

