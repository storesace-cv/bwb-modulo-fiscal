package adminauth

import (
	"os"
	"strings"
)

// AuthModeName is the configured Admin API auth mode (process config only).
type AuthModeName string

const (
	AuthModeFailClosed AuthModeName = "fail_closed"
	AuthModeInjected   AuthModeName = "injected"
	AuthModeOIDCJWT    AuthModeName = "oidc_jwt"
)

// AuthReadiness is a sanitized diagnostics snapshot for /admin/v1/ready (RM-BO-018).
// Never includes issuer URLs with credentials, JWKS bodies, tokens, or subjects.
type AuthReadiness struct {
	Mode              AuthModeName
	OIDCConfigured    bool
	InteractiveLogin  bool // browser IdP redirect / session mint ready
	MissingConfigKeys []string
	Notes             []string
}

// DiagnoseAuthReadiness inspects process env for provider-neutral OIDC readiness.
// Does not contact the IdP. fail_closed remains a valid ready mode (explicit deny).
func DiagnoseAuthReadiness(fiscalEnv string) AuthReadiness {
	modeRaw := strings.TrimSpace(os.Getenv("FISCAL_ADMIN_AUTH_MODE"))
	if modeRaw == "" {
		modeRaw = string(AuthModeFailClosed)
	}
	out := AuthReadiness{Mode: AuthModeName(modeRaw)}
	switch out.Mode {
	case AuthModeFailClosed:
		out.OIDCConfigured = false
		out.InteractiveLogin = false
		out.Notes = []string{
			"fail_closed: Admin API/UI deny until oidc_jwt is configured",
			"no local test IdP; configure a real provider (see admin-idp-onboarding.md)",
		}
		return out
	case AuthModeInjected:
		out.OIDCConfigured = false
		out.InteractiveLogin = false
		if strings.ToLower(strings.TrimSpace(fiscalEnv)) != "development" {
			out.Notes = append(out.Notes, "injected is invalid outside development (startup should fail-closed)")
		} else {
			out.Notes = append(out.Notes, "injected is local-dev only; not interactive IdP login")
		}
		return out
	case AuthModeOIDCJWT:
		required := []string{
			"FISCAL_ADMIN_OIDC_ISSUER",
			"FISCAL_ADMIN_OIDC_AUDIENCE",
			"FISCAL_ADMIN_OIDC_JWKS_URL",
			"FISCAL_ADMIN_OIDC_ROLE_MAP",
		}
		for _, k := range required {
			if strings.TrimSpace(os.Getenv(k)) == "" {
				out.MissingConfigKeys = append(out.MissingConfigKeys, k)
			}
		}
		jwks := strings.TrimSpace(os.Getenv("FISCAL_ADMIN_OIDC_JWKS_URL"))
		if jwks != "" && !strings.HasPrefix(strings.ToLower(jwks), "https://") {
			out.MissingConfigKeys = append(out.MissingConfigKeys, "FISCAL_ADMIN_OIDC_JWKS_URL(https_required)")
		}
		out.OIDCConfigured = len(out.MissingConfigKeys) == 0
		// Interactive browser login requires oidc configured + session mint; redirect authorize still deferred.
		out.InteractiveLogin = false
		out.Notes = []string{
			"oidc_jwt validates Bearer JWT (JWKS); browser redirect authorize not wired yet",
			"session mint: POST /admin/ui/auth/session with Bearer after IdP token issuance",
		}
		if out.OIDCConfigured {
			out.Notes = append(out.Notes, "oidc env present; interactive redirect still unavailable")
		} else {
			out.Notes = append(out.Notes, "oidc env incomplete; remaining fail-closed for JWT auth")
		}
		return out
	default:
		out.OIDCConfigured = false
		out.InteractiveLogin = false
		out.Notes = []string{"unknown FISCAL_ADMIN_AUTH_MODE; treat as fail_closed"}
		return out
	}
}
