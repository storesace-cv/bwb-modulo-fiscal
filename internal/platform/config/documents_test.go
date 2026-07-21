package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/config"
)

func clearDocumentsEnv(t *testing.T) {
	t.Helper()
	for _, key := range config.DocumentsEnvKeys() {
		prev, existed := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, prev)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}

func validDocumentsEnv(t *testing.T) {
	t.Helper()
	t.Setenv("FISCAL_ENV", "development")
	t.Setenv("FISCAL_AUTH_MODE", "dev_static")
	t.Setenv("FISCAL_AUTH_DEV_TOKEN", strings.Repeat("a", 32))
	t.Setenv("FISCAL_AUTH_DEV_SCOPE_ID", "scope-dev")
	t.Setenv("FISCAL_SERIES_MODE", "static")
	t.Setenv("FISCAL_SERIES_EFFECTIVE_CODE", "A")
	t.Setenv("FISCAL_DATABASE_DRIVER", "sqlite")
	t.Setenv("FISCAL_DATABASE_URL", "./tmp/fiscal.db")
}

func TestLoadDocumentsRuntimeOK(t *testing.T) {
	clearDocumentsEnv(t)
	validDocumentsEnv(t)
	cfg, err := config.LoadDocumentsRuntime()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SeriesEffectiveCode != "A" {
		t.Fatalf("%q", cfg.SeriesEffectiveCode)
	}
}

func TestLoadDocumentsRuntimeFailClosedMissingAuth(t *testing.T) {
	clearDocumentsEnv(t)
	validDocumentsEnv(t)
	t.Setenv("FISCAL_AUTH_MODE", "")
	_ = os.Unsetenv("FISCAL_AUTH_MODE")
	_, err := config.LoadDocumentsRuntime()
	if err == nil {
		t.Fatal("expected fail-closed")
	}
}

func TestLoadDocumentsRuntimeRejectsDevStaticOutsideDevelopment(t *testing.T) {
	clearDocumentsEnv(t)
	validDocumentsEnv(t)
	t.Setenv("FISCAL_ENV", "production")
	_, err := config.LoadDocumentsRuntime()
	if err == nil {
		t.Fatal("expected rejection")
	}
}

func TestLoadDocumentsRuntimeRejectsShortToken(t *testing.T) {
	clearDocumentsEnv(t)
	validDocumentsEnv(t)
	t.Setenv("FISCAL_AUTH_DEV_TOKEN", "short")
	_, err := config.LoadDocumentsRuntime()
	if err == nil {
		t.Fatal("expected rejection")
	}
}
