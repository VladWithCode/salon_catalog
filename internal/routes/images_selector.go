package routes

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
)

func RegisterImageSelectorRoutes(router *customServeMux) {
	router.HandleFunc("GET /imagenes/selector", auth.ValidateAuth(RenderImageSelector))
	router.HandleFunc("GET /api/images/selector", auth.ValidateAuth(GetImagesForSelector))
}

func RenderImageSelector(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Parse query parameters for selector configuration
	filters := parseImageFilters(r)
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "single"
	}

	updateEndpoint := r.URL.Query().Get("update_endpoint")
	title := r.URL.Query().Get("title")
	if title == "" {
		title = "Seleccionar Imagen"
	}

	maxSelection := db.DefaultImageSelectorLimit
	if maxStr := r.URL.Query().Get("max_selection"); maxStr != "" {
		if maxSelect, err := strconv.Atoi(maxStr); err == nil && maxSelect > 0 {
			maxSelection = maxSelect
		}
	}

	selectedIds := []string{}
	if selectedStr := r.URL.Query().Get("selected_ids"); selectedStr != "" {
		selectedIds = strings.Split(selectedStr, ",")
	}

	if filters.Limit == 0 {
		filters.Limit = db.DefaultImageSelectorLimit
	}
	successTarget := r.URL.Query().Get("success_target")

	config := dashboard.ImageSelectorConfig{
		Mode:           mode,
		UpdateEndpoint: updateEndpoint,
		Title:          title,
		MaxSelection:   maxSelection,
		SelectedIds:    selectedIds,
		AllowUpload:    true,
		SuccessTarget:  successTarget,
	}

	filters.Pinned = config.SelectedIds
	// Get images
	result, err := db.FilterImages(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		result := &db.ImageFilterResult{
			HasError: true,
			Error:    "Algo salió mal",
		}
		dashboard.ImageSelectorModal(config, result).Render(r.Context(), w)
		log.Printf("failed to filter images: %v\n", err)
		return
	}

	component := dashboard.ImageSelectorModal(config, result)
	component.Render(r.Context(), w)
}

func GetImagesForSelector(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Parse filter parameters
	filters := parseImageFilters(r)
	if filters.Limit == 0 {
		filters.Limit = db.DefaultImageSelectorLimit
	}

	var err error
	config := dashboard.ImageSelectorConfig{}
	rawConfig := r.URL.Query().Get("selectorConfig")
	if rawConfig == "" {
		w.WriteHeader(http.StatusBadRequest)
		result := &db.ImageFilterResult{
			HasError: true,
			Error:    "Algo salió mal",
		}
		templ.RenderFragments(
			r.Context(),
			w,
			dashboard.ImageSelectorModal(config, result),
			"imagesGrid",
		)
		log.Println("selector config is empty")
		return
	} else {
		err = config.FromJSONString(rawConfig)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			result := &db.ImageFilterResult{
				HasError: true,
				Error:    "Algo salió mal",
			}
			templ.RenderFragments(
				r.Context(),
				w,
				dashboard.ImageSelectorModal(config, result),
				"imagesGrid",
			)
			log.Printf("failed to parse selector config: %v\n", err)
			return
		}
	}

	if addToSelection := r.URL.Query()["add_to_selection"]; len(addToSelection) > 0 {
		if config.Mode == "multiple" {
			filters.Pinned = append(filters.Pinned, addToSelection...)
		} else {
			filters.Pinned = []string{addToSelection[0]}
		}
	}

	if removeFromSelection := r.URL.Query()["remove_from_selection"]; len(removeFromSelection) > 0 {
		newSelection := []string{}
		for _, id := range filters.Pinned {
			if id != removeFromSelection[0] {
				newSelection = append(newSelection, id)
			}
		}
		filters.Pinned = newSelection
		config.SelectedIds = newSelection
	}

	if len(filters.Pinned) > 10 {
		w.WriteHeader(http.StatusBadRequest)
		result := &db.ImageFilterResult{
			HasError: true,
			Error:    "El máximo de imágenes permitidas es de 10",
		}
		templ.RenderFragments(
			r.Context(),
			w,
			dashboard.ImageSelectorModal(config, result),
			"imagesGrid",
		)
		log.Printf("failed to filter images for selector API: %v\n", err)
	}

	// Get images
	result, err := db.FilterImages(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		result := &db.ImageFilterResult{
			HasError: true,
			Error:    "Algo salió mal",
		}
		templ.RenderFragments(
			r.Context(),
			w,
			dashboard.ImageSelectorModal(config, result),
			"imagesGrid",
		)
		log.Printf("failed to filter images for selector API: %v\n", err)
		return
	}

	config.SelectedIds = filters.Pinned
	// Return JSON response
	err = templ.RenderFragments(
		r.Context(),
		w,
		dashboard.ImageSelectorModal(config, result),
		"imagesGrid",
	)
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render images grid: %v\n", err)
		return
	}
}
