package config_test

import (
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/config"
)

func TestLoadDefaults(t *testing.T) {
	// Assumes FISCAL_* are unset in the process environment (CI and local defaults).
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.Version != "0.0.0-dev" {
		t.Fatalf("Version = %q, want 0.0.0-dev", cfg.Version)
	}
	if cfg.FiscalPackage != "AO-UNDECLARED" {
		t.Fatalf("FiscalPackage = %q, want AO-UNDECLARED", cfg.FiscalPackage)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Fatalf("ReadTimeout = %v, want 5s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 10*time.Second {
		t.Fatalf("WriteTimeout = %v, want 10s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 60*time.Second {
		t.Fatalf("IdleTimeout = %v, want 60s", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Fatalf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("FISCAL_HTTP_ADDR", "127.0.0.1:9090")
	t.Setenv("FISCAL_APP_VERSION", "1.2.3")
	t.Setenv("FISCAL_PACKAGE", "AO-TEST-1")
	t.Setenv("FISCAL_HTTP_READ_TIMEOUT", "1500")
	t.Setenv("FISCAL_HTTP_WRITE_TIMEOUT", "2s")
	t.Setenv("FISCAL_HTTP_IDLE_TIMEOUT", "30s")
	t.Setenv("FISCAL_HTTP_SHUTDOWN_TIMEOUT", "2500")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if cfg.HTTPAddr != "127.0.0.1:9090" {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.Version != "1.2.3" {
		t.Fatalf("Version = %q", cfg.Version)
	}
	if cfg.FiscalPackage != "AO-TEST-1" {
		t.Fatalf("FiscalPackage = %q", cfg.FiscalPackage)
	}
	if cfg.ReadTimeout != 1500*time.Millisecond {
		t.Fatalf("ReadTimeout = %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 2*time.Second {
		t.Fatalf("WriteTimeout = %v", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 30*time.Second {
		t.Fatalf("IdleTimeout = %v", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 2500*time.Millisecond {
		t.Fatalf("ShutdownTimeout = %v", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsEmptyVersion(t *testing.T) {
	t.Setenv("FISCAL_APP_VERSION", "   ")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() expected error for empty version")
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	t.Setenv("FISCAL_HTTP_READ_TIMEOUT", "not-a-duration")

	_, err := config.Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid timeout")
	}
}

func TestValidateRejectsZeroTimeout(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:        ":8080",
		Version:         "1.0.0",
		FiscalPackage:   "AO-X",
		ReadTimeout:     0,
		WriteTimeout:    time.Second,
		IdleTimeout:     time.Second,
		ShutdownTimeout: time.Second,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() expected error for zero ReadTimeout")
	}
}
