package routes

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	dashboardComponents "github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
	dashboardPages "github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
	"github.com/vladwithcode/salon_catalog/internal/whatsapp"
)

// maxQuoteRequestJSONBytes bounds the public quote-request JSON contract
// the same way maxCartAPIRequestBytes bounds the cart API — a public,
// unauthenticated endpoint must never let a client force unbounded body
// reads.
const maxQuoteRequestJSONBytes int64 = 8 * 1024

var quoteRequestEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// quoteRequestJSONBody is the public JSON contract used only by the Next
// frontend's server-to-server Server Action — never exposed to the browser
// directly (same same-origin/no-CORS design as internal/routes/cart_api.go).
// DisallowUnknownFields at decode time rejects any field this contract does
// not know about, per Fase 11 section 9 ("rechazar campos desconocidos
// cuando sea compatible").
type quoteRequestJSONBody struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	EventDate string `json:"event_date"`
	EventType string `json:"event_type"`
}

type quoteRequestErrorCode string

const (
	quoteErrInvalidRequest         quoteRequestErrorCode = "invalid_request"
	quoteErrRequestTooLarge        quoteRequestErrorCode = "request_too_large"
	quoteErrUnsupportedMediaType   quoteRequestErrorCode = "unsupported_media_type"
	quoteErrCartEmpty              quoteRequestErrorCode = "cart_empty"
	quoteErrProductRemoved         quoteRequestErrorCode = "product_removed"
	quoteErrProductUnavailable     quoteRequestErrorCode = "product_unavailable"
	quoteErrInvalidQuantity        quoteRequestErrorCode = "invalid_quantity"
	quoteErrCatalogUnavailable     quoteRequestErrorCode = "catalog_unavailable"
	quoteErrDBError                quoteRequestErrorCode = "db_error"
	quoteErrIdempotencyKeyRequired quoteRequestErrorCode = "idempotency_key_required"
	quoteErrInvalidIdempotencyKey  quoteRequestErrorCode = "invalid_idempotency_key"
	quoteErrIdempotencyConflict    quoteRequestErrorCode = "idempotency_conflict"
)

// quoteIdempotencyKeyPattern mirrors idempotencyKeyPattern
// (internal/routes/cart_api_mutations.go) exactly — same client-generated
// opaque key contract, same 16-128 character ASCII allowlist.
var quoteIdempotencyKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{16,128}$`)

// hashQuoteIdempotencyKey / hashQuoteRequest mirror
// hashCartAPIIdempotencyKey / hashCartAPIAddItemRequest
// (internal/routes/cart_api_mutations.go): SHA-256 of the raw key (never
// stored raw), and SHA-256 of a canonical representation built from
// validated, parsed fields — never the raw JSON body, so formatting
// differences never change the hash while a genuinely different request
// always does. No personal data beyond what the caller already sent is
// added, and neither hash is ever logged.
func hashQuoteIdempotencyKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

func hashQuoteRequest(cartID, name, phone, email, eventDate, eventType string) []byte {
	canonical := "POST /solicitar-cotizacion\n" + cartID + "\n" + name + "\n" + phone + "\n" + email + "\n" + eventDate + "\n" + eventType
	sum := sha256.Sum256([]byte(canonical))
	return sum[:]
}

// validateQuoteCart reloads the cart strictly from PostgreSQL by the
// signed-cookie cart ID (never a client-supplied identifier) and confirms
// it is safe to attach to a new quote: not empty, every item's product
// still available, every item's quantity still within current stock. Go —
// not the browser, not Next — is the sole source of truth for all of this
// (Fase 11 section 2/9).
//
// Note: cart_items.product_id has ON DELETE CASCADE onto products (see
// sql/migrations/20250726033344_add_carts_table.sql) — a deleted product's
// cart_items row is removed by PostgreSQL itself, not left orphaned. That
// makes "producto eliminado" indistinguishable, at this layer, from the
// customer simply having an empty/smaller cart; both surface as
// quoteErrCartEmpty (or a shorter valid item list) rather than a separate
// error code. Distinguishing them would require an audit trail the schema
// does not have — documented as a residual limitation, not silently
// papered over with an invented table.
func validateQuoteCart(cartID string, r *http.Request) (*db.Cart, quoteRequestErrorCode) {
	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		log.Printf("validateQuoteCart: failed to load cart: %v\n", err)
		return nil, quoteErrCatalogUnavailable
	}

	if len(cart.Items) == 0 {
		return cart, quoteErrCartEmpty
	}
	for _, item := range cart.Items {
		if !item.Available {
			return cart, quoteErrProductUnavailable
		}
		if item.Quantity <= 0 || item.Quantity > item.MaxQty {
			return cart, quoteErrInvalidQuantity
		}
	}
	return cart, ""
}

func writeQuoteRequestJSONError(w http.ResponseWriter, status int, code quoteRequestErrorCode) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": string(code)})
}

func quoteRequestErrorStatus(code quoteRequestErrorCode) int {
	switch code {
	case quoteErrRequestTooLarge:
		return http.StatusRequestEntityTooLarge
	case quoteErrUnsupportedMediaType:
		return http.StatusUnsupportedMediaType
	case quoteErrCartEmpty, quoteErrProductRemoved, quoteErrProductUnavailable, quoteErrInvalidQuantity, quoteErrInvalidRequest:
		return http.StatusUnprocessableEntity
	case quoteErrCatalogUnavailable, quoteErrDBError:
		return http.StatusServiceUnavailable
	case quoteErrIdempotencyKeyRequired, quoteErrInvalidIdempotencyKey:
		return http.StatusBadRequest
	case quoteErrIdempotencyConflict:
		return http.StatusConflict
	default:
		return http.StatusBadRequest
	}
}

// handleQuoteRequestJSON is the contract the Next frontend's Server Action
// actually uses: server-to-server, same-origin design, JSON in/out, never
// touched by a browser directly. It never accepts cart_id, products,
// quantities, or totals from the request body — those all come from
// validateQuoteCart reloading PostgreSQL by the signed cart cookie.
func handleQuoteRequestJSON(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxQuoteRequestJSONBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body quoteRequestJSONBody
	if err := decoder.Decode(&body); err != nil {
		if err.Error() == "http: request body too large" {
			writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrRequestTooLarge), quoteErrRequestTooLarge)
			return
		}
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrInvalidRequest), quoteErrInvalidRequest)
		return
	}

	name := strings.TrimSpace(body.Name)
	phone := strings.TrimSpace(body.Phone)
	email := strings.TrimSpace(body.Email)
	eventDate := strings.TrimSpace(body.EventDate)
	eventType := strings.TrimSpace(body.EventType)

	if len(name) < 2 || len(phone) < 10 {
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrInvalidRequest), quoteErrInvalidRequest)
		return
	}
	if email != "" && !quoteRequestEmailPattern.MatchString(email) {
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrInvalidRequest), quoteErrInvalidRequest)
		return
	}

	var eventTime time.Time
	if eventDate != "" {
		var err error
		eventTime, err = time.Parse("2006-01-02", eventDate)
		if err != nil || eventTime.Before(time.Now()) {
			writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrInvalidRequest), quoteErrInvalidRequest)
			return
		}
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrIdempotencyKeyRequired), quoteErrIdempotencyKeyRequired)
		return
	}
	if !quoteIdempotencyKeyPattern.MatchString(idempotencyKey) {
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrInvalidIdempotencyKey), quoteErrInvalidIdempotencyKey)
		return
	}

	cartID, err := cartIDFromRequestContext(r)
	if err != nil {
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrCatalogUnavailable), quoteErrCatalogUnavailable)
		return
	}

	cart, cartErrCode := validateQuoteCart(cartID, r)
	if cartErrCode != "" {
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(cartErrCode), cartErrCode)
		return
	}

	quote := &db.Quote{
		CustomerName:  name,
		CustomerPhone: phone,
		RequestType:   db.QuoteRequestTypeBudget,
		Status:        db.QuoteStatusPending,
	}
	if !eventTime.IsZero() {
		quote.TimeStart = &eventTime
	}
	if cart.ID != "" {
		quote.CartID = sql.NullString{String: cart.ID, Valid: true}
	}
	if eventType != "" && eventType != "otro" {
		quote.EventKindID = sql.NullString{String: eventType, Valid: true}
	}

	keyHash := hashQuoteIdempotencyKey(idempotencyKey)
	requestHash := hashQuoteRequest(cartID, name, phone, email, eventDate, eventType)

	outcome, err := db.SubmitQuoteIdempotent(r.Context(), cartID, keyHash, requestHash, quote, time.Now().UTC())
	if errors.Is(err, db.ErrQuoteIdempotencyConflict) {
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrIdempotencyConflict), quoteErrIdempotencyConflict)
		return
	}
	if err != nil {
		log.Printf("handleQuoteRequestJSON: failed to submit quote: %v\n", err)
		writeQuoteRequestJSONError(w, quoteRequestErrorStatus(quoteErrDBError), quoteErrDBError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":   "success",
		"replayed": outcome == db.QuoteSubmitReplayed,
	})

	if outcome == db.QuoteSubmitApplied {
		go sendQuoteRequestConfirmation(quote)
	}
}

func sendQuoteRequestConfirmation(quote *db.Quote) {
	formatted, err := internal.FormatPhone(quote.CustomerPhone)
	if err != nil {
		log.Printf("failed to format phone number: %v\n", err)
		return
	}
	quote.CustomerPhone = formatted
	if err := whatsapp.SendQuoteRequestConfirmation(quote); err != nil {
		log.Printf("failed to send quote request confirmation: %v\n", err)
	}
}

func quoteRequestCartErrorMessage(code quoteRequestErrorCode) string {
	switch code {
	case quoteErrCartEmpty:
		return "Tu selección está vacía. Agrega productos antes de solicitar cotización."
	case quoteErrProductRemoved:
		return "Uno de los productos de tu selección ya no está disponible en el catálogo."
	case quoteErrProductUnavailable:
		return "Uno de los productos de tu selección ya no está disponible."
	case quoteErrInvalidQuantity:
		return "La cantidad de un producto en tu selección ya no es válida."
	default:
		return "No se pudo verificar tu selección. Intenta de nuevo."
	}
}

// wantsQuoteRequestJSON decides content negotiation for POST
// /solicitar-cotizacion: only a request that both declares a JSON body and
// asks for a JSON response gets the JSON contract — this is exactly what
// Next's server-to-server fetch does and a real browser form submission
// (with or without JavaScript) never does, so the existing HTMX/PRG form
// path below is untouched for every real end-user submission.
func wantsQuoteRequestJSON(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return false
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func RegisterContactRequestsRoutes(router *customServeMux, cartSessions *session.CartManager, csrfGuard *appsecurity.CSRFGuard) {
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
	router.HandleFunc("GET /solicitar-cotizacion", withCartSession(cartSessions, RenderQuoteRequest))
	router.HandleFunc("POST /solicitar-cotizacion", withProtectedCartSession(csrfGuard, cartSessions, HandleQuoteRequestSubmission))
	router.HandleFunc("POST /solicitar-contacto", HandleContactFormSubmission)

	// Cart management endpoints for quote form
	router.HandleFunc("PUT /cotizacion/carrito/items/{id}", withProtectedCartSession(csrfGuard, cartSessions, UpdateQuoteCartItem))
	router.HandleFunc("DELETE /cotizacion/carrito/items/{id}", withProtectedCartSession(csrfGuard, cartSessions, RemoveFromQuoteCart))
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
	cartID, err := cartIDFromRequestContext(r)
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to get cart session: %v\n", err)
		return
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
	if wantsQuoteRequestJSON(r) {
		handleQuoteRequestJSON(w, r)
		return
	}

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
	cartID, err := cartIDFromRequestContext(r)
	if err != nil {
		http.Error(w, "Ocurrió un error inesperado", http.StatusInternalServerError)
		log.Printf("failed to get cart session: %v\n", err)
		return
	}
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
		eventTime, err = time.Parse("2006-01-02", eventDate)
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

	// Reload the cart strictly from PostgreSQL by the signed cart cookie
	// and confirm it is still safe to attach to a quote — never trust the
	// cart.Items already loaded above for form display, which can be
	// stale by the time this handler runs.
	freshCart, cartErrCode := validateQuoteCart(cartID, r)
	if cartErrCode != "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		toastData.Type = components.ToastError
		toastData.Message = "Error al enviar la solicitud"
		formState.CartError = quoteRequestCartErrorMessage(cartErrCode)
		formState.Cart = freshCart.Items
		combined := templ.Join(
			components.ToasterToast(toastData),
			pages.QuoteRequest(formState),
		)
		templ.RenderFragments(r.Context(), w, combined, "toaster-toast", "quoteRequestForm")
		log.Printf("quote cart validation failed: %s\n", cartErrCode)
		return
	}
	cart = freshCart

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

	// A real browser form submission without JavaScript never sends
	// HX-Request — for that case, follow Post/Redirect/Get with a 303 so a
	// page refresh never resubmits the quote, matching the cart's own
	// no-JS fallback pattern (routes registered elsewhere already redirect
	// this way). An HTMX request keeps its existing fragment-swap
	// response unchanged.
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, "/solicitar-cotizacion?enviado=1", http.StatusSeeOther)
		go sendQuoteRequestConfirmation(quote)
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

	go sendQuoteRequestConfirmation(quote)
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
	cartID, err := cartIDFromRequestContext(r)
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
	cartID, err := cartIDFromRequestContext(r)
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
