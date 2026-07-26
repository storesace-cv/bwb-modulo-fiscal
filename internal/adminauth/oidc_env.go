package adminauth

import (
	"fmt"
	"os"
	"strings"
)

// OIDCConfigFromEnv loads OIDC/JWT settings from FISCAL_ADMIN_OIDC_* (fail-closed).
// Does not enable HTTP JWKS — that remains test-only via AllowHTTPJWKS.
func OIDCConfigFromEnv() (OIDCConfig, error) {
	roleMap, err := ParseRoleMap(os.Getenv("FISCAL_ADMIN_OIDC_ROLE_MAP"))
	if err != nil {
		return OIDCConfig{}, fmt.Errorf("FISCAL_ADMIN_OIDC_ROLE_MAP: %w", err)
	}
	algs := ParseCSVList(os.Getenv("FISCAL_ADMIN_OIDC_ALGS"))
	cfg := OIDCConfig{
		Issuer:        strings.TrimSpace(os.Getenv("FISCAL_ADMIN_OIDC_ISSUER")),
		Audience:      strings.TrimSpace(os.Getenv("FISCAL_ADMIN_OIDC_AUDIENCE")),
		JWKSURL:       strings.TrimSpace(os.Getenv("FISCAL_ADMIN_OIDC_JWKS_URL")),
		AllowedAlgs:   algs,
		RoleClaim:     strings.TrimSpace(os.Getenv("FISCAL_ADMIN_OIDC_ROLE_CLAIM")),
		RoleMap:       roleMap,
		OwnerSubjects: ParseCSVList(os.Getenv("FISCAL_ADMIN_OIDC_OWNER_SUBJECTS")),
	}
	if cfg.RoleClaim == "" {
		cfg.RoleClaim = "groups"
	}
	if err := cfg.Validate(); err != nil {
		return OIDCConfig{}, err
	}
	return cfg, nil
}
