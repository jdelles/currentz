package main

import (
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

type Config struct {
	Addr string `env:"APP_ADDR" envDefault:":8080"`
	Env  string `env:"APP_ENV"  envDefault:"dev"`
}

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Logger, chimw.Recoverer, chimw.Timeout(60*time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	srv := &http.Server{Addr: cfg.Addr, Handler: r}
	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())
}
