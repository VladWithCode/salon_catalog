package routes

import (
	"log"
	"net/http"

	"github.com/vladwithcode/salon_catalog/internal/auth"
)

func RegisterUserRoutes(router *customServeMux) {
	router.HandleFunc("PUT /panel/usuario", auth.ValidateAuth(UpdateUser))
}

func UpdateUser(w http.ResponseWriter, r *http.Request, a *auth.Auth) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to parse request body"))
		log.Printf("failed to parse request body: %v\n", err)
		return
	}
}
