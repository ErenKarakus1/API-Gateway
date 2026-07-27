package main

import (
	"log"
	"os"

	"github.com/ErenKarakus1/API-Gateway/internal/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := server.New()

	if err := app.Run(":" + port); err != nil {
		log.Fatalf("failed to start gateway: %v", err)
	}
}
