// Package series resolve a SeriesCode efetiva autorizada pelo módulo.
package series

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrUnresolved indicates the requested series reference is not authorized.
	ErrUnresolved = errors.New("series: unresolved")
)

// Resolver maps (scope, requested_series reference) to an authorized effective SeriesCode.
// Implementations must never return requestedSeries as SeriesCode without an explicit mapping policy.
type Resolver interface {
	Resolve(scopeID, requestedSeries string) (seriesCode string, err error)
}

// StaticConfig configures a development static resolver.
type StaticConfig struct {
	EffectiveCode string // required; the only authorized SeriesCode returned
}

// Static always returns EffectiveCode. requestedSeries never becomes SeriesCode.
type Static struct {
	effective string
}

// NewStatic builds a Static resolver.
func NewStatic(cfg StaticConfig) (*Static, error) {
	code := strings.TrimSpace(cfg.EffectiveCode)
	if code == "" {
		return nil, errors.New("series: effective series code required")
	}
	return &Static{effective: code}, nil
}

// Resolve returns the configured effective series. requestedSeries is ignored for the code value
// (it remains a POS reference only; SealInTx stores it separately as requested_series).
func (s *Static) Resolve(scopeID, requestedSeries string) (string, error) {
	if strings.TrimSpace(scopeID) == "" {
		return "", fmt.Errorf("%w: empty scope", ErrUnresolved)
	}
	_ = requestedSeries
	return s.effective, nil
}
