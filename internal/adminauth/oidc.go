package adminauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OIDCAuthenticator validates Bearer JWTs against a configured JWKS (provider-neutral).
// Never logs tokens or secret material.
type OIDCAuthenticator struct {
	cfg     OIDCConfig
	jwks    *jwksCache
	allowed map[string]struct{}
	owners  map[string]struct{}
}

// NewOIDCAuthenticator builds a fail-closed JWT authenticator.
func NewOIDCAuthenticator(cfg OIDCConfig) (*OIDCAuthenticator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg = cfg.withDefaults()
	allowed := make(map[string]struct{}, len(cfg.AllowedAlgs))
	for _, a := range cfg.AllowedAlgs {
		allowed[a] = struct{}{}
	}
	owners := make(map[string]struct{}, len(cfg.OwnerSubjects))
	for _, s := range cfg.OwnerSubjects {
		owners[strings.TrimSpace(s)] = struct{}{}
	}
	return &OIDCAuthenticator{
		cfg:     cfg,
		jwks:    newJWKSCache(cfg.JWKSURL, cfg.HTTPClient, cfg.JWKSMaxBytes, cfg.JWKSCacheTTL, cfg.JWKSMinRefresh),
		allowed: allowed,
		owners:  owners,
	}, nil
}

// Authenticate implements Authenticator.
func (a *OIDCAuthenticator) Authenticate(ctx context.Context, r *http.Request) (Claims, error) {
	if a == nil {
		return Claims{}, ErrUnauthorized
	}
	raw, err := bearerToken(r, a.cfg.MaxTokenBytes)
	if err != nil {
		return Claims{}, err
	}
	kid, err := peekKid(raw)
	if err != nil {
		return Claims{}, err
	}
	key, err := a.jwks.get(ctx, kid)
	if err != nil {
		return Claims{}, err
	}
	claimsMap, err := verifyAndParseClaims(raw, key, a.allowed, a.cfg.Clock(), a.cfg.Issuer, a.cfg.Audience)
	if err != nil {
		return Claims{}, err
	}
	sub, _ := claimsMap["sub"].(string)
	sub = strings.TrimSpace(sub)
	roles, err := a.mapRoles(claimsMap, sub)
	if err != nil {
		return Claims{}, err
	}
	return Claims{Subject: sub, Roles: roles}, nil
}

func bearerToken(r *http.Request, maxBytes int) (string, error) {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if h == "" {
		return "", ErrUnauthorized
	}
	const prefix = "Bearer "
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", ErrUnauthorized
	}
	tok := strings.TrimSpace(h[len(prefix):])
	if tok == "" || len(tok) > maxBytes {
		return "", ErrUnauthorized
	}
	// Reject embedded whitespace / newlines (header smuggling oddities).
	if strings.ContainsAny(tok, " \t\r\n") {
		return "", ErrUnauthorized
	}
	return tok, nil
}

func peekKid(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("%w: jwt partes", ErrUnauthorized)
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("%w: jwt header", ErrUnauthorized)
	}
	var hdr jwtHeader
	if err := json.Unmarshal(hb, &hdr); err != nil {
		return "", fmt.Errorf("%w: jwt header json", ErrUnauthorized)
	}
	kid := strings.TrimSpace(hdr.Kid)
	if kid == "" {
		return "", fmt.Errorf("%w: kid obrigatório", ErrUnauthorized)
	}
	if strings.EqualFold(strings.TrimSpace(hdr.Alg), "none") {
		return "", fmt.Errorf("%w: alg proibido", ErrUnauthorized)
	}
	return kid, nil
}

func (a *OIDCAuthenticator) mapRoles(claims map[string]any, subject string) ([]Role, error) {
	values := claimStringList(claims[a.cfg.RoleClaim])
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: sem roles/grupos", ErrUnauthorized)
	}
	seen := map[Role]struct{}{}
	out := make([]Role, 0, len(values))
	for _, v := range values {
		role, ok := a.cfg.RoleMap[v]
		if !ok {
			continue // unmapped → ignore (no implicit elevation)
		}
		if role == RoleOwner {
			if _, allowed := a.owners[subject]; !allowed {
				continue // owner requires explicit subject allowlist
			}
		}
		if _, dup := seen[role]; dup {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: sem roles mapeadas", ErrUnauthorized)
	}
	return out, nil
}

func claimStringList(raw any) []string {
	switch v := raw.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		// Support space-separated single claim.
		if strings.Contains(v, " ") {
			return ParseCSVList(strings.ReplaceAll(v, " ", ","))
		}
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
