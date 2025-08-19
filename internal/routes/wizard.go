package routes

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/templates/components"
	"github.com/vladwithcode/salon_catalog/internal/templates/components/dashboard"
	dashboardPages "github.com/vladwithcode/salon_catalog/internal/templates/pages/dashboard"
)

func RegisterWizardRoutes(router *customServeMux) {
	// Dashboard page
	router.HandleFunc("GET /panel/asistentes", auth.ValidateAuth(RenderWizard))
	router.HandleFunc("GET /panel/asistentes/pasos", auth.ValidateAuth(RenderWizardSteps))

	// Table routes
	router.HandleFunc("GET /panel/asistentes/table", auth.ValidateAuth(RenderWizardsTable))
	router.HandleFunc("POST /panel/asistentes/nuevo", auth.ValidateAuth(CreateWizardAndReturnTable))
	router.HandleFunc("GET /panel/asistentes/modal/nuevo", auth.ValidateAuth(RenderNewWizardForm))
	router.HandleFunc("GET /panel/asistentes/modal/{id}", auth.ValidateAuth(RenderEditWizardForm))
	router.HandleFunc("PUT /panel/asistentes/{id}", auth.ValidateAuth(UpdateWizardAndReturnTable))
	router.HandleFunc("DELETE /panel/asistentes", auth.ValidateAuth(DeleteWizardsAndReturnTable))
	router.HandleFunc("DELETE /panel/asistentes/{id}", auth.ValidateAuth(DeleteWizardAndReturnTable))

	router.HandleFunc("GET /panel/asistentes/pasos/table", auth.ValidateAuth(RenderWizardStepsTable))
	router.HandleFunc("POST /panel/asistentes/pasos/nuevo", auth.ValidateAuth(CreateWizardStepAndReturnTable))
	router.HandleFunc("GET /panel/asistentes/pasos/modal/nuevo", auth.ValidateAuth(RenderNewWizardStepForm))
	router.HandleFunc("GET /panel/asistentes/pasos/modal/{id}", auth.ValidateAuth(RenderEditWizardStepForm))
	router.HandleFunc("PUT /panel/asistentes/pasos/{id}", auth.ValidateAuth(UpdateWizardStepAndReturnTable))
	router.HandleFunc("DELETE /panel/asistentes/pasos", auth.ValidateAuth(DeleteWizardStepsAndReturnTable))
	router.HandleFunc("DELETE /panel/asistentes/pasos/{id}", auth.ValidateAuth(DeleteWizardStepAndReturnTable))

	// Wizard step management routes
	router.HandleFunc("GET /panel/asistentes/{wizard_id}/pasos/modal/nuevo", auth.ValidateAuth(RenderAddStepToWizardForm))
	router.HandleFunc("GET /panel/asistentes/{wizard_id}/pasos/modal/{step_id}", auth.ValidateAuth(RenderEditWizardStepParamsForm))
	router.HandleFunc("POST /panel/asistentes/{wizard_id}/pasos", auth.ValidateAuth(AttachStepToWizardAndReturn))
	router.HandleFunc("PUT /panel/asistentes/{wizard_id}/pasos/{step_id}", auth.ValidateAuth(UpdateWizardStepParamsAndReturn))
	router.HandleFunc("DELETE /panel/asistentes/{wizard_id}/pasos/{step_id}", auth.ValidateAuth(DetachStepFromWizardAndReturn))
}

func RenderWizard(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := dashboardPages.Wizards().Render(r.Context(), w)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("Something went wrong"))
		log.Printf("failed to render Wizard err: %v\n", err)
	}
}

func RenderWizardSteps(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := dashboardPages.WizardSteps().Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Algo salió mal", 500)
		log.Printf("failed to render Wizard err: %v\n", err)
	}
}

func RenderWizardsTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	params, err := parseWizardFilterParams(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to parse request parameters"))
		log.Printf("failed to parse request parameters: %v\n", err)
		return
	}

	wizards, err := db.FilterWizards(*params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find wizards"))
		log.Printf("failed to find wizards: %v\n", err)
		return
	}

	component := dashboard.WizardsTable(wizards)
	component.Render(r.Context(), w)
}

func RenderNewWizardForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Get available event kinds
	eventKinds, err := db.FindAllEventKinds()
	if err != nil {
		log.Printf("failed to find event kinds: %v\n", err)
		component := dashboard.WizardCreateModal(&dashboard.WizardCreateModalState{Error: "Ocurrió un error inesperado."})
		component.Render(r.Context(), w)
		return
	}

	// Get available wizard steps
	wizardSteps, err := db.GetAllWizardSteps(r.Context())
	if err != nil {
		log.Printf("failed to find wizard steps: %v\n", err)
		component := dashboard.WizardCreateModal(&dashboard.WizardCreateModalState{Error: "Ocurrió un error inesperado."})
		component.Render(r.Context(), w)
		return
	}

	formState := &dashboard.WizardCreateModalState{
		Kinds: eventKinds,
		Steps: wizardSteps,
	}

	component := dashboard.WizardCreateModal(formState)
	component.Render(r.Context(), w)
}

func RenderEditWizardForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	wizardID := r.PathValue("id")
	if wizardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		component := dashboard.WizardModal(nil, nil, "ID de asistente inválido")
		component.Render(r.Context(), w)
		return
	}

	// Get wizard with steps
	wizard, err := db.GetWizardWithSteps(r.Context(), wizardID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		component := dashboard.WizardModal(nil, nil, "Asistente no encontrado")
		component.Render(r.Context(), w)
		log.Printf("failed to find wizard: %v\n", err)
		return
	}

	// Get available event kinds
	eventKinds, err := db.FindAllEventKinds()
	if err != nil {
		log.Printf("failed to find event kinds: %v\n", err)
		component := dashboard.WizardModal(wizard, nil, "Error al cargar tipos de evento")
		component.Render(r.Context(), w)
		return
	}

	component := dashboard.WizardModal(wizard, eventKinds, "")
	component.Render(r.Context(), w)
}

func CreateWizardAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se creó el asistente exitosamente", components.ToastSuccess, 3000, true, false)

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		formState := &dashboard.WizardCreateModalState{
			Error: "Error al procesar el formulario",
		}
		comp := templ.Join(
			components.ToasterToast(toastData),
			dashboard.WizardCreateModal(formState),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	eventKinds, err := db.FindAllEventKinds()
	if err != nil {
		log.Printf("failed to find event kinds: %v\n", err)
		component := dashboard.WizardCreateModal(&dashboard.WizardCreateModalState{Error: "Ocurrió un error inesperado."})
		component.Render(r.Context(), w)
		return
	}

	// Get form values
	name := strings.TrimSpace(r.FormValue("name"))
	eventKindID := strings.TrimSpace(r.FormValue("event_kind"))
	description := strings.TrimSpace(r.FormValue("description"))
	selectedStepIDs := r.Form["selected_steps"]

	// Validate required fields
	if name == "" || eventKindID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Nombre y tipo de evento son requeridos"
		toastData.Type = components.ToastError

		comp := templ.Join(
			dashboard.WizardCreateModal(&dashboard.WizardCreateModalState{
				Kinds: eventKinds,
				Error: "Nombre y tipo de evento son requeridos",
			}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	formState := &dashboard.WizardCreateModalState{
		Wizard: &db.Wizard{
			Name:        name,
			EventKindID: eventKindID,
		},
		SelectedKindIdx:   0,
		SelectedStepsIdxs: []int{},
	}
	// Create new wizard
	wizard := &db.Wizard{
		Name:        name,
		Description: description,
		EventKindID: eventKindID,
	}

	// Add selected steps if any
	if len(selectedStepIDs) > 0 {
		for i, stepID := range selectedStepIDs {
			wizard.Steps = append(wizard.Steps, &db.WizardStep{
				ID:          stepID,
				StepOrder:   i + 1,
				Required:    false,
				MultiSelect: false,
				MinSelected: 0,
				MaxSelected: 1,
			})
		}
	}

	// Create wizard in database
	err = db.CreateWizard(r.Context(), wizard)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Ocurrió un error inesperado."
		toastData.Type = components.ToastError
		formState.Error = "Ocurrió un error inesperado."

		comp := templ.Join(
			dashboard.WizardCreateModal(formState),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to create wizard: %v\n", err)
		return
	}

	// Success - return table with toast
	params := &db.WizardFilterParams{Page: 1, Limit: 20}
	wizards, err := db.FilterWizards(*params)
	if err != nil {
		wizards = &db.WizardFilterResult{HasError: true, Error: "Error al recargar asistentes"}
	}

	comp := templ.Join(
		dashboard.WizardsTable(wizards),
		components.ToasterToast(toastData),
	)
	comp.Render(r.Context(), w)
}

func UpdateWizardAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se actualizó el asistente exitosamente", components.ToastSuccess, 3000, true, false)

	// Get wizard ID from URL
	wizardID := r.PathValue("id")
	if wizardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "ID de asistente inválido"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "ID de asistente inválido"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Get form values
	name := strings.TrimSpace(r.FormValue("name"))
	eventKindID := strings.TrimSpace(r.FormValue("event_kind"))

	// Validate required fields
	if name == "" || eventKindID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Nombre y tipo de evento son requeridos"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Nombre y tipo de evento son requeridos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Update wizard
	wizard := &db.Wizard{
		ID:          wizardID,
		Name:        name,
		EventKindID: eventKindID,
	}

	err = db.UpdateWizard(r.Context(), wizard)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el asistente"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Error al actualizar el asistente"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to update wizard: %v\n", err)
		return
	}

	// Success - return table with toast
	params := &db.WizardFilterParams{Page: 1, Limit: 20}
	wizards, err := db.FilterWizards(*params)
	if err != nil {
		wizards = &db.WizardFilterResult{HasError: true, Error: "Error al recargar asistentes"}
	}

	comp := templ.Join(
		dashboard.WizardsTable(wizards),
		components.ToasterToast(toastData),
	)
	comp.Render(r.Context(), w)
}

func DeleteWizardAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminó el asistente exitosamente", components.ToastSuccess, 3000, true, false)

	// Get wizard ID from URL
	wizardID := r.PathValue("id")
	if wizardID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "ID de asistente inválido"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "ID de asistente inválido"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Delete wizard
	err := db.DeleteWizard(r.Context(), wizardID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al eliminar el asistente"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Error al eliminar el asistente"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to delete wizard: %v\n", err)
		return
	}

	// Success - return table with toast
	params := &db.WizardFilterParams{Page: 1, Limit: 20}
	wizards, err := db.FilterWizards(*params)
	if err != nil {
		wizards = &db.WizardFilterResult{HasError: true, Error: "Error al recargar asistentes"}
	}

	comp := templ.Join(
		dashboard.WizardsTable(wizards),
		components.ToasterToast(toastData),
	)
	comp.Render(r.Context(), w)
}

func DeleteWizardsAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// TODO: Implement batch delete for wizards
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Batch delete not implemented yet"))
}

func RenderWizardStepsTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	params, err := parseWizardStepFilterParams(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to parse request parameters"))
		log.Printf("failed to parse request parameters: %v\n", err)
		return
	}

	wizardSteps, err := db.FilterWizardSteps(*params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to find wizard steps"))
		log.Printf("failed to find wizard steps: %v\n", err)
		return
	}

	component := dashboard.WizardStepsTable(wizardSteps)
	component.Render(r.Context(), w)
}

func RenderNewWizardStepForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	// Get available categories
	categories, err := db.FindAllCategories()
	if err != nil {
		log.Printf("failed to find categories: %v\n", err)
		component := dashboard.WizardStepCreateModal(&dashboard.WizardStepCreateModalState{Error: "Ocurrió un error inesperado."})
		component.Render(r.Context(), w)
		return
	}
	formState := &dashboard.WizardStepCreateModalState{
		Categories: categories,
	}

	component := dashboard.WizardStepCreateModal(formState)
	component.Render(r.Context(), w)
}

func RenderEditWizardStepForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	stepID := r.PathValue("id")
	if stepID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("ID de paso inválido"))
		return
	}

	// Get the wizard step
	step, err := db.FindWizardStep(r.Context(), stepID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Paso no encontrado"))
		log.Printf("failed to find wizard step: %v\n", err)
		return
	}

	// Get available categories
	categories, err := db.FindAllCategories()
	if err != nil {
		log.Printf("failed to find categories: %v\n", err)
		component := dashboard.WizardStepEditModal(step, &dashboard.WizardStepEditModalState{Error: "Ocurrió un error inesperado."})
		component.Render(r.Context(), w)
		return
	}

	formState := &dashboard.WizardStepEditModalState{
		Categories: categories,
	}

	component := dashboard.WizardStepEditModal(step, formState)
	component.Render(r.Context(), w)
}

func CreateWizardStepAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se creó el paso exitosamente", components.ToastSuccess, 3000, true, false)

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		formState := &dashboard.WizardStepCreateModalState{
			Error: "Error al procesar el formulario",
		}
		comp := templ.Join(
			components.ToasterToast(toastData),
			dashboard.WizardStepCreateModal(formState),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	categories, err := db.FindAllCategories()
	if err != nil {
		log.Printf("failed to find categories: %v\n", err)
		component := dashboard.WizardStepCreateModal(&dashboard.WizardStepCreateModalState{Error: "Ocurrió un error inesperado."})
		component.Render(r.Context(), w)
		return
	}

	// Get form values
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	required := r.FormValue("required") == "on"
	multiSelect := r.FormValue("multi_select") == "on"

	minSelected, _ := strconv.Atoi(r.FormValue("min_selected"))
	maxSelected, _ := strconv.Atoi(r.FormValue("max_selected"))
	stepOrder, _ := strconv.Atoi(r.FormValue("step_order"))
	categoryIDs := r.Form["category_ids"]

	// Validate required fields
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "El nombre es requerido"
		toastData.Type = components.ToastError

		comp := templ.Join(
			dashboard.WizardStepCreateModal(&dashboard.WizardStepCreateModalState{
				Categories: categories,
				Error:      "El nombre es requerido",
			}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Create new wizard step
	wizardStep := &db.WizardStep{
		Name:        name,
		Description: description,
		Required:    required,
		MultiSelect: multiSelect,
		MinSelected: minSelected,
		MaxSelected: maxSelected,
		StepOrder:   stepOrder,
		CategoryIDs: categoryIDs,
	}

	// Create wizard step in database
	err = db.CreateWizardStep(r.Context(), wizardStep)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Ocurrió un error inesperado."
		toastData.Type = components.ToastError
		formState := &dashboard.WizardStepCreateModalState{
			Categories: categories,
			Error:      "Ocurrió un error inesperado.",
		}

		comp := templ.Join(
			dashboard.WizardStepCreateModal(formState),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to create wizard step: %v\n", err)
		return
	}

	// Success - return table with toast
	params := &db.WizardStepFilterParams{Page: 1, Limit: 20}
	wizardSteps, err := db.FilterWizardSteps(*params)
	if err != nil {
		wizardSteps = &db.WizardStepFilterResult{HasError: true, Error: "Error al recargar pasos"}
	}

	comp := templ.Join(
		dashboard.WizardStepsTable(wizardSteps),
		components.ToasterToast(toastData),
	)
	comp.Render(r.Context(), w)
}

func UpdateWizardStepAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se actualizó el paso exitosamente", components.ToastSuccess, 3000, true, false)

	// Get step ID from URL
	stepID := r.PathValue("id")
	if stepID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "ID de paso inválido"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "ID de paso inválido"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to parse form: %v\n", err)
		return
	}

	// Get form values
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	required := r.FormValue("required") == "on"
	multiSelect := r.FormValue("multi_select") == "on"

	minSelected, _ := strconv.Atoi(r.FormValue("min_selected"))
	maxSelected, _ := strconv.Atoi(r.FormValue("max_selected"))
	stepOrder, _ := strconv.Atoi(r.FormValue("step_order"))
	categoryIDs := r.Form["category_ids"]

	// Validate required fields
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "El nombre es requerido"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "El nombre es requerido"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Update wizard step
	wizardStep := &db.WizardStep{
		ID:          stepID,
		Name:        name,
		Description: description,
		Required:    required,
		MultiSelect: multiSelect,
		MinSelected: minSelected,
		MaxSelected: maxSelected,
		StepOrder:   stepOrder,
		CategoryIDs: categoryIDs,
	}

	err = db.UpdateWizardStep(r.Context(), wizardStep)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al actualizar el paso"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "Error al actualizar el paso"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to update wizard step: %v\n", err)
		return
	}

	// Success - return table with toast
	params := &db.WizardStepFilterParams{Page: 1, Limit: 20}
	wizardSteps, err := db.FilterWizardSteps(*params)
	if err != nil {
		wizardSteps = &db.WizardStepFilterResult{HasError: true, Error: "Error al recargar pasos"}
	}

	comp := templ.Join(
		dashboard.WizardStepsTable(wizardSteps),
		components.ToasterToast(toastData),
	)
	comp.Render(r.Context(), w)
}

func DeleteWizardStepAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminó el paso exitosamente", components.ToastSuccess, 3000, true, false)

	// Get step ID from URL
	stepID := r.PathValue("id")
	if stepID == "" {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "ID de paso inválido"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "ID de paso inválido"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Delete wizard step
	err := db.DeleteWizardStep(r.Context(), stepID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		toastData.Message = "Error al eliminar el paso"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "Error al eliminar el paso"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		log.Printf("failed to delete wizard step: %v\n", err)
		return
	}

	// Success - return table with toast
	params := &db.WizardStepFilterParams{Page: 1, Limit: 20}
	wizardSteps, err := db.FilterWizardSteps(*params)
	if err != nil {
		wizardSteps = &db.WizardStepFilterResult{HasError: true, Error: "Error al recargar pasos"}
	}

	comp := templ.Join(
		dashboard.WizardStepsTable(wizardSteps),
		components.ToasterToast(toastData),
	)
	comp.Render(r.Context(), w)
}

func DeleteWizardStepsAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	w.Header().Add("X-Includes-Toast", "true")
	toastData := components.NewToastData("Se eliminaron los pasos exitosamente", components.ToastSuccess, 3000, true, false)

	// Parse form data to get selected step IDs
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "Error al procesar el formulario"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	stepIDs := r.Form["wizard-step-selection"]
	if len(stepIDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		toastData.Message = "No se seleccionaron pasos para eliminar"
		toastData.Type = components.ToastError
		comp := templ.Join(
			dashboard.WizardStepsTable(&db.WizardStepFilterResult{HasError: true, Error: "No se seleccionaron pasos"}),
			components.ToasterToast(toastData),
		)
		comp.Render(r.Context(), w)
		return
	}

	// Delete each selected wizard step
	for _, stepID := range stepIDs {
		err := db.DeleteWizardStep(r.Context(), stepID)
		if err != nil {
			log.Printf("failed to delete wizard step %s: %v\n", stepID, err)
		}
	}

	// Success - return table with toast
	params := &db.WizardStepFilterParams{Page: 1, Limit: 20}
	wizardSteps, err := db.FilterWizardSteps(*params)
	if err != nil {
		wizardSteps = &db.WizardStepFilterResult{HasError: true, Error: "Error al recargar pasos"}
	}

	comp := templ.Join(
		dashboard.WizardStepsTable(wizardSteps),
		components.ToasterToast(toastData),
	)
	comp.Render(r.Context(), w)
}

// parseWizardFilterParams parses filter parameters from the request
func parseWizardFilterParams(r *http.Request) (*db.WizardFilterParams, error) {
	params := &db.WizardFilterParams{
		Search:     strings.TrimSpace(r.URL.Query().Get("search")),
		SearchMode: db.SearchMode(r.URL.Query().Get("search_mode")),
		EventKind:  r.URL.Query().Get("event_kind_filter"),
		Sort:       r.URL.Query().Get("sort"),
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}
	if params.Page == 0 {
		params.Page = 1
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	// Set default search mode
	if params.SearchMode == "" {
		params.SearchMode = db.SearchModeFullText
	}

	return params, nil
}

func parseWizardStepFilterParams(r *http.Request) (*db.WizardStepFilterParams, error) {
	params := &db.WizardStepFilterParams{
		Search:     strings.TrimSpace(r.URL.Query().Get("search")),
		SearchMode: db.SearchMode(r.URL.Query().Get("search_mode")),
		Categories: r.URL.Query()["categories"],
		Sort:       r.URL.Query().Get("sort"),
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}
	if params.Page == 0 {
		params.Page = 1
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			params.Limit = limit
		}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	// Set default search mode
	if params.SearchMode == "" {
		params.SearchMode = db.SearchModeFullText
	}

	return params, nil
}
