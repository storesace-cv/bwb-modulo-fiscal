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

func TestAdminUISessionCookieNoJWTInBrowser(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui-session.db")
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
	h.CookieSecure = false
	h.Sessions = adminui.NewSessionStore(nil, false)
	h.CSRF.SetSecure(false)
	tokenAuth := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "sess-user", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}
	h.TokenAuth = tokenAuth

	uiAuth := adminui.BuildUIAuthenticator(adminauth.FailClosedAuthenticator{}, h.Sessions, "development", "", nil)
	mux := http.NewServeMux()
	adminui.Mount(mux, uiAuth, h)

	// Mint session from Bearer (token not returned in Set-Cookie).
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/ui/auth/session", nil)
	req.Header.Set("Authorization", "Bearer synthetic-not-logged")
	// StaticAuthenticator ignores Bearer content — still proves no JWT in cookie.
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("session create %d %s", rr.Code, rr.Body.String())
	}
	var sessionCookie string
	for _, c := range rr.Result().Cookies() {
		if c.Name == "fiscal_admin_session" {
			sessionCookie = c.Value
			if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
				t.Fatalf("cookie flags: %+v", c)
			}
			if strings.Contains(strings.ToLower(c.Value), "bearer") || strings.Count(c.Value, ".") == 2 {
				t.Fatal("cookie looks like a JWT")
			}
		}
	}
	if sessionCookie == "" {
		t.Fatal("missing session cookie")
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/ui/", nil)
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_session", Value: sessionCookie})
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("dashboard with session %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "sess-user") {
		t.Fatalf("body: %s", rr.Body.String())
	}
	csrf := csrfFromSetCookie(rr)
	if csrf == "" {
		t.Fatal("csrf for logout")
	}

	rr = httptest.NewRecorder()
	form := url.Values{"csrf_token": {csrf}}
	req = httptest.NewRequest(http.MethodPost, "/admin/ui/auth/logout", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_session", Value: sessionCookie})
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusSeeOther {
		t.Fatalf("logout %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/admin/ui/", nil)
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_session", Value: sessionCookie})
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("after logout want 401 got %d", rr.Code)
	}
}
