package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "single"
	}

	targetField := r.URL.Query().Get("target_field")
	title := r.URL.Query().Get("title")
	if title == "" {
		title = "Seleccionar Imagen"
	}

	maxSelection := 15
	if maxStr := r.URL.Query().Get("max_selection"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil && max > 0 {
			maxSelection = max
		}
	}

	selectedIds := []string{}
	if selectedStr := r.URL.Query().Get("selected_ids"); selectedStr != "" {
		selectedIds = strings.Split(selectedStr, ",")
	}

	// Parse filter parameters for initial load
	filters := parseImageFilters(r)
	if filters.Limit == 0 {
		filters.Limit = 12 // Default grid size for selector
	}

	// Get images
	result, err := db.FilterImages(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get images"))
		log.Printf("failed to filter images for selector: %v\n", err)
		return
	}

	config := dashboard.ImageSelectorConfig{
		Mode:         mode,
		TargetField:  targetField,
		Title:        title,
		MaxSelection: maxSelection,
		SelectedIds:  selectedIds,
		AllowUpload:  true,
	}

	component := dashboard.ImageSelectorModal(config, result)
	component.Render(r.Context(), w)
}

func GetImagesForSelector(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Parse filter parameters
	filters := parseImageFilters(r)
	if filters.Limit == 0 {
		filters.Limit = 12 // Default grid size for selector
	}

	// Get images
	result, err := db.FilterImages(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get images"))
		log.Printf("failed to filter images for selector API: %v\n", err)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
