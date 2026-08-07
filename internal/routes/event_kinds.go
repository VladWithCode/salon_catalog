package routes

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
)

func RegisterEventKindsRoutes(router *customServeMux) {
	// HTMX routes that respond with templ components (for AJAX requests)
	router.HandleFunc("GET /panel/tipos-eventos/table", auth.ValidateAuth(RenderEventKindsTable))
	router.HandleFunc("GET /panel/tipos-eventos/modal/nuevo", auth.ValidateAuth(RenderNewEventKindForm))
	router.HandleFunc("POST /panel/tipos-eventos/nuevo", auth.ValidateAuth(CreateEventKindAndReturnTable))
	router.HandleFunc("GET /panel/tipos-eventos/modal/{id}", auth.ValidateAuth(RenderEventKind))
	router.HandleFunc("PUT /panel/tipos-eventos/{id}", auth.ValidateAuth(UpdateEventKindAndReturnTable))
	router.HandleFunc("DELETE /panel/tipos-eventos/{id}", auth.ValidateAuth(DeleteEventKindAndReturnTable))

	// API routes for external access
	router.HandleFunc("GET /api/event-kinds", GetEventKinds)
	router.HandleFunc("GET /api/event-kinds/{id}", GetEventKindByID)
	router.HandleFunc("POST /api/event-kinds", auth.ValidateAuth(CreateEventKindAPI))
	router.HandleFunc("PUT /api/event-kinds/{id}", auth.ValidateAuth(UpdateEventKindAPI))
	router.HandleFunc("DELETE /api/event-kinds/{id}", auth.ValidateAuth(DeleteEventKindAPI))
}

func parseEventKindFilterParams(r *http.Request) (*db.EventKindFilterParams, error) {
	params := &db.EventKindFilterParams{}
	params.Search = r.URL.Query().Get("search")
	params.Page, _ = strconv.Atoi(r.FormValue("page"))
	params.Limit, _ = strconv.Atoi(r.FormValue("limit"))

	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Limit <= 0 {
		params.Limit = 20
	}

	return params, nil
}

func RenderEventKindsTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	filters, err := parseEventKindFilterParams(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to parse filters"))
		log.Printf("failed to parse filters: %v\n", err)
		return
	}

	result, err := db.FilterEventKinds(*filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get event kinds"))
		log.Printf("failed to filter event kinds: %v\n", err)
		return
	}

	component := dashboard.EventKindsTable(result)
	component.Render(r.Context(), w)
}

func RenderNewEventKindForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	if r.Header.Get("HX-Request") == "true" {
		component := dashboard.EventKindCreateModal("")
		component.Render(r.Context(), w)
	} else {
		// TODO: render page
	}
}

func CreateEventKindAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se creó el tipo de evento exitosamente", components.ToastSuccess, 3000, true, false)

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindCreateModal("Error al procesar el formulario"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "El nombre es requerido"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindCreateModal("El nombre es requerido"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	eventKind := &db.EventKind{
		Name:        name,
		Description: description,
	}

	err = db.CreateEventKind(eventKind)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al crear el tipo de evento"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindCreateModal("Error al crear el tipo de evento"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to create event kind: %v\n", err)
		return
	}

	filters, _ := parseEventKindFilterParams(r)
	result, err := db.FilterEventKinds(*filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar los tipos de eventos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindsTable(&db.EventKindFilterResult{HasError: true, Error: "Error al recuperar los tipos de eventos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get event kinds after create: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.EventKindsTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func RenderEventKind(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	eventKind, err := db.FindEventKindByID(id)

	if r.Header.Get("HX-Request") == "true" {
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			dashboard.EventKindCreateModal("Tipo de evento no encontrado").Render(r.Context(), w)
			log.Printf("failed to find event kind: %v\n", err)
			return
		}

		formState := forms.NewEventKindFormStateFromEventKind("edit", eventKind)
		component := dashboard.EventKindEditModal(formState, "")
		component.Render(r.Context(), w)
	} else {
		// TODO: render page
	}
}

func UpdateEventKindAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se actualizó el tipo de evento", components.ToastSuccess, 3000, true, false)

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindsTable(&db.EventKindFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	eventKind, err := db.FindEventKindByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		toastData.Message = "Tipo de evento no encontrado"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindsTable(&db.EventKindFilterResult{HasError: true, Error: "Tipo de evento no encontrado"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to find event kind: %v\n", err)
		return
	}

	eventKind.Name = strings.TrimSpace(r.FormValue("name"))
	eventKind.Description = strings.TrimSpace(r.FormValue("description"))

	if eventKind.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "El nombre es requerido"
		toastData.Type = components.ToastError
		formState := forms.NewEventKindFormStateFromEventKind("edit", eventKind)
		comp := templ.Join(
			dashboard.EventKindEditModal(formState, "El nombre es requerido"),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	err = db.UpdateEventKind(eventKind)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el tipo de evento"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindsTable(&db.EventKindFilterResult{HasError: true, Error: "Error al actualizar el tipo de evento"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to update event kind: %v\n", err)
		return
	}

	filters, _ := parseEventKindFilterParams(r)
	result, err := db.FilterEventKinds(*filters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar los tipos de eventos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindsTable(&db.EventKindFilterResult{HasError: true, Error: "Error al recuperar los tipos de eventos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get event kinds after update: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.EventKindsTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

func DeleteEventKindAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminó el tipo de evento", components.ToastSuccess, 3000, true, false)

	err := db.DeleteEventKind(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al eliminar el tipo de evento"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindsTable(&db.EventKindFilterResult{HasError: true, Error: "Error al eliminar el tipo de evento"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to delete event kind: %v\n", err)
		return
	}

	filters, _ := parseEventKindFilterParams(r)
	result, err := db.FilterEventKinds(*filters)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recuperar los tipos de eventos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.EventKindsTable(&db.EventKindFilterResult{HasError: true, Error: "Error al recuperar los tipos de eventos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to get event kinds after delete: %v\n", err)
		return
	}

	comp := templ.Join(
		dashboard.EventKindsTable(result),
		components.ToasterToast(toastData),
	)
	err = comp.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("failed to render response: %v\n", err)
		return
	}
}

// API endpoints for external access
func GetEventKinds(w http.ResponseWriter, r *http.Request) {
	eventKinds, err := db.FindAllEventKinds()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find event kinds"))
		log.Printf("failed to find event kinds: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Simple JSON response
	w.Write([]byte("["))
	for i, eventKind := range eventKinds {
		if i > 0 {
			w.Write([]byte(","))
		}
		w.Write([]byte(`{"id":"` + eventKind.ID + `","name":"` + eventKind.Name + `","description":"` + eventKind.Description + `"}`))
	}
	w.Write([]byte("]"))
}

func GetEventKindByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	eventKind, err := db.FindEventKindByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Event kind not found"))
		log.Printf("failed to find event kind: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"id":"` + eventKind.ID + `","name":"` + eventKind.Name + `","description":"` + eventKind.Description + `"}`))
}

func CreateEventKindAPI(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse form"))
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Name is required"))
		return
	}

	eventKind := &db.EventKind{
		Name:        name,
		Description: description,
	}

	err = db.CreateEventKind(eventKind)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create event kind"))
		log.Printf("failed to create event kind: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"id":"` + eventKind.ID + `","name":"` + eventKind.Name + `","description":"` + eventKind.Description + `"}`))
}

func UpdateEventKindAPI(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse form"))
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	eventKind, err := db.FindEventKindByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Event kind not found"))
		log.Printf("failed to find event kind: %v\n", err)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))

	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Name is required"))
		return
	}

	eventKind.Name = name
	eventKind.Description = description

	err = db.UpdateEventKind(eventKind)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to update event kind"))
		log.Printf("failed to update event kind: %v\n", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"id":"` + eventKind.ID + `","name":"` + eventKind.Name + `","description":"` + eventKind.Description + `"}`))
}

func DeleteEventKindAPI(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")

	err := db.DeleteEventKind(id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to delete event kind"))
		log.Printf("failed to delete event kind: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
