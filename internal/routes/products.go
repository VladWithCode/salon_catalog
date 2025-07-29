package routes

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterProductsRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/products", GetProducts)
	router.HandleFunc("GET /api/products/table", GetProductsTable)
	router.HandleFunc("GET /api/products/list", GetProductsList)
	router.HandleFunc("GET /api/products/{slug}", GetProductBySlug)

	router.HandleFunc("POST /api/products", auth.ValidateAuth(CreateProduct))
	router.HandleFunc("PUT /api/products/{id}", auth.ValidateAuth(UpdateProduct))
	router.HandleFunc("PUT /api/products/{id}/images", auth.ValidateAuth(UpdateProductImages))
	router.HandleFunc("DELETE /api/products/{id}", auth.ValidateAuth(DeleteProduct))
	router.HandleFunc("DELETE /api/products/{id}/images", auth.ValidateAuth(DeleteProductImages))
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
	prods, err := db.FindAllProducts()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find products"))
		log.Printf("failed to find products: %v\n", err)
		return
	}

	data, err := json.Marshal(prods)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal products"))
		log.Printf("failed to marshal products: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func GetProductsTable(w http.ResponseWriter, r *http.Request) {
	params, err := parseProductFilterParams(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	prods, err := db.FilterProducts(*params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find products"))
		log.Printf("failed to find products: %v\n", err)
		return
	}

	err = dashboard.ProductsTable(*prods).Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render ProductsTable err: %v\n", err)
		return
	}
}

func GetProductsList(w http.ResponseWriter, r *http.Request) {
	prods, err := db.FindAllProducts()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find products"))
		log.Printf("failed to find products: %v\n", err)
		return
	}

	data, err := json.Marshal(prods)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal products"))
		log.Printf("failed to marshal products: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func GetProductBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	prod, err := db.FindProductBySlug(slug)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	data, err := json.Marshal(prod)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal product"))
		log.Printf("failed to marshal product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func CreateProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	var data map[string]string
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}
	defer r.Body.Close()

	formState := forms.NewProductFormStateFromMap("create", data)
	err = formState.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render CreateProduct err: %v\n", err)
		return
	}
	var product db.Product
	product.Name = formState.Values.Name
	product.Description = formState.Values.Description
	product.MainImg = formState.Values.MainImg
	product.CategoryID = formState.Values.CategoryID
	product.Available = formState.Values.Available
	product.Gallery = formState.Values.Gallery

	product.Slug = internal.Slugify(product.Name)

	err = db.CreateProduct(&product)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create product"))
		log.Printf("failed to create product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	var data db.Product
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	id := r.PathValue("id")
	prod, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	if data.Name != "" {
		prod.Name = data.Name
		prod.Slug = internal.Slugify(prod.Name)
	}
	if data.Description != "" {
		prod.Description = data.Description
	}
	if data.Category != "" {
		prod.Category = data.Category
	}

	if data.MainImg != "" {
		if data.MainImg == uploads.RemoveImgFlag {
			prod.MainImg = ""
		}

		prod.MainImg = data.MainImg
	}

	err = db.UpdateProduct(prod)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to update product"))
		log.Printf("failed to update product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateProductImages(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	var imgIDs []string
	err := json.NewDecoder(r.Body).Decode(&imgIDs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	prod, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	err = db.LinkImagesToProduct(imgIDs, prod.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to link images to product"))
		log.Printf("failed to link images to product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteProduct(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	err := db.DeleteProduct(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete product"))
		log.Printf("failed to delete product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteProductImages(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	var imgIDs []string
	err := json.NewDecoder(r.Body).Decode(&imgIDs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	prod, err := db.FindProductByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find product"))
		log.Printf("failed to find product: %v\n", err)
		return
	}

	err = db.UnlinkImagesFromProduct(imgIDs, prod.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to unlink images from product"))
		log.Printf("failed to unlink images from product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func parseProductFilterParams(r *http.Request) (*db.ProductFilterParams, error) {
	params := &db.ProductFilterParams{}
	params.Search = r.URL.Query().Get("search")
	params.Category = r.URL.Query().Get("category")
	params.Sort = r.URL.Query().Get("sort")
	params.Page, _ = strconv.Atoi(r.FormValue("page"))
	params.Limit, _ = strconv.Atoi(r.FormValue("limit"))

	return params, nil
}
