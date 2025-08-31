package config

import (
	"fmt"
	"os"
	"os/user"
)

type Config struct {
	DatabaseURL string
	Host        string
	Port        string
	User        string
	Password    string
	DBName      string
	SSLMode     string
}

func Load() (*Config, error) {
	currentUser, err := user.Current()
	defaultUser := "postgres"
	if err == nil {
		defaultUser = currentUser.Username
	}

	cfg := &Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", defaultUser),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", "personal_finance"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	// Build connection string
	if cfg.Password != "" {
		cfg.DatabaseURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)
	} else {
		cfg.DatabaseURL = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.DBName, cfg.SSLMode)
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
