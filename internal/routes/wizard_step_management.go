package routes

import (
	"log"
	"net/http"
	"strconv"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
)

func RenderAddStepToWizardForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	wizardID := r.PathValue("wizard_id")
	if wizardID == "" {
		http.Error(w, "ID de asistente inválido", http.StatusBadRequest)
		return
	}

	// Get available wizard steps
	allSteps, err := db.GetAllWizardSteps(r.Context())
	if err != nil {
		http.Error(w, "Error al cargar pasos disponibles", http.StatusInternalServerError)
		log.Printf("failed to get wizard steps: %v\n", err)
		return
	}

	// Get currently attached steps
	attachedSteps, err := db.GetWizardSteps(r.Context(), wizardID)
	if err != nil {
		http.Error(w, "Error al cargar pasos actuales", http.StatusInternalServerError)
		log.Printf("failed to get wizard steps: %v\n", err)
		return
	}

	// Filter out already attached steps
	attachedStepIDs := make(map[string]bool)
	for _, step := range attachedSteps {
		attachedStepIDs[step.ID] = true
	}

	var availableSteps []*db.WizardStep
	for _, step := range allSteps {
		if !attachedStepIDs[step.ID] {
			availableSteps = append(availableSteps, step)
		}
	}

	component := dashboard.AddStepToWizardModal(wizardID, availableSteps)
	component.Render(r.Context(), w)
}

func RenderEditWizardStepParamsForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	wizardID := r.PathValue("wizard_id")
	stepID := r.PathValue("step_id")

	if wizardID == "" || stepID == "" {
		http.Error(w, "IDs inválidos", http.StatusBadRequest)
		return
	}

	// Get wizard with steps to find the specific step parameters
	wizard, err := db.GetWizardWithSteps(r.Context(), wizardID)
	if err != nil {
		http.Error(w, "Asistente no encontrado", http.StatusNotFound)
		log.Printf("failed to find wizard: %v\n", err)
		return
	}

	// Find the specific step
	var wizardStep *db.WizardStep
	for _, step := range wizard.Steps {
		if step.ID == stepID {
			wizardStep = step
			break
		}
	}

	if wizardStep == nil {
		http.Error(w, "Paso no encontrado en el asistente", http.StatusNotFound)
		return
	}

	component := dashboard.EditWizardStepParamsModal(wizardID, wizardStep)
	component.Render(r.Context(), w)
}

func AttachStepToWizardAndReturn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	wizardID := r.PathValue("wizard_id")
	if wizardID == "" {
		http.Error(w, "ID de asistente inválido", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al procesar formulario", http.StatusBadRequest)
		return
	}

	stepID := r.FormValue("step_id")
	if stepID == "" {
		http.Error(w, "ID de paso requerido", http.StatusBadRequest)
		return
	}

	// Parse step parameters
	stepParams := &forms.WizardStepParams{
		Required:    r.FormValue("required") == "on",
		MultiSelect: r.FormValue("multi_select") == "on",
		StepOrder:   1,
		MinSelected: 0,
		MaxSelected: 1,
	}

	if order := r.FormValue("step_order"); order != "" {
		if o, err := strconv.Atoi(order); err == nil {
			stepParams.StepOrder = o
		}
	}
	if min := r.FormValue("min_selected"); min != "" {
		if m, err := strconv.Atoi(min); err == nil {
			stepParams.MinSelected = m
		}
	}
	if max := r.FormValue("max_selected"); max != "" {
		if m, err := strconv.Atoi(max); err == nil {
			stepParams.MaxSelected = m
		}
	}

	if !stepParams.Validate() {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	// Attach step to wizard
	err = db.AttachStepToWizard(r.Context(), wizardID, stepID, stepParams.ToWizardStep())
	if err != nil {
		http.Error(w, "Error al agregar paso", http.StatusInternalServerError)
		log.Printf("failed to attach step to wizard: %v\n", err)
		return
	}

	w.Write([]byte("Paso agregado exitosamente"))
}

func UpdateWizardStepParamsAndReturn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	wizardID := r.PathValue("wizard_id")
	stepID := r.PathValue("step_id")

	if wizardID == "" || stepID == "" {
		http.Error(w, "IDs inválidos", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error al procesar formulario", http.StatusBadRequest)
		return
	}

	// Parse step parameters
	stepParams := &forms.WizardStepParams{
		Required:    r.FormValue("required") == "on",
		MultiSelect: r.FormValue("multi_select") == "on",
		StepOrder:   1,
		MinSelected: 0,
		MaxSelected: 1,
	}

	if order := r.FormValue("step_order"); order != "" {
		if o, err := strconv.Atoi(order); err == nil {
			stepParams.StepOrder = o
		}
	}
	if min := r.FormValue("min_selected"); min != "" {
		if m, err := strconv.Atoi(min); err == nil {
			stepParams.MinSelected = m
		}
	}
	if max := r.FormValue("max_selected"); max != "" {
		if m, err := strconv.Atoi(max); err == nil {
			stepParams.MaxSelected = m
		}
	}

	if !stepParams.Validate() {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	// Update wizard step parameters
	err = db.UpdateWizardStepParams(r.Context(), wizardID, stepID, stepParams.ToWizardStep())
	if err != nil {
		http.Error(w, "Error al actualizar parámetros", http.StatusInternalServerError)
		log.Printf("failed to update wizard step params: %v\n", err)
		return
	}

	w.Write([]byte("Parámetros actualizados"))
}

func DetachStepFromWizardAndReturn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	wizardID := r.PathValue("wizard_id")
	stepID := r.PathValue("step_id")

	if wizardID == "" || stepID == "" {
		http.Error(w, "IDs inválidos", http.StatusBadRequest)
		return
	}

	// Detach step from wizard
	err := db.DetachStepFromWizard(r.Context(), wizardID, stepID)
	if err != nil {
		http.Error(w, "Error al quitar paso", http.StatusInternalServerError)
		log.Printf("failed to detach step from wizard: %v\n", err)
		return
	}

	// Return empty content to remove the step row
	w.WriteHeader(http.StatusOK)
}
