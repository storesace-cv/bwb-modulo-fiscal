// Package config carrega e valida a configuração do processo a partir do ambiente.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultHTTPAddr          = ":8080"
	defaultReadTimeout       = 5 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
	defaultVersion           = "0.0.0-dev"
	defaultFiscalPackage     = "AO-UNDECLARED"
	envHTTPAddr              = "FISCAL_HTTP_ADDR"
	envVersion               = "FISCAL_APP_VERSION"
	envFiscalPackage         = "FISCAL_PACKAGE"
	envReadTimeout           = "FISCAL_HTTP_READ_TIMEOUT"
	envReadHeaderTimeout     = "FISCAL_HTTP_READ_HEADER_TIMEOUT"
	envWriteTimeout          = "FISCAL_HTTP_WRITE_TIMEOUT"
	envIdleTimeout           = "FISCAL_HTTP_IDLE_TIMEOUT"
	envShutdownTimeout       = "FISCAL_HTTP_SHUTDOWN_TIMEOUT"
)

// EnvKeys lista as variáveis de ambiente usadas por Load (útil em testes herméticos).
func EnvKeys() []string {
	return []string{
		envHTTPAddr,
		envVersion,
		envFiscalPackage,
		envReadTimeout,
		envReadHeaderTimeout,
		envWriteTimeout,
		envIdleTimeout,
		envShutdownTimeout,
	}
}

// Config contém parâmetros de arranque do serviço fiscal-api.
type Config struct {
	HTTPAddr          string
	Version           string
	FiscalPackage     string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// Load lê variáveis de ambiente, aplica defaults e valida o resultado.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:          getenv(envHTTPAddr, defaultHTTPAddr),
		Version:           getenv(envVersion, defaultVersion),
		FiscalPackage:     getenv(envFiscalPackage, defaultFiscalPackage),
		ReadTimeout:       defaultReadTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ShutdownTimeout:   defaultShutdownTimeout,
	}

	var err error
	if cfg.ReadTimeout, err = durationFromEnv(envReadTimeout, defaultReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ReadHeaderTimeout, err = durationFromEnv(envReadHeaderTimeout, defaultReadHeaderTimeout); err != nil {
		return Config{}, err
	}
	if cfg.WriteTimeout, err = durationFromEnv(envWriteTimeout, defaultWriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.IdleTimeout, err = durationFromEnv(envIdleTimeout, defaultIdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = durationFromEnv(envShutdownTimeout, defaultShutdownTimeout); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate verifica restrições mínimas de arranque.
func (c Config) Validate() error {
	if strings.TrimSpace(c.HTTPAddr) == "" {
		return fmt.Errorf("%s must not be empty", envHTTPAddr)
	}
	if strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("%s must not be empty", envVersion)
	}
	if strings.TrimSpace(c.FiscalPackage) == "" {
		return fmt.Errorf("%s must not be empty", envFiscalPackage)
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("%s must be > 0", envReadTimeout)
	}
	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("%s must be > 0", envReadHeaderTimeout)
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("%s must be > 0", envWriteTimeout)
	}
	if c.IdleTimeout <= 0 {
		return fmt.Errorf("%s must be > 0", envIdleTimeout)
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("%s must be > 0", envShutdownTimeout)
	}
	return nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if ms <= 0 {
			return 0, fmt.Errorf("%s must be a positive integer (milliseconds)", key)
		}
		return time.Duration(ms) * time.Millisecond, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid duration %q: use milliseconds or Go duration (e.g. 5s)", key, raw)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be > 0", key)
	}
	return d, nil
}
