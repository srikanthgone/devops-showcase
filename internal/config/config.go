// Package config loads runtime configuration from environment variables.
// Twelve-factor style: everything is configurable via the environment so the
// same image runs unchanged across local, staging and production.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all tunable runtime settings.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string
	// LogLevel controls slog verbosity: debug, info, warn, error.
	LogLevel string
	// ReadTimeout / WriteTimeout / IdleTimeout guard the HTTP server against
	// slow-client (Slowloris) style resource exhaustion.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// ShutdownTimeout bounds how long we wait for in-flight requests to drain
	// during graceful shutdown.
	ShutdownTimeout time.Duration
	// Environment is a free-form label (dev/staging/prod) surfaced in metrics
	// and the root endpoint.
	Environment string
}

// Load reads configuration from the environment, applying sane defaults.
func Load() (Config, error) {
	cfg := Config{
		Port:            getEnv("PORT", "8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Environment:     getEnv("APP_ENV", "development"),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getEnvDuration("IDLE_TIMEOUT", 120*time.Second),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}

	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("invalid PORT %q: %w", cfg.Port, err)
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
