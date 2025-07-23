// Package routes contains the routing logic for the application
package routes

import (
	"context"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
)

func NewRouter() http.Handler {
	router := NewCustomServeMux()

	router.HandleFunc("GET /{$}", RenderIndex)
	router.HandleFunc("GET /catalogo", RenderCatalaog)
	router.HandleFunc("GET /iniciar-sesion", RenderSignIn)

	RegisterImagesRoutes(router)
	RegisterCategoriesRoutes(router)
	RegisterProductsRoutes(router)

	// Api
	router.HandleFunc("POST /api/sign-in", auth.PopulateAuth(SignIn))

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

func RenderCatalaog(w http.ResponseWriter, r *http.Request) {
	err := pages.Catalog().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Catalog err: %v\n", err)
	}
}

func RenderSignIn(w http.ResponseWriter, r *http.Request) {
	err := pages.SignIn().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render SignIn err: %v\n", err)
	}
}

func SignIn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	if a.ID != "" {
		w.Header().Add("HX-Redirect", "/dashboard")
	}

	signinPage := pages.SignIn
	err := r.ParseForm()
	if err != nil {
		err = signinPage().Render(context.TODO(), w)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error inesperado"))
			return
		}

		return
	}

}

func render404Page(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 page not found"))
}
