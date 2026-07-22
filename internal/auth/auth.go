// Package auth autentica identidades POS/módulo (não credenciais AGT).
package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
)

var (
	// ErrUnauthorized indicates missing or invalid credentials.
	ErrUnauthorized = errors.New("auth: unauthorized")
	// ErrForbidden indicates an authenticated principal without authorization.
	ErrForbidden = errors.New("auth: forbidden")
	// ErrInternal indicates authenticator infrastructure failure (DB/timeout); map to HTTP 500.
	ErrInternal = errors.New("auth: internal")
)

// ScopeBinding is the authenticated fiscal scope identity (no token/hash).
type ScopeBinding struct {
	ScopeID             string
	TaxpayerNIF         string
	IANATimezone        string
	SeriesEffectiveCode string
	Environment         string
	CredentialID        string
}

// Authenticator validates POS/module credentials.
type Authenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (ScopeBinding, error)
}

// DevStaticConfig configures development-only static bearer auth.
type DevStaticConfig struct {
	Token               string // required; min 32 bytes; never log
	ScopeID             string // required non-empty
	ForbiddenToken      string // optional; min 32 bytes if set
	TaxpayerNIF         string
	IANATimezone        string
	SeriesEffectiveCode string
	Environment         string // development
	CredentialID        string // synthetic id for binding
}

// DevStatic is an explicit development Authenticator.
type DevStatic struct {
	tokenHash          [sha256.Size]byte
	hasForbidden       bool
	forbiddenTokenHash [sha256.Size]byte
	binding            ScopeBinding
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
	nif := strings.TrimSpace(cfg.TaxpayerNIF)
	if nif == "" {
		return nil, errors.New("auth: dev taxpayer_nif required")
	}
	tz := strings.TrimSpace(cfg.IANATimezone)
	if tz == "" {
		return nil, errors.New("auth: dev iana_timezone required")
	}
	series := strings.TrimSpace(cfg.SeriesEffectiveCode)
	if series == "" {
		return nil, errors.New("auth: dev series_effective_code required")
	}
	env := strings.TrimSpace(cfg.Environment)
	if env == "" {
		env = "development"
	}
	credID := strings.TrimSpace(cfg.CredentialID)
	if credID == "" {
		credID = "dev-static"
	}
	d := &DevStatic{
		tokenHash: sha256.Sum256([]byte(token)),
		binding: ScopeBinding{
			ScopeID:             scope,
			TaxpayerNIF:         nif,
			IANATimezone:        tz,
			SeriesEffectiveCode: series,
			Environment:         env,
			CredentialID:        credID,
		},
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
func (d *DevStatic) Authenticate(ctx context.Context, r *http.Request) (ScopeBinding, error) {
	_ = ctx
	raw := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return ScopeBinding{}, ErrUnauthorized
	}
	got := []byte(strings.TrimSpace(strings.TrimPrefix(raw, prefix)))
	if len(got) == 0 {
		return ScopeBinding{}, ErrUnauthorized
	}
	gotHash := sha256.Sum256(got)
	if d.hasForbidden && subtle.ConstantTimeCompare(gotHash[:], d.forbiddenTokenHash[:]) == 1 {
		return ScopeBinding{}, ErrForbidden
	}
	if subtle.ConstantTimeCompare(gotHash[:], d.tokenHash[:]) != 1 {
		return ScopeBinding{}, ErrUnauthorized
	}
	return d.binding, nil
}

// CredentialVerifier is the persistence surface used by CredentialStoreAuthenticator.
type CredentialVerifier interface {
	VerifyCredentialTokenHash(ctx context.Context, computedHash []byte) (*persistence.CredentialAuthRecord, error)
	RecordAuthAudit(ctx context.Context, ev persistence.AuthAuditEvent) error
}

// CredentialStoreAuthenticator authenticates sandbox Bearer tokens against the credential store.
type CredentialStoreAuthenticator struct {
	store     CredentialVerifier
	fiscalEnv string
	now       func() time.Time
}

// NewCredentialStoreAuthenticator builds a credential_store authenticator.
func NewCredentialStoreAuthenticator(store CredentialVerifier, fiscalEnv string) (*CredentialStoreAuthenticator, error) {
	env := strings.TrimSpace(fiscalEnv)
	if env != "homologation" && env != "development" {
		return nil, errors.New("auth: fiscal env must be homologation or development")
	}
	if store == nil {
		return nil, errors.New("auth: credential store required")
	}
	return &CredentialStoreAuthenticator{
		store:     store,
		fiscalEnv: env,
		now:       func() time.Time { return time.Now().UTC() },
	}, nil
}

// SetClock injects a clock for tests.
func (a *CredentialStoreAuthenticator) SetClock(now func() time.Time) {
	if now == nil {
		a.now = func() time.Time { return time.Now().UTC() }
		return
	}
	a.now = now
}

// Authenticate validates Bearer sandbox tokens.
func (a *CredentialStoreAuthenticator) Authenticate(ctx context.Context, r *http.Request) (ScopeBinding, error) {
	reqID := requestIDFromContext(ctx)
	token, ok := extractBearer(r)
	if !ok || !validSandboxTokenFormat(token) {
		a.auditReject(ctx, reqID, "malformed")
		return ScopeBinding{}, ErrUnauthorized
	}

	computed := persistence.HashCredentialToken(token)
	rec, err := a.store.VerifyCredentialTokenHash(ctx, computed)
	if err != nil {
		if errors.Is(err, persistence.ErrCredentialNotFound) {
			a.auditReject(ctx, reqID, "not_found")
			return ScopeBinding{}, ErrUnauthorized
		}
		return ScopeBinding{}, errors.Join(ErrInternal, err)
	}
	if !a.credentialAcceptable(rec) {
		a.auditReject(ctx, reqID, "policy")
		return ScopeBinding{}, ErrUnauthorized
	}

	_ = a.store.RecordAuthAudit(ctx, persistence.AuthAuditEvent{
		Action:       persistence.AuditActionAuthAccept,
		Result:       "success",
		RequestID:    reqID,
		CredentialID: rec.CredentialID,
		ScopeID:      rec.ScopeID,
	})
	return ScopeBinding{
		ScopeID:             rec.ScopeID,
		TaxpayerNIF:         rec.TaxpayerNIF,
		IANATimezone:        rec.IANATimezone,
		SeriesEffectiveCode: rec.SeriesEffectiveCode,
		Environment:         rec.ScopeEnvironment,
		CredentialID:        rec.CredentialID,
	}, nil
}

func (a *CredentialStoreAuthenticator) credentialAcceptable(rec *persistence.CredentialAuthRecord) bool {
	if rec == nil {
		return false
	}
	if rec.ScopeStatus != "active" {
		return false
	}
	if rec.ScopeEnvironment != a.fiscalEnv {
		return false
	}
	now := a.now().UTC()
	if rec.RevokedAt != nil {
		return false
	}
	expired := rec.ExpiresAt != nil && !rec.ExpiresAt.After(now)
	switch rec.Status {
	case "active":
		return !expired
	case "grace":
		if expired {
			return false
		}
		return rec.GraceUntil != nil && now.Before(*rec.GraceUntil)
	default:
		return false
	}
}

func (a *CredentialStoreAuthenticator) auditReject(ctx context.Context, reqID, reason string) {
	_ = a.store.RecordAuthAudit(ctx, persistence.AuthAuditEvent{
		Action:     persistence.AuditActionAuthReject,
		Result:     "failure",
		ReasonCode: reason,
		RequestID:  reqID,
	})
}

func extractBearer(r *http.Request) (string, bool) {
	raw := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return "", false
	}
	tok := strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	return tok, tok != ""
}

func validSandboxTokenFormat(token string) bool {
	if len(token) != persistence.CredentialTokenExactLen {
		return false
	}
	if !strings.HasPrefix(token, persistence.CredentialTokenPrefix) {
		return false
	}
	// Byte-wise Base64URL alphabet only (A-Z, a-z, 0-9, '-', '_'); no Unicode.
	for i := len(persistence.CredentialTokenPrefix); i < len(token); i++ {
		c := token[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '_':
			continue
		default:
			return false
		}
	}
	return true
}

type ctxKey int

const requestIDCtxKey ctxKey = 1

// ContextWithRequestID stores the server-generated request ID for auth/audit.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDCtxKey, id)
}

func requestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDCtxKey).(string); ok {
		return v
	}
	return ""
}
