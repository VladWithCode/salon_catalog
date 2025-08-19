package forms

import (
	"strconv"
	"strings"

	"github.com/vladwithcode/salon_catalog/internal/db"
)

type WizardForm struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	EventKindID string            `json:"event_kind_id"`
	Enabled     bool              `json:"enabled"`
	StepIDs     []string          `json:"step_ids"`
	Errors      map[string]string `json:"errors"`
}

func NewWizardForm() *WizardForm {
	return &WizardForm{
		Errors: make(map[string]string),
	}
}

func (f *WizardForm) Validate() bool {
	f.Errors = make(map[string]string)

	// Validate name
	f.Name = strings.TrimSpace(f.Name)
	if f.Name == "" {
		f.Errors["name"] = "El nombre es requerido"
	} else if len(f.Name) > 200 {
		f.Errors["name"] = "El nombre no puede exceder 200 caracteres"
	}

	// Validate description
	f.Description = strings.TrimSpace(f.Description)
	if len(f.Description) > 512 {
		f.Errors["description"] = "La descripción no puede exceder 512 caracteres"
	}

	// Validate event kind ID
	f.EventKindID = strings.TrimSpace(f.EventKindID)
	if f.EventKindID == "" {
		f.Errors["event_kind_id"] = "El tipo de evento es requerido"
	}

	return len(f.Errors) == 0
}

func (f *WizardForm) ToWizard() *db.Wizard {
	return &db.Wizard{
		Name:        f.Name,
		Description: f.Description,
		EventKindID: f.EventKindID,
		Enabled:     f.Enabled,
	}
}

func (f *WizardForm) FromWizard(wizard *db.Wizard) {
	f.Name = wizard.Name
	f.Description = wizard.Description
	f.EventKindID = wizard.EventKindID
	f.Enabled = wizard.Enabled
}

type WizardStepForm struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Required    bool              `json:"required"`
	MultiSelect bool              `json:"multi_select"`
	MinSelected int               `json:"min_selected"`
	MaxSelected int               `json:"max_selected"`
	StepOrder   int               `json:"step_order"`
	CategoryIDs []string          `json:"category_ids"`
	Errors      map[string]string `json:"errors"`
}

func NewWizardStepForm() *WizardStepForm {
	return &WizardStepForm{
		Errors:      make(map[string]string),
		MaxSelected: 1,
		StepOrder:   1,
	}
}

func (f *WizardStepForm) Validate() bool {
	f.Errors = make(map[string]string)

	// Validate name
	f.Name = strings.TrimSpace(f.Name)
	if f.Name == "" {
		f.Errors["name"] = "El nombre es requerido"
	} else if len(f.Name) > 200 {
		f.Errors["name"] = "El nombre no puede exceder 200 caracteres"
	}

	// Validate description
	f.Description = strings.TrimSpace(f.Description)
	if len(f.Description) > 512 {
		f.Errors["description"] = "La descripción no puede exceder 512 caracteres"
	}

	// Validate selection limits
	if f.MinSelected < 0 {
		f.Errors["min_selected"] = "El mínimo de selecciones no puede ser negativo"
	}

	if f.MaxSelected < 1 {
		f.Errors["max_selected"] = "El máximo de selecciones debe ser al menos 1"
	}

	if f.MinSelected > f.MaxSelected {
		f.Errors["min_selected"] = "El mínimo no puede ser mayor que el máximo"
	}

	// Validate step order
	if f.StepOrder < 1 {
		f.Errors["step_order"] = "El orden del paso debe ser al menos 1"
	}

	return len(f.Errors) == 0
}

func (f *WizardStepForm) ToWizardStep() *db.WizardStep {
	return &db.WizardStep{
		Name:        f.Name,
		Description: f.Description,
		Required:    f.Required,
		MultiSelect: f.MultiSelect,
		MinSelected: f.MinSelected,
		MaxSelected: f.MaxSelected,
		StepOrder:   f.StepOrder,
		CategoryIDs: f.CategoryIDs,
	}
}

func (f *WizardStepForm) FromWizardStep(step *db.WizardStep) {
	f.Name = step.Name
	f.Description = step.Description
	f.Required = step.Required
	f.MultiSelect = step.MultiSelect
	f.MinSelected = step.MinSelected
	f.MaxSelected = step.MaxSelected
	f.StepOrder = step.StepOrder
	f.CategoryIDs = step.CategoryIDs
}

type WizardStepParams struct {
	Required    bool              `json:"required"`
	MultiSelect bool              `json:"multi_select"`
	MinSelected int               `json:"min_selected"`
	MaxSelected int               `json:"max_selected"`
	StepOrder   int               `json:"step_order"`
	Errors      map[string]string `json:"errors"`
}

func NewWizardStepParams() *WizardStepParams {
	return &WizardStepParams{
		Errors:      make(map[string]string),
		MaxSelected: 1,
		StepOrder:   1,
	}
}

func (p *WizardStepParams) Validate() bool {
	p.Errors = make(map[string]string)

	// Validate selection limits
	if p.MinSelected < 0 {
		p.Errors["min_selected"] = "El mínimo de selecciones no puede ser negativo"
	}

	if p.MaxSelected < 1 {
		p.Errors["max_selected"] = "El máximo de selecciones debe ser al menos 1"
	}

	if p.MinSelected > p.MaxSelected {
		p.Errors["min_selected"] = "El mínimo no puede ser mayor que el máximo"
	}

	// Validate step order
	if p.StepOrder < 1 {
		p.Errors["step_order"] = "El orden del paso debe ser al menos 1"
	}

	return len(p.Errors) == 0
}

func (p *WizardStepParams) ToWizardStep() *db.WizardStep {
	return &db.WizardStep{
		Required:    p.Required,
		MultiSelect: p.MultiSelect,
		MinSelected: p.MinSelected,
		MaxSelected: p.MaxSelected,
		StepOrder:   p.StepOrder,
	}
}

// Helper function to parse form values into WizardStepParams
func ParseWizardStepParams(formValues map[string][]string, prefix string) *WizardStepParams {
	params := NewWizardStepParams()

	if val := formValues[prefix+"required"]; len(val) > 0 {
		params.Required = val[0] == "on" || val[0] == "true"
	}

	if val := formValues[prefix+"multi_select"]; len(val) > 0 {
		params.MultiSelect = val[0] == "on" || val[0] == "true"
	}

	if val := formValues[prefix+"min_selected"]; len(val) > 0 {
		if min, err := strconv.Atoi(val[0]); err == nil {
			params.MinSelected = min
		}
	}

	if val := formValues[prefix+"max_selected"]; len(val) > 0 {
		if max, err := strconv.Atoi(val[0]); err == nil {
			params.MaxSelected = max
		}
	}

	if val := formValues[prefix+"step_order"]; len(val) > 0 {
		if order, err := strconv.Atoi(val[0]); err == nil {
			params.StepOrder = order
		}
	}

	return params
}
