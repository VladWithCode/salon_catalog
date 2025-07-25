package routes

import (
	"context"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
)

func RegisterCatalogRoutes(router *customServeMux) {
	router.HandleFunc("GET /catalog/categories", GetCatalogCategories)
	router.HandleFunc("GET /catalog/products", GetCatalogProducts)
}

func GetCatalogCategories(w http.ResponseWriter, r *http.Request) {
	responseTempl := components.CategoryFilters
	categories, err := db.FindCatalogCategories()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find catalog categories"))
		log.Printf("failed to find catalog categories: %v\n", err)
		return
	}

	plainCtgs := make([]db.CatalogCtg, len(categories))
	for i, ctg := range categories {
		plainCtgs[i] = *ctg
	}
	err = responseTempl(plainCtgs, "").Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to render catalog categories"))
		log.Printf("failed to render catalog categories: %v\n", err)
		return
	}
}

func GetCatalogProducts(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")
	products, err := db.FindCatalogProducts(categoryID, search)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find catalog products"))
		log.Printf("failed to find catalog products: %v\n", err)
		return
	}

	plainProducts := make([]db.CatalogProd, len(products))
	for i, prod := range products {
		plainProducts[i] = *prod
	}

	err = components.ProductGrid(plainProducts, false, 0).Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to render catalog products"))
		log.Printf("failed to render catalog products: %v\n", err)
		return
	}
}
