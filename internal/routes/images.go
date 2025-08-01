package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterImagesRoutes(router *customServeMux) {
	// This routes respond with templ components (i.e. HTML or text/HTML mime type responses)
	router.HandleFunc("GET /imagenes/subir", auth.ValidateAuth(RenderNewImageForm))
	router.HandleFunc("POST /imagenes/subir", auth.ValidateAuth(UploadImagesForm))
	router.HandleFunc("GET /imagenes/{id}", auth.PopulateAuth(RenderImage))
	router.HandleFunc("GET /imagenes/table", auth.ValidateAuth(RenderImagesTable))
	router.HandleFunc("DELETE /imagenes/{id}", auth.PopulateAuth(DeleteImageAndReturnTable))

	router.HandleFunc("POST /api/images", auth.ValidateAuth(UploadImages))
	// This routes respond with JSON
	// TODO: update to use JSON responses
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

func RenderNewImageForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	fs := forms.NewImagesFormState()
	err := dashboard.ImagesNewForm(fs).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to render new image form"))
		log.Printf("failed to render new image form: %v\n", err)
		return
	}
}

func UploadImagesForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Create states and indicate we're including toasts
	w.Header().Add("X-Includes-Toast", "true")
	fs, err := forms.NewImagesFormStateFromReq(r)
	fs.SetSuccessMessage("Imágenes subidas exitosamente")
	td := components.NewToastData(
		"Se subieron las imágenes exitosamente",
		components.ToastSuccess,
		3000,
		true,
		false,
	)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		fs := forms.NewImagesFormState()
		fs.SetErrorMessage("El tamaño máximo de subida es de 64MB")
		td := components.NewToastData("El peso total excede el máximo permitido", components.ToastError, 3000, true, false)
		td.Message = "Imágenes muy grandes"
		td.Type = components.ToastError
		templ.Join(
			dashboard.ImagesNewForm(fs),
			components.ToasterToast(td),
		).Render(r.Context(), w)
		log.Printf("image upload failed: %v\n", err)
		return
	}

	err = fs.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		fs.SetErrorMessage("Hay errores en el formulario")
		td.Message = "Hay errores en el formulario"
		td.Type = components.ToastError
		component := templ.Join(
			dashboard.ImagesNewForm(fs),
			components.ToasterToast(td),
		)
		component.Render(r.Context(), w)
		log.Printf("image upload failed: %v\n", err)
		return
	}

	files := r.MultipartForm.File
	if len(files) == 0 {
		w.WriteHeader(http.StatusBadRequest)

		fs.SetErrorMessage("No se encontraron imágenes")
		td.Message = "No se encontraron imágenes"
		td.Type = components.ToastError
		component := templ.Join(
			dashboard.ImagesNewForm(fs),
			components.ToasterToast(td),
		)
		component.Render(r.Context(), w)
		log.Printf("image upload failed: %v\n", err)
		return
	}

	imgs, wrtFiles, err := parseImageUploads(r)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		td.Type = components.ToastError
		component := templ.Join(
			dashboard.ImagesNewForm(fs),
			components.ToasterToast(td),
		)

		if errors.Is(err, ErrInvalidImageUpload) {
			fs.SetErrorMessage("Hay imagenes mal formadas")
			td.Message = "Hay imagenes mal formadas"
			component.Render(r.Context(), w)
			return
		} else if errors.Is(err, ErrTooManyImages) {
			fs.SetErrorMessage("Hay demasiadas imagenes")
			td.Message = "Hay demasiadas imagenes"
			component.Render(r.Context(), w)
			return
		}
	}

	writtenFiles, err := uploads.UploadMultiple(wrtFiles)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		fs.SetErrorMessage("Algo salió mal")
		td.Message = "Algo salió mal"
		td.Type = components.ToastError
		component := templ.Join(
			dashboard.ImagesNewForm(fs),
			components.ToasterToast(td),
		)
		component.Render(r.Context(), w)
		log.Printf("image upload failed: %v\n", err)
		return
	}

	for i, img := range imgs {
		img.Filename = writtenFiles[i].Filename
		img.Size = int(writtenFiles[i].Size)
	}

	err = db.CreateImages(imgs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		fs.SetErrorMessage("Algo salió mal")
		td.Message = "Algo salió mal"
		td.Type = components.ToastError
		component := templ.Join(
			dashboard.ImagesNewForm(fs),
			components.ToasterToast(td),
		)
		component.Render(r.Context(), w)
		log.Printf("image upload failed: %v\n", err)
		return
	}

	w.Header().Add("Hx-Reswap", "innerHTML")
	filters := parseImageFilters(r)
	imagesResult, err := db.FilterImages(filters)
	if err != nil {
		w.Header().Add("Hx-Retarget", "#images-table")
		w.WriteHeader(http.StatusInternalServerError)

		imagesResult.HasError = true
		imagesResult.Error = "Algo salió mal al recuperar las imágenes"
		td.Message = "Se subieron las imágenes exitosamente"
		td.Type = components.ToastError
		component := templ.Join(
			dashboard.ImagesTable(imagesResult),
			components.ToasterToast(td),
		)
		component.Render(r.Context(), w)
		log.Printf("image upload error while querying images: %v\n", err)
		return
	}

	err = templ.Join(
		dashboard.ImagesTable(imagesResult),
		components.ToasterToast(td),
	).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		fs.SetErrorMessage("Algo salió mal")
		td.Message = "Algo salió mal"
		td.Type = components.ToastError
		component := templ.Join(
			dashboard.ImagesNewForm(fs),
			components.ToasterToast(td),
		)
		component.Render(r.Context(), w)
		log.Printf("image upload failed: %v\n", err)
		return
	}
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

	writtenFiles, err := uploads.UploadMultiple(imgs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to upload images"))
		log.Printf("image upload failed: %v\n", err)
		return
	}

	var uploadedImages []*db.Image
	imageNames := r.Form["img_names"]

	for i, fileName := range writtenFiles {
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
			Filename:   fileName.Filename,
			Size:       int(fileName.Size),
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

var (
	ErrInvalidImageUpload = errors.New("las imágenes subidas no son válidas")
	ErrTooManyImages      = errors.New("el máximo de imágenes subidas es de 10")
)

func parseImageUploads(r *http.Request) ([]*db.Image, []*multipart.FileHeader, error) {
	imgs := []*db.Image{}
	wrtFiles := []*multipart.FileHeader{}
	createDate := time.Now()
	files := r.MultipartForm.File
	count := 0

	for name, fileHdl := range files {
		parts := strings.SplitN(name, "_", 3)
		if len(parts) != 3 {
			return nil, nil, ErrInvalidImageUpload
		}

		inpName := fmt.Sprintf("%s_name_%s", parts[0], parts[2])
		imgName := r.FormValue(inpName)
		if imgName == "" {
			imgName = createDate.Format("2006-01-02_15-04-05")
		}

		if len(fileHdl) > 1 {
			for i, handle := range fileHdl {
				count++
				if count > 10 {
					return nil, nil, ErrTooManyImages
				}
				id := uuid.Must(uuid.NewV7())
				img := &db.Image{
					ID:         id.String(),
					Name:       fmt.Sprintf("%s %d", imgName, i+1),
					Filename:   "",
					Size:       0,
					CreatedAt:  createDate,
					NoOptimize: false,
				}

				imgs = append(imgs, img)
				wrtFiles = append(wrtFiles, handle)
			}
		} else {
			id := uuid.Must(uuid.NewV7())
			img := &db.Image{
				ID:         id.String(),
				Name:       imgName,
				Filename:   "",
				Size:       0,
				CreatedAt:  createDate,
				NoOptimize: false,
			}

			imgs = append(imgs, img)
			wrtFiles = append(wrtFiles, fileHdl[0])
		}
	}

	return imgs, wrtFiles, nil
}
