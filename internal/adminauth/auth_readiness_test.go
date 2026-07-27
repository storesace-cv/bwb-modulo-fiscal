package adminauth_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

func TestDiagnoseAuthReadinessFailClosed(t *testing.T) {
	t.Setenv("FISCAL_ADMIN_AUTH_MODE", "")
	r := adminauth.DiagnoseAuthReadiness("homologation")
	if r.Mode != adminauth.AuthModeFailClosed || r.OIDCConfigured || r.InteractiveLogin {
		t.Fatalf("%+v", r)
	}
	if len(r.Notes) == 0 {
		t.Fatal("expected notes")
	}
}

func TestDiagnoseAuthReadinessOIDCIncomplete(t *testing.T) {
	t.Setenv("FISCAL_ADMIN_AUTH_MODE", "oidc_jwt")
	t.Setenv("FISCAL_ADMIN_OIDC_ISSUER", "https://idp.example/realms/bwb")
	t.Setenv("FISCAL_ADMIN_OIDC_AUDIENCE", "bwb-fiscal-admin")
	t.Setenv("FISCAL_ADMIN_OIDC_JWKS_URL", "") // missing
	t.Setenv("FISCAL_ADMIN_OIDC_ROLE_MAP", "admins:admin")
	r := adminauth.DiagnoseAuthReadiness("homologation")
	if r.OIDCConfigured || r.InteractiveLogin {
		t.Fatalf("%+v", r)
	}
	if len(r.MissingConfigKeys) == 0 {
		t.Fatal("expected missing keys")
	}
}

func TestDiagnoseAuthReadinessOIDCCompleteStillNoInteractive(t *testing.T) {
	t.Setenv("FISCAL_ADMIN_AUTH_MODE", "oidc_jwt")
	t.Setenv("FISCAL_ADMIN_OIDC_ISSUER", "https://idp.example/realms/bwb")
	t.Setenv("FISCAL_ADMIN_OIDC_AUDIENCE", "bwb-fiscal-admin")
	t.Setenv("FISCAL_ADMIN_OIDC_JWKS_URL", "https://idp.example/jwks")
	t.Setenv("FISCAL_ADMIN_OIDC_ROLE_MAP", "admins:admin")
	r := adminauth.DiagnoseAuthReadiness("homologation")
	if !r.OIDCConfigured {
		t.Fatalf("%+v", r)
	}
	if r.InteractiveLogin {
		t.Fatal("interactive redirect must stay false until IdP authorize is wired")
	}
}
