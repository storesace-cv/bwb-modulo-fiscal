package adminapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func mountProbeHandler(t *testing.T, fiscalEnv, authorityMode string) http.Handler {
	t.Helper()
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "probe.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	h := &adminapi.Handler{
		Registry:      adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:         adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:           adminops.New(sqlDB, adminops.DialectSQLite),
		AuthorityMode: authorityMode,
		FiscalEnv:     fiscalEnv,
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)
	return mux
}

func TestProbeAuthorityConfigSimulatorOK(t *testing.T) {
	mux := mountProbeHandler(t, "development", "simulator")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/authority/probe-config", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] != true || body["external_verified"] != false || body["mode"] != "simulator" {
		t.Fatalf("%v", body)
	}
	if body["simulator_reachable"] != true {
		t.Fatalf("%v", body)
	}
}

func TestProbeAuthorityConfigProductionSimulatorForbidden(t *testing.T) {
	mux := mountProbeHandler(t, "production", "simulator")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/authority/probe-config", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ADMIN_AUTHORITY_FAIL_CLOSED") {
		t.Fatalf("%s", rr.Body.String())
	}
}

func TestProbeAuthorityConfigAGTHMLForbidden(t *testing.T) {
	mux := mountProbeHandler(t, "homologation", "agt-hml")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/authority/probe-config", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", rr.Code)
	}
}

func TestProbeAuthorityConfigAGTPRDForbidden(t *testing.T) {
	mux := mountProbeHandler(t, "production", "agt-prd")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/authority/probe-config", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 got %d", rr.Code)
	}
}
