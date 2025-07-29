package routes

import (
	"context"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	dashcomps "github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
)

func RegisterDashboardRoutes(router *customServeMux) {
	router.HandleFunc("GET /panel", auth.ValidateAuth(RenderDashboard))
	router.HandleFunc("GET /panel/productos", auth.ValidateAuth(RenderProducts))
	router.HandleFunc("GET /panel/productos/nuevo", auth.ValidateAuth(RenderCreateProduct))
}

func RenderDashboard(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := dashboard.Dashboard().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Dashboard err: %v\n", err)
	}
}

func RenderProducts(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := dashboard.Products().Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Products err: %v\n", err)
	}
}

func RenderCreateProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	ctgPtrs, err := db.FindAllCategories()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to find categories err: %v\n", err)
		return
	}
	ctgs := make([]db.Category, len(ctgPtrs))
	for i, ctg := range ctgPtrs {
		ctgs[i] = *ctg
	}

	formState := forms.NewProductFormState("create")
	formState.Fields["name"] = forms.FieldState{
		IsTouched:       false,
		IsValid:         false,
		HasError:        false,
		HelpText:        "Nombre descriptivo del producto",
		ValidationClass: "",
	}
	formState.Fields["description"] = forms.FieldState{
		IsTouched:       false,
		IsValid:         false,
		HasError:        false,
		HelpText:        "Breve descripción del producto",
		ValidationClass: "",
	}
	err = dashboard.CreateProduct(dashcomps.CreateProductForm(formState, ctgs)).Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render CreateProduct err: %v\n", err)
	}
}
