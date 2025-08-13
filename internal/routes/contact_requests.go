package routes

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	dashboardComponents "github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
	dashboardPages "github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/whatsapp"
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

	// Public quote request form routes
	router.HandleFunc("GET /solicitar-cotizacion", RenderQuoteRequest)
	router.HandleFunc("POST /solicitar-cotizacion", HandleQuoteRequestSubmission)
	router.HandleFunc("POST /solicitar-contacto", HandleContactFormSubmission)

	// Cart management endpoints for quote form
	router.HandleFunc("PUT /cotizacion/carrito/items/{id}", UpdateQuoteCartItem)
	router.HandleFunc("DELETE /cotizacion/carrito/items/{id}", RemoveFromQuoteCart)
}

func RenderContactRequestsPage(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := dashboardPages.ContactRequests().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
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
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render ContactRequestModal err: %v\n", err)
	}
}

func RenderEditContactRequestForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	id := r.PathValue("id")

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
	}
	quote.RequestType = db.QuoteRequestType(r.FormValue("request_type"))
	quote.Status = db.QuoteStatus(r.FormValue("status"))

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
		quote.EventKindID = sql.NullString{String: eventKindID, Valid: true}
	}

	// Parse optional comments
	if comments := r.FormValue("comments"); comments != "" {
		quote.Comments = sql.NullString{String: comments, Valid: true}
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

// Public Quote Request Form Handlers

func RenderQuoteRequest(w http.ResponseWriter, r *http.Request) {
	// Get or create cart from session
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		// Create new cart ID if none exists
		cartID = ""
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		log.Printf("failed to get or create cart: %v\n", err)
		cart = &db.Cart{Items: []*db.CartItem{}}
	}

	// Get event kinds
	eventKinds, err := db.FindAllEventKinds()
	if err != nil {
		log.Printf("failed to find event kinds: %v\n", err)
		eventKinds = []*db.EventKind{}
	}

	formState := &pages.QuoteRequestFormState{
		Cart:       cart.Items,
		EventKinds: eventKinds,
	}

	err = pages.QuoteRequest(formState).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render QuoteRequest err: %v\n", err)
	}
}

func HandleQuoteRequestSubmission(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Solicitud enviada con éxito. Serás redirigido en breve.", components.ToastSuccess, 3000, true, false)
	formState := &pages.QuoteRequestFormState{}
	err := r.ParseForm()
	if err != nil {
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "Error al actualizar el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "quoteRequestForm")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Extract form values
	name := strings.TrimSpace(r.FormValue("name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	eventDate := strings.TrimSpace(r.FormValue("event_date"))
	eventType := strings.TrimSpace(r.FormValue("event_type"))

	// Get cart and event kinds for form state
	cartID, _ := db.GetCartIDFromRequest(r)
	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		cart = &db.Cart{Items: []*db.CartItem{}}
	}

	eventKinds, err := db.FindAllEventKinds()
	if err != nil {
		eventKinds = []*db.EventKind{}
	}

	// Validate form data
	formState.NameValue = name
	formState.PhoneValue = phone
	formState.EventDateValue = eventDate
	formState.EventTypeValue = eventType
	formState.Cart = cart.Items
	formState.EventKinds = eventKinds

	hasErrors := false

	// Validate required fields
	if name == "" {
		formState.NameError = "El nombre es requerido"
		hasErrors = true
	} else if len(name) < 2 {
		formState.NameError = "El nombre debe tener al menos 2 caracteres"
		hasErrors = true
	}

	if phone == "" {
		formState.PhoneError = "El teléfono es requerido"
		hasErrors = true
	} else if len(phone) < 10 {
		formState.PhoneError = "El teléfono debe tener al menos 10 dígitos"
		hasErrors = true
	}

	var eventTime time.Time
	if eventDate != "" {
		eventTime, err = time.Parse("2006-01-02T15:04", eventDate)
		if err != nil {
			formState.EventDateError = "La fecha del evento no es válida"
			hasErrors = true
		} else if eventTime.Before(time.Now()) {
			formState.EventDateError = "La fecha del evento debe ser en el futuro"
			hasErrors = true
		}
	}

	if hasErrors {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Type = components.ToastError
		toastData.Message = "Error al enviar la solicitud"
		formState.ServerError = "Ocurrió un error al procesar el formulario"
		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "quoteRequestForm")
		log.Println("form validation failed")
		return
	}

	// Create quote object
	quote := &db.Quote{
		CustomerName:  name,
		CustomerPhone: phone,
		TimeStart:     &eventTime,
		RequestType:   db.QuoteRequestTypeBudget,
		Status:        db.QuoteStatusPending,
	}

	// Add cart ID if available
	if cart.ID != "" {
		quote.CartID = sql.NullString{String: cart.ID, Valid: true}
	}

	// Add event kind ID if selected
	if eventType != "" && eventType != "otro" {
		quote.EventKindID = sql.NullString{String: eventType, Valid: true}
	}

	// Save quote to database
	err = db.CreateQuote(quote)
	if err != nil {
		toastData.Type = components.ToastError
		toastData.Message = "Error al guardar la solicitud"
		formState.ServerError = "Ocurrió un error al guardar la solicitud"
		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "quoteRequestForm")
		return
	}

	formState.IsSuccessful = true
	combined := templ.Join(
		components.ToasterToast(toastData),
		pages.QuoteRequest(formState),
	)
	err = templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "quoteRequestForm")
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render success form: %v\n", err)
	}

	// Send notification to customer
	go func() {
		quote.CustomerPhone, err = internal.FormatPhone(quote.CustomerPhone)
		if err != nil {
			log.Printf("failed to format phone number: %v\n", err)
			return
		}

		err := whatsapp.SendQuoteRequestConfirmation(quote)
		if err != nil {
			log.Printf("failed to send quote request confirmation: %v\n", err)
		}
	}()
}

func HandleContactFormSubmission(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Solicitud enviada con éxito", components.ToastSuccess, 3000, true, false)
	formState := &components.ContactFormState{}
	err := r.ParseForm()
	if err != nil {
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"

		templ.Join(
			components.ToasterToast(toastData),
			components.ContactForm(formState),
		).Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Extract form values
	name := strings.TrimSpace(r.FormValue("name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	hasErrors := false
	formState.NameValue = name
	formState.PhoneValue = phone

	// Validate
	if name == "" {
		formState.NameError = "El nombre es requerido"
		hasErrors = true
	} else if len(name) < 2 {
		formState.NameError = "El nombre debe tener al menos 2 caracteres"
		hasErrors = true
	}

	if phone == "" {
		formState.PhoneError = "El teléfono es requerido"
		hasErrors = true
	} else if len(phone) < 10 {
		formState.PhoneError = "El teléfono debe tener al menos 10 dígitos"
		hasErrors = true
	}

	if hasErrors {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Type = components.ToastError
		toastData.Message = "Error al enviar la solicitud"
		formState.ServerError = "El formulario contiene información inválida"
		templ.Join(
			components.ToasterToast(toastData),
			components.ContactForm(formState),
		).Render(r.Context(), w)
		log.Printf("form validation failed: %v\n", formState)
		return
	}

	// Create quote object
	quote := &db.Quote{
		CustomerName:  name,
		CustomerPhone: phone,
		RequestType:   db.QuoteRequestTypeContact,
		Status:        db.QuoteStatusPending,
	}

	// Save quote to database
	err = db.CreateQuote(quote)
	if err != nil {
		toastData.Type = components.ToastError
		toastData.Message = "Error al guardar la solicitud"
		formState.ServerError = "Ocurrió un error al guardar la solicitud"
		templ.Join(
			components.ToasterToast(toastData),
			components.ContactForm(formState),
		).Render(r.Context(), w)
		return
	}

	formState.IsSuccessful = true
	err = templ.Join(
		components.ToasterToast(toastData),
		components.ContactForm(formState),
	).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render success form: %v\n", err)
	}
}

// UpdateQuoteCartItem updates the quantity of a cart item
func UpdateQuoteCartItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se actualizó el carrito", components.ToastSuccess, 3000, true, false)
	formState := &pages.QuoteRequestFormState{ShouldOobSwapItems: true}

	itemID := r.PathValue("id")
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "Error al actualizar el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to get cart id: %v\n", err)
		return
	}

	err = r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "Error al actualizar el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	quantityStr := r.FormValue("quantity")
	qty, err := strconv.Atoi(quantityStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.CartError = "La cantidad no es válida"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "No se encontró el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	cart.UpdateItemQty(itemID, qty)
	err = cart.Save(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "No se encontró el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	formState.Cart = cart.Items
	combined := templ.Join(
		components.ToasterToast(toastData),
		pages.QuoteRequest(formState),
	)
	err = templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render cartItems: %v\n", err)
	}
}

// RemoveFromQuoteCart removes an item from the cart
func RemoveFromQuoteCart(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminó el producto del carrito", components.ToastSuccess, 3000, true, false)
	formState := &pages.QuoteRequestFormState{ShouldOobSwapItems: true}
	itemID := r.PathValue("id")

	// Get cart
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "No se encontró el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "No se encontró el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Remove item
	cart.RemoveItem(itemID)
	err = cart.Save(r.Context())
	if err != nil {
		toastData.Type = components.ToastError
		toastData.Message = "Error al actualizar el carrito"
		formState.ServerError = "Ocurrió un error inesperado"
		formState.CartError = "No se encontró el carrito"

		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	formState.Cart = cart.Items
	combined := templ.Join(
		components.ToasterToast(toastData),
		pages.QuoteRequest(formState),
	)
	err = templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "cartItems")
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to render cartItems: %v\n", err)
	}
}
