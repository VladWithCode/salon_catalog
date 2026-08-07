package routes

import (
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
)

// RegisterQuotesRoutes' POST and PUT handlers were found registered without
// auth.ValidateAuth — an unauthenticated caller could hit
// db.CreateQuote/db.UpdateQuote directly. No consumer (Go template, HTMX
// route, or Next code) references /api/quotes anywhere in this repo, so
// gating them costs nothing and closes an unauthenticated write path the
// same way categories.go's equivalent leak was closed.
func RegisterQuotesRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/quotes", auth.ValidateAuth(GetQuotes))
	router.HandleFunc("GET /api/quotes/{id}", GetQuoteByID)
	router.HandleFunc("POST /api/quotes", auth.ValidateAuth(CreateQuote))
	router.HandleFunc("PUT /api/quotes/{id}", auth.ValidateAuth(UpdateQuote))
	router.HandleFunc("DELETE /api/quotes/{id}", auth.ValidateAuth(DeleteQuote))
}

func GetQuotes(w http.ResponseWriter, r *http.Request, a *auth.Auth) {

}

func GetQuoteByID(w http.ResponseWriter, r *http.Request) {

}

func CreateQuote(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	var data db.Quote
	err := r.ParseForm()
	if err != nil {
		// TODO: respond with error template
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}

	err = db.CreateQuote(&data)
	if err != nil {
		// TODO: respond with error template
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create quote"))
		log.Printf("failed to create quote: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func UpdateQuote(w http.ResponseWriter, r *http.Request, a *auth.Auth) {

}

func DeleteQuote(w http.ResponseWriter, r *http.Request, a *auth.Auth) {

}
