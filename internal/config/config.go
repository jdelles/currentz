package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
}

func Load() (*Config, error) {
	dbURL := strings.TrimSpace(os.Getenv("DB_URL"))
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL not set. Run `make dev-setup` or create .env from .env.example")
	}
	return &Config{DatabaseURL: dbURL}, nil
}
