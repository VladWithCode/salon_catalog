package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/pages"
)

func RegisterQuoteRequestsRoutes(router *customServeMux) {
	router.HandleFunc("GET /solicitar-cotizacion", RenderQuoteRequest)
	router.HandleFunc("POST /api/solicitar-cotizacion", HandleQuoteRequestSubmission)
}

func RenderQuoteRequest(w http.ResponseWriter, r *http.Request) {
	err := pages.QuoteRequest(&pages.QuoteRequestFormState{}).Render(context.Background(), w)
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
	email := strings.TrimSpace(r.FormValue("email"))
	eventDate := strings.TrimSpace(r.FormValue("event_date"))
	eventType := strings.TrimSpace(r.FormValue("event_type"))
	comments := strings.TrimSpace(r.FormValue("comments"))
	cartData := strings.TrimSpace(r.FormValue("cart_data"))

	// Validate form data
	formState := &pages.QuoteRequestFormState{
		NameValue:      name,
		PhoneValue:     phone,
		EmailValue:     email,
		EventDateValue: eventDate,
		EventTypeValue: eventType,
		CommentsValue:  comments,
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

	if email == "" {
		formState.EmailError = "El correo electrónico es requerido"
		hasErrors = true
	} else if !isValidEmail(email) {
		formState.EmailError = "El correo electrónico no es válido"
		hasErrors = true
	}

	if eventDate == "" {
		formState.EventDateError = "La fecha del evento es requerida"
		hasErrors = true
	} else {
		// Parse and validate event date
		eventTime, err := time.Parse("2006-01-02T15:04", eventDate)
		if err != nil {
			formState.EventDateError = "La fecha del evento no es válida"
			hasErrors = true
		} else if eventTime.Before(time.Now()) {
			formState.EventDateError = "La fecha del evento debe ser en el futuro"
			hasErrors = true
		}
	}

	if eventType == "" {
		formState.EventTypeError = "El tipo de evento es requerido"
		hasErrors = true
	}

	if hasErrors {
		w.WriteHeader(http.StatusBadRequest)
		renderQuoteRequestError(w, formState)
		return
	}

	// Parse event date
	eventTime, err := time.Parse("2006-01-02T15:04", eventDate)
	if err != nil {
		formState.EventDateError = "La fecha del evento no es válida"
		w.WriteHeader(http.StatusBadRequest)
		renderQuoteRequestError(w, formState)
		return
	}

	// Create quote object
	quote := &db.Quote{
		CustomerName:  name,
		CustomerPhone: phone,
		TimeStart:     &eventTime,
		RequestType:   string(db.QuoteRequestTypeBudget),
		Status:        string(db.QuoteStatusPending),
		Comments:      buildQuoteComments(email, eventType, comments, cartData),
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
	err = components.ToasterToast(successToast).Render(context.Background(), w)
	if err != nil {
		log.Printf("failed to render success toast: %v\n", err)
	}

	// Render success form state
	successForm := &pages.QuoteRequestFormState{}
	err = pages.QuoteRequest(successForm).Render(context.Background(), w)
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

func isValidEmail(email string) bool {
	// Simple email validation
	return strings.Contains(email, "@") && strings.Contains(email, ".") && len(email) > 5
}
