package main

import (
	"log"
	"os"

	"github.com/ErenKarakus1/API-Gateway/internal/config"
	"github.com/ErenKarakus1/API-Gateway/internal/server"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app, err := server.New(cfg)
	if err != nil {
		log.Fatalf("failed to build server: %v", err)
	}

	if err := app.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("failed to start gateway: %v", err)
	}
}
