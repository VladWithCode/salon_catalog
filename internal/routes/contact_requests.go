package routes

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	dashboardComponents "github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	dashboardPages "github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
)

func RegisterContactRequestsRoutes(router *customServeMux) {
	// Main dashboard page
	router.HandleFunc("GET /panel/solicitudes", auth.ValidateAuth(RenderContactRequestsPage))

	// HTMX routes that respond with templ components
	router.HandleFunc("GET /solicitudes/table", auth.ValidateAuth(RenderContactRequestsTable))
	router.HandleFunc("GET /solicitudes/filters", auth.ValidateAuth(RenderContactRequestsFiltersModal))
	router.HandleFunc("GET /solicitudes/{id}", auth.ValidateAuth(RenderContactRequest))
	router.HandleFunc("GET /solicitudes/{id}/editar", auth.ValidateAuth(RenderEditContactRequestForm))
	router.HandleFunc("PUT /solicitudes/{id}", auth.ValidateAuth(UpdateContactRequestAndReturnTable))
	router.HandleFunc("DELETE /solicitudes", auth.ValidateAuth(DeleteContactRequestsAndReturnTable))
	router.HandleFunc("DELETE /solicitudes/{id}", auth.ValidateAuth(DeleteContactRequestAndReturnTable))
	router.HandleFunc("PATCH /solicitudes/status", auth.ValidateAuth(UpdateContactRequestsStatusAndReturnTable))
}

func RenderContactRequestsPage(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := dashboardPages.ContactRequests().Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render ContactRequests page err: %v\n", err)
	}
}

func RenderContactRequestsTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Parse query parameters
	filters := db.QuoteFilterParams{
		CustomerName:   r.URL.Query().Get("customer_name"),
		Phone:          r.URL.Query().Get("phone"),
		CreatedFrom:    r.URL.Query().Get("created_from"),
		CreatedTo:      r.URL.Query().Get("created_to"),
		EventStartFrom: r.URL.Query().Get("event_start_from"),
		EventStartTo:   r.URL.Query().Get("event_start_to"),
		Status:         r.URL.Query().Get("status"),
		RequestType:    r.URL.Query().Get("request_type"),
		Comments:       r.URL.Query().Get("comments"),
		Sort:           r.URL.Query().Get("sort"),
		Page:           1,
		Limit:          20,
	}

	// Parse page parameter
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filters.Page = page
		}
	}

	// Parse limit parameter
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filters.Limit = limit
		}
	}

	// Get filtered quotes
	result, err := db.FilterQuotes(filters)
	if err != nil {
		log.Printf("failed to filter quotes: %v\n", err)
		result = &db.QuoteFilterResult{
			Quotes:   []*db.Quote{},
			Total:    0,
			HasError: true,
			Error:    "Error al cargar las solicitudes",
		}
	}

	// Render table component
	component := dashboardComponents.ContactRequestsTable(result)
	err = component.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render ContactRequestsTable err: %v\n", err)
	}
}

func RenderContactRequest(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Quote ID is required"))
		return
	}

	quote, err := db.FindQuoteByID(id)
	if err != nil {
		log.Printf("failed to find quote by id %s: %v\n", id, err)
		component := dashboardComponents.ContactRequestModal(nil, "Solicitud no encontrada")
		component.Render(r.Context(), w)
		return
	}

	component := dashboardComponents.ContactRequestModal(quote, "")
	err = component.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render ContactRequestModal err: %v\n", err)
	}
}

func RenderEditContactRequestForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Quote ID is required"))
		return
	}

	quote, err := db.FindQuoteByID(id)
	if err != nil {
		log.Printf("failed to find quote by id %s: %v\n", id, err)
		component := dashboardComponents.ContactRequestEditModal(nil, nil, "Solicitud no encontrada")
		component.Render(r.Context(), w)
		return
	}

	// Get event kinds for the dropdown
	eventKinds, err := db.FindAllEventKinds()
	if err != nil {
		log.Printf("failed to find event kinds: %v\n", err)
		eventKinds = []*db.EventKind{}
	}

	component := dashboardComponents.ContactRequestEditModal(quote, eventKinds, "")
	err = component.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render ContactRequestEditModal err: %v\n", err)
	}
}

func UpdateContactRequestAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Quote ID is required"))
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid form data"))
		return
	}

	// Parse form data
	quote := &db.Quote{
		ID:            id,
		CustomerName:  r.FormValue("customer_name"),
		CustomerPhone: r.FormValue("customer_phone"),
		RequestType:   r.FormValue("request_type"),
		Status:        r.FormValue("status"),
		Comments:      r.FormValue("comments"),
	}

	// Parse optional datetime fields
	if timeStartStr := r.FormValue("time_start"); timeStartStr != "" {
		if timeStart, err := time.Parse("2006-01-02T15:04", timeStartStr); err == nil {
			quote.TimeStart = &timeStart
		}
	}

	if timeEndStr := r.FormValue("time_end"); timeEndStr != "" {
		if timeEnd, err := time.Parse("2006-01-02T15:04", timeEndStr); err == nil {
			quote.TimeEnd = &timeEnd
		}
	}

	// Parse optional event kind ID
	if eventKindID := r.FormValue("event_kind_id"); eventKindID != "" {
		quote.EventKindID = &eventKindID
	}

	// Update quote
	err = db.UpdateQuote(quote)
	if err != nil {
		log.Printf("failed to update quote: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Error al actualizar la solicitud"))
		return
	}

	// Return updated table
	RenderContactRequestsTable(w, r, a)
}

func DeleteContactRequestAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Quote ID is required"))
		return
	}

	err := db.DeleteQuote(id)
	if err != nil {
		log.Printf("failed to delete quote: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Error al eliminar la solicitud"))
		return
	}

	// Return updated table
	RenderContactRequestsTable(w, r, a)
}

func DeleteContactRequestsAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid form data"))
		return
	}

	ids := r.Form["ids"]
	if len(ids) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("No quotes selected"))
		return
	}

	err = db.DeleteQuotes(ids)
	if err != nil {
		log.Printf("failed to delete quotes: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Error al eliminar las solicitudes"))
		return
	}

	// Return updated table
	RenderContactRequestsTable(w, r, a)
}

func UpdateContactRequestsStatusAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid form data"))
		return
	}

	ids := r.Form["ids"]
	if len(ids) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("No quotes selected"))
		return
	}

	status := r.FormValue("status")
	if status == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Status is required"))
		return
	}

	err = db.UpdateQuoteStatus(ids, status)
	if err != nil {
		log.Printf("failed to update quote status: %v\n", err)
		w.WriteHeader(500)
		w.Write([]byte("Error al actualizar el estado de las solicitudes"))
		return
	}

	// Return updated table
	RenderContactRequestsTable(w, r, a)
}


func RenderContactRequestsFiltersModal(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	component := dashboardComponents.ContactRequestsFiltersModal()
	err := component.Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render ContactRequestsFiltersModal err: %v\n", err)
	}
}
