package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	dashboardComponents "github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
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

	// Public quote request form routes
	router.HandleFunc("GET /solicitar-cotizacion", RenderQuoteRequest)
	router.HandleFunc("POST /solicitar-cotizacion", HandleQuoteRequestSubmission)

	// Cart management endpoints for quote form
	router.HandleFunc("GET /cotizacion/carrito", GetQuoteCart)
	router.HandleFunc("PUT /cotizacion/carrito/{id}", UpdateQuoteCartItem)
	router.HandleFunc("DELETE /cotizacion/carrito/{id}", RemoveFromQuoteCart)
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
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		renderQuoteRequestError(w, &pages.QuoteRequestFormState{
			ServerError: "Error inesperado al procesar el formulario",
		})
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
	formState := &pages.QuoteRequestFormState{
		NameValue:      name,
		PhoneValue:     phone,
		EventDateValue: eventDate,
		EventTypeValue: eventType,
		Cart:           cart.Items,
		EventKinds:     eventKinds,
	}

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

	if eventDate == "" {
		formState.EventDateError = "La fecha del evento es requerida"
		hasErrors = true
	}
	eventTime, err := time.Parse("2006-01-02T15:04", eventDate)
	if err != nil {
		formState.EventDateError = "La fecha del evento no es válida"
		hasErrors = true
	} else if eventTime.Before(time.Now()) {
		formState.EventDateError = "La fecha del evento debe ser en el futuro"
		hasErrors = true
	}

	if hasErrors {
		w.WriteHeader(http.StatusBadRequest)
		renderQuoteRequestError(w, formState)
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
		log.Printf("failed to create quote: %v\n", err)
		formState.ServerError = "Error al guardar la cotización. Por favor, inténtalo de nuevo."
		w.WriteHeader(http.StatusInternalServerError)
		renderQuoteRequestError(w, formState)
		return
	}

	// Success response with toast
	w.Header().Set("X-Includes-Toast", "true")
	w.WriteHeader(http.StatusOK)

	// Render success form with toast
	successToast := components.NewToastData(
		"¡Cotización enviada exitosamente! Nos pondremos en contacto contigo pronto.",
		components.ToastSuccess,
		5000,
		true,
		false,
	)

	// Render both the form and the toast
	err = components.ToasterToast(successToast).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render success toast: %v\n", err)
	}

	// Render success form state
	successForm := &pages.QuoteRequestFormState{}
	err = pages.QuoteRequest(successForm).Render(r.Context(), w)
	if err != nil {
		log.Printf("failed to render success form: %v\n", err)
	}
}

func renderQuoteRequestError(w http.ResponseWriter, formState *pages.QuoteRequestFormState) {
	err := pages.QuoteRequest(formState).Render(context.Background(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Error inesperado"))
		log.Printf("failed to render QuoteRequest error form: %v\n", err)
	}
}

// GetQuoteCart returns the cart items for the quote form
func GetQuoteCart(w http.ResponseWriter, r *http.Request) {
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		cartID = ""
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		log.Printf("failed to get cart: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al cargar el carrito"))
		return
	}

	// Render cart items
	cartItemsHTML := ""
	for _, item := range cart.Items {
		// This would need to be implemented as a separate template component
		// For now, return a simple response
		cartItemsHTML += fmt.Sprintf(`
			<div class="cart-item" data-product-id="%s">
				<span>%s - Cantidad: %d</span>
			</div>
		`, item.ProductID, item.Name, item.Quantity)
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(cartItemsHTML))
}

// UpdateQuoteCartItem updates the quantity of a cart item
func UpdateQuoteCartItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")
	if itemID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Item ID is required"))
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid form data"))
		return
	}

	quantityStr := r.FormValue("quantity")
	if quantityStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Quantity is required"))
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid quantity"))
		return
	}

	// Get cart and update item
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cart not found"))
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		log.Printf("failed to get cart: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al actualizar el carrito"))
		return
	}

	// Update item quantity
	for _, item := range cart.Items {
		if item.ProductID == itemID {
			item.Quantity = quantity
			break
		}
	}

	cart.SetField("items", cart.Items)
	err = cart.Save(r.Context())
	if err != nil {
		log.Printf("failed to save cart: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al guardar el carrito"))
		return
	}

	// Return updated cart items (redirect to GetQuoteCart)
	GetQuoteCart(w, r)
}

// RemoveFromQuoteCart removes an item from the cart
func RemoveFromQuoteCart(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")
	if itemID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Item ID is required"))
		return
	}

	// Get cart
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cart not found"))
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		log.Printf("failed to get cart: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al cargar el carrito"))
		return
	}

	// Remove item
	cart.RemoveItem(itemID)
	err = cart.Save(r.Context())
	if err != nil {
		log.Printf("failed to save cart: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al guardar el carrito"))
		return
	}

	// Return updated cart items
	GetQuoteCart(w, r)
}

func isValidEmail(email string) bool {
	// Simple email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".") && len(email) > 5
}

func buildQuoteComments(email, eventType, comments, cartData string) string {
	var parts []string

	parts = append(parts, "Email: "+email)
	parts = append(parts, "Tipo de evento: "+eventType)

	if comments != "" {
		parts = append(parts, "Comentarios: "+comments)
	}

	// Parse and include cart data if available
	if cartData != "" {
		var cart map[string]interface{}
		if err := json.Unmarshal([]byte(cartData), &cart); err == nil {
			if items, ok := cart["items"].([]interface{}); ok && len(items) > 0 {
				parts = append(parts, "Productos seleccionados:")
				for _, item := range items {
					if itemMap, ok := item.(map[string]interface{}); ok {
						name, _ := itemMap["name"].(string)
						quantity, _ := itemMap["quantity"].(float64)
						if name != "" {
							parts = append(parts, "- "+name+" (Cantidad: "+formatQuantity(quantity)+")")
						}
					}
				}
			}
		}
	}

	return strings.Join(parts, "\n")
}

func formatQuantity(quantity float64) string {
	if quantity == float64(int(quantity)) {
		return fmt.Sprintf("%.0f", quantity)
	}
	return fmt.Sprintf("%.1f", quantity)
}
