package main

import (
	"log"
	"os"

	"github.com/jdelles/currentz/internal/app"
	"github.com/jdelles/currentz/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	financeApp, err := app.NewFinanceApp(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}
	defer func() {
		if err := financeApp.Close(); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	if err := financeApp.Run(); err != nil {
		log.Fatalf("Application error: %v", err)
		os.Exit(1)
	}
}
