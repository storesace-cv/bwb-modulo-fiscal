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
	envScopeTimezone    = "FISCAL_SCOPE_TIMEZONE"
	envSeriesMode       = "FISCAL_SERIES_MODE"
	envSeriesEffective  = "FISCAL_SERIES_EFFECTIVE_CODE"
	envDatabaseDriver   = "FISCAL_DATABASE_DRIVER"
	envDatabaseURL      = "FISCAL_DATABASE_URL"
	authModeDevStatic   = "dev_static"
	seriesModeStatic    = "static"
	envDevelopment      = "development"
	minDevTokenBytes    = 32
	// scopeTimezoneAngola is the only authorized development timezone in this increment (DEC-TIME-001).
	scopeTimezoneAngola = "Africa/Luanda"
)

// DocumentsRuntime is the fail-closed configuration required to serve POST /v1/documents.
type DocumentsRuntime struct {
	Env                   string
	AuthMode              string
	AuthDevToken          string
	AuthDevForbiddenToken string
	AuthDevScopeID        string
	ScopeTimezone         string // IANA; development: Africa/Luanda
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
		envScopeTimezone,
		envSeriesMode,
		envSeriesEffective,
		envDatabaseDriver,
		envDatabaseURL,
	}
}

// LoadDocumentsRuntime loads and validates documents API runtime config.
// Missing or invalid configuration returns an error (fail-closed; no “disabled” accept mode).
func LoadDocumentsRuntime() (DocumentsRuntime, error) {
	cfg := DocumentsRuntime{
		Env:                   strings.TrimSpace(os.Getenv(envEnv)),
		AuthMode:              strings.TrimSpace(os.Getenv(envAuthMode)),
		AuthDevToken:          os.Getenv(envAuthDevToken),
		AuthDevForbiddenToken: os.Getenv(envAuthDevForbidden),
		AuthDevScopeID:        strings.TrimSpace(os.Getenv(envAuthDevScope)),
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
	if c.AuthMode != authModeDevStatic {
		return fmt.Errorf("%s=%q is not supported; only %q is available in this increment", envAuthMode, c.AuthMode, authModeDevStatic)
	}
	if c.Env != envDevelopment {
		return fmt.Errorf("%s must be %q when %s=%s", envEnv, envDevelopment, envAuthMode, authModeDevStatic)
	}
	if len(c.AuthDevToken) < minDevTokenBytes {
		return fmt.Errorf("%s must be at least %d bytes", envAuthDevToken, minDevTokenBytes)
	}
	if strings.TrimSpace(c.AuthDevScopeID) == "" {
		return fmt.Errorf("%s is required", envAuthDevScope)
	}
	if c.AuthDevForbiddenToken != "" && len(c.AuthDevForbiddenToken) < minDevTokenBytes {
		return fmt.Errorf("%s must be at least %d bytes when set", envAuthDevForbidden, minDevTokenBytes)
	}
	if c.ScopeTimezone == "" {
		return fmt.Errorf("%s is required (fail-closed)", envScopeTimezone)
	}
	if c.ScopeTimezone != scopeTimezoneAngola {
		return fmt.Errorf("%s=%q is not supported in this increment; only %q (Cabo Verde runtime not implemented)", envScopeTimezone, c.ScopeTimezone, scopeTimezoneAngola)
	}
	if c.SeriesMode == "" {
		return fmt.Errorf("%s is required (fail-closed)", envSeriesMode)
	}
	if c.SeriesMode != seriesModeStatic {
		return fmt.Errorf("%s=%q is not supported; only %q is available in this increment", envSeriesMode, c.SeriesMode, seriesModeStatic)
	}
	if c.SeriesEffectiveCode == "" {
		return fmt.Errorf("%s is required", envSeriesEffective)
	}
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

// AuthModeDevStatic is the only auth mode in this increment.
func AuthModeDevStatic() string { return authModeDevStatic }

// SeriesModeStatic is the only series mode in this increment.
func SeriesModeStatic() string { return seriesModeStatic }

// EnvDevelopment is the only environment that may enable dev_static.
func EnvDevelopment() string { return envDevelopment }
