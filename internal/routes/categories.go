package routes

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterCategoriesRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/categories", GetCategories)
	router.HandleFunc("POST /api/categories", CreateCategory)
	router.HandleFunc("PUT /api/categories/{id}", UpdateCategory)
	router.HandleFunc("DELETE /api/categories/{id}", DeleteCategory)
}

func GetCategories(w http.ResponseWriter, r *http.Request) {
	ctgs, err := db.FindAllCategories()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find categories"))
		log.Printf("failed to find categories: %v\n", err)
		return
	}

	data, err := json.Marshal(ctgs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal categories"))
		log.Printf("failed to marshal categories: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func CreateCategory(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	ctg := db.Category{
		Name:        r.Form.Get("name"),
		Slug:        r.Form.Get("slug"),
		Description: r.Form.Get("description"),
		HeaderImg:   r.Form.Get("headerImg"),
		DisplayImg:  r.Form.Get("displayImg"),
	}

	if ctg.Slug == "" {
		ctg.Slug = internal.Slugify(ctg.Name)
	}

	err = db.CreateCategory(&ctg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create category"))
		log.Printf("failed to create category: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	ctg, err := db.FindCategoryByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find category"))
		log.Printf("failed to find category: %v\n", err)
		return
	}

	if r.Form.Get("slug") != "" {
		ctg.Slug = r.Form.Get("slug")
	}
	if r.Form.Get("name") != "" {
		ctg.Name = r.Form.Get("name")
		ctg.Slug = internal.Slugify(ctg.Name)
	}
	if r.Form.Get("description") != "" {
		ctg.Description = r.Form.Get("description")
	}
	if himg := r.Form.Get("headerImg"); himg != "" {
		if himg == uploads.RemoveImgFlag {
			ctg.HeaderImg = ""
		}
		ctg.HeaderImg = r.Form.Get("headerImg")
	}
	if dimg := r.Form.Get("displayImg"); dimg != "" {
		if dimg == uploads.RemoveImgFlag {
			ctg.DisplayImg = ""
		}
		ctg.DisplayImg = r.Form.Get("displayImg")
	}

	err = db.UpdateCategory(ctg)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to update category"))
		log.Printf("failed to update category: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := db.DeleteCategory(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete category"))
		log.Printf("failed to delete category: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
