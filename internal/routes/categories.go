package routes

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/uploads"
)

func RegisterCategoriesRoutes(router *customServeMux) {
	// HTMX routes that respond with templ components
	router.HandleFunc("GET /categorias", auth.ValidateAuth(RenderCategoriesTable))
	router.HandleFunc("GET /categorias/nueva", auth.ValidateAuth(RenderNewCategoryForm))
	router.HandleFunc("POST /categorias/nueva", auth.ValidateAuth(CreateCategoryAndReturnTable))
	router.HandleFunc("GET /categorias/{id}", auth.ValidateAuth(RenderCategory))
	router.HandleFunc("PUT /categorias/{id}", auth.ValidateAuth(UpdateCategoryAndReturnTable))
	router.HandleFunc("DELETE /categorias", auth.ValidateAuth(DeleteCategoriesAndReturnTable))
	router.HandleFunc("DELETE /categorias/{id}", auth.ValidateAuth(DeleteCategoryAndReturnTable))

	// Dashboard specific routes
	router.HandleFunc("GET /panel/categorias/select", RenderCategorySelect)

	// Legacy JSON API routes
	router.HandleFunc("GET /api/categories", GetCategories)
	router.HandleFunc("POST /api/categories", CreateCategory)
	router.HandleFunc("PUT /api/categories/{id}", UpdateCategory)
	router.HandleFunc("DELETE /api/categories/{id}", DeleteCategory)
}

func RenderNewCategoryForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// If request is AJAX, render components
	if r.Header.Get("HX-Request") == "true" {
		component := dashboard.CategoryCreateModal("")
		component.Render(r.Context(), w)
	} else {
		// Else render page
		// TODO: render page
	}
}

func CreateCategoryAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se creó la categoría exitosamente", components.ToastSuccess, 3000, true, false)

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Get form values
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	longDescription := strings.TrimSpace(r.FormValue("long_description"))

	// Validate required fields
	if name == "" || description == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Nombre y descripción son requeridos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoryCreateModal("Nombre y descripción son requeridos"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Create new category
	category := &db.Category{
		Name:            name,
		Slug:            internal.Slugify(name),
		Description:     description,
		LongDescription: longDescription,
	}

	// Create category in database
	err = db.CreateCategory(category)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al crear la categoría"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoryCreateModal("Error al crear la categoría"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to create category: %v\n", err)
		return
	}

	// Get updated categories list
	filters := parseCategoryFilters(r)
	result, err := db.FilterCategories(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar las categorías"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al recuperar las categorías"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get categories after create: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.CategoriesTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func RenderCategory(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	category, err := db.FindCategoryByID(id)

	// If request is AJAX, render components
	if r.Header.Get("HX-Request") == "true" {
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			dashboard.CategoryModal(category, "Categoría no encontrada").Render(r.Context(), w)
			log.Printf("failed to find category: %v\n", err)
			return
		}

		component := dashboard.CategoryModal(category, "")
		component.Render(r.Context(), w)
	} else {
		// Else render page
		// TODO: render page
	}
}

func RenderCategoriesTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Parse query parameters
	filters := parseCategoryFilters(r)

	// Get filtered categories
	result, err := db.FilterCategories(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get categories"))
		log.Printf("failed to filter categories: %v\n", err)
		return
	}

	// Render the template with the results
	component := dashboard.CategoriesTable(result)
	component.Render(r.Context(), w)
}

func UpdateCategoryAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se actualizó la categoría", components.ToastSuccess, 3000, true, false)

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Get existing category
	category, err := db.FindCategoryByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		toastData.Message = "Categoría no encontrada"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Categoría no encontrada"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to find category: %v\n", err)
		return
	}

	// Update category fields
	category.Name = strings.TrimSpace(r.FormValue("name"))
	category.Slug = strings.TrimSpace(r.FormValue("slug"))
	category.Description = strings.TrimSpace(r.FormValue("description"))
	category.LongDescription = strings.TrimSpace(r.FormValue("long_description"))

	// Validate required fields
	if category.Name == "" || category.Slug == "" || category.Description == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Nombre, slug y descripción son requeridos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Campos requeridos faltantes"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Update category in database
	err = db.UpdateCategory(category)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar la categoría"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al actualizar la categoría"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to update category: %v\n", err)
		return
	}

	// Get updated categories list
	filters := parseCategoryFilters(r)
	result, err := db.FilterCategories(filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar las categorías"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al recuperar las categorías"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get categories after update: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.CategoriesTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func DeleteCategoryAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminó la categoría", components.ToastSuccess, 3000, true, false)

	err := db.DeleteCategory(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al eliminar la categoría"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al eliminar la categoría"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to delete category: %v\n", err)
		return
	}

	filters := parseCategoryFilters(r)
	result, err := db.FilterCategories(filters)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar las categorías"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al recuperar las categorías"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get categories after delete: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.CategoriesTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func DeleteCategoriesAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	td := components.NewToastData("Se eliminaron las categorías", components.ToastSuccess, 3000, true, false)

	var err error
	categoryIds := r.URL.Query()["ids"]
	if len(categoryIds) > 0 {
		for _, id := range categoryIds {
			err = db.DeleteCategory(id)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				td.Message = "Error al eliminar las categorías"
				td.Type = components.ToastError
				comp := templ.Join(
					dashboard.CategoriesTable(&db.CategoryFilterResult{HasError: true, Error: "Error al eliminar las categorías"}),
					components.ToasterToast(td),
				)
				comp.Render(r.Context(), w)
				log.Printf("failed to delete categories: %v\n", err)
				return
			}
		}
	}

	filters := parseCategoryFilters(r)
	result, err := db.FilterCategories(filters)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusInternalServerError)
		td.Message = "Error al recuperar las categorías"
		td.Type = components.ToastError
		result := &db.CategoryFilterResult{HasError: true, Error: "Error al recuperar las categorías"}
		comp := templ.Join(
			dashboard.CategoriesTable(result),
			components.ToasterToast(td),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get categories after bulk delete: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.CategoriesTable(result),
		components.ToasterToast(td),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func RenderCategorySelect(w http.ResponseWriter, r *http.Request) {
	selectedCategory := r.URL.Query().Get("selected")
	params := components.CategorySelectParams{
		Selected:  selectedCategory,
		ValueType: components.CategorySelectValueTypeID,
		Args:      map[string]string{},
	}

	var err error
	rawArgs := r.URL.Query().Get("withHtmxAttrs")
	args := map[string]string{}
	if rawArgs != "" {
		err = json.Unmarshal([]byte(rawArgs), &args)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			components.CategorySelect([]db.Category{}, &params).Render(r.Context(), w)
			log.Printf("failed to unmarshal withHtmxAttrs: %v\n", err)
			return
		}
	}
	params.Args = args

	ctgs, err := db.FindAllCategories()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		components.CategorySelect([]db.Category{}, &params).Render(r.Context(), w)
		log.Printf("failed to find categories: %v\n", err)
		return
	}

	components.CategorySelect(internal.PtrSliceToPlainSlice(ctgs), &params).Render(r.Context(), w)
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

// parseCategoryFilters extracts and parses filter parameters from the request
func parseCategoryFilters(r *http.Request) db.CategoryFilterParams {
	query := r.URL.Query()

	filters := db.CategoryFilterParams{
		Search:     strings.TrimSpace(query.Get("search")),
		SearchMode: db.SearchModeFullText, // Only fulltext for now
	}

	// Parse sorting
	rawSort := query.Get("sort")
	filters.Sort = parseCategorySort(rawSort)

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

	return filters
}

// parseCategorySort converts sort parameter to appropriate sort string
func parseCategorySort(s string) string {
	switch strings.ToLower(s) {
	case "name_asc", "name", "":
		return "name_asc"
	case "name_desc":
		return "name_desc"
	case "newest":
		return "newest"
	case "oldest":
		return "oldest"
	default:
		return "name_asc"
	}
}
