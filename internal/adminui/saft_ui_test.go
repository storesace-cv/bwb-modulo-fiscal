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

func TestAdminUISAFStatusReadOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui-saft.db")
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
		Subject: "ops-saft", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, h)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/saft", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("saft %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, want := range []string{
		"AO-SAFT-XSD-1.01_01",
		"pending_validation",
		"GAP-SAFT-PAY-PERSIST",
		"SalesInvoices",
		"Estrutura XSD",
		"GAP-SAFT-GLE-PERSIST",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body: %s", want, body)
		}
	}
	if strings.Contains(body, "BEGIN RSA") || strings.Contains(strings.ToLower(body), "<script") {
		t.Fatal("forbidden content in saft page")
	}
	if strings.Contains(strings.ToLower(body), `download="`) || strings.Contains(body, "href=\"/admin/ui/saft/export") {
		t.Fatal("unexpected download affordance")
	}

	closed := http.NewServeMux()
	adminui.Mount(closed, adminauth.FailClosedAuthenticator{}, h)
	rr = httptest.NewRecorder()
	closed.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/saft", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rr.Code)
	}

	// Subject without roles must fail closed (no ops.read).
	norole := http.NewServeMux()
	adminui.Mount(norole, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "norole", Roles: nil,
	}}, h)
	rr = httptest.NewRecorder()
	norole.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/saft", nil))
	if rr.Code == http.StatusOK {
		t.Fatalf("subject without roles must not access saft: %d", rr.Code)
	}
}
