package routes

import (
	"net/http"
)

func RegisterWizardRoutes(router *customServeMux) {
	router.HandleFunc("GET /wizard/categories", GetWizardCategories)
}

func GetWizardCategories(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("HX-Trigger", "load")
	w.Header().Add("HX-Target", "wizard-modal")
	w.Header().Add("HX-Swap", "innerHTML")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
