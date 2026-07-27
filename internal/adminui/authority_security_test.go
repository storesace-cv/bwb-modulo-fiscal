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

// RM-AGTPREP-013: surface authority UI — RBAC owner-only, CSRF, no secret leaks.
func TestAuthoritySurfaceSecurity_RBAC_CSRF_NoLeaks(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "auth-sec.db")
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
		Subject: "owner-sec", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	paths := []string{
		"/admin/ui/authority-profiles",
		"/admin/ui/authority-profiles/wizard",
		"/admin/ui/authority-profiles/new",
	}
	for _, roleName := range []struct {
		name  string
		roles []adminauth.Role
	}{
		{"admin", []adminauth.Role{adminauth.RoleAdmin}},
		{"operator", []adminauth.Role{adminauth.RoleOperator}},
		{"auditor", []adminauth.Role{adminauth.RoleAuditor}},
	} {
		mux := http.NewServeMux()
		adminui.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
			Subject: roleName.name, Roles: roleName.roles,
		}}, h)
		for _, p := range paths {
			rr := httptest.NewRecorder()
			mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, p, nil))
			if rr.Code != http.StatusForbidden {
				t.Fatalf("%s GET %s want 403 got %d", roleName.name, p, rr.Code)
			}
		}
	}

	// CSRF fail-closed on wizard step1.
	rr := httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/wizard", nil))
	csrf := csrfFromSetCookie(rr)
	if csrf == "" {
		t.Fatal("csrf")
	}
	rr = httptest.NewRecorder()
	bad := url.Values{
		"csrf_token":   {"not-the-cookie"},
		"display_name": {"x"},
		"environment":  {"homologation"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles/wizard", strings.NewReader(bad.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("bad csrf want 403 got %d", rr.Code)
	}

	// Create via wizard with valid CSRF; pages must not leak PEM/password fields.
	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/authority-profiles/wizard", nil))
	csrf = csrfFromSetCookie(rr)
	rr = httptest.NewRecorder()
	okForm := url.Values{
		"csrf_token":   {csrf},
		"display_name": {"Sec HML"},
		"environment":  {"homologation"},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/authority-profiles/wizard", strings.NewReader(okForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("wizard create %d %s", rr.Code, rr.Body.String())
	}
	list, err := reg.ListAuthorityProfiles(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("%v %#v", err, list)
	}
	p := list[0]
	if p.ExternalVerified || p.Status == adminregistry.AuthorityStatusActive {
		t.Fatalf("%+v", p)
	}

	for _, path := range []string{
		"/admin/ui/authority-profiles/" + p.ID + "/readiness",
		"/admin/ui/authority-profiles/" + p.ID + "/history",
		"/admin/ui/authority-profiles/" + p.ID + "/wizard?step=3",
	} {
		rr = httptest.NewRecorder()
		ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s %d", path, rr.Code)
		}
		body := rr.Body.String()
		if strings.Contains(body, "-----BEGIN") || strings.Contains(body, `type="password"`) {
			t.Fatalf("leak in %s", path)
		}
		if !strings.Contains(body, "external_verified") {
			t.Fatalf("missing external_verified hint in %s", path)
		}
	}
}
