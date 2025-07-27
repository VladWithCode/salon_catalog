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
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
)

func NewRouter() http.Handler {
	router := NewCustomServeMux()

	router.HandleFunc("GET /{$}", RenderIndex)
	router.HandleFunc("GET /catalogo", RenderCatalaog)
	router.HandleFunc("GET /servicios", RenderServices)
	router.HandleFunc("GET /salon", RenderSalon)
	router.HandleFunc("GET /iniciar-sesion", auth.PopulateAuth(RenderSignIn))

	RegisterDashboardRoutes(router)
	RegisterImagesRoutes(router)
	RegisterCategoriesRoutes(router)
	RegisterProductsRoutes(router)
	RegisterWizardRoutes(router)
	RegisterCatalogRoutes(router)
	RegisterCartRoutes(router)

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
	cartID, err := uuid.NewV7()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to generate cart id err: %v\n", err)
		return
	}

	_, err = r.Cookie("cart_id")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:    "cart_id",
			Value:   cartID.String(),
			Expires: time.Now().Add(time.Hour * 24 * 30),
			Path:    "/",
			Secure:  true,
		})
	}
	err = pages.Catalog().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Catalog err: %v\n", err)
	}
}

func RenderServices(w http.ResponseWriter, r *http.Request) {
	err := pages.Services().Render(context.Background(), w)
	if err != nil {
		log.Printf("failed to render Services err: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
	}
}

func RenderSalon(w http.ResponseWriter, r *http.Request) {
	err := pages.Salon().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Gallery err: %v\n", err)
	}
}

func RenderSignIn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	if a.ID != "" && a.ID != auth.InvalidTokenID {
		internal.HandleRedirect("/panel", http.StatusFound, w, r)
		return
	}

	err := pages.SignIn(
		&pages.FormState{},
	).Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render SignIn err: %v\n", err)
	}
}

func SignIn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	if a.ID != "" && a.ID != auth.InvalidTokenID {
		internal.HandleRedirect("/panel", http.StatusFound, w, r)
		return
	}

	signinPage := pages.SignIn
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err = signinPage(&pages.FormState{
			ServerError: "Error inesperado",
		}).Render(context.Background(), w)
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
		}).Render(context.Background(), w)
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
		}).Render(context.Background(), w)
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
		}).Render(context.Background(), w)
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
		}).Render(context.Background(), w)
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
		Secure:   true,
	})

	w.Header().Add("HX-Redirect", "/panel")
	w.WriteHeader(http.StatusFound)
}

func render404Page(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 page not found"))
}
