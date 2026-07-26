package adminui_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminui"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAdminUIDashboardReadOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	tp, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{
		NIF: "5000000100", LegalName: "UI Demo Lda",
	})
	if err != nil {
		t.Fatal(err)
	}
	est, err := reg.CreateEstablishment(ctx, adminregistry.CreateEstablishmentInput{
		TaxpayerID: tp.ID, Code: "L1", Name: "Loja",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = reg.CreateScopeBinding(ctx, adminregistry.CreateScopeBindingInput{
		ScopeID: "scope-ui-1", TaxpayerID: tp.ID, EstablishmentID: est.ID,
		Environment: adminregistry.EnvHomologation, IANATimezone: "Africa/Luanda",
		SeriesEffectiveCode: "A",
	})
	if err != nil {
		t.Fatal(err)
	}

	h, err := adminui.New(reg, "development")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	adminui.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "ops-ui", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, h)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("dashboard %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "UI Demo Lda") || !strings.Contains(body, "scope-ui-1") {
		t.Fatalf("missing data: %s", body)
	}
	if !strings.Contains(rr.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatalf("csp=%q", rr.Header().Get("Content-Security-Policy"))
	}
	if strings.Contains(strings.ToLower(body), "<script") {
		t.Fatal("unexpected script tag")
	}
	// No secrets / credentials patterns.
	for _, bad := range []string{"plaintext", "BEGIN RSA", "password=", "authorization:"} {
		if strings.Contains(strings.ToLower(body), bad) {
			t.Fatalf("forbidden token %q in body", bad)
		}
	}

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/taxpayers", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "5000000100") {
		t.Fatalf("taxpayers %d", rr.Code)
	}

	// Fail-closed
	closed := http.NewServeMux()
	adminui.Mount(closed, adminauth.FailClosedAuthenticator{}, h)
	rr = httptest.NewRecorder()
	closed.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Não autenticado") {
		t.Fatalf("html 401: %s", rr.Body.String())
	}

	// Static CSS public
	rr = httptest.NewRecorder()
	closed.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/static/admin.css", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "--accent") {
		t.Fatalf("css %d", rr.Code)
	}
}

func TestDevCookieAuthenticator(t *testing.T) {
	secret := strings.Repeat("a", 32)
	authn := adminui.DevCookieAuthenticator{
		AllowDev: true, CookieVal: secret,
		Claims: adminauth.Claims{Subject: "dev", Roles: []adminauth.Role{adminauth.RoleAdmin}},
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_ui_session", Value: secret})
	claims, err := authn.Authenticate(t.Context(), req)
	if err != nil || claims.Subject != "dev" {
		t.Fatalf("%v %+v", err, claims)
	}
	bad := httptest.NewRequest(http.MethodGet, "/", nil)
	bad.AddCookie(&http.Cookie{Name: "fiscal_admin_ui_session", Value: "wrong"})
	if _, err := authn.Authenticate(t.Context(), bad); err == nil {
		t.Fatal("want unauthorized")
	}
}
