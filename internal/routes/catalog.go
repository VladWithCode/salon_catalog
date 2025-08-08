package routes

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
)

func RegisterCatalogRoutes(router *customServeMux) {
	router.HandleFunc("GET /catalog/categories", GetCatalogCategories)
	router.HandleFunc("GET /catalog/products", GetCatalogProducts)
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
