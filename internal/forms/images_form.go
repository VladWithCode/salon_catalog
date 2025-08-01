package forms

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
)

const MinImageNameLength = 3
const MaxImageNameLength = 120

var (
	ErrNameRequired = errors.New("el nombre es requerido")
	ErrNameTooShort = fmt.Errorf("el nombre debe tener al menos %d caracteres", MinImageNameLength)
	ErrNameTooLong  = fmt.Errorf("el nombre no puede exceder %d caracteres", MaxImageNameLength)
	ErrFileRequired = errors.New("el archivo es requerido")
)

type ImagesFormState struct {
	fields map[string]FieldState

	generalError   string
	successMessage string

	// Form state
	isSubmitted bool
	isValid     bool
	hasErrors   bool
	hasWarnings bool
	hasSuccess  bool
	isLoading   bool
}

func NewImagesFormState() *ImagesFormState {
	return &ImagesFormState{
		fields:      make(map[string]FieldState),
		isSubmitted: false,
		isValid:     false,
		hasErrors:   false,
		hasWarnings: false,
		hasSuccess:  false,
		isLoading:   false,
	}
}

func NewImagesFormStateFromReq(r *http.Request) *ImagesFormState {
	fs := NewImagesFormState()
	for k, v := range r.Form {
		if !strings.HasPrefix(k, "img_name_") {
			continue
		}

		fs.fields[k] = FieldState{
			Value:     v[0],
			FieldType: FieldTypeText,
		}
	}
	for k, v := range r.MultipartForm.File {
		if !strings.HasPrefix(k, "img_file_") {
			continue
		}

		fs.fields[k] = FieldState{
			File:      v[0],
			FieldType: FieldTypeFile,
		}
	}

	return fs
}

func (fs *ImagesFormState) GetFieldValue(field string) string {
	fld, exists := fs.fields[field]
	if !exists {
		return ""
	}

	return fld.Value
}

func (fs *ImagesFormState) GetFieldState(field string) FieldState {
	fld, exists := fs.fields[field]
	if !exists {
		return FieldState{}
	}

	return fld
}

func (fs *ImagesFormState) SetFieldState(field string, state FieldState) {
	fs.fields[field] = state
}

func (fs *ImagesFormState) HasFieldError(field string) bool {
	fld, exists := fs.fields[field]
	if !exists {
		return false
	}
	return fld.HasError
}

func (fs *ImagesFormState) SetFieldError(field, message string) {
	fld, exists := fs.fields[field]
	if !exists {
		return
	}
	fld.HasError = true
	fld.ErrorMessage = message
	fld.ValidationClass = "error"

	fs.SetFieldState(field, fld)
}

func (fs *ImagesFormState) ClearFieldError(field string) {
	fld, exists := fs.fields[field]
	if !exists {
		return
	}
	fld.HasError = false
	fld.ErrorMessage = ""
	if fld.ValidationClass == "error" {
		fld.ValidationClass = ""
	}

	fs.SetFieldState(field, fld)
}

func (fs *ImagesFormState) HasFieldWarning(field string) bool {
	fld, exists := fs.fields[field]
	if !exists {
		return false
	}
	return fld.HasWarning
}

func (fs *ImagesFormState) GetFieldWarning(field string) string {
	fld, exists := fs.fields[field]
	if !exists {
		return ""
	}
	return fld.WarningText
}

func (fs *ImagesFormState) SetFieldWarning(field, message string) {
	fld, exists := fs.fields[field]
	if !exists {
		return
	}
	fld.HasWarning = true
	fld.WarningText = message
	fld.ValidationClass = "warning"

	fs.SetFieldState(field, fld)
}

func (fs *ImagesFormState) GetFieldError(field string) string {
	fld, exists := fs.fields[field]
	if !exists {
		return ""
	}
	return fld.ErrorMessage
}

func (fs *ImagesFormState) GetFieldClass(field string) string {
	fld, exists := fs.fields[field]
	if !exists {
		return "border-gray-300 focus:ring-accent focus:border-transparent"
	}

	baseClass := "transition-all duration-200"

	switch fld.ValidationClass {
	case "error":
		return baseClass + " border-red-500 focus:ring-red-500 focus:border-red-500 bg-red-50"
	case "valid":
		return baseClass + " border-green-500 focus:ring-green-500 focus:border-green-500 bg-green-50"
	case "warning":
		return baseClass + " border-yellow-500 focus:ring-yellow-500 focus:border-yellow-500 bg-yellow-50"
	default:
		if fld.IsTouched {
			return baseClass + " border-gray-400 focus:ring-accent focus:border-transparent"
		}
		return baseClass + " border-gray-300 focus:ring-accent focus:border-transparent hover:border-gray-400"
	}
}

func (fs *ImagesFormState) IsValid() bool {
	return !fs.HasErrors()
}

func (fs *ImagesFormState) IsSuccessful() bool {
	return fs.isSubmitted && fs.IsValid() && !fs.HasErrors() && fs.successMessage != ""
}

func (fs *ImagesFormState) GetSuccessMessage() string {
	return fs.successMessage
}

func (fs *ImagesFormState) SetSuccessMessage(message string) {
	fs.successMessage = message
}

func (fs *ImagesFormState) HasErrors() bool {
	for _, state := range fs.fields {
		if state.HasError {
			return true
		}
	}
	return false
}

func (fs *ImagesFormState) GetErrorMessage() string {
	return fs.generalError
}

func (fs *ImagesFormState) SetErrorMessage(message string) {
	fs.generalError = message
}

func (fs *ImagesFormState) ClearErrors() {
	fs.generalError = ""

	// Clear field error states
	for field := range fs.fields {
		state := fs.fields[field]
		state.HasError = false
		state.ErrorMessage = ""
		if state.ValidationClass == "error" {
			state.ValidationClass = ""
		}

		fs.SetFieldState(field, state)
	}
}

func (fs *ImagesFormState) ResetFieldState(fields ...[]string) {
	var fieldsToReset []string

	if len(fields) > 0 && len(fields[0]) > 0 {
		fieldsToReset = fields[0]
	} else {
		// Reset all fields
		for k := range fs.fields {
			fieldsToReset = append(fieldsToReset, k)
		}
	}

	for _, field := range fieldsToReset {
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

func (fs *ImagesFormState) Validate() error {
	hasErrors := false

	for k, fld := range fs.fields {
		switch fld.FieldType {
		case FieldTypeText:
			if err := ValidateImageName(fld.Value); err != nil {
				fs.SetFieldError(k, err.Error())
				hasErrors = true
			}
		case FieldTypeFile:
			if err := ValidateImageFile(fld.File); err != nil {
				fs.SetFieldError(k, err.Error())
				hasErrors = true
			}
		default:
			fs.SetFieldError(k, "Campo inválido")
			hasErrors = true
		}
	}

	if hasErrors {
		return ErrValidationFailed
	}

	return nil
}

func ValidateImageName(name string) error {
	if l := len(name); strings.TrimSpace(name) == "" {
		return ErrNameRequired
	} else if l < MinImageNameLength {
		return ErrNameTooShort
	} else if l > MaxImageNameLength {
		return ErrNameTooLong
	}
	return nil
}

func ValidateImageFile(file *multipart.FileHeader) error {
	if file == nil {
		return ErrFileRequired
	}
	return nil
}
