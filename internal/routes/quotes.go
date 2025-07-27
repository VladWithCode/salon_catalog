package routes

import (
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/auth"
	"github.com/vladwithcode/salon_catalog/internal/db"
)

func RegisterQuotesRoutes(router *customServeMux) {
	router.HandleFunc("GET /api/quotes", auth.ValidateAuth(GetQuotes))
	router.HandleFunc("GET /api/quotes/{id}", GetQuoteByID)
	router.HandleFunc("POST /api/quotes", CreateQuote)
	router.HandleFunc("PUT /api/quotes/{id}", UpdateQuote)
	router.HandleFunc("DELETE /api/quotes/{id}", auth.ValidateAuth(DeleteQuote))
}

func GetQuotes(w http.ResponseWriter, r *http.Request, a *auth.Auth) {

}

func GetQuoteByID(w http.ResponseWriter, r *http.Request) {

}

func CreateQuote(w http.ResponseWriter, r *http.Request) {
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

func UpdateQuote(w http.ResponseWriter, r *http.Request) {

}

func DeleteQuote(w http.ResponseWriter, r *http.Request, a *auth.Auth) {

}
