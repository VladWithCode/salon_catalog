package forms

import (
	"fmt"
	"strings"
)

const (
	MinCategoryNameLength            = 3
	MaxCategoryNameLength            = 120
	MaxCategoryDescriptionLength     = 128
	MaxCategoryLongDescriptionLength = 360
)

type CategoryFormState struct {
	fields map[string]FieldState

	successMessage string
	generalError   string

	isValid      bool
	hasErrors    bool
	hasWarnings  bool
	hasSucceeded bool
}

func NewCategoryFormState(action string) *CategoryFormState {
	return &CategoryFormState{
		fields: map[string]FieldState{
			"name": {
				FieldType:  FieldTypeText,
				IsRequired: true,
			},
			"description": {
				FieldType:  FieldTypeText,
				IsRequired: true,
			},
			"long_description": {
				FieldType: FieldTypeText,
			},
			"header_img": {
				FieldType: FieldTypeFile,
			},
			"display_img": {
				FieldType: FieldTypeFile,
			},
		},
	}
}

func (fs *CategoryFormState) GetFieldValue(field string) string {
	return fs.fields[field].Value
}

func (fs *CategoryFormState) GetFieldState(field string) FieldState {
	state, exists := fs.fields[field]
	if !exists {
		return FieldState{}
	}

	return state
}

func (fs *CategoryFormState) SetFieldState(field string, state FieldState) {
	fs.fields[field] = state
}

func (fs *CategoryFormState) SetFieldError(field, message string) {
	state, exists := fs.fields[field]
	if !exists {
		return
	}

	state.IsTouched = true
	state.IsValid = false
	state.HasError = true
	state.ErrorMessage = message
	state.ValidationClass = "error"

	fs.fields[field] = state
}

func (fs *CategoryFormState) ClearFieldError(field string) {
	state, exists := fs.fields[field]
	if !exists {
		return
	}

	state.HasError = false
	state.ErrorMessage = ""
	state.ValidationClass = ""
	fs.fields[field] = state
}

func (fs *CategoryFormState) HasFieldError(field string) bool {
	return fs.fields[field].HasError
}

func (fs *CategoryFormState) SetFieldWarning(field, message string) {
	state, exists := fs.fields[field]
	if !exists {
		return
	}

	state.HasWarning = true
	state.WarningText = message
	state.ValidationClass = "warning"
	fs.fields[field] = state
}

func (fs *CategoryFormState) HasFieldWarning(field string) bool {
	return fs.fields[field].HasWarning
}

func (fs *CategoryFormState) GetFieldWarning(field string) string {
	return fs.fields[field].WarningText
}

func (fs *CategoryFormState) GetFieldHelpText(field string) string {
	return fs.fields[field].HelpText
}

func (fs *CategoryFormState) GetFieldError(field string) string {
	return fs.fields[field].ErrorMessage
}

func (fs *CategoryFormState) GetFieldClass(field string) string {
	state, exists := fs.fields[field]
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

func (fs *CategoryFormState) IsValid() bool {
	return !fs.HasErrors()
}

func (fs *CategoryFormState) IsSuccessful() bool {
	return fs.hasSucceeded
}

func (fs *CategoryFormState) GetSuccessMessage() string {
	return fs.successMessage
}

func (fs *CategoryFormState) SetSuccessMessage(message string) {
	fs.successMessage = message
	fs.hasSucceeded = true
}

func (fs *CategoryFormState) HasErrors() bool {
	for _, state := range fs.fields {
		if state.HasError {
			return true
		}
	}
	return false
}

func (fs *CategoryFormState) GetErrorMessage() string {
	return fs.generalError
}

func (fs *CategoryFormState) SetErrorMessage(message string) {
	fs.generalError = message
	fs.hasErrors = true
}

func (fs *CategoryFormState) ClearErrors() {
	fs.generalError = ""
	fs.hasErrors = false

	// Clear field error states
	for field := range fs.fields {
		state := fs.fields[field]
		state.HasError = false
		state.ErrorMessage = ""
		if state.ValidationClass == "error" {
			state.ValidationClass = ""
		}

		fs.fields[field] = state
	}
}

func (fs *CategoryFormState) ResetFieldState(fields ...string) {

	if len(fields) > 0 && len(fields[0]) > 0 {
		for _, field := range fields {
			fs.fields[field] = FieldState{
				Value:           fs.GetFieldValue(field),
				IsTouched:       false,
				IsValid:         false,
				HasError:        false,
				ErrorMessage:    "",
				HasWarning:      false,
				WarningText:     "",
				HelpText:        "",
				ValidationClass: "",
			}
		}
	} else {
		// Reset all fields
		for field := range fs.fields {
			fs.fields[field] = FieldState{
				Value:           fs.GetFieldValue(field),
				IsTouched:       false,
				IsValid:         false,
				HasError:        false,
				ErrorMessage:    "",
				HasWarning:      false,
				WarningText:     "",
				HelpText:        "",
				ValidationClass: "",
			}
		}
	}

}

func (fs *CategoryFormState) Validate() error {
	hasErrors := false

	name := fs.GetFieldValue("name")
	if l := len(strings.TrimSpace(name)); l == 0 {
		fs.SetFieldError("name", "El nombre es requerido")
		hasErrors = true
	} else if l < MinCategoryNameLength {
		fs.SetFieldError("name", fmt.Sprintf("El nombre debe tener al menos %d caracteres", MinCategoryNameLength))
		hasErrors = true
	} else if l > MaxCategoryNameLength {
		fs.SetFieldError("name", fmt.Sprintf("El nombre no puede exceder %d caracteres", MaxCategoryNameLength))
		hasErrors = true
	} else {
		fs.SetFieldValid("name")
	}

	desc := fs.GetFieldValue("description")
	if l := len(strings.TrimSpace(desc)); l == 0 {
		fs.SetFieldError("description", "La descripción es requerida")
		hasErrors = true
	} else if l > MaxCategoryDescriptionLength {
		fs.SetFieldError("description", fmt.Sprintf("La descripción no puede exceder %d caracteres", MaxCategoryDescriptionLength))
		hasErrors = true
	} else {
		fs.SetFieldValid("description")
	}

	if len(fs.GetFieldValue("long_description")) > MaxCategoryLongDescriptionLength {
		fs.SetFieldError("long_description", fmt.Sprintf("La descripción larga no puede exceder %d caracteres", MaxCategoryLongDescriptionLength))
		hasErrors = true
	} else {
		fs.SetFieldValid("long_description")
	}

	if hasErrors {
		return ErrValidationFailed
	}

	return nil
}

// Helper methods

// SetFieldValid sets the valid state for a specific field
// and updates the field state accordingly
func (fs *CategoryFormState) SetFieldValid(field string) {
	fs.ClearFieldError(field)
	state := fs.GetFieldState(field)
	state.IsTouched = true
	state.IsValid = true
	state.HasError = false
	state.ValidationClass = "valid"
	fs.fields[field] = state
}

func (fs *CategoryFormState) isFieldRequired(field string) bool {
	return fs.fields[field].IsRequired
}
