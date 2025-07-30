package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterImagesRoutes(router *customServeMux) {
	// router.HandleFunc("GET /api/images", RenderIndex)
	router.HandleFunc("GET /api/images/table", auth.ValidateAuth(RenderImagesTable))

	router.HandleFunc("POST /api/images", auth.ValidateAuth(UploadImages))
	router.HandleFunc("PUT /api/images/{id}", auth.ValidateAuth(UpdateImage))
	router.HandleFunc("DELETE /api/images", auth.ValidateAuth(DeleteImages))
	router.HandleFunc("DELETE /api/images/{id}", auth.ValidateAuth(DeleteImage))
}

func GetImages(w http.ResponseWriter, r *http.Request) {
	sortOrder := r.URL.Query().Get("sortOrder")
	sortBy := r.URL.Query().Get("sortBy")
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortBy == "" {
		sortBy = "created_at"

func GetImages(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// rawSort := r.URL.Query().Get("images_sort")
	// sortOrder, sortBy := parseImagesSort(rawSort)

	images, err := db.FindAllImages([]string{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get images"))
		log.Printf("failed to get images: %v\n", err)
		return
	}

	data, err := json.Marshal(images)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func UploadImages(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Max 64MB file upload
	err := r.ParseMultipartForm(64 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse multipart form"))
		log.Printf("image upload failed: %v\n", err)
		return
	}

	imgs := r.MultipartForm.File["imgs"]
	if len(imgs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("No files uploaded"))
		return
	}

	var sizes []int64
	for _, file := range imgs {
		sizes = append(sizes, file.Size)
	}

	fileNames, err := uploads.UploadMultiple(imgs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to upload images"))
		log.Printf("image upload failed: %v\n", err)
		return
	}

	var uploadedImages []*db.Image
	imageNames := r.Form["img_names"]

	for i, fileName := range fileNames {
		id, err := uuid.NewV7()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Failed to generate new uuid"))
			log.Printf("failed to generate new uuid: %v\n", err)
			return
		}

		img := &db.Image{
			ID:         id.String(),
			Name:       imageNames[i],
			Filename:   fileName,
			Size:       int(sizes[i]),
			CreatedAt:  time.Now(),
			NoOptimize: false,
		}

		uploadedImages = append(uploadedImages, img)
	}

	err = db.CreateImages(uploadedImages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create images"))
		log.Printf("failed to create images: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateImage(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	// Max 10MB single file upload
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse multipart form"))
		log.Printf("image upload failed: %v\n", err)
		return
	}
	newFile := r.MultipartForm.File["img"][0]

	img, err := db.FindImageByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find image"))
		log.Printf("failed to find image: %v\n", err)
		return
	}

	err = uploads.Update(img.Filename, newFile)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to update file"))
		log.Printf("failed to update image: %v\n", err)
		return
	}

	img.Size = int(newFile.Size)
	err = db.UpdateImage(img)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to update image record"))
		log.Printf("failed to update image: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteImage(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	img, err := db.FindImageByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find image"))
		log.Printf("failed to find image: %v\n", err)
		return
	}

	err = db.DeleteImage(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete image record"))
		log.Printf("failed to delete image: %v\n", err)
		return
	}

	err = uploads.Delete(img.Filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete file"))
		log.Printf("failed to delete image: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func DeleteImages(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	ids := r.URL.Query()["ids"]

	imgs, err := db.FindAllImages(ids)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find images"))
		log.Printf("failed to find images: %v\n", err)
		return
	}

	err = db.DeleteImages(ids)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete image"))
		log.Printf("failed to delete image: %v\n", err)
		return
	}

	var filenames = make([]string, len(imgs))
	for i, img := range imgs {
		filenames[i] = img.Filename
	}
	err = uploads.DeleteMultiple(filenames)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete images"))
		log.Printf("failed to delete images: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
