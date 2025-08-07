package forms

import (
	"fmt"
	"regexp"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

const MinUserNameLength = 3
const MaxUserNameLength = 255
const MinUserPasswordLength = 8
const MaxUserPasswordLength = 120

type UserFormState struct {
	fields map[string]FieldState

	generalError   string
	successMessage string

	isSubmitted bool
	hasErrors   bool
	hasWarnings bool
}

func NewUserFormState() *UserFormState {
	return &UserFormState{
		fields: map[string]FieldState{},
	}
}

func NewUserFormStateFromUser(user *db.User) *UserFormState {
	return &UserFormState{
		fields: map[string]FieldState{
			"fullname": {
				Value: user.Fullname,
			},
			"username": {
				Value: user.Username,
			},
			"email": {
				Value: user.Email,
			},
			"role": {
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

	baseClass := "transition-all duration-200 placeholder:text-gray-500"

	switch fld.ValidationClass {
	case "error":
		return baseClass + " border-red-500 focus:ring-red-500 focus:border-red-500 bg-red-50"
	case "valid":
		return baseClass + " border-green-500 focus:ring-green-500 focus:border-green-500 bg-green-50"
	case "warning":
		return baseClass + " border-yellow-500 focus:ring-yellow-500 focus:border-yellow-500 bg-yellow-50"
	default:
		return baseClass + " border-gray-200 focus:ring-accent focus:border-transparent bg-light"
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

func (fs *UserFormState) Validate() error {
	hasErrors := false

	if nameFld, ok := fs.fields["fullname"]; ok {
		exp := regexp.MustCompile("[a-zA-Z ]+$")
		if nameFld.Value == "" && len(nameFld.Value) == 0 {
			fs.SetFieldError("fullname", "El nombre es requerido")
			hasErrors = true
		} else if len(nameFld.Value) < MinUserNameLength {
			fs.SetFieldError("fullname", fmt.Sprintf("El nombre debe tener al menos %d caracteres", MinUserNameLength))
			hasErrors = true
		} else if len(nameFld.Value) > MaxUserNameLength {
			fs.SetFieldError("fullname", fmt.Sprintf("El nombre no puede exceder %d caracteres", MaxUserNameLength))
			hasErrors = true
		} else if exp.Match([]byte(nameFld.Value)) == false {
			fs.SetFieldError("fullname", "El nombre no es válido. Solo letras y espacios son permitidos")
			hasErrors = true
		}
	}

	if usernameFld, ok := fs.fields["username"]; ok {
		exp := regexp.MustCompile("[a-zA-Z0-9_]+$")
		if usernameFld.Value == "" && len(usernameFld.Value) == 0 {
			fs.SetFieldError("username", "El nombre es requerido")
			hasErrors = true
		} else if len(usernameFld.Value) < MinUserNameLength {
			fs.SetFieldError("username", fmt.Sprintf("El nombre debe tener al menos %d caracteres", MinUserNameLength))
			hasErrors = true
		} else if len(usernameFld.Value) > MaxUserNameLength {
			fs.SetFieldError("username", fmt.Sprintf("El nombre no puede exceder %d caracteres", MaxUserNameLength))
			hasErrors = true
		} else if exp.Match([]byte(usernameFld.Value)) == false {
			fs.SetFieldError("username", "El nombre no es válido. Solo caracteres alfanuméricos y guiones son permitidos")
			hasErrors = true
		}
	}

	if roleFld, ok := fs.fields["role"]; ok {
		if roleFld.Value == "" {
			fs.SetFieldError("role", "El rol es requerido")
			hasErrors = true
		} else if roleFld.Value != db.RoleAdmin && roleFld.Value != db.RoleEditor && roleFld.Value != db.RoleUser {
			fs.SetFieldError("role", "El rol no es válido")
			hasErrors = true
		}
	}

	if roleFld, ok := fs.fields["role"]; ok {
		if roleFld.Value == "" {
			fs.SetFieldError("role", "El rol es requerido")
			hasErrors = true
		}

		if roleFld.Value != db.RoleAdmin && roleFld.Value != db.RoleEditor && roleFld.Value != db.RoleUser {
			fs.SetFieldError("role", "El rol no es válido")
			hasErrors = true
		}
	}

	if passwordFld, ok := fs.fields["current_passwd"]; ok {
		if passwordFld.Value == "" {
			fs.SetFieldError("current_passwd", "Debes ingresar tu contraseña actual.")
			hasErrors = true
		}

		newPasswordFld, _ := fs.fields["new_passwd"]
		confirmPasswd, _ := fs.fields["confirm_passwd"]
		if newPasswordFld.Value == "" {
			fs.SetFieldError("new_passwd", "La nueva contraseña es requerida")
			hasErrors = true
		} else if len(newPasswordFld.Value) < MinUserPasswordLength {
			fs.SetFieldError("new_passwd", fmt.Sprintf("La nueva contraseña debe tener al menos %d caracteres", MinUserPasswordLength))
			hasErrors = true
		} else if len(newPasswordFld.Value) > MaxUserPasswordLength {
			fs.SetFieldError("new_passwd", fmt.Sprintf("La nueva contraseña no puede exceder %d caracteres", MaxUserPasswordLength))
			hasErrors = true
		}

		if confirmPasswd.Value == "" || confirmPasswd.Value != newPasswordFld.Value {
			fs.SetFieldError("confirm_passwd", "Las contraseñas no coinciden")
			hasErrors = true
		}
	}

	if hasErrors {
		return ErrValidationFailed
	}

	return nil
}

func (fs *UserFormState) SetIsSubmitted() {
	fs.isSubmitted = true
}

func (fs *UserFormState) IsSubmitted() bool {
	return fs.isSubmitted
}
