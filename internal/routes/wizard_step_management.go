package routes

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/forms"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
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
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Paso agregado exitosamente", components.ToastSuccess, 3000, true, false)

	wizardID := r.PathValue("wizard_id")
	if wizardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "ID de asistente inválido"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar formulario"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		return
	}

	stepID := r.FormValue("step_id")
	if stepID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "ID de paso requerido"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
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
	if minSelected := r.FormValue("min_selected"); minSelected != "" {
		if m, err := strconv.Atoi(minSelected); err == nil {
			stepParams.MinSelected = m
		}
	}
	if maxSelected := r.FormValue("max_selected"); maxSelected != "" {
		if m, err := strconv.Atoi(maxSelected); err == nil {
			stepParams.MaxSelected = m
		}
	}

	if !stepParams.Validate() {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Parámetros inválidos"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		return
	}

	// Attach step to wizard
	err = db.AttachStepToWizard(r.Context(), wizardID, stepID, stepParams.ToWizardStep())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al agregar paso"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to attach step to wizard: %v\n", err)
		return
	}

	// Get updated wizard with steps to return refreshed UI
	wizard, err := db.GetWizardWithSteps(r.Context(), wizardID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recargar pasos del asistente"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to get wizard with steps: %v\n", err)
		return
	}

	// Return updated wizard steps section with success toast
	comp := templ.Join(
		dashboard.WizardEditForm(wizard, []*db.EventKind{}),
		components.ToasterToast(toastData),
	)
	w.Header().Set("HX-Trigger", `{"app:closeStepsModal": {}}`)
	reCtx := context.WithValue(r.Context(), "swapOOBStepSection", true)
	templ.RenderFragments(reCtx, w, comp, "toaster-toast", "wizardStepsSection")
}

func UpdateWizardStepParamsAndReturn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Paso actualizado exitosamente", components.ToastSuccess, 3000, true, false)

	wizardID := r.PathValue("wizard_id")
	stepID := r.PathValue("step_id")

	if wizardID == "" || stepID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "IDs inválidos"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar formulario"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
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
	if minSelected := r.FormValue("min_selected"); minSelected != "" {
		if m, err := strconv.Atoi(minSelected); err == nil {
			stepParams.MinSelected = m
		}
	}
	if maxSelected := r.FormValue("max_selected"); maxSelected != "" {
		if m, err := strconv.Atoi(maxSelected); err == nil {
			stepParams.MaxSelected = m
		}
	}

	if !stepParams.Validate() {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Parámetros inválidos"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		return
	}

	// Update wizard step parameters
	err = db.UpdateWizardStepParams(r.Context(), wizardID, stepID, stepParams.ToWizardStep())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar parámetros"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to update wizard step params: %v\n", err)
		return
	}

	// Get updated wizard with steps to return refreshed UI
	wizard, err := db.GetWizardWithSteps(r.Context(), wizardID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recargar pasos del asistente"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to get wizard with steps: %v\n", err)
		return
	}

	// Return updated wizard steps section with success toast
	comp := templ.Join(
		dashboard.WizardEditForm(wizard, []*db.EventKind{}),
		components.ToasterToast(toastData),
	)
	w.Header().Set("HX-Trigger", `{"app:closeStepsModal": {}}`)
	reCtx := context.WithValue(r.Context(), "swapOOBStepSection", true)
	templ.RenderFragments(reCtx, w, comp, "toaster-toast", "wizardStepsSection")
}

func DetachStepFromWizardAndReturn(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Paso eliminado exitosamente", components.ToastSuccess, 3000, true, false)

	wizardID := r.PathValue("wizard_id")
	stepID := r.PathValue("step_id")

	if wizardID == "" || stepID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "IDs inválidos"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		return
	}

	// Detach step from wizard
	err := db.DetachStepFromWizard(r.Context(), wizardID, stepID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al quitar paso"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to detach step from wizard: %v\n", err)
		return
	}

	// Get updated wizard with steps to return refreshed UI
	wizard, err := db.GetWizardWithSteps(r.Context(), wizardID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al recargar pasos del asistente"
		toastData.Type = components.ToastError
		components.ToasterToast(toastData).Render(r.Context(), w)
		log.Printf("failed to get wizard with steps: %v\n", err)
		return
	}

	// Return updated wizard steps section with success toast
	comp := templ.Join(
		dashboard.WizardEditForm(wizard, []*db.EventKind{}),
		components.ToasterToast(toastData),
	)
	reCtx := context.WithValue(r.Context(), "swapOOBStepSection", true)
	templ.RenderFragments(reCtx, w, comp, "toaster-toast", "wizardStepsSection")
}

func RenderWizardStepsFragment(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	wizardID := r.PathValue("wizard_id")
	if wizardID == "" {
		http.Error(w, "ID de asistente inválido", http.StatusBadRequest)
		return
	}

	// Get wizard with steps
	wizard, err := db.GetWizardWithSteps(r.Context(), wizardID)
	if err != nil {
		http.Error(w, "Asistente no encontrado", http.StatusNotFound)
		log.Printf("failed to find wizard: %v\n", err)
		return
	}

	// Render just the wizard steps management section
	component := dashboard.WizardStepsManagementSection(wizard)
	component.Render(r.Context(), w)
}
