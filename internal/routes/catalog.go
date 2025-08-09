package routes

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
)

func RegisterCatalogRoutes(router *customServeMux) {
	router.HandleFunc("GET /catalog/categories", GetCatalogCategories)
	router.HandleFunc("GET /catalog/products", GetCatalogProducts)
	router.HandleFunc("GET /catalogo/producto/{id}", GetProductDetail)
}

func GetCatalogCategories(w http.ResponseWriter, r *http.Request) {
	activeCtg := r.URL.Query().Get("categoria")

	ctgs, err := db.FindCatalogCategories("")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		components.CategoryFilters(internal.PtrSliceToPlainSlice(ctgs), activeCtg).Render(r.Context(), w)
		return
	}

	err = components.CategoryFilters(internal.PtrSliceToPlainSlice(ctgs), activeCtg).Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to render catalog categories"))
		log.Printf("failed to render catalog categories: %v\n", err)
		return
	}
}

func GetCatalogProducts(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("categoria")
	search := r.URL.Query().Get("buscar")

	// Parse pagination parameters
	page := 1
	limit := db.DefaultCatalogPageSize

	if pageStr := r.URL.Query().Get("pagina"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("por_pagina"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	result, err := db.FindCatalogProducts(categoryID, search, page, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find catalog products"))
		log.Printf("failed to find catalog products: %v\n", err)
		return
	}

	err = components.ProductGrid(result).Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to render catalog products"))
		log.Printf("failed to render catalog products: %v\n", err)
		return
	}
}

func GetProductDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	isAjax := r.Header.Get("HX-Request") == "true"
	modalState := &components.ProductModalState{}
	product, err := db.FindCatalogProductDetail(id)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		modalState.Error = "No se encontró el producto"

		if isAjax {
			templ.RenderFragments(
				r.Context(),
				w,
				components.ProductModal(modalState),
				"productModalContent",
			)
		} else {
			renderCatalogWithModal(w, r, modalState)
		}
		log.Printf("/catalogo/producto/%s: %v\n", id, err)
		return
	}

	modalState.Product = *product
	modalState.RelatedProducts, err = db.FindRelatedProducts(product.ID, 6)
	if err != nil {
		log.Printf("failed to find related products: %v\n", err)
	}

	if isAjax {
		err = templ.RenderFragments(
			r.Context(),
			w,
			components.ProductModal(modalState),
			"productModalContent",
		)
	} else {
		renderCatalogWithModal(w, r, modalState)
		return
	}
	if err != nil {
		reqWithErr := r.WithContext(context.WithValue(r.Context(), "error", err))
		http.Redirect(w, reqWithErr, "/error", http.StatusInternalServerError)
		log.Printf("failed to render product detail: %v\n", err)
	}
}

func renderCatalogWithModal(w http.ResponseWriter, r *http.Request, modalState *components.ProductModalState) {
	state := &pages.CatalogState{
		Search:            r.URL.Query().Get("buscar"),
		ActiveCategory:    r.URL.Query().Get("categoria"),
		Page:              r.URL.Query().Get("pagina"),
		Limit:             r.URL.Query().Get("por_pagina"),
		ProductModalState: modalState,
	}

	err := pages.Catalog(state).Render(r.Context(), w)
	if err != nil {
		reqWithErr := r.WithContext(context.WithValue(r.Context(), "error", err))
		http.Redirect(w, reqWithErr, "/error", http.StatusInternalServerError)
	}
}
