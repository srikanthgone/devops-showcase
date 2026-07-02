package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure a clean environment so defaults apply.
	for _, k := range []string{"PORT", "LOG_LEVEL", "APP_ENV", "READ_TIMEOUT", "SHUTDOWN_TIMEOUT"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %q", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default log level info, got %q", cfg.LogLevel)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected default read timeout 5s, got %s", cfg.ReadTimeout)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("expected default shutdown timeout 15s, got %s", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-number")
	if _, err := Load(); err == nil {
		t.Fatal("expected an error for a non-numeric PORT, got nil")
	}
}

func TestLoadHonoursOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("APP_ENV", "staging")
	t.Setenv("SHUTDOWN_TIMEOUT", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %q", cfg.Port)
	}
	if cfg.Environment != "staging" {
		t.Errorf("expected env staging, got %q", cfg.Environment)
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("expected shutdown timeout 30s, got %s", cfg.ShutdownTimeout)
	}
}

func TestGetEnvDurationFallsBackOnGarbage(t *testing.T) {
	t.Setenv("READ_TIMEOUT", "definitely-not-a-duration")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("expected fallback read timeout 5s on bad input, got %s", cfg.ReadTimeout)
	}
}
