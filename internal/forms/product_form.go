package forms

import (
	"fmt"
	"strings"
	"time"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

const MinProductNameLength = 3
const MaxProductNameLength = 120
const MaxProductDescriptionLength = 128
const MaxProductLongDescriptionLength = 360

type ProductFormState struct {
	// Form data preservation and error handling
	Values ProductFormValues `json:"values"`
	Errors ProductFormErrors `json:"errors"`

	// Field states for UI feedback
	Fields map[string]FieldState `json:"fields"`

	// Global form state
	IsSubmitted    bool   `json:"is_submitted"`
	SuccessMessage string `json:"success_message,omitempty"`
	GeneralError   string `json:"general_error,omitempty"`

	// UI state
	IsLoading      bool `json:"is_loading"`
	SubmitAttempts int  `json:"submit_attempts"`

	// Metadata for enhanced UX
	LastUpdated time.Time `json:"last_updated"`
	FormMode    string    `json:"form_mode"` // "create", "edit", "view"
}

type ProductFormValues struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	LongDescription string   `json:"long_description"`
	CategoryID      string   `json:"category_id"`
	Available       bool     `json:"available"`
	MainImg         string   `json:"main_img"`
	Gallery         []string `json:"gallery"`
}

type ProductFormErrors struct {
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	LongDescription string `json:"long_description,omitempty"`
	CategoryID      string `json:"category_id,omitempty"`
	MainImg         string `json:"main_img,omitempty"`
	Gallery         string `json:"gallery,omitempty"`
}

// Helper methods

// NewProductFormState creates a new ProductFormState instance
func NewProductFormState(mode string) *ProductFormState {
	return &ProductFormState{
		Values: ProductFormValues{
			Available: true, // Default to available
			Gallery:   []string{},
		},
		Errors:      ProductFormErrors{},
		Fields:      make(map[string]FieldState),
		FormMode:    mode,
		LastUpdated: time.Now(),
	}
}

func NewProductFormStateFromMap(mode string, values map[string]string) *ProductFormState {
	newState := &ProductFormState{
		Values: ProductFormValues{
			Name:            values["name"],
			Description:     values["description"],
			LongDescription: values["long_description"],
			CategoryID:      values["category_id"],
			MainImg:         values["main_img"],
			Available:       values["available"] == "on",
		},
		Errors:      ProductFormErrors{},
		Fields:      make(map[string]FieldState),
		FormMode:    mode,
		LastUpdated: time.Now(),
	}

	if values["gallery"] != "" {
		newState.Values.Gallery = strings.Split(values["gallery"], ",")
	}

	return newState
}

func NewProductFormStateFromProduct(mode string, product *db.Product) *ProductFormState {
	return &ProductFormState{
		Values: ProductFormValues{
			Name:            product.Name,
			Description:     product.Description,
			LongDescription: product.LongDescription,
			CategoryID:      product.CategoryID,
			Available:       product.Available,
			Gallery:         product.Gallery,
		},
		Errors:      ProductFormErrors{},
		Fields:      make(map[string]FieldState),
		FormMode:    mode,
		LastUpdated: time.Now(),
	}
}

// FormState interface implementation

func (pfs *ProductFormState) GetFormState() *ProductFormState {
	return pfs
}

// GetFieldValue returns the current value of a field
func (pfs *ProductFormState) GetFieldValue(field string) string {
	switch field {
	case "name":
		return pfs.Values.Name
	case "description":
		return pfs.Values.Description
	case "long_description":
		return pfs.Values.LongDescription
	case "category_id":
		return pfs.Values.CategoryID
	case "main_img":
		return pfs.Values.MainImg
	case "available":
		if pfs.Values.Available {
			return "on"
		}
		return ""
	case "gallery":
		return strings.Join(pfs.Values.Gallery, ",")
	default:
		return ""
	}
}

// GetFieldState returns the FieldState for a specific field
func (pfs *ProductFormState) GetFieldState(field string) FieldState {
	state, exists := pfs.Fields[field]
	if !exists {
		// Return default state for fields that haven't been touched
		return FieldState{
			Value:           pfs.GetFieldValue(field),
			IsTouched:       false,
			IsValid:         false,
			HasError:        false,
			ErrorMessage:    "",
			HasWarning:      false,
			WarningText:     "",
			HelpText:        "",
			ValidationClass: "border-gray-300 focus:ring-accent focus:border-transparent",
			IsRequired:      pfs.isFieldRequired(field),
		}
	}

	// Update value in case it changed
	state.Value = pfs.GetFieldValue(field)
	state.ErrorMessage = pfs.GetFieldError(field)
	state.ValidationClass = pfs.GetFieldClass(field)
	state.IsRequired = pfs.isFieldRequired(field)

	return state
}

// SetFieldState sets the complete field state
func (pfs *ProductFormState) SetFieldState(field string, state FieldState) {
	pfs.Fields[field] = state
}

// SetFieldError sets the error message for a specific field
// and updates the field state accordingly
func (pfs *ProductFormState) SetFieldError(field, message string) {
	// Set error using reflection or switch
	switch field {
	case "name":
		pfs.Errors.Name = message
	case "description":
		pfs.Errors.Description = message
	case "long_description":
		pfs.Errors.LongDescription = message
	case "category_id":
		pfs.Errors.CategoryID = message
	case "main_img":
		pfs.Errors.MainImg = message
	case "gallery":
		pfs.Errors.Gallery = message
	}

	state := pfs.GetFieldState(field)
	state.IsTouched = true
	state.IsValid = false
	state.HasError = true
	state.ErrorMessage = message
	state.ValidationClass = "error"

	pfs.Fields[field] = state
}

// ClearFieldError clears the error for a specific field
func (pfs *ProductFormState) ClearFieldError(field string) {
	switch field {
	case "name":
		pfs.Errors.Name = ""
	case "description":
		pfs.Errors.Description = ""
	case "long_description":
		pfs.Errors.LongDescription = ""
	case "category_id":
		pfs.Errors.CategoryID = ""
	case "main_img":
		pfs.Errors.MainImg = ""
	case "gallery":
		pfs.Errors.Gallery = ""
	}

	state := pfs.GetFieldState(field)
	state.HasError = false
	state.ErrorMessage = ""
	state.ValidationClass = ""
	pfs.Fields[field] = state
}

// HasFieldError returns true if the field has an error
func (pfs *ProductFormState) HasFieldError(field string) bool {
	return pfs.GetFieldError(field) != ""
}

// SetFieldWarning sets a warning message for a field
func (pfs *ProductFormState) SetFieldWarning(field, message string) {
	state := pfs.GetFieldState(field)
	state.HasWarning = true
	state.WarningText = message
	state.ValidationClass = "warning"
	pfs.Fields[field] = state
}

// HasFieldWarning returns true if the field has a warning
func (pfs *ProductFormState) HasFieldWarning(field string) bool {
	state := pfs.GetFieldState(field)
	return state.HasWarning
}

// GetFieldWarning returns the warning message for a field
func (pfs *ProductFormState) GetFieldWarning(field string) string {
	fieldState := pfs.GetFieldState(field)
	return fieldState.WarningText
}

func (pfs *ProductFormState) GetFieldHelpText(field string) string {
	fieldState := pfs.GetFieldState(field)
	return fieldState.HelpText
}

// GetFieldError returns the error message for a field
func (pfs *ProductFormState) GetFieldError(field string) string {
	switch field {
	case "name":
		return pfs.Errors.Name
	case "description":
		return pfs.Errors.Description
	case "long_description":
		return pfs.Errors.LongDescription
	case "category_id":
		return pfs.Errors.CategoryID
	case "main_img":
		return pfs.Errors.MainImg
	case "gallery":
		return pfs.Errors.Gallery
	default:
		return ""
	}
}

// GetFieldClass returns the CSS class for a field based on its state
func (pfs *ProductFormState) GetFieldClass(field string) string {
	state, exists := pfs.Fields[field]
	if !exists {
		return "border-gray-300 focus:ring-accent focus:border-transparent"
	}

	baseClass := "transition-all duration-200"

	switch state.ValidationClass {
	case "error":
		return baseClass + " border-red-500 focus:ring-red-500 focus:border-red-500 bg-red-50"
	case "valid":
		return baseClass + " border-green-500 focus:ring-green-500 focus:border-green-500 bg-green-50"
	case "warning":
		return baseClass + " border-yellow-500 focus:ring-yellow-500 focus:border-yellow-500 bg-yellow-50"
	default:
		if state.IsTouched {
			return baseClass + " border-gray-400 focus:ring-accent focus:border-transparent"
		}
		return baseClass + " border-gray-300 focus:ring-accent focus:border-transparent hover:border-gray-400"
	}
}

// IsValid returns true if the form is valid (no errors)
func (pfs *ProductFormState) IsValid() bool {
	return !pfs.HasErrors()
}

// IsSuccessful returns true if the form was successfully submitted
func (pfs *ProductFormState) IsSuccessful() bool {
	return pfs.IsSubmitted && pfs.IsValid() && !pfs.HasErrors() && pfs.SuccessMessage != ""
}

// GetSuccessMessage returns the success message
func (pfs *ProductFormState) GetSuccessMessage() string {
	return pfs.SuccessMessage
}

// SetSuccessMessage sets the success message
func (pfs *ProductFormState) SetSuccessMessage(message string) {
	pfs.SuccessMessage = message
}

// HasErrors returns true if the form has any errors
func (pfs *ProductFormState) HasErrors() bool {
	for _, state := range pfs.Fields {
		if state.HasError {
			return true
		}
	}
	return false
}

// GetErrorMessage returns the general error message
func (pfs *ProductFormState) GetErrorMessage() string {
	return pfs.GeneralError
}

// SetErrorMessage sets the general error message
func (pfs *ProductFormState) SetErrorMessage(message string) {
	pfs.GeneralError = message
}

// ClearErrors clears all errors in the form
func (pfs *ProductFormState) ClearErrors() {
	pfs.Errors = ProductFormErrors{}
	pfs.GeneralError = ""

	// Clear field error states
	for field := range pfs.Fields {
		state := pfs.Fields[field]
		state.HasError = false
		state.ErrorMessage = ""
		if state.ValidationClass == "error" {
			state.ValidationClass = ""
		}
		pfs.Fields[field] = state
	}
}

// ResetFieldState resets the state of specified fields, or all fields if none specified
func (pfs *ProductFormState) ResetFieldState(fields ...string) {
	if len(fields) > 0 && len(fields[0]) > 0 {
		for _, field := range fields {
			pfs.Fields[field] = FieldState{
				Value:           pfs.GetFieldValue(field),
				IsTouched:       false,
				IsValid:         false,
				HasError:        false,
				ErrorMessage:    "",
				HasWarning:      false,
				WarningText:     "",
				HelpText:        "",
				ValidationClass: "",
				IsRequired:      pfs.isFieldRequired(field),
			}
		}
	} else {
		for field := range pfs.Fields {
			pfs.Fields[field] = FieldState{
				Value:           pfs.GetFieldValue(field),
				IsTouched:       false,
				IsValid:         false,
				HasError:        false,
				ErrorMessage:    "",
				HasWarning:      false,
				WarningText:     "",
				HelpText:        "",
				ValidationClass: "",
				IsRequired:      pfs.isFieldRequired(field),
			}
		}
	}

}

// Validate validates the form and returns an error if validation fails
func (pfs *ProductFormState) Validate() error {
	hasErrors := false

	if l := len(pfs.Values.Name); strings.TrimSpace(pfs.Values.Name) == "" {
		pfs.SetFieldError("name", "El nombre es requerido")
		hasErrors = true
	} else if l < MinProductNameLength {
		pfs.SetFieldError("name", fmt.Sprintf("El nombre debe tener al menos %d caracteres", MinProductNameLength))
		hasErrors = true
	} else if l > MaxProductNameLength {
		pfs.SetFieldError("name", fmt.Sprintf("El nombre no puede exceder %d caracteres", MaxProductNameLength))
		hasErrors = true
	} else {
		pfs.SetFieldValid("name")
	}

	if strings.TrimSpace(pfs.Values.Description) == "" {
		pfs.SetFieldError("description", "La descripción es requerida")
		hasErrors = true
	} else if len(pfs.Values.Description) > MaxProductDescriptionLength {
		pfs.SetFieldError("description", fmt.Sprintf("La descripción no puede exceder %d caracteres", MaxProductDescriptionLength))
		hasErrors = true
	} else {
		pfs.SetFieldValid("description")
	}

	if hasErrors {
		return ErrValidationFailed
	}

	return nil
}

// Helper methods

// SetFieldValid sets the valid state for a specific field
// and updates the field state accordingly
func (pfs *ProductFormState) SetFieldValid(field string) {
	pfs.ClearFieldError(field)
	state := pfs.GetFieldState(field)
	state.IsTouched = true
	state.IsValid = true
	state.HasError = false
	state.ValidationClass = "valid"
	pfs.Fields[field] = state
}

// isFieldRequired returns true if the field is required
func (pfs *ProductFormState) isFieldRequired(field string) bool {
	switch field {
	case "name", "description":
		return true
	default:
		return false
	}
}
