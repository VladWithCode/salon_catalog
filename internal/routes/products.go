package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterProductsRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/products", GetProducts)
	router.HandleFunc("POST /api/products", CreateProduct)
	router.HandleFunc("PUT /api/products/{id}", UpdateProduct)
	router.HandleFunc("PUT /api/products/{id}/images", UpdateProductImages)
	router.HandleFunc("DELETE /api/products/{id}", DeleteProduct)
	router.HandleFunc("DELETE /api/products/{id}/images", DeleteProductImages)
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	var data db.Product
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	fmt.Printf("data: %+v\n", data)

	err = db.CreateProduct(&data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create product"))
		log.Printf("failed to create product: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
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
	if data.Price != 0 {
		prod.Price = data.Price
	}
	if data.Features != nil {
		prod.Features = data.Features
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

func UpdateProductImages(w http.ResponseWriter, r *http.Request) {
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

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
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

func DeleteProductImages(w http.ResponseWriter, r *http.Request) {
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
