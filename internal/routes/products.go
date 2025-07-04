package routes

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/db"
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
}

func UpdateProductImages(w http.ResponseWriter, r *http.Request) {
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
}

func DeleteProductImages(w http.ResponseWriter, r *http.Request) {
}
