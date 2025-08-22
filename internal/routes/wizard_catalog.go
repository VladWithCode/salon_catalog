package routes

import (
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
)

func RenderWizardModal(w http.ResponseWriter, r *http.Request) {
	filters := db.WizardFilterParams{
		Enabled: 1, // Enabled wizards only
	}
	wizards, err := db.FilterWizards(filters)
	if err != nil {
		w.Header().Add("X-Includes-Toast", "true")
		w.Header().Add("HX-Retarget", "#toaster-container")
		w.Header().Add("HX-Reswap", "beforeend")
		components.ToasterToast(
			components.NewToastData(
				"Algo salió mal",
				components.ToastError,
				3000,
				true,
				false,
			),
		).Render(r.Context(), w)
		log.Printf("failed to find wizards: %v\n", err)
		return
	}

	state := components.WizardModalState{
		CurrentView: "selection",
		Wizards:     wizards.Wizards,
	}
	err = components.WizardModal(&state).Render(r.Context(), w)
	if err != nil {
		w.Header().Add("X-Includes-Toast", "true")
		w.Header().Add("HX-Retarget", "#toaster-container")
		w.Header().Add("HX-Reswap", "beforeend")
		components.ToasterToast(
			components.NewToastData(
				"Algo salió mal",
				components.ToastError,
				3000,
				true,
				false,
			),
		).Render(r.Context(), w)
		log.Printf("failed to find wizards: %v\n", err)
		return
	}
}

func RenderWizardStep(w http.ResponseWriter, r *http.Request) {
	wizardID := r.PathValue("wizard_id")
	stepID := r.PathValue("step_id")
	state := components.WizardModalState{
		CurrentView: "step",
	}

	wizard, err := db.FindWizard(r.Context(), wizardID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Asistente no encontrado"))
		log.Printf("failed to find wizard: %v\n", err)
		return
	}
	steps, err := db.GetWizardSteps(r.Context(), wizardID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al recuperar pasos"))
		log.Printf("failed to find wizard step: %v\n", err)
		return
	}

	wizard.Steps = steps
	if stepID == "" {
		state.CurrentStep = steps[0]
	} else {
		for i, step := range steps {
			if step.ID == stepID {
				state.CurrentStep = steps[i]
				break
			}
		}
	}

	products, err := db.FilterCatalogProductsForWizard(state.CurrentStep.CategoryIDs, []string{}, 10)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al recuperar productos"))
		log.Printf("failed to find wizard step: %v\n", err)
		return
	}

	state.CurrentWizard = wizard
	state.Products = products.Products
	state.StepIndex = state.CurrentStep.StepOrder

	err = components.WizardModal(&state).Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error al renderizar modal"))
		log.Printf("failed to render wizard modal: %v\n", err)
		return
	}
}
