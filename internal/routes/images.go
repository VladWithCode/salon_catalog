package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterImagesRoutes(router *customServeMux) {
	router.HandleFunc("GET /imagenes/{id}", auth.PopulateAuth(RenderImage))
	router.HandleFunc("DELETE /imagenes/{id}", auth.PopulateAuth(DeleteImageAndReturnTable))

	// router.HandleFunc("GET /api/images", RenderIndex)
	router.HandleFunc("GET /api/images/table", auth.ValidateAuth(RenderImagesTable))

	router.HandleFunc("POST /api/images", auth.ValidateAuth(UploadImages))
	router.HandleFunc("PUT /api/images/{id}", auth.ValidateAuth(UpdateImage))
	router.HandleFunc("DELETE /api/images", auth.ValidateAuth(DeleteImages))
	router.HandleFunc("DELETE /api/images/{id}", auth.ValidateAuth(DeleteImage))
}

func RenderImage(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	img, err := db.FindImageByID(id)

	// If request is AJAX, render components
	if r.Header.Get("HX-Request") == "true" {
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			dashboard.ImageModal(img, "Imágen no encontrada").Render(r.Context(), w)
			log.Printf("failed to find image: %v\n", err)
			return
		}

		component := dashboard.ImageModal(img, "")
		component.Render(r.Context(), w)
	} else {
		// Else render page
		// TODO: render page
	}
}

func RenderImagesTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Parse query parameters
	filters := parseImageFilters(r)

	// Get filtered images
	result, err := db.FilterImages(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get images"))
		log.Printf("failed to filter images: %v\n", err)
		return
	}

	// Render the template with the results
	component := dashboard.ImagesTable(result)
	component.Render(r.Context(), w)
}

func DeleteImageAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminó la imagen", components.ToastSuccess, 3000, true, false)
	fname, err := db.DeleteImage(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		toastData.Message = "Algo salió mal"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImagesTable(&db.ImageFilterResult{HasError: true, Error: "Algo salió mal"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)

		log.Printf("failed to delete image: %v\n", err)
		return
	}

	err = uploads.Delete(fname)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		toastData.Message = "Algo salió mal"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImagesTable(&db.ImageFilterResult{HasError: true, Error: "Algo salió mal"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)

		log.Printf("failed to delete image: %v\n", err)
		return
	}

	filters := parseImageFilters(r)
	result, err := db.FilterImages(filters)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusInternalServerError)

		toastData.Message = "Algo salió mal"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImagesTable(&db.ImageFilterResult{HasError: true, Error: "Algo salió mal"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)

		log.Printf("failed to delete image: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.ImagesTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		toastData.Message = "Algo salió mal"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.ImagesTable(&db.ImageFilterResult{HasError: true, Error: "Algo salió mal"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)

		log.Printf("failed to delete image: %v\n", err)
		return
	}
}

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
	fname, err := db.DeleteImage(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete image record"))
		log.Printf("failed to delete image: %v\n", err)
		return
	}

	err = uploads.Delete(fname)
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

func parseImagesSort(s string) (sortOrder, sortBy string) {
	parts := strings.Split(s, "_")
	sortOrder = "DESC"
	sortBy = "name"

	if len(parts) == 0 {
		return
	} else if len(parts) == 2 {
		sortBy = strings.ToLower(parts[0])
		sortOrder = strings.ToUpper(parts[1])
	}

	if sortBy == "date" {
		sortBy = "created_at"
	}

	return
}

// parseImageFilters extracts and parses filter parameters from the request
func parseImageFilters(r *http.Request) db.ImageFilterParams {
	query := r.URL.Query()

	filters := db.ImageFilterParams{
		Name: strings.TrimSpace(query.Get("name")),
	}

	// Parse sorting
	rawSort := query.Get("sort")
	filters.SortOrder, filters.SortBy = parseImagesSort(rawSort)

	// Parse pagination
	if pageStr := query.Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filters.Page = page
		}
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	// Parse date filters
	if exactDateStr := query.Get("date"); exactDateStr != "" {
		if exactDate, err := time.Parse("2006-01-02", exactDateStr); err == nil {
			filters.ExactDate = exactDate
		}
	}

	if afterDateStr := query.Get("date_after"); afterDateStr != "" {
		if afterDate, err := time.Parse("2006-01-02", afterDateStr); err == nil {
			filters.DateAfter = afterDate
		}
	}

	if beforeDateStr := query.Get("date_before"); beforeDateStr != "" {
		if beforeDate, err := time.Parse("2006-01-02", beforeDateStr); err == nil {
			filters.DateBefore = beforeDate
		}
	}

	return filters
}
