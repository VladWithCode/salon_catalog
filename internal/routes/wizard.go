package routes

func RegisterWizardRoutes(router *customServeMux) {
	// Dashboard page
	// router.HandleFunc("GET /panel/asistentes", auth.ValidateAuth(RenderWizard))

	// Table routes
	// router.HandleFunc("GET /panel/asistentes/table", auth.ValidateAuth(RenderWizardsTable))
	// router.HandleFunc("POST /panel/asistentes/nuevo", auth.ValidateAuth(CreateWizardAndReturnTable))
	// router.HandleFunc("GET /panel/asistentes/modal/nuevo", auth.ValidateAuth(RenderNewWizardForm))
	// router.HandleFunc("GET /panel/asistentes/modal/{id}", auth.ValidateAuth(RenderWizard))
	// router.HandleFunc("PUT /panel/asistentes/{id}", auth.ValidateAuth(UpdateWizardAndReturnTable))
	// router.HandleFunc("DELETE /panel/asistentes", auth.ValidateAuth(DeleteWizardsAndReturnTable))
	// router.HandleFunc("DELETE /panel/asistentes/{id}", auth.ValidateAuth(DeleteWizardAndReturnTable))
	//
	// // Legacy API routes
	// router.HandleFunc("GET /api/wizard", InitWizard)
	// router.HandleFunc("GET /api/wizard/categories", GetWizardCategories)
}

//
// func RenderWizard(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
// 	err := dashboardPages.Wizards().Render(r.Context(), w)
// 	if err != nil {
// 		w.WriteHeader(500)
// 		w.Write([]byte("Something went wrong"))
// 		log.Printf("failed to render Wizard err: %v\n", err)
// 	}
// }
//
// func RenderWizardsTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
// 	params, err := parseWizardFilterParams(r)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		w.Write([]byte("Failed to parse request parameters"))
// 		log.Printf("failed to parse request parameters: %v\n", err)
// 		return
// 	}
//
// 	wizards, err := db.FilterWizards(*params)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		w.Write([]byte("Failed to find wizards"))
// 		log.Printf("failed to find wizards: %v\n", err)
// 		return
// 	}
//
// 	component := dashboard.WizardsTable(wizards)
// 	component.Render(r.Context(), w)
// }
//
// func RenderNewWizardForm(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
// 	// Get available event kinds
// 	eventKinds, err := db.FindAllEventKinds()
// 	if err != nil {
// 		log.Printf("failed to find event kinds: %v\n", err)
// 		component := dashboard.WizardCreateModal(nil, "Error al cargar tipos de evento")
// 		component.Render(r.Context(), w)
// 		return
// 	}
//
// 	component := dashboard.WizardCreateModal(eventKinds, "")
// 	component.Render(r.Context(), w)
// }
//
// func CreateWizardAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
// 	w.Header().Add("X-Includes-Toast", "true")
// 	toastData := components.NewToastData("Se creó el asistente exitosamente", components.ToastSuccess, 3000, true, false)
//
// 	// Parse form data
// 	err := r.ParseForm()
// 	if err != nil {
// 		w.WriteHeader(http.StatusBadRequest)
// 		toastData.Message = "Error al procesar el formulario"
// 		toastData.Type = components.ToastError
// 		comp := templ.Join(
// 			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		log.Printf("failed to parse form: %v\n", err)
// 		return
// 	}
//
// 	// Get form values
// 	name := strings.TrimSpace(r.FormValue("name"))
// 	eventKindID := strings.TrimSpace(r.FormValue("event_kind"))
//
// 	// Validate required fields
// 	if name == "" || eventKindID == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 		toastData.Message = "Nombre y tipo de evento son requeridos"
// 		toastData.Type = components.ToastError
//
// 		eventKinds, _ := db.FindAllEventKinds()
// 		comp := templ.Join(
// 			dashboard.WizardCreateModal(eventKinds, "Nombre y tipo de evento son requeridos"),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		return
// 	}
//
// 	// Create new wizard
// 	wizard := &db.Wizard{
// 		Name:        name,
// 		EventKindID: eventKindID,
// 	}
//
// 	// Create wizard in database (no steps for now)
// 	err = db.CreateWizard(wizard, []*db.WizardStep{})
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		toastData.Message = "Error al crear el asistente"
// 		toastData.Type = components.ToastError
//
// 		eventKinds, _ := db.FindAllEventKinds()
// 		comp := templ.Join(
// 			dashboard.WizardCreateModal(eventKinds, "Error al crear el asistente"),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		log.Printf("failed to create wizard: %v\n", err)
// 		return
// 	}
//
// 	// Success - return table with toast
// 	params := &db.WizardFilterParams{Page: 1, Limit: 20}
// 	wizards, err := db.FilterWizards(*params)
// 	if err != nil {
// 		wizards = &db.WizardFilterResult{HasError: true, Error: "Error al recargar asistentes"}
// 	}
//
// 	comp := templ.Join(
// 		dashboard.WizardsTable(wizards),
// 		components.ToasterToast(toastData),
// 	)
// 	comp.Render(r.Context(), w)
// }
//
// func UpdateWizardAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
// 	w.Header().Add("X-Includes-Toast", "true")
// 	toastData := components.NewToastData("Se actualizó el asistente exitosamente", components.ToastSuccess, 3000, true, false)
//
// 	// Get wizard ID from URL
// 	wizardID := r.PathValue("id")
// 	if wizardID == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 		toastData.Message = "ID de asistente inválido"
// 		toastData.Type = components.ToastError
// 		comp := templ.Join(
// 			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "ID de asistente inválido"}),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		return
// 	}
//
// 	// Parse form data
// 	err := r.ParseForm()
// 	if err != nil {
// 		w.WriteHeader(http.StatusBadRequest)
// 		toastData.Message = "Error al procesar el formulario"
// 		toastData.Type = components.ToastError
// 		comp := templ.Join(
// 			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Error al procesar el formulario"}),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		log.Printf("failed to parse form: %v\n", err)
// 		return
// 	}
//
// 	// Get form values
// 	name := strings.TrimSpace(r.FormValue("name"))
// 	eventKindID := strings.TrimSpace(r.FormValue("event_kind"))
//
// 	// Validate required fields
// 	if name == "" || eventKindID == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 		toastData.Message = "Nombre y tipo de evento son requeridos"
// 		toastData.Type = components.ToastError
// 		comp := templ.Join(
// 			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Nombre y tipo de evento son requeridos"}),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		return
// 	}
//
// 	// Update wizard
// 	wizard := &db.Wizard{
// 		ID:          wizardID,
// 		Name:        name,
// 		EventKindID: eventKindID,
// 	}
//
// 	err = db.UpdateWizard(wizard)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		toastData.Message = "Error al actualizar el asistente"
// 		toastData.Type = components.ToastError
// 		comp := templ.Join(
// 			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Error al actualizar el asistente"}),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		log.Printf("failed to update wizard: %v\n", err)
// 		return
// 	}
//
// 	// Success - return table with toast
// 	params := &db.WizardFilterParams{Page: 1, Limit: 20}
// 	wizards, err := db.FilterWizards(*params)
// 	if err != nil {
// 		wizards = &db.WizardFilterResult{HasError: true, Error: "Error al recargar asistentes"}
// 	}
//
// 	comp := templ.Join(
// 		dashboard.WizardsTable(wizards),
// 		components.ToasterToast(toastData),
// 	)
// 	comp.Render(r.Context(), w)
// }
//
// func DeleteWizardAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
// 	w.Header().Add("X-Includes-Toast", "true")
// 	toastData := components.NewToastData("Se eliminó el asistente exitosamente", components.ToastSuccess, 3000, true, false)
//
// 	// Get wizard ID from URL
// 	wizardID := r.PathValue("id")
// 	if wizardID == "" {
// 		w.WriteHeader(http.StatusBadRequest)
// 		toastData.Message = "ID de asistente inválido"
// 		toastData.Type = components.ToastError
// 		comp := templ.Join(
// 			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "ID de asistente inválido"}),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		return
// 	}
//
// 	// Delete wizard
// 	err := db.DeleteWizard(wizardID)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		toastData.Message = "Error al eliminar el asistente"
// 		toastData.Type = components.ToastError
// 		comp := templ.Join(
// 			dashboard.WizardsTable(&db.WizardFilterResult{HasError: true, Error: "Error al eliminar el asistente"}),
// 			components.ToasterToast(toastData),
// 		)
// 		comp.Render(r.Context(), w)
// 		log.Printf("failed to delete wizard: %v\n", err)
// 		return
// 	}
//
// 	// Success - return table with toast
// 	params := &db.WizardFilterParams{Page: 1, Limit: 20}
// 	wizards, err := db.FilterWizards(*params)
// 	if err != nil {
// 		wizards = &db.WizardFilterResult{HasError: true, Error: "Error al recargar asistentes"}
// 	}
//
// 	comp := templ.Join(
// 		dashboard.WizardsTable(wizards),
// 		components.ToasterToast(toastData),
// 	)
// 	comp.Render(r.Context(), w)
// }
//
// func DeleteWizardsAndReturnTable(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
// 	// TODO: Implement batch delete for wizards
// 	w.WriteHeader(http.StatusNotImplemented)
// 	w.Write([]byte("Batch delete not implemented yet"))
// }
//
// // parseWizardFilterParams parses filter parameters from the request
// func parseWizardFilterParams(r *http.Request) (*db.WizardFilterParams, error) {
// 	params := &db.WizardFilterParams{
// 		Search:     strings.TrimSpace(r.URL.Query().Get("search")),
// 		SearchMode: db.SearchMode(r.URL.Query().Get("search_mode")),
// 		EventKind:  r.URL.Query().Get("event_kind"),
// 		Sort:       r.URL.Query().Get("sort"),
// 	}
//
// 	// Parse page
// 	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
// 		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
// 			params.Page = page
// 		}
// 	}
// 	if params.Page == 0 {
// 		params.Page = 1
// 	}
//
// 	// Parse limit
// 	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
// 		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
// 			params.Limit = limit
// 		}
// 	}
// 	if params.Limit == 0 {
// 		params.Limit = 20
// 	}
//
// 	// Set default search mode
// 	if params.SearchMode == "" {
// 		params.SearchMode = db.SearchModeFullText
// 	}
//
// 	return params, nil
// }
