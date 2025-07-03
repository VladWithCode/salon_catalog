// Package routes contains the routing logic for the application
package routes

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/templates"
)

func NewRouter() http.Handler {
	router := NewCustomServeMux()

	router.HandleFunc("GET /{$}", RenderIndex)
	router.HandleFunc("GET /test", TestFn)

	return router
}

func RenderIndex(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	response := fmt.Sprintf("Hello, %s!", name)
	w.Write([]byte(response))
}

func TestFn(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	t := templates.Demo(name)

	err := t.Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("failed to render template"))
		fmt.Printf("err: %v\n", err)
	}
}
