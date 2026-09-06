package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

const defaultDevJWTSecret = "dev-secret-change-me"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	JWTSecret   string
	JWTExpiry   time.Duration
	SeedDemo    bool
	SLAHours    int
	AppEnv      string
}

func Load() (Config, error) {
	slaHours, err := strconv.Atoi(getEnv("SLA_HOURS", "48"))
	if err != nil || slaHours <= 0 {
		return Config{}, errors.New("SLA_HOURS must be a positive integer")
	}

	cfg := Config{
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://helpdesk:helpdesk@localhost:5433/helpdesk?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", defaultDevJWTSecret),
		JWTExpiry:   24 * time.Hour,
		SeedDemo:    getEnv("SEED_DEMO", "false") == "true",
		SLAHours:    slaHours,
		AppEnv:      getEnv("APP_ENV", "dev"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.AppEnv != "dev" && (cfg.JWTSecret == "" || cfg.JWTSecret == defaultDevJWTSecret) {
		return Config{}, errors.New("JWT_SECRET must be set to a non-default value when APP_ENV is not dev")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
