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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestAdminUISecAdmMetadataOwnerOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui-secadm.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte("synthetic-ui-secret-not-real")
	put, err := mem.Put(ctx, secretstore.Ref{
		Kind: "producer_credential", Environment: "homologation",
		SubjectID: "platform", Name: "agt",
	}, plaintext, nil)
	if err != nil {
		t.Fatal(err)
	}

	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	h, err := adminui.New(reg, "development")
	if err != nil {
		t.Fatal(err)
	}
	h.SecretsMeta = mem

	ownerMux := http.NewServeMux()
	adminui.Mount(ownerMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-ui", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	rr := httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/secadm/metadata", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("owner get %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "apenas metadados") || !strings.Contains(body, "SecAdm") {
		t.Fatalf("nav/body: %s", body)
	}
	if strings.Contains(body, string(plaintext)) {
		t.Fatal("plaintext leaked on GET")
	}
	csrf := csrfFromSetCookie(rr)
	if csrf == "" {
		t.Fatal("missing csrf")
	}

	rr = httptest.NewRecorder()
	form := url.Values{
		"csrf_token": {csrf},
		"kind":       {"producer_credential"}, "environment": {"homologation"},
		"subject_id": {"platform"}, "name": {"agt"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/ui/secadm/metadata", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	ownerMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("lookup %d %s", rr.Code, rr.Body.String())
	}
	out := rr.Body.String()
	if !strings.Contains(out, put.Metadata.Fingerprint) || !strings.Contains(out, put.Metadata.Status) {
		t.Fatalf("missing metadata: %s", out)
	}
	if strings.Contains(out, string(plaintext)) || strings.Contains(strings.ToLower(out), "plaintext") {
		t.Fatal("plaintext leaked on POST result")
	}

	adminMux := http.NewServeMux()
	adminui.Mount(adminMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-ui", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)
	rr = httptest.NewRecorder()
	adminMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/secadm/metadata", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("admin want 403 got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "apenas owner") {
		t.Fatalf("deny body: %s", rr.Body.String())
	}
}
