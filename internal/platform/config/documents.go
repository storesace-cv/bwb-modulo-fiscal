package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	envEnv              = "FISCAL_ENV"
	envAuthMode         = "FISCAL_AUTH_MODE"
	envAuthDevToken     = "FISCAL_AUTH_DEV_TOKEN"
	envAuthDevForbidden = "FISCAL_AUTH_DEV_FORBIDDEN_TOKEN"
	envAuthDevScope     = "FISCAL_AUTH_DEV_SCOPE_ID"
	envAuthDevNIF       = "FISCAL_AUTH_DEV_TAXPAYER_NIF"
	envScopeTimezone    = "FISCAL_SCOPE_TIMEZONE"
	envSeriesMode       = "FISCAL_SERIES_MODE"
	envSeriesEffective  = "FISCAL_SERIES_EFFECTIVE_CODE"
	envDatabaseDriver   = "FISCAL_DATABASE_DRIVER"
	envDatabaseURL      = "FISCAL_DATABASE_URL"

	authModeDevStatic       = "dev_static"
	authModeCredentialStore = "credential_store"
	seriesModeStatic        = "static"
	envDevelopment          = "development"
	envHomologation         = "homologation"
	minDevTokenBytes        = 32
	scopeTimezoneAngola     = "Africa/Luanda"
)

// DocumentsRuntime is the fail-closed configuration required to serve POST /v1/documents.
type DocumentsRuntime struct {
	Env                   string
	AuthMode              string
	AuthDevToken          string
	AuthDevForbiddenToken string
	AuthDevScopeID        string
	AuthDevTaxpayerNIF    string
	ScopeTimezone         string
	SeriesMode            string
	SeriesEffectiveCode   string
	DatabaseDriver        string
	DatabaseURL           string
}

// DocumentsEnvKeys lists env vars for documents runtime (tests).
func DocumentsEnvKeys() []string {
	return []string{
		envEnv,
		envAuthMode,
		envAuthDevToken,
		envAuthDevForbidden,
		envAuthDevScope,
		envAuthDevNIF,
		envScopeTimezone,
		envSeriesMode,
		envSeriesEffective,
		envDatabaseDriver,
		envDatabaseURL,
	}
}

// LoadDocumentsRuntime loads and validates documents API runtime config.
func LoadDocumentsRuntime() (DocumentsRuntime, error) {
	cfg := DocumentsRuntime{
		Env:                   strings.TrimSpace(os.Getenv(envEnv)),
		AuthMode:              strings.TrimSpace(os.Getenv(envAuthMode)),
		AuthDevToken:          os.Getenv(envAuthDevToken),
		AuthDevForbiddenToken: os.Getenv(envAuthDevForbidden),
		AuthDevScopeID:        strings.TrimSpace(os.Getenv(envAuthDevScope)),
		AuthDevTaxpayerNIF:    strings.TrimSpace(os.Getenv(envAuthDevNIF)),
		ScopeTimezone:         strings.TrimSpace(os.Getenv(envScopeTimezone)),
		SeriesMode:            strings.TrimSpace(os.Getenv(envSeriesMode)),
		SeriesEffectiveCode:   strings.TrimSpace(os.Getenv(envSeriesEffective)),
		DatabaseDriver:        strings.TrimSpace(os.Getenv(envDatabaseDriver)),
		DatabaseURL:           strings.TrimSpace(os.Getenv(envDatabaseURL)),
	}
	if err := cfg.Validate(); err != nil {
		return DocumentsRuntime{}, err
	}
	return cfg, nil
}

// Validate enforces fail-closed documents runtime rules.
func (c DocumentsRuntime) Validate() error {
	if c.AuthMode == "" {
		return fmt.Errorf("%s is required (fail-closed)", envAuthMode)
	}
	switch c.Env {
	case envDevelopment, envHomologation:
	default:
		return fmt.Errorf("%s must be %q or %q", envEnv, envDevelopment, envHomologation)
	}
	switch c.AuthMode {
	case authModeDevStatic:
		return c.validateDevStatic()
	case authModeCredentialStore:
		return c.validateCredentialStore()
	default:
		return fmt.Errorf("%s=%q is not supported; use %q or %q", envAuthMode, c.AuthMode, authModeCredentialStore, authModeDevStatic)
	}
}

func (c DocumentsRuntime) validateDevStatic() error {
	if c.Env != envDevelopment {
		return fmt.Errorf("%s must be %q when %s=%s", envEnv, envDevelopment, envAuthMode, authModeDevStatic)
	}
	if len(c.AuthDevToken) < minDevTokenBytes {
		return fmt.Errorf("%s must be at least %d bytes", envAuthDevToken, minDevTokenBytes)
	}
	if strings.TrimSpace(c.AuthDevScopeID) == "" {
		return fmt.Errorf("%s is required", envAuthDevScope)
	}
	if c.AuthDevTaxpayerNIF == "" {
		return fmt.Errorf("%s is required", envAuthDevNIF)
	}
	if c.AuthDevForbiddenToken != "" && len(c.AuthDevForbiddenToken) < minDevTokenBytes {
		return fmt.Errorf("%s must be at least %d bytes when set", envAuthDevForbidden, minDevTokenBytes)
	}
	if c.ScopeTimezone == "" {
		return fmt.Errorf("%s is required (fail-closed)", envScopeTimezone)
	}
	if c.ScopeTimezone != scopeTimezoneAngola {
		return fmt.Errorf("%s=%q is not supported in this increment; only %q", envScopeTimezone, c.ScopeTimezone, scopeTimezoneAngola)
	}
	if c.SeriesMode == "" {
		return fmt.Errorf("%s is required (fail-closed)", envSeriesMode)
	}
	if c.SeriesMode != seriesModeStatic {
		return fmt.Errorf("%s=%q is not supported; only %q", envSeriesMode, c.SeriesMode, seriesModeStatic)
	}
	if c.SeriesEffectiveCode == "" {
		return fmt.Errorf("%s is required", envSeriesEffective)
	}
	return c.validateDatabase()
}

func (c DocumentsRuntime) validateCredentialStore() error {
	if c.Env == envHomologation && c.AuthMode != authModeCredentialStore {
		return fmt.Errorf("%s=%s requires %s=%s", envEnv, envHomologation, envAuthMode, authModeCredentialStore)
	}
	if c.AuthDevToken != "" {
		return fmt.Errorf("%s must not be set when %s=%s", envAuthDevToken, envAuthMode, authModeCredentialStore)
	}
	if c.AuthDevForbiddenToken != "" {
		return fmt.Errorf("%s must not be set when %s=%s", envAuthDevForbidden, envAuthMode, authModeCredentialStore)
	}
	return c.validateDatabase()
}

func (c DocumentsRuntime) validateDatabase() error {
	switch c.DatabaseDriver {
	case "postgres", "sqlite":
	default:
		return fmt.Errorf("%s must be postgres or sqlite", envDatabaseDriver)
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("%s is required", envDatabaseURL)
	}
	return nil
}

// AuthModeDevStatic is the development static bearer mode.
func AuthModeDevStatic() string { return authModeDevStatic }

// AuthModeCredentialStore is the sandbox credential store mode.
func AuthModeCredentialStore() string { return authModeCredentialStore }

// SeriesModeStatic is the only series mode for dev_static.
func SeriesModeStatic() string { return seriesModeStatic }

// EnvDevelopment is the development environment name.
func EnvDevelopment() string { return envDevelopment }

// EnvHomologation is the homologation environment name.
func EnvHomologation() string { return envHomologation }
