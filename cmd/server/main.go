package main

import (
	"context"
	"log"
	"os"

	"github.com/jdelles/currentz/internal/api"
	"github.com/jdelles/currentz/internal/service"
)

func main() {
	// Get database URL from environment variable or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://user:password@localhost/dbname?sslmode=disable"
		log.Println("DATABASE_URL not set, using default:", dbURL)
	}

	// Get server port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	// Create finance service
	financeService, err := service.NewFinanceServiceFromURL(ctx, dbURL)
	if err != nil {
		log.Fatal("Failed to create finance service:", err)
	}
	defer func() {
		if err := financeService.Close(); err != nil {
			// at least log it, or handle gracefully
			log.Printf("error closing financeService: %v", err)
		}
	}()

	// Create API server
	server := api.NewAPIServer(financeService)

	// Start server
	log.Printf("Starting server on port %s", port)
	if err := server.Start(":" + port); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
