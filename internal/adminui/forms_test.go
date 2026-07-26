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

func TestAdminUIFormsCSRFAndCreate(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui-forms.db")
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
	mux := http.NewServeMux()
	adminui.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-ui", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)

	// GET form → CSRF cookie
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/taxpayers/new", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("form %d", rr.Code)
	}
	csrf := csrfFromSetCookie(rr)
	if csrf == "" {
		t.Fatal("missing csrf cookie")
	}
	if !strings.Contains(rr.Body.String(), `name="csrf_token"`) {
		t.Fatal("missing csrf field")
	}

	// POST without CSRF → 403
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/ui/taxpayers", strings.NewReader(url.Values{
		"nif": {"5000000200"}, "legal_name": {"Form Lda"}, "status": {"active"},
	}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("no csrf want 403 got %d", rr.Code)
	}

	// POST with CSRF → redirect
	rr = httptest.NewRecorder()
	form := url.Values{
		"csrf_token": {csrf}, "nif": {"5000000200"}, "legal_name": {"Form Lda"}, "status": {"active"},
	}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/taxpayers", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("create want 303 got %d body=%s", rr.Code, rr.Body.String())
	}

	// Operator cannot open write form
	opMux := http.NewServeMux()
	adminui.Mount(opMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "op", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, h)
	rr = httptest.NewRecorder()
	opMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/taxpayers/new", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("operator write want 403 got %d", rr.Code)
	}
}

func csrfFromSetCookie(rr *httptest.ResponseRecorder) string {
	for _, c := range rr.Result().Cookies() {
		if c.Name == "fiscal_admin_csrf" {
			return c.Value
		}
	}
	return ""
}
