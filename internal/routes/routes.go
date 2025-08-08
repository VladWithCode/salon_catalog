// Package routes contains the routing logic for the application
package routes

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
)

func NewRouter() http.Handler {
	router := NewCustomServeMux()

	router.HandleFunc("GET /{$}", publicMiddleware(RenderIndex))
	router.HandleFunc("GET /catalogo", publicMiddleware(RenderCatalog))
	router.HandleFunc("GET /productos/{slug}", publicMiddleware(RenderProductDetail))
	router.HandleFunc("GET /servicios", publicMiddleware(RenderServices))
	router.HandleFunc("GET /reservaciones", publicMiddleware(RenderReservations))
	router.HandleFunc("GET /experiencia", publicMiddleware(RenderSalon))
	router.HandleFunc("GET /iniciar-sesion", publicMiddleware(auth.PopulateAuth(RenderSignIn)))

	RegisterDashboardRoutes(router)
	RegisterImagesRoutes(router)
	RegisterImageSelectorRoutes(router)
	RegisterCategoriesRoutes(router)
	RegisterProductsRoutes(router)
	RegisterEventKindsRoutes(router)
	RegisterContactRequestsRoutes(router)
	RegisterWizardRoutes(router)
	RegisterCatalogRoutes(router)
	RegisterCartRoutes(router)
	RegisterUserRoutes(router)
	RegisterQuoteRequestsRoutes(router)

	// Api
	router.HandleFunc("POST /api/sign-in", auth.PopulateAuth(SignIn))

	// Serve static files
	fs := http.FileServer(http.Dir("web/static/"))
	router.Handle("GET /static/", http.StripPrefix("/static/", fs))

	router.NotFoundHandleFunc(render404Page)

	return router
}

func RenderIndex(w http.ResponseWriter, r *http.Request) {
	rawListings, err := db.FindCatalogListings()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Index err: %v\n", err)
		return
	}

	listings := make(components.CatalogListings)
	for ctg, list := range rawListings {
		listings[ctg] = make([]db.CatalogProd, len(list))
		for i, prod := range list {
			listings[ctg][i] = *prod
		}
	}

	err = pages.Index(listings).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Index err: %v\n", err)
	}
}

func RenderCatalog(w http.ResponseWriter, r *http.Request) {
	cartID := uuid.Must(uuid.NewV7())

	search := r.URL.Query().Get("buscar")
	category := r.URL.Query().Get("categoria")

	_, err := r.Cookie("cart_id")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:    "cart_id",
			Value:   cartID.String(),
			Expires: time.Now().Add(time.Hour * 24 * 30),
			Path:    "/",
		})
	}

	state := &pages.CatalogState{
		Search:         search,
		ActiveCategory: category,
	}
	err = pages.Catalog(state).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Algo salió mal"))
		log.Printf("failed to render Catalog err: %v\n", err)
	}
}

func RenderServices(w http.ResponseWriter, r *http.Request) {
	err := pages.Services().Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render Services err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
	}
}

func RenderSalon(w http.ResponseWriter, r *http.Request) {
	err := pages.Salon().Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Gallery err: %v\n", err)
	}
}

func RenderReservations(w http.ResponseWriter, r *http.Request) {
	err := pages.Reservations().Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Gallery err: %v\n", err)
	}
}

func RenderSignIn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	if a.ID != "" && a.ID != auth.InvalidTokenID && a.ID != auth.ExpiredTokenID {
		internal.HandleRedirect("/panel", http.StatusFound, w, r)
		return
	}

	err := pages.SignIn(
		&pages.FormState{},
	).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render SignIn err: %v\n", err)
	}
}

func SignIn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	if a.ID != "" && a.ID != auth.InvalidTokenID && a.ID != auth.ExpiredTokenID {
		internal.HandleRedirect("/panel", http.StatusFound, w, r)
		return
	}

	signinPage := pages.SignIn
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = signinPage(&pages.FormState{
			ServerError: "Error inesperado",
		}).Render(r.Context(), w)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error inesperado"))
		}
		return
	}

	username := r.FormValue("user")
	password := r.FormValue("password")

	if username == "" || password == "" {
		w.WriteHeader(http.StatusBadRequest)
		err = signinPage(&pages.FormState{
			UserError:     "El nombre de usuario es requerido",
			UserValue:     username,
			PasswordError: "La contraseña es requerida",
		}).Render(r.Context(), w)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error inesperado"))
		}
		return
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		err = signinPage(&pages.FormState{
			UserError:     "Revisa que el nombre de usuario sea correcto",
			UserValue:     username,
			PasswordError: "Revisa que la contraseña sea correcta",
		}).Render(r.Context(), w)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error inesperado"))
		}
		return
	}

	err = user.ValidatePass(password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		err = signinPage(&pages.FormState{
			UserError:     "Revisa que el nombre de usuario sea correcto",
			UserValue:     username,
			PasswordError: "Revisa que la contraseña sea correcta",
		}).Render(r.Context(), w)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error inesperado"))
		}
		return
	}

	token, err := auth.CreateToken(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = signinPage(&pages.FormState{
			ServerError: "Error inesperado",
		}).Render(r.Context(), w)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Error inesperado"))
		}
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		Path:     "/",
		HttpOnly: true,
		// Secure:   true,
	})

	internal.HandleRedirect("/panel", http.StatusFound, w, r)
}

func RenderProductDetail(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Product slug is required"))
		return
	}

	product, err := db.FindProductBySlug(slug)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Product not found"))
		log.Printf("failed to find product by slug %s: %v\n", slug, err)
		return
	}

	err = pages.Product(product).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Product page err: %v\n", err)
	}
}

func render404Page(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 page not found"))
}

func publicMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqWithPath := r.WithContext(context.WithValue(r.Context(), "urlPath", r.URL.Path))
		next(w, reqWithPath)
	})
}
