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
