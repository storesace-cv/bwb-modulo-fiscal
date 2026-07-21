// Package fiscaltz resolve a timezone fiscal IANA autorizada por scope (DEC-TIME-001).
package fiscaltz

import (
	"errors"
	"fmt"
	"strings"
	"time"

	_ "time/tzdata"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaltime"
)

var (
	// ErrUnresolved means the scope has no authorized fiscal timezone (fail-closed).
	ErrUnresolved = errors.New("fiscaltz: unresolved")
)

// Resolver maps scope_id → IANA timezone.
type Resolver interface {
	Resolve(scopeID string) (ianaTimezone string, err error)
}

// StaticConfig configures a development-only static resolver.
type StaticConfig struct {
	ScopeID  string
	Timezone string // e.g. Africa/Luanda
}

// Static resolves a single configured scope to a fixed IANA zone.
type Static struct {
	scopeID  string
	timezone string
}

// NewStatic builds a fail-closed static resolver for development.
func NewStatic(cfg StaticConfig) (*Static, error) {
	scope := strings.TrimSpace(cfg.ScopeID)
	tz := strings.TrimSpace(cfg.Timezone)
	if scope == "" {
		return nil, errors.New("fiscaltz: scope_id required")
	}
	if tz == "" {
		return nil, errors.New("fiscaltz: timezone required")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return nil, fmt.Errorf("fiscaltz: invalid timezone %q: %w", tz, err)
	}
	return &Static{scopeID: scope, timezone: tz}, nil
}

// NewStaticAfricaLuanda is the Angola development helper.
func NewStaticAfricaLuanda(scopeID string) (*Static, error) {
	return NewStatic(StaticConfig{ScopeID: scopeID, Timezone: fiscaltime.AfricaLuanda})
}

// Resolve returns the IANA timezone for scopeID or ErrUnresolved.
func (s *Static) Resolve(scopeID string) (string, error) {
	if strings.TrimSpace(scopeID) != s.scopeID {
		return "", fmt.Errorf("%w: scope %q", ErrUnresolved, scopeID)
	}
	return s.timezone, nil
}
