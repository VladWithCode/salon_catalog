package routes

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterImagesRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/images", RenderIndex)

	router.HandleFunc("POST /api/images", UploadImages)
	router.HandleFunc("PUT /api/images/{id}", UpdateImage)
	router.HandleFunc("DELETE /api/images/{id}", DeleteImages)
}

func GetImages(w http.ResponseWriter, r *http.Request) {
	sortOrder := r.URL.Query().Get("sortOrder")
	sortBy := r.URL.Query().Get("sortBy")
	if sortOrder == "" {
		sortOrder = "desc"
	}
	if sortBy == "" {
		sortBy = "created_at"
	}
}

func UploadImages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Upload Imgs")
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

func UpdateImage(w http.ResponseWriter, r *http.Request) {
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

func DeleteImages(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	img, err := db.FindImageByID(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find image"))
		log.Printf("failed to find image: %v\n", err)
		return
	}

	err = uploads.Delete(img.Filename)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete file"))
		log.Printf("failed to delete image: %v\n", err)
		return
	}

	err = db.DeleteImage(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete image record"))
		log.Printf("failed to delete image: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
