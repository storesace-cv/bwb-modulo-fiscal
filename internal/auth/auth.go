// Package auth autentica identidades POS/módulo (não credenciais AGT).
package auth

import (
	"context"
	"crypto/sha256"
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

// DevStatic is an explicit development Authenticator.
// Tokens are compared via SHA-256 digests with subtle.ConstantTimeCompare (fixed 32-byte width).
type DevStatic struct {
	tokenHash          [sha256.Size]byte
	hasForbidden       bool
	forbiddenTokenHash [sha256.Size]byte
	scopeID            string
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
		tokenHash: sha256.Sum256([]byte(token)),
		scopeID:   scope,
	}
	if cfg.ForbiddenToken != "" {
		if len(cfg.ForbiddenToken) < 32 {
			return nil, errors.New("auth: forbidden token must be at least 32 bytes when set")
		}
		d.hasForbidden = true
		d.forbiddenTokenHash = sha256.Sum256([]byte(cfg.ForbiddenToken))
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
	gotHash := sha256.Sum256(got)
	if d.hasForbidden && subtle.ConstantTimeCompare(gotHash[:], d.forbiddenTokenHash[:]) == 1 {
		return Principal{}, ErrForbidden
	}
	if subtle.ConstantTimeCompare(gotHash[:], d.tokenHash[:]) != 1 {
		return Principal{}, ErrUnauthorized
	}
	return Principal{ScopeID: d.scopeID}, nil
}
