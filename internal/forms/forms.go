// Package forms provides form state and validation logic for the application
// various forms.
package forms

import (
	"errors"
	"os"
)

var (
	ErrValidationFailed = errors.New("validation failed")
)

type FieldType string

const (
	FieldTypeText FieldType = "text"
	FieldTypeFile FieldType = "file"
)

type FieldState struct {
	Value     string    `json:"value"`
	File      *os.File  `json:"file,omitempty"`
	FieldType FieldType `json:"field_type"`

	IsTouched       bool   `json:"is_touched"`
	IsValid         bool   `json:"is_valid"`
	HasError        bool   `json:"has_error"`
	ErrorMessage    string `json:"error_message,omitempty"`
	HasWarning      bool   `json:"has_warning"`
	WarningText     string `json:"warning_text,omitempty"`
	HelpText        string `json:"help_text,omitempty"`
	ValidationClass string `json:"validation_class"`
	IsRequired      bool   `json:"is_required"`
}

type FormState interface {
	// Fields
	GetFieldValue(field string) string
	GetFieldState(field string) FieldState
	SetFieldState(field string, state FieldState)
	HasFieldError(field string) bool
	SetFieldError(field, message string)
	ClearFieldError(field string)
	HasFieldWarning(field string) bool
	SetFieldWarning(field, message string)
	GetFieldError(field string) string
	GetFieldClass(field string) string

	// Form
	IsValid() bool
	IsSuccessful() bool
	GetSuccessMessage() string
	HasErrors() bool
	GetErrorMessage() string
	ClearErrors()
	ResetFieldState(fields ...[]string)
	Validate() error
}
