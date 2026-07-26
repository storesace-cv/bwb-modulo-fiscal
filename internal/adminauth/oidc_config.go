package adminauth

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultMaxTokenBytes = 8192
	defaultJWKSTimeout   = 5 * time.Second
	defaultJWKSCacheTTL  = 10 * time.Minute
	defaultJWKSMaxBytes  = 1 << 20 // 1 MiB
	defaultMinRSABits    = 2048
)

// OIDCConfig is provider-neutral JWT validation config (DEC-BO-002 / RM-BO-006).
// Fail-closed: Validate rejects missing/insecure settings before any token is accepted.
type OIDCConfig struct {
	Issuer   string // exact iss match (required)
	Audience string // exact aud match (required; string or array containing this value)
	JWKSURL  string // required; https in production paths

	// AllowedAlgs explicit allowlist. Empty → default RS256 only. Never includes "none".
	AllowedAlgs []string

	// RoleClaim is the JWT claim holding group/role strings (e.g. "groups", "roles").
	RoleClaim string
	// RoleMap maps claim values → RBAC roles. No implicit elevation: unmapped values ignored.
	RoleMap map[string]Role
	// OwnerSubjects allowlists subjects that may receive RoleOwner when mapped.
	// Required whenever RoleMap contains RoleOwner.
	OwnerSubjects []string

	Clock          func() time.Time
	HTTPClient     *http.Client
	MaxTokenBytes  int
	JWKSCacheTTL   time.Duration
	JWKSMaxBytes   int
	JWKSMinRefresh time.Duration // min interval between forced refreshes on unknown kid

	// AllowHTTPJWKS permits http:// JWKS (tests only). Production wiring must leave false.
	AllowHTTPJWKS bool
}

// Validate fails closed on insecure or incomplete config.
func (c OIDCConfig) Validate() error {
	if strings.TrimSpace(c.Issuer) == "" {
		return fmt.Errorf("%w: issuer obrigatório", ErrValidation)
	}
	if strings.TrimSpace(c.Audience) == "" {
		return fmt.Errorf("%w: audience obrigatório", ErrValidation)
	}
	jwks := strings.TrimSpace(c.JWKSURL)
	if jwks == "" {
		return fmt.Errorf("%w: JWKS URL obrigatória", ErrValidation)
	}
	u, err := url.Parse(jwks)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: JWKS URL inválida", ErrValidation)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		// ok
	case "http":
		if !c.AllowHTTPJWKS {
			return fmt.Errorf("%w: JWKS deve ser https", ErrValidation)
		}
	default:
		return fmt.Errorf("%w: esquema JWKS não suportado", ErrValidation)
	}

	algs := c.AllowedAlgs
	if len(algs) == 0 {
		algs = []string{"RS256"}
	}
	for _, a := range algs {
		a = strings.TrimSpace(a)
		if a == "" || strings.EqualFold(a, "none") {
			return fmt.Errorf("%w: algoritmo proibido", ErrValidation)
		}
		switch a {
		case "RS256", "ES256":
			// supported
		default:
			return fmt.Errorf("%w: algoritmo não suportado %q", ErrValidation, a)
		}
	}

	claim := strings.TrimSpace(c.RoleClaim)
	if claim == "" {
		return fmt.Errorf("%w: role claim obrigatório", ErrValidation)
	}
	if len(c.RoleMap) == 0 {
		return fmt.Errorf("%w: role map obrigatório (sem elevação implícita)", ErrValidation)
	}
	ownerMapped := false
	for k, role := range c.RoleMap {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("%w: chave de role map vazia", ErrValidation)
		}
		if !ValidRole(role) {
			return fmt.Errorf("%w: role map inválida %q", ErrValidation, role)
		}
		if role == RoleOwner {
			ownerMapped = true
		}
	}
	if ownerMapped {
		if len(c.OwnerSubjects) == 0 {
			return fmt.Errorf("%w: OwnerSubjects obrigatório quando owner está no RoleMap", ErrValidation)
		}
		for _, s := range c.OwnerSubjects {
			if strings.TrimSpace(s) == "" {
				return fmt.Errorf("%w: OwnerSubjects contém vazio", ErrValidation)
			}
		}
	}
	return nil
}

func (c OIDCConfig) withDefaults() OIDCConfig {
	out := c
	if out.Clock == nil {
		out.Clock = func() time.Time { return time.Now().UTC() }
	}
	if out.MaxTokenBytes <= 0 {
		out.MaxTokenBytes = defaultMaxTokenBytes
	}
	if out.JWKSCacheTTL <= 0 {
		out.JWKSCacheTTL = defaultJWKSCacheTTL
	}
	if out.JWKSMaxBytes <= 0 {
		out.JWKSMaxBytes = defaultJWKSMaxBytes
	}
	if out.JWKSMinRefresh <= 0 {
		out.JWKSMinRefresh = 30 * time.Second
	}
	if len(out.AllowedAlgs) == 0 {
		out.AllowedAlgs = []string{"RS256"}
	}
	if strings.TrimSpace(out.RoleClaim) == "" {
		out.RoleClaim = "groups"
	}
	if out.HTTPClient == nil {
		out.HTTPClient = &http.Client{Timeout: defaultJWKSTimeout}
	} else if out.HTTPClient.Timeout == 0 {
		clone := *out.HTTPClient
		clone.Timeout = defaultJWKSTimeout
		out.HTTPClient = &clone
	}
	return out
}

// ParseRoleMap parses "idp-value:role,other:admin" (fail-closed on bad entries).
func ParseRoleMap(raw string) (map[string]Role, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%w: role map vazio", ErrValidation)
	}
	out := make(map[string]Role)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, ":")
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if !ok || k == "" || v == "" {
			return nil, fmt.Errorf("%w: entrada role map inválida", ErrValidation)
		}
		role := Role(v)
		if !ValidRole(role) {
			return nil, fmt.Errorf("%w: role inválida %q", ErrValidation, v)
		}
		out[k] = role
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: role map vazio", ErrValidation)
	}
	return out, nil
}

// ParseCSVList splits comma-separated non-empty trimmed values.
func ParseCSVList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}
