package forms

import "time"

type ProductFormState struct {
	// Form data preservation and error handling
	Values ProductFormValues `json:"values"`
	Errors ProductFormErrors `json:"errors"`

	// Field states for UI feedback
	Fields map[string]FieldState `json:"fields"`

	// Global form state
	IsSubmitted    bool   `json:"is_submitted"`
	IsValid        bool   `json:"is_valid"`
	HasErrors      bool   `json:"has_errors"`
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

type FieldState struct {
	IsTouched       bool   `json:"is_touched"`
	IsValid         bool   `json:"is_valid"`
	HasError        bool   `json:"has_error"`
	HasWarning      bool   `json:"has_warning"`
	IsRequired      bool   `json:"is_required"`
	ValidationClass string `json:"validation_class"`
	HelpText        string `json:"help_text,omitempty"`
	WarningText     string `json:"warning_text,omitempty"`
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

// SetFieldError sets the error message for a specific field
// and updates the field state accordingly
func (pfs *ProductFormState) SetFieldError(field, message string) {
	// Set error using reflection or switch
	switch field {
	case "name":
		pfs.Errors.Name = message
	case "description":
		pfs.Errors.Description = message
	case "category_id":
		pfs.Errors.CategoryID = message
	case "main_img":
		pfs.Errors.MainImg = message
	case "gallery":
		pfs.Errors.Gallery = message
	}

	pfs.Fields[field] = FieldState{
		IsTouched:       true,
		IsValid:         false,
		HasError:        true,
		ValidationClass: "error",
	}
	pfs.HasErrors = true
}

// SetFieldValid sets the valid state for a specific field
// and updates the field state accordingly
func (pfs *ProductFormState) SetFieldValid(field string) {
	pfs.ClearFieldError(field)
	pfs.Fields[field] = FieldState{
		IsTouched:       true,
		IsValid:         true,
		HasError:        false,
		ValidationClass: "valid",
	}
}

func (pfs *ProductFormState) SetFieldWarning(field, message string) {
	state := pfs.Fields[field]
	state.HasWarning = true
	state.WarningText = message
	state.ValidationClass = "warning"
	pfs.Fields[field] = state
}

func (pfs *ProductFormState) ClearFieldError(field string) {
	switch field {
	case "name":
		pfs.Errors.Name = ""
	case "description":
		pfs.Errors.Description = ""
	case "category_id":
		pfs.Errors.CategoryID = ""
	case "main_img":
		pfs.Errors.MainImg = ""
	case "gallery":
		pfs.Errors.Gallery = ""
	}

	state := pfs.Fields[field]
	state.HasError = false
	state.ValidationClass = ""
	pfs.Fields[field] = state
}

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

func (pfs *ProductFormState) GetFieldError(field string) string {
	switch field {
	case "name":
		return pfs.Errors.Name
	case "description":
		return pfs.Errors.Description
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

func (pfs *ProductFormState) HasFieldError(field string) bool {
	return pfs.GetFieldError(field) != ""
}
