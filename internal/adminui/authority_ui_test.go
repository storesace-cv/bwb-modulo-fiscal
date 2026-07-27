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

func TestAdminUIAuthorityProfilesOwnerOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui-authority.db")
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
		Subject: "owner-ui", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	rr := httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("owner list %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Perfis de autoridade") || !strings.Contains(body, "external_verified") {
		t.Fatalf("list body: %s", body)
	}
	if strings.Contains(body, "BEGIN ") || strings.Contains(strings.ToLower(body), "password") && strings.Contains(body, "type=\"password\"") {
		t.Fatal("secret-like fields leaked")
	}

	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/new", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("new form %d", rr.Code)
	}
	csrf := csrfFromSetCookie(rr)
	if csrf == "" {
		t.Fatal("missing csrf")
	}
	if !strings.Contains(rr.Body.String(), "registarFactura") {
		t.Fatal("missing known ops")
	}

	rr = httptest.NewRecorder()
	form := url.Values{
		"csrf_token":              {csrf},
		"display_name":            {"HML prep"},
		"environment":             {"homologation"},
		"status":                  {"draft"},
		"allowed_operations":      {"registarFactura", "obterEstado"},
		"algorithm_declared":      {"RS256"},
		"key_id_sanitized":        {"kid-test"},
		"fingerprint_sanitized":   {"sha256:deadbeef"},
		"producer_credential_ref": {"agt-cred"},
		"pending_external":        {"unknown_endpoint=pending_external"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}

	list, err := reg.ListAuthorityProfiles(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list after create: %v %#v", err, list)
	}
	p := list[0]
	if p.DisplayName != "HML prep" || p.AlgorithmDeclared != "RS256" || p.ExternalVerified {
		t.Fatalf("profile: %#v", p)
	}
	if p.FingerprintSanitized != "sha256:deadbeef" || p.KeyIDSanitized != "kid-test" {
		t.Fatalf("sanitized fields: %#v", p)
	}

	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles", nil))
	out := rr.Body.String()
	if !strings.Contains(out, "HML prep") || !strings.Contains(out, "sha256:deadbeef") || !strings.Contains(out, "kid-test") {
		t.Fatalf("list missing sanitized meta: %s", out)
	}
	if !strings.Contains(out, "cfg=false") || !strings.Contains(out, "ext=false") {
		t.Fatalf("readiness missing: %s", out)
	}

	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/"+p.ID+"/edit", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("edit get %d", rr.Code)
	}
	csrf = csrfFromSetCookie(rr)
	editBody := rr.Body.String()
	if !strings.Contains(editBody, "config_ready") || strings.Contains(editBody, "-----BEGIN") {
		t.Fatalf("edit form bad: %s", editBody)
	}

	rr = httptest.NewRecorder()
	patch := url.Values{
		"csrf_token":            {csrf},
		"display_name":          {"HML prep"},
		"status":                {"validated"},
		"allowed_operations":    {"registarFactura"},
		"algorithm_declared":    {"RS256"},
		"key_id_sanitized":      {"kid-test"},
		"fingerprint_sanitized": {"sha256:deadbeef"},
		"config_ready":          {"on"},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles/"+p.ID, strings.NewReader(patch.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("patch %d %s", rr.Code, rr.Body.String())
	}
	updated, err := reg.GetAuthorityProfile(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "validated" || !updated.ConfigReady || updated.ExternalVerified {
		t.Fatalf("updated: %#v", updated)
	}

	// Reject PEM-like fingerprint
	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/new", nil))
	csrf = csrfFromSetCookie(rr)
	rr = httptest.NewRecorder()
	bad := url.Values{
		"csrf_token":            {csrf},
		"display_name":          {"bad"},
		"environment":           {"homologation"},
		"fingerprint_sanitized": {"-----BEGIN PRIVATE KEY-----"},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles", strings.NewReader(bad.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code == http.StatusSeeOther {
		t.Fatal("PEM fingerprint must be rejected")
	}
	if strings.Contains(rr.Body.String(), "BEGIN PRIVATE") {
		t.Fatal("rejected PEM echoed in HTML")
	}

	adminMux := http.NewServeMux()
	adminui.Mount(adminMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-ui", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)
	rr = httptest.NewRecorder()
	adminMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("admin want 403 got %d", rr.Code)
	}
}
