package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

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

	// Handle localStorage selections if provided
	selectionsJSON := r.FormValue("wizard_localStorage")
	var savedSelections map[string][]string
	if selectionsJSON != "" {
		if err := json.Unmarshal([]byte(selectionsJSON), &savedSelections); err == nil {
			state.Selections = savedSelections
		}
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

func CompleteWizardAndAddToCart(w http.ResponseWriter, r *http.Request) {
	wizardID := r.PathValue("wizard_id")

	// Parse localStorage selections from request
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		log.Printf("failed to parse form: %v", err)
		return
	}

	selectionsJSON := r.FormValue("wizard_localStorage")
	var selections map[string][]string
	if selectionsJSON != "" {
		if err := json.Unmarshal([]byte(selectionsJSON), &selections); err != nil {
			http.Error(w, "Invalid selections data", http.StatusBadRequest)
			log.Printf("failed to unmarshal selections: %v", err)
			return
		}
	}

	// Get all product IDs from all steps
	var allProductIDs []string
	for _, productIDs := range selections {
		allProductIDs = append(allProductIDs, productIDs...)
	}

	// Add all products to cart
	cartID, err := db.GetCartIDFromRequest(r)
	if err != nil {
		http.Error(w, "Cart error", http.StatusBadRequest)
		log.Printf("failed to get cart ID: %v", err)
		return
	}

	cart, err := db.GetOrCreateCart(r.Context(), cartID)
	if err != nil {
		http.Error(w, "Cart error", http.StatusInternalServerError)
		log.Printf("failed to get or create cart: %v", err)
		return
	}

	// Add each product to cart
	addedCount := 0
	for _, productID := range allProductIDs {
		// Check if already exists, if not add new item
		existingItem := false
		for _, item := range cart.Items {
			if item.ProductID == productID {
				// Update quantity of existing item
				newQty := item.Quantity + 1
				cart.UpdateItemQty(productID, newQty)
				existingItem = true
				addedCount++
				break
			}
		}

		if !existingItem {
			prod, err := db.FindCatalogProductDetail(productID)
			if err != nil {
				log.Printf("failed to find product %s: %v", productID, err)
				continue // Skip invalid products
			}

			cartItem := &db.CartItem{
				ProductID: productID,
				Source:    "wizard",
				Name:      prod.Name,
				Category:  prod.CategoryName,
				ImageURL:  prod.ImageURL,
				Quantity:  1,
				MaxQty:    prod.Quantity,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			cart.AddItem(cartItem)
			addedCount++
		}
	}

	// Save cart
	err = cart.Save(r.Context())
	if err != nil {
		http.Error(w, "Error saving cart", http.StatusInternalServerError)
		log.Printf("failed to save cart: %v", err)
		return
	}

	// Get wizard details for completion view
	wizard, err := db.FindWizard(r.Context(), wizardID)
	if err != nil {
		log.Printf("failed to find wizard: %v", err)
		wizard = &db.Wizard{ID: wizardID, Name: "Asistente"}
	}

	// Return completion view
	state := components.WizardModalState{
		CurrentView:   "completion",
		CurrentWizard: wizard,
		Selections:    selections,
	}

	err = components.WizardModal(&state).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering completion view", http.StatusInternalServerError)
		log.Printf("failed to render completion view: %v", err)
		return
	}
}
