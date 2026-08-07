package forms

import (
	"fmt"
	"strings"
	"time"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

const MaxSocialNameLength = 60
const MaxSocialLinkLength = 512

type SocialFormState struct {
	Values         SocialFormValues      `json:"values"`
	Errors         SocialFormErrors      `json:"errors"`
	Fields         map[string]FieldState `json:"fields"`
	IsSubmitted    bool                  `json:"is_submitted"`
	SuccessMessage string                `json:"success_message,omitempty"`
	GeneralError   string                `json:"general_error,omitempty"`
	IsLoading      bool                  `json:"is_loading"`
	SubmitAttempts int                   `json:"submit_attempts"`
	LastUpdated    time.Time             `json:"last_updated"`
	FormMode       string                `json:"form_mode"` // "create", "edit", "view"
}

type SocialFormValues struct {
	Name      string `json:"name"`
	Link      string `json:"link"`
	IconID    string `json:"icon_id"`
	SectionID string `json:"section_id"`
}

type SocialFormErrors struct {
	Name      string `json:"name,omitempty"`
	Link      string `json:"link,omitempty"`
	IconID    string `json:"icon_id,omitempty"`
	SectionID string `json:"section_id,omitempty"`
}

func NewSocialFormState(mode string) *SocialFormState {
	return &SocialFormState{
		Values:      SocialFormValues{},
		Errors:      SocialFormErrors{},
		Fields:      make(map[string]FieldState),
		FormMode:    mode,
		LastUpdated: time.Now(),
	}
}

func NewSocialFormStateFromMap(mode string, values map[string]string) *SocialFormState {
	return &SocialFormState{
		Values: SocialFormValues{
			Name:      values["name"],
			Link:      values["link"],
			IconID:    values["icon_id"],
			SectionID: values["section_id"],
		},
		Errors:      SocialFormErrors{},
		Fields:      make(map[string]FieldState),
		FormMode:    mode,
		LastUpdated: time.Now(),
	}
}

func NewSocialFormStateFromSocialLink(mode string, link *db.SocialLink) *SocialFormState {
	return &SocialFormState{
		Values: SocialFormValues{
			Name: link.Name,
			Link: link.Link,
		},
		Errors:      SocialFormErrors{},
		Fields:      make(map[string]FieldState),
		FormMode:    mode,
		LastUpdated: time.Now(),
	}
}

// FormState interface implementation
func (sfs *SocialFormState) GetFormState() *SocialFormState {
	return sfs
}

func (sfs *SocialFormState) GetFieldValue(field string) string {
	switch field {
	case "name":
		return sfs.Values.Name
	case "link":
		return sfs.Values.Link
	case "icon_id":
		return sfs.Values.IconID
	case "section_id":
		return sfs.Values.SectionID
	default:
		return ""
	}
}

func (sfs *SocialFormState) GetFieldState(field string) FieldState {
	state, exists := sfs.Fields[field]
	if !exists {
		return FieldState{
			Value:           sfs.GetFieldValue(field),
			IsTouched:       false,
			IsValid:         false,
			HasError:        false,
			ErrorMessage:    "",
			HasWarning:      false,
			WarningText:     "",
			HelpText:        "",
			ValidationClass: "border-gray-300 focus:ring-accent focus:border-transparent",
			IsRequired:      sfs.isFieldRequired(field),
		}
	}

	state.Value = sfs.GetFieldValue(field)
	state.ErrorMessage = sfs.GetFieldError(field)
	state.ValidationClass = sfs.GetFieldClass(field)
	state.IsRequired = sfs.isFieldRequired(field)

	return state
}

func (sfs *SocialFormState) SetFieldState(field string, state FieldState) {
	sfs.Fields[field] = state
}

func (sfs *SocialFormState) GetFieldError(field string) string {
	switch field {
	case "name":
		return sfs.Errors.Name
	case "link":
		return sfs.Errors.Link
	case "icon_id":
		return sfs.Errors.IconID
	case "section_id":
		return sfs.Errors.SectionID
	default:
		return ""
	}
}

func (sfs *SocialFormState) SetFieldError(field, message string) {
	switch field {
	case "name":
		sfs.Errors.Name = message
	case "link":
		sfs.Errors.Link = message
	case "icon_id":
		sfs.Errors.IconID = message
	case "section_id":
		sfs.Errors.SectionID = message
	}

	state := sfs.GetFieldState(field)
	state.IsTouched = true
	state.IsValid = false
	state.HasError = true
	state.ErrorMessage = message
	state.ValidationClass = "error"

	sfs.Fields[field] = state
}

func (sfs *SocialFormState) ClearFieldError(field string) {
	switch field {
	case "name":
		sfs.Errors.Name = ""
	case "link":
		sfs.Errors.Link = ""
	case "icon_id":
		sfs.Errors.IconID = ""
	case "section_id":
		sfs.Errors.SectionID = ""
	}

	state := sfs.GetFieldState(field)
	state.HasError = false
	state.ErrorMessage = ""
	state.ValidationClass = ""
	sfs.Fields[field] = state
}

func (sfs *SocialFormState) HasFieldError(field string) bool {
	return sfs.GetFieldError(field) != ""
}

func (sfs *SocialFormState) SetFieldWarning(field, message string) {
	state := sfs.GetFieldState(field)
	state.HasWarning = true
	state.WarningText = message
	state.ValidationClass = "warning"
	sfs.Fields[field] = state
}

func (sfs *SocialFormState) HasFieldWarning(field string) bool {
	state := sfs.GetFieldState(field)
	return state.HasWarning
}

func (sfs *SocialFormState) GetFieldWarning(field string) string {
	fieldState := sfs.GetFieldState(field)
	return fieldState.WarningText
}

func (sfs *SocialFormState) GetFieldClass(field string) string {
	state, exists := sfs.Fields[field]
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

func (sfs *SocialFormState) IsValid() bool {
	return !sfs.HasErrors()
}

func (sfs *SocialFormState) IsSuccessful() bool {
	return sfs.IsSubmitted && sfs.IsValid() && !sfs.HasErrors() && sfs.SuccessMessage != ""
}

func (sfs *SocialFormState) GetSuccessMessage() string {
	return sfs.SuccessMessage
}

func (sfs *SocialFormState) SetSuccessMessage(message string) {
	sfs.SuccessMessage = message
}

func (sfs *SocialFormState) HasErrors() bool {
	for _, state := range sfs.Fields {
		if state.HasError {
			return true
		}
	}
	return false
}

func (sfs *SocialFormState) GetErrorMessage() string {
	return sfs.GeneralError
}

func (sfs *SocialFormState) SetErrorMessage(message string) {
	sfs.GeneralError = message
}

func (sfs *SocialFormState) ClearErrors() {
	sfs.Errors = SocialFormErrors{}
	sfs.GeneralError = ""

	for field := range sfs.Fields {
		state := sfs.Fields[field]
		state.HasError = false
		state.ErrorMessage = ""
		if state.ValidationClass == "error" {
			state.ValidationClass = ""
		}
		sfs.Fields[field] = state
	}
}

func (sfs *SocialFormState) ResetFieldState(fields ...string) {
	if len(fields) > 0 && len(fields[0]) > 0 {
		for _, field := range fields {
			sfs.Fields[field] = FieldState{
				Value:           sfs.GetFieldValue(field),
				IsTouched:       false,
				IsValid:         false,
				HasError:        false,
				ErrorMessage:    "",
				HasWarning:      false,
				WarningText:     "",
				HelpText:        "",
				ValidationClass: "",
				IsRequired:      sfs.isFieldRequired(field),
			}
		}
	} else {
		for field := range sfs.Fields {
			sfs.Fields[field] = FieldState{
				Value:           sfs.GetFieldValue(field),
				IsTouched:       false,
				IsValid:         false,
				HasError:        false,
				ErrorMessage:    "",
				HasWarning:      false,
				WarningText:     "",
				HelpText:        "",
				ValidationClass: "",
				IsRequired:      sfs.isFieldRequired(field),
			}
		}
	}
}

func (sfs *SocialFormState) Validate() error {
	hasErrors := false

	if strings.TrimSpace(sfs.Values.Name) == "" {
		sfs.SetFieldError("name", "El nombre es requerido")
		hasErrors = true
	} else if len(sfs.Values.Name) > MaxSocialNameLength {
		sfs.SetFieldError("name", fmt.Sprintf("El nombre no puede exceder %d caracteres", MaxSocialNameLength))
		hasErrors = true
	} else {
		sfs.SetFieldValid("name")
	}

	if strings.TrimSpace(sfs.Values.Link) == "" {
		sfs.SetFieldError("link", "El enlace es requerido")
		hasErrors = true
	} else if len(sfs.Values.Link) > MaxSocialLinkLength {
		sfs.SetFieldError("link", fmt.Sprintf("El enlace no puede exceder %d caracteres", MaxSocialLinkLength))
		hasErrors = true
	} else {
		sfs.SetFieldValid("link")
	}

	if hasErrors {
		return ErrValidationFailed
	}

	return nil
}

// Helper methods
func (sfs *SocialFormState) SetFieldValid(field string) {
	sfs.ClearFieldError(field)
	state := sfs.GetFieldState(field)
	state.IsTouched = true
	state.IsValid = true
	state.HasError = false
	state.ValidationClass = "valid"
	sfs.Fields[field] = state
}

func (sfs *SocialFormState) isFieldRequired(field string) bool {
	switch field {
	case "name", "link":
		return true
	default:
		return false
	}
}
