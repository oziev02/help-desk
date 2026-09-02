package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
	SeedDemo    bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://helpdesk:helpdesk@localhost:5433/helpdesk?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
		JWTExpiry:   24 * time.Hour,
		SeedDemo:    getEnv("SEED_DEMO", "true") == "true",
	}

	if cfg.JWTSecret == "dev-secret-change-me" {
		// acceptable for local dev only
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
