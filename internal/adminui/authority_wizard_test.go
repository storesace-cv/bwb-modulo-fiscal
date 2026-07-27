package adminui_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminui"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAdminUIAuthorityWizardOwnerOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui-wizard.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	h, err := adminui.New(reg, "development")
	if err != nil {
		t.Fatal(err)
	}

	ownerMux := http.NewServeMux()
	adminui.Mount(ownerMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-wiz", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	rr := httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/wizard", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("wizard start %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Wizard") || !strings.Contains(body, "external_verified") {
		t.Fatalf("wizard body: %s", body)
	}
	if strings.Contains(body, "-----BEGIN") || strings.Contains(body, `type="password"`) {
		t.Fatal("secrets UI leaked in wizard")
	}
	csrf := csrfFromSetCookie(rr)
	if csrf == "" {
		t.Fatal("missing csrf")
	}

	rr = httptest.NewRecorder()
	form := url.Values{
		"csrf_token":         {csrf},
		"display_name":       {"Wizard HML"},
		"environment":        {"homologation"},
		"allowed_operations": {"registarFactura"},
		"algorithm_declared": {"RS256"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles/wizard", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("step1 %d %s", rr.Code, rr.Body.String())
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "/wizard?step=2") {
		t.Fatalf("step1 redirect: %s", loc)
	}

	list, err := reg.ListAuthorityProfiles(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("after step1: %v %#v", err, list)
	}
	p := list[0]
	if p.Status != adminregistry.AuthorityStatusDraft || p.ExternalVerified {
		t.Fatalf("must stay draft/unverified: %#v", p)
	}
	if p.Status == adminregistry.AuthorityStatusActive {
		t.Fatal("wizard must never activate")
	}

	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/"+p.ID+"/wizard?step=2", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("step2 get %d", rr.Code)
	}
	csrf = csrfFromSetCookie(rr)

	rr = httptest.NewRecorder()
	step2 := url.Values{
		"csrf_token":              {csrf},
		"producer_credential_ref": {"agt-cred-wiz"},
		"producer_key_ref":        {"agt-key-wiz"},
		"certificate_ref":         {"agt-cert-wiz"},
		"algorithm_declared":      {"RS256"},
		"key_id_sanitized":        {"kid-wiz"},
		"fingerprint_sanitized":   {"sha256:aabb"},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles/"+p.ID+"/wizard/step2", strings.NewReader(step2.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther || !strings.Contains(rr.Header().Get("Location"), "step=3") {
		t.Fatalf("step2 %d loc=%s body=%s", rr.Code, rr.Header().Get("Location"), rr.Body.String())
	}

	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/"+p.ID+"/wizard?step=3", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("step3 get %d", rr.Code)
	}
	step3Body := rr.Body.String()
	if !strings.Contains(step3Body, "não</strong> activa") && !strings.Contains(step3Body, "não activa") {
		// HTML may escape — check Portuguese hint
		if !strings.Contains(step3Body, "não") || !strings.Contains(step3Body, "external_verified") {
			t.Fatalf("step3 missing fail-closed hints: %s", step3Body)
		}
	}
	csrf = csrfFromSetCookie(rr)

	rr = httptest.NewRecorder()
	step3 := url.Values{"csrf_token": {csrf}}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles/"+p.ID+"/wizard/step3", strings.NewReader(step3.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther || !strings.Contains(rr.Header().Get("Location"), "/readiness") {
		t.Fatalf("step3 %d loc=%s", rr.Code, rr.Header().Get("Location"))
	}

	final, err := reg.GetAuthorityProfile(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.Status != adminregistry.AuthorityStatusDraft {
		t.Fatalf("wizard must leave draft, got %s", final.Status)
	}
	if !final.ConfigReady || final.ExternalVerified || final.Status == adminregistry.AuthorityStatusActive {
		t.Fatalf("fail-closed violated: %#v", final)
	}
	if final.ProducerCredentialRef != "agt-cred-wiz" || final.KeyIDSanitized != "kid-wiz" {
		t.Fatalf("refs: %#v", final)
	}

	adminMux := http.NewServeMux()
	adminui.Mount(adminMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-wiz", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)
	rr = httptest.NewRecorder()
	adminMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/wizard", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("admin want 403 got %d", rr.Code)
	}
}
