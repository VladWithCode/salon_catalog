package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/routes"
	appsecurity "github.com/vladwithcode/salon_catalog/internal/security"
	"github.com/vladwithcode/salon_catalog/internal/session"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("failed to set enviroment from file\n%v\n", err)
	}

	cartSessions, err := session.NewCartManagerFromEnv()
	if err != nil {
		log.Fatalf("invalid cart cookie configuration: %v\n", err)
	}
	csrfGuard, err := appsecurity.NewCSRFGuardFromEnv()
	if err != nil {
		log.Fatalf("invalid CSRF configuration: %v\n", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("required env var PORT is not set")
	}

	dbPool, err := db.Connect()
	if err != nil {
		log.Fatalf("failed to connect to DB:\n%v\n", err)
	}
	defer dbPool.Close()

	router := routes.NewRouter(cartSessions, csrfGuard)
	fmt.Printf("Starting server on port http://localhost:%s\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), router)
	if err != nil {
		log.Fatalf("failed to listen and serve %v\n", err)
	}
}
