// Package auth autentica identidades POS/módulo (não credenciais AGT).
package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
)

var (
	// ErrUnauthorized indicates missing or invalid credentials.
	ErrUnauthorized = errors.New("auth: unauthorized")
	// ErrForbidden indicates an authenticated principal without authorization.
	ErrForbidden = errors.New("auth: forbidden")
)

// Principal is the authenticated caller identity.
type Principal struct {
	ScopeID string
}

// Authenticator validates POS/module credentials.
type Authenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (Principal, error)
}

// DevStaticConfig configures development-only static bearer auth.
type DevStaticConfig struct {
	Token          string // required; min 32 bytes; never log
	ScopeID        string // required non-empty
	ForbiddenToken string // optional; min 32 bytes if set; authenticates but forbids
}

// DevStatic is an explicit development Authenticator (constant-time token compare).
type DevStatic struct {
	token          []byte
	forbiddenToken []byte
	scopeID        string
}

// NewDevStatic builds a DevStatic authenticator. Caller must enforce FISCAL_ENV=development.
func NewDevStatic(cfg DevStaticConfig) (*DevStatic, error) {
	token := cfg.Token
	if len(token) < 32 {
		return nil, errors.New("auth: dev token must be at least 32 bytes")
	}
	scope := strings.TrimSpace(cfg.ScopeID)
	if scope == "" {
		return nil, errors.New("auth: dev scope_id required")
	}
	d := &DevStatic{
		token:   []byte(token),
		scopeID: scope,
	}
	if cfg.ForbiddenToken != "" {
		if len(cfg.ForbiddenToken) < 32 {
			return nil, errors.New("auth: forbidden token must be at least 32 bytes when set")
		}
		d.forbiddenToken = []byte(cfg.ForbiddenToken)
	}
	return d, nil
}

// Authenticate validates the Authorization Bearer token.
func (d *DevStatic) Authenticate(ctx context.Context, r *http.Request) (Principal, error) {
	_ = ctx
	raw := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return Principal{}, ErrUnauthorized
	}
	got := []byte(strings.TrimSpace(strings.TrimPrefix(raw, prefix)))
	if len(got) == 0 {
		return Principal{}, ErrUnauthorized
	}
	if len(d.forbiddenToken) > 0 && constantTimeEqual(got, d.forbiddenToken) {
		return Principal{}, ErrForbidden
	}
	if !constantTimeEqual(got, d.token) {
		return Principal{}, ErrUnauthorized
	}
	return Principal{ScopeID: d.scopeID}, nil
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		// Compare against itself to keep timing closer when lengths differ.
		subtle.ConstantTimeCompare(a, a)
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
