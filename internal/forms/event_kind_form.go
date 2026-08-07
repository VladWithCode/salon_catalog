package forms

import (
	"strings"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

type EventKindFormValues struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type EventKindFormState struct {
	Mode           string                `json:"mode"`
	Fields         map[string]FieldState `json:"fields"`
	Values         EventKindFormValues   `json:"values"`
	GeneralError   string                `json:"general_error"`
	SuccessMessage string                `json:"success_message"`
	EventKind      *db.EventKind         `json:"event_kind,omitempty"`
}

func NewEventKindFormState(mode string) *EventKindFormState {
	return &EventKindFormState{
		Mode:   mode,
		Fields: make(map[string]FieldState),
	}
}

func NewEventKindFormStateFromMap(mode string, data map[string]string) *EventKindFormState {
	state := NewEventKindFormState(mode)

	if name, ok := data["name"]; ok {
		state.Values.Name = name
	}
	if description, ok := data["description"]; ok {
		state.Values.Description = description
	}

	return state
}

func NewEventKindFormStateFromEventKind(mode string, eventKind *db.EventKind) *EventKindFormState {
	state := NewEventKindFormState(mode)
	state.EventKind = eventKind

	if eventKind != nil {
		state.Values.Name = eventKind.Name
		state.Values.Description = eventKind.Description
	}

	return state
}

func (s *EventKindFormState) GetFieldValue(field string) string {
	switch field {
	case "name":
		return s.Values.Name
	case "description":
		return s.Values.Description
	default:
		return ""
	}
}

func (s *EventKindFormState) GetFieldState(field string) FieldState {
	if state, ok := s.Fields[field]; ok {
		return state
	}
	return FieldState{}
}

func (s *EventKindFormState) SetFieldState(field string, state FieldState) {
	s.Fields[field] = state
}

func (s *EventKindFormState) HasFieldError(field string) bool {
	if state, ok := s.Fields[field]; ok {
		return state.HasError
	}
	return false
}

func (s *EventKindFormState) SetFieldError(field, message string) {
	state := s.GetFieldState(field)
	state.HasError = true
	state.ErrorMessage = message
	state.IsValid = false
	state.ValidationClass = "error"
	s.SetFieldState(field, state)
}

func (s *EventKindFormState) ClearFieldError(field string) {
	state := s.GetFieldState(field)
	state.HasError = false
	state.ErrorMessage = ""
	state.ValidationClass = ""
	s.SetFieldState(field, state)
}

func (s *EventKindFormState) HasFieldWarning(field string) bool {
	if state, ok := s.Fields[field]; ok {
		return state.HasWarning
	}
	return false
}

func (s *EventKindFormState) SetFieldWarning(field, message string) {
	state := s.GetFieldState(field)
	state.HasWarning = true
	state.WarningText = message
	state.ValidationClass = "warning"
	s.SetFieldState(field, state)
}

func (s *EventKindFormState) GetFieldWarning(field string) string {
	if state, ok := s.Fields[field]; ok {
		return state.WarningText
	}
	return ""
}

func (s *EventKindFormState) GetFieldError(field string) string {
	if state, ok := s.Fields[field]; ok {
		return state.ErrorMessage
	}
	return ""
}

func (s *EventKindFormState) GetFieldClass(field string) string {
	if state, ok := s.Fields[field]; ok {
		return state.ValidationClass
	}
	return ""
}

func (s *EventKindFormState) IsValid() bool {
	for _, state := range s.Fields {
		if state.HasError {
			return false
		}
	}
	return s.GeneralError == ""
}

func (s *EventKindFormState) IsSuccessful() bool {
	return s.SuccessMessage != ""
}

func (s *EventKindFormState) SetSuccessMessage(message string) {
	s.SuccessMessage = message
}

func (s *EventKindFormState) GetSuccessMessage() string {
	return s.SuccessMessage
}

func (s *EventKindFormState) HasErrors() bool {
	return s.GeneralError != "" || !s.IsValid()
}

func (s *EventKindFormState) SetErrorMessage(message string) {
	s.GeneralError = message
}

func (s *EventKindFormState) GetErrorMessage() string {
	return s.GeneralError
}

func (s *EventKindFormState) ClearErrors() {
	s.GeneralError = ""
	for field := range s.Fields {
		s.ClearFieldError(field)
	}
}

func (s *EventKindFormState) ResetFieldState(fields ...string) {
	if len(fields) == 0 {
		for field := range s.Fields {
			state := s.GetFieldState(field)
			state.HasError = false
			state.ErrorMessage = ""
			state.HasWarning = false
			state.WarningText = ""
			state.ValidationClass = ""
			s.SetFieldState(field, state)
		}
	} else {
		for _, field := range fields {
			state := s.GetFieldState(field)
			state.HasError = false
			state.ErrorMessage = ""
			state.HasWarning = false
			state.WarningText = ""
			state.ValidationClass = ""
			s.SetFieldState(field, state)
		}
	}
}

func (s *EventKindFormState) Validate() error {
	s.ClearErrors()

	// Validate name
	if strings.TrimSpace(s.Values.Name) == "" {
		s.SetFieldError("name", "El nombre es requerido")
	} else if len(s.Values.Name) > 256 {
		s.SetFieldError("name", "El nombre no puede exceder 256 caracteres")
	}

	// Validate description (optional but with length limit)
	if len(s.Values.Description) > 512 {
		s.SetFieldError("description", "La descripción no puede exceder 512 caracteres")
	}

	if !s.IsValid() {
		return ErrValidationFailed
	}

	return nil
}
