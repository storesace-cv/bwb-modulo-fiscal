// Package adminauth implements DEC-BO-002 operator auth: OIDC/JWT RBAC contract
// with an injectable Authenticator. Fail-closed when no authenticator is configured.
// Distinct from POS Bearer credential_store. No improvised local login.
package adminauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Role is an admin RBAC role (DEC-BO-002).
type Role string

const (
	RoleOwner    Role = "owner"
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleAuditor  Role = "auditor"
)

var (
	// ErrUnauthorized means missing/invalid authentication.
	ErrUnauthorized = errors.New("adminauth: não autenticado")
	// ErrForbidden means authenticated but lacking required role.
	ErrForbidden = errors.New("adminauth: proibido")
	// ErrValidation is fail-closed config/input.
	ErrValidation = errors.New("adminauth: validação")
)

type ctxKey int

const claimsKey ctxKey = 1

// Claims are operator identity claims (never secrets).
type Claims struct {
	Subject string
	Roles   []Role
}

// HasRole reports whether claims include role.
func (c Claims) HasRole(role Role) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasAnyRole reports whether any of the roles match.
func (c Claims) HasAnyRole(roles ...Role) bool {
	for _, want := range roles {
		if c.HasRole(want) {
			return true
		}
	}
	return false
}

// Authenticator validates an admin request into Claims (OIDC/JWT in production;
// injectable stub in tests). Must fail closed on error.
type Authenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (Claims, error)
}

// StaticAuthenticator returns fixed claims (tests / explicit injection only).
type StaticAuthenticator struct {
	Claims Claims
}

// Authenticate implements Authenticator.
func (s StaticAuthenticator) Authenticate(_ context.Context, _ *http.Request) (Claims, error) {
	if strings.TrimSpace(s.Claims.Subject) == "" || len(s.Claims.Roles) == 0 {
		return Claims{}, fmt.Errorf("%w: claims estáticas inválidas", ErrValidation)
	}
	for _, role := range s.Claims.Roles {
		if !ValidRole(role) {
			return Claims{}, fmt.Errorf("%w: role inválida %q", ErrValidation, role)
		}
	}
	return s.Claims, nil
}

// FailClosedAuthenticator always rejects (default when IdP not wired).
type FailClosedAuthenticator struct{}

// Authenticate implements Authenticator.
func (FailClosedAuthenticator) Authenticate(context.Context, *http.Request) (Claims, error) {
	return Claims{}, ErrUnauthorized
}

// ValidRole reports whether role is in the initial set.
func ValidRole(role Role) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleOperator, RoleAuditor:
		return true
	default:
		return false
	}
}

// Middleware authenticates and stores Claims in context.
func Middleware(authn Authenticator) func(http.Handler) http.Handler {
	if authn == nil {
		authn = FailClosedAuthenticator{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := authn.Authenticate(r.Context(), r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Unauthorized","status":401,"code":"ADMIN_UNAUTHORIZED","request_id":""}`))
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAnyRole rejects with 403 unless claims have one of the roles.
func RequireAnyRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Unauthorized","status":401,"code":"ADMIN_UNAUTHORIZED","request_id":""}`))
				return
			}
			if !claims.HasAnyRole(roles...) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Forbidden","status":403,"code":"ADMIN_FORBIDDEN","request_id":""}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission rejects with 403 unless Allows(claims, perm).
// Prefer this over RequireAnyRole for Admin API routes (canonical matrix).
func RequirePermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Unauthorized","status":401,"code":"ADMIN_UNAUTHORIZED","request_id":""}`))
				return
			}
			if !Allows(claims, perm) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"type":"about:blank","title":"Forbidden","status":403,"code":"ADMIN_FORBIDDEN","request_id":""}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext returns Claims previously set by Middleware.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}

// ContextWithClaims stores Claims for RequirePermission / handlers (UI HTML auth).
func ContextWithClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ParseRoles parses comma-separated role names.
func ParseRoles(raw string) ([]Role, error) {
	parts := strings.Split(raw, ",")
	out := make([]Role, 0, len(parts))
	seen := map[Role]struct{}{}
	for _, p := range parts {
		r := Role(strings.TrimSpace(p))
		if r == "" {
			continue
		}
		if !ValidRole(r) {
			return nil, fmt.Errorf("%w: role inválida %q", ErrValidation, r)
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: sem roles", ErrValidation)
	}
	return out, nil
}
