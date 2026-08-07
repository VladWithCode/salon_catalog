package routes

import (
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	componentsDashboard "github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
)

func RegisterSocialRoutes(router *customServeMux) {
	// Admin routes
	router.HandleFunc("GET /panel/socials", auth.ValidateAuth(handleSocialsIndex))
	router.HandleFunc("GET /panel/socials/new", auth.ValidateAuth(handleSocialsNew))
	router.HandleFunc("POST /panel/socials", auth.ValidateAuth(handleSocialsCreate))
	router.HandleFunc("GET /panel/socials/{id}/edit", auth.ValidateAuth(handleSocialsEdit))
	router.HandleFunc("PUT /panel/socials/{id}", auth.ValidateAuth(handleSocialsUpdate))
	router.HandleFunc("DELETE /panel/socials/{id}", auth.ValidateAuth(handleSocialsDelete))

	// Section assignment routes
	router.HandleFunc("GET /panel/socials/sections", auth.ValidateAuth(handleSocialSectionsIndex))
	router.HandleFunc("GET /panel/socials/sections/assign", auth.ValidateAuth(handleSocialAssignmentModal))
	router.HandleFunc("POST /panel/socials/sections/assign", auth.ValidateAuth(handleAssignSocialToSection))
	router.HandleFunc("GET /panel/socials/icons/selector", auth.ValidateAuth(handleSocialIconSelector))
	router.HandleFunc("PUT /panel/socials/icons/selected", auth.ValidateAuth(handleIconSelected))
	router.HandleFunc("DELETE /panel/socials/sections/{linkId}/{sectionId}", auth.ValidateAuth(handleUnassignSocialFromSection))
}

func handleSocialsIndex(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	links, err := db.GetSocialLinks()
	if err != nil {
		log.Printf("Error fetching social links: %v", err)
		http.Error(w, "Failed to fetch social links", http.StatusInternalServerError)
		return
	}

	sections, err := db.GetSocialSections()
	if err != nil {
		log.Printf("Error fetching social sections: %v", err)
		http.Error(w, "Failed to fetch social sections", http.StatusInternalServerError)
		return
	}

	sectionLinks, err := db.FilterSocialSectionLinks(nil)
	if err != nil {
		log.Printf("Error fetching section assignments: %v", err)
		http.Error(w, "Failed to fetch section assignments", http.StatusInternalServerError)
		return
	}

	dashboard.SocialsIndex(links, sections, sectionLinks.SocialSectionLink).Render(r.Context(), w)
}

func handleSocialsNew(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	formState := forms.NewSocialFormState("create")
	dashboard.SocialsNew(formState).Render(r.Context(), w)
}

func handleSocialsCreate(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	formValues := make(map[string]string)
	for key, values := range r.Form {
		if len(values) > 0 {
			formValues[key] = values[0]
		}
	}

	formState := forms.NewSocialFormStateFromMap("create", formValues)

	if err := formState.Validate(); err != nil {
		w.Header().Set("HX-Reswap", "outerHTML")
		dashboard.SocialsNew(formState).Render(r.Context(), w)
		return
	}

	link := &db.SocialLink{
		Name: formState.Values.Name,
		Link: formState.Values.Link,
	}

	if err := db.CreateSocialLink(link); err != nil {
		log.Printf("Error creating social link: %v", err)
		formState.SetErrorMessage("Error al crear el enlace social")
		w.Header().Set("HX-Reswap", "outerHTML")
		dashboard.SocialsNew(formState).Render(r.Context(), w)
		return
	}

	w.Header().Set("HX-Redirect", "/panel/socials")
	w.WriteHeader(http.StatusCreated)
}

func handleSocialsEdit(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	link, err := db.FindSocialLinkByID(id)
	if err != nil {
		log.Printf("Error finding social link by ID %s: %v", id, err)
		http.Error(w, "Social link not found", http.StatusNotFound)
		return
	}

	formState := forms.NewSocialFormStateFromSocialLink("edit", link)
	dashboard.SocialsEdit(formState, link).Render(r.Context(), w)
}

func handleSocialsUpdate(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	formValues := make(map[string]string)
	for key, values := range r.Form {
		if len(values) > 0 {
			formValues[key] = values[0]
		}
	}

	formState := forms.NewSocialFormStateFromMap("edit", formValues)

	if err := formState.Validate(); err != nil {
		link, _ := db.FindSocialLinkByID(id)
		w.Header().Set("HX-Reswap", "outerHTML")
		dashboard.SocialsEdit(formState, link).Render(r.Context(), w)
		return
	}

	link := &db.SocialLink{
		ID:   id,
		Name: formState.Values.Name,
		Link: formState.Values.Link,
	}

	if err := db.UpdateSocialLink(link); err != nil {
		log.Printf("Error updating social link: %v", err)
		formState.SetErrorMessage("Error al actualizar el enlace social")
		link, _ := db.FindSocialLinkByID(id)
		w.Header().Set("HX-Reswap", "outerHTML")
		dashboard.SocialsEdit(formState, link).Render(r.Context(), w)
		return
	}

	w.Header().Set("HX-Redirect", "/panel/socials")
	w.WriteHeader(http.StatusOK)
}

func handleSocialsDelete(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	if err := db.DeleteSocialLink(id); err != nil {
		log.Printf("Error deleting social link: %v", err)
		http.Error(w, "Failed to delete social link", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/panel/socials")
	w.WriteHeader(http.StatusOK)
}

func handleSocialSectionsIndex(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	// This would be for a dedicated sections management page
	// For now, we'll redirect to the main socials page
	http.Redirect(w, r, "/panel/socials", http.StatusSeeOther)
}

func handleAssignSocialToSection(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	linkID := r.FormValue("link_id")
	sectionID := r.FormValue("section_id")
	iconID := r.FormValue("icon_id")

	if linkID == "" || sectionID == "" || iconID == "" {
		// Return error in modal format
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
			<div class="relative top-20 mx-auto p-5 border w-11/12 md:w-1/2 shadow-lg rounded-md bg-white">
				<div class="text-center">
					<h3 class="text-lg font-bold text-red-600 mb-4">Error</h3>
					<p class="text-gray-700 mb-4">Todos los campos son obligatorios. Por favor, selecciona un enlace, una sección y un icono.</p>
					<button onclick="this.closest('.fixed').remove()" class="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700">
						Cerrar
					</button>
				</div>
			</div>
		</div>`))
		return
	}

	if err := db.CreateSocialSectionLink(linkID, sectionID, iconID); err != nil {
		log.Printf("Error assigning social link to section: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
			<div class="relative top-20 mx-auto p-5 border w-11/12 md:w-1/2 shadow-lg rounded-md bg-white">
				<div class="text-center">
					<h3 class="text-lg font-bold text-red-600 mb-4">Error</h3>
					<p class="text-gray-700 mb-4">Error al asignar el enlace a la sección.</p>
					<button onclick="this.closest('.fixed').remove()" class="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700">
						Cerrar
					</button>
				</div>
			</div>
		</div>`))
		return
	}

	// Success - redirect to main socials page
	w.Header().Set("HX-Redirect", "/panel/socials")
	w.WriteHeader(http.StatusCreated)
}

func handleSocialAssignmentModal(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	sectionID := r.URL.Query().Get("section_id")

	links, err := db.GetSocialLinks()
	if err != nil {
		log.Printf("Error fetching social links: %v", err)
		http.Error(w, "Failed to fetch social links", http.StatusInternalServerError)
		return
	}

	sections, err := db.GetSocialSections()
	if err != nil {
		log.Printf("Error fetching social sections: %v", err)
		http.Error(w, "Failed to fetch social sections", http.StatusInternalServerError)
		return
	}

	dashboard.SocialAssignmentModal(links, sections, sectionID).Render(r.Context(), w)
}

func handleSocialIconSelector(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	// Create image selector specifically for SVG icons
	// This will render the image selector modal with SVG filtering

	filters := db.ImageFilterParams{
		FileType:  "image/svg+xml", // Only SVG files
		Page:      1,
		Limit:     20,
		SortBy:    "createdAt",
		SortOrder: "desc",
	}

	result, err := db.FilterImages(filters)
	if err != nil {
		log.Printf("Error fetching SVG images: %v", err)
		http.Error(w, "Failed to load icons", http.StatusInternalServerError)
		return
	}

	// Create image selector config
	config := componentsDashboard.ImageSelectorConfig{
		Mode:           "single",
		UpdateEndpoint: "/panel/socials/icons/selected",
		Title:          "Seleccionar Icono SVG",
		MaxSelection:   1,
		SelectedIds:    []string{},
		AllowUpload:    true,
		SuccessTarget:  "#socials-assignment-modal",
	}

	componentsDashboard.ImageSelectorModal(config, result).Render(r.Context(), w)
}

func handleIconSelected(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	// Handle icon selection from the image selector
	// This endpoint receives the selected icon IDs and returns JavaScript to update the assignment modal

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	selectedIcons := r.Form["selected"]
	if len(selectedIcons) == 0 {
		http.Error(w, "No icon selected", http.StatusBadRequest)
		return
	}

	iconID := selectedIcons[0]
	if iconID == "" {
		http.Error(w, "Empty icon ID", http.StatusBadRequest)
		return
	}

	// Get icon details from database
	icon, err := db.FindImageByID(iconID)
	if err != nil {
		log.Printf("Error finding icon by ID %s: %v", iconID, err)
		http.Error(w, "Icon not found", http.StatusNotFound)
		return
	}

	// Return JavaScript to update the assignment modal and close the image selector
	response := `
	<script>
		// Close the image selector modal
		const imageModal = document.getElementById('image-selector-modal');
		if (imageModal) {
			imageModal.remove();
		}
		
		// Update the assignment modal with selected icon
		const iconIdInput = document.getElementById('icon_id');
		const selectedIconPreview = document.getElementById('selected-icon-preview');
		const selectedIconName = document.getElementById('selected-icon-name');
		const selectIconBtn = document.getElementById('select-icon-btn');
		
		if (iconIdInput && selectedIconPreview && selectedIconName && selectIconBtn) {
			iconIdInput.value = '` + iconID + `';
			selectedIconName.textContent = '` + icon.Name + `';
			selectedIconPreview.classList.remove('hidden');
			selectIconBtn.textContent = 'Cambiar Icono';
		}
	</script>
	`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func handleUnassignSocialFromSection(w http.ResponseWriter, r *http.Request, auth *auth.Auth) {
	linkID := r.PathValue("linkId")
	sectionID := r.PathValue("sectionId")

	if linkID == "" || sectionID == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	if err := db.DeleteSocialSectionLink(linkID, sectionID); err != nil {
		log.Printf("Error unassigning social link from section: %v", err)
		http.Error(w, "Failed to unassign social link from section", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/panel/socials")
	w.WriteHeader(http.StatusOK)
}
