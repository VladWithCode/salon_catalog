package forms

import "github.com/vladwithcode/salon_catalog/internal/db"

type UserFormState struct {
	fields map[string]FieldState

	generalError   string
	successMessage string

	hasErrors   bool
	hasWarnings bool
}

func NewUserFormState() *UserFormState {
	return &UserFormState{}
}

func NewUserFormStateFromUser(user *db.User) *UserFormState {
	return &UserFormState{
		fields: map[string]FieldState{
			"fullname": FieldState{
				Value: user.Fullname,
			},
			"username": FieldState{
				Value: user.Username,
			},
			"email": FieldState{
				Value: user.Email,
			},
			"role": FieldState{
				Value: user.Role,
			},
		},
	}
}

func (fs *UserFormState) GetFieldValue(field string) string {
	fld, exists := fs.fields[field]
	if !exists {
		return ""
	}

	return fld.Value
}

func (fs *UserFormState) GetFieldState(field string) FieldState {
	fld, exists := fs.fields[field]
	if !exists {
		return FieldState{}
	}

	return fld
}

func (fs *UserFormState) SetFieldState(field string, state FieldState) {
	fs.fields[field] = state
}

func (fs *UserFormState) HasFieldError(field string) bool {
	fld, exists := fs.fields[field]
	if !exists {
		return false
	}
	return fld.HasError
}

func (fs *UserFormState) SetFieldError(field, message string) {
	fld, exists := fs.fields[field]
	if !exists {
		return
	}
	fld.HasError = true
	fld.ErrorMessage = message
	fld.ValidationClass = "error"

	fs.SetFieldState(field, fld)
}

func (fs *UserFormState) ClearFieldError(field string) {
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

func (fs *UserFormState) HasFieldWarning(field string) bool {
	fld, exists := fs.fields[field]
	if !exists {
		return false
	}
	return fld.HasWarning
}

func (fs *UserFormState) GetFieldWarning(field string) string {
	fld, exists := fs.fields[field]
	if !exists {
		return ""
	}
	return fld.WarningText
}

func (fs *UserFormState) SetFieldWarning(field, message string) {
	fld, exists := fs.fields[field]
	if !exists {
		return
	}
	fld.HasWarning = true
	fld.WarningText = message
	fld.ValidationClass = "warning"

	fs.SetFieldState(field, fld)
}

func (fs *UserFormState) GetFieldError(field string) string {
	fld, exists := fs.fields[field]
	if !exists {
		return ""
	}
	return fld.ErrorMessage
}

func (fs *UserFormState) GetFieldClass(field string) string {
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
			return baseClass
		}
		return baseClass + " border-gray-300 focus:ring-accent focus:border-transparent hover:border-gray-400"
	}
}

func (fs *UserFormState) IsValid() bool {
	return fs.successMessage != ""
}

func (fs *UserFormState) IsSuccessful() bool {
	return fs.IsValid() && !fs.HasErrors()
}

func (fs *UserFormState) GetSuccessMessage() string {
	return fs.successMessage
}

func (fs *UserFormState) SetSuccessMessage(message string) {
	fs.successMessage = message
}

func (fs *UserFormState) HasErrors() bool {
	if fs.hasErrors {
		return true
	}
	for _, state := range fs.fields {
		if state.HasError {
			return true
		}
	}
	return false
}

func (fs *UserFormState) GetErrorMessage() string {
	return fs.generalError
}

func (fs *UserFormState) SetErrorMessage(message string) {
	fs.generalError = message
	fs.hasErrors = true
}

func (fs *UserFormState) ClearErrors() {
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

func (fs *UserFormState) ResetFieldState(fields ...string) {
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
