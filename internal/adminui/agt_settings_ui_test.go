package adminui_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminui"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestAGTSettingsHubOwnerOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "agt-hub.db")
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
	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	h.SecretsMeta = mem
	h.AuthorityMode = "simulator"
	h.FiscalEnv = "development"

	ownerMux := http.NewServeMux()
	adminui.Mount(ownerMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-hub", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	rr := httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/agt-settings?environment=homologation", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "registarFactura") || !strings.Contains(body, "solicitarSerie") {
		t.Fatalf("missing catalog ops")
	}
	if !strings.Contains(body, "RS256") || strings.Contains(body, "BEGIN PRIVATE") {
		t.Fatal("jws scaffold / leak")
	}
	if !strings.Contains(body, "Preparação AGT") {
		t.Fatal("missing hub title")
	}
	if !strings.Contains(body, "Correr probe (simulador)") {
		t.Fatal("missing probe form")
	}
	if !strings.Contains(body, "Bindings") {
		t.Fatal("missing bindings column")
	}
	if !strings.Contains(body, "SecretStore (estado sanitizado)") || !strings.Contains(body, "ready_for_homologation") {
		t.Fatal("missing vault status section")
	}
	if !strings.Contains(body, "SecAdm gate") || !strings.Contains(body, "secadm-gate-status") {
		t.Fatal("missing secadm gate section")
	}

	adminMux := http.NewServeMux()
	adminui.Mount(adminMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-1", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)
	rr = httptest.NewRecorder()
	adminMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/agt-settings", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("non-owner want 403, got %d", rr.Code)
	}
}

func TestAGTSettingsProbeCSRFAndFailClosed(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "agt-probe.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	audit := adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil)
	h, err := adminui.New(reg, "development")
	if err != nil {
		t.Fatal(err)
	}
	h.Audit = audit
	h.AuthorityMode = "simulator"
	h.FiscalEnv = "homologation"

	mux := http.NewServeMux()
	adminui.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-probe", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/agt-settings?environment=homologation", nil))
	csrf := csrfFromSetCookie(rr)
	if csrf == "" {
		t.Fatal("missing csrf")
	}

	// Bad CSRF
	form := url.Values{"csrf_token": {"bad"}, "environment": {"homologation"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/ui/agt-settings/probe", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("bad csrf want 403 got %d", rr.Code)
	}

	// Happy path simulator
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/agt-settings?environment=homologation", nil))
	csrf = csrfFromSetCookie(rr)
	form = url.Values{"csrf_token": {csrf}, "environment": {"homologation"}}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/agt-settings/probe", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("probe want 303 got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	if !strings.Contains(loc, "probe_ok=") || strings.Contains(loc, "BEGIN ") {
		t.Fatalf("redirect=%s", loc)
	}

	// Fail-closed: production + simulator
	h.FiscalEnv = "production"
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/agt-settings?environment=homologation", nil))
	csrf = csrfFromSetCookie(rr)
	form = url.Values{"csrf_token": {csrf}, "environment": {"homologation"}}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/agt-settings/probe", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("fail-closed want 303 got %d", rr.Code)
	}
	loc = rr.Header().Get("Location")
	if !strings.Contains(loc, "probe_err=") || !strings.Contains(loc, "fail-closed") {
		t.Fatalf("fail-closed redirect=%s", loc)
	}
}
