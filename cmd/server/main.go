package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jdelles/currentz/internal/api"
	"github.com/jdelles/currentz/internal/config"
	"github.com/jdelles/currentz/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	svc, err := service.NewFinanceServiceFromURL(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("service: %v", err)
	}
	defer func() {
		if err := svc.Close(); err != nil {
			log.Printf("service close error: %v", err)
		}
	}()

	api := api.New(svc)
	addr := getenv("HTTP_ADDR", ":8080")

	srv := &http.Server{
		Addr:         addr,
		Handler:      api.Router(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("HTTP listening on %s", addr)
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("shutdown signal received")
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("listen error: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		log.Printf("shutdown error: (%T) %v", err, err)
	}
	log.Println("bye 👋")
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
