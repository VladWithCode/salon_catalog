package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/vladwithcode/salon_catalog/internal/db"
	"github.com/vladwithcode/salon_catalog/internal/routes"
)

func main() {
	err := godotenv.Overload()
	if err != nil {
		log.Printf("failed to set enviroment from file\n%v\n", err)
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

	router := routes.NewRouter()
	fmt.Printf("Starting server on port :%s\n", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), router)
	if err != nil {
		log.Fatalf("failed to listen and serve %v\n", err)
	}
}
