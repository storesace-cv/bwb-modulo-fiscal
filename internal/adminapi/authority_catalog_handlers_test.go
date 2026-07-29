package adminapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestAuthorityEndpointCatalogAndJWSScaffoldOwner(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "cat.db")
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
	authn := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}
	mux := http.NewServeMux()
	adminapi.Mount(mux, authn, &adminapi.Handler{
		Registry:    adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:       adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:         adminops.New(sqlDB, adminops.DialectSQLite),
		SecretsMeta: mem,
		AuthMode:    "injected",
		FiscalEnv:   "development",
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/endpoint-catalog?environment=homologation", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", rr.Code, rr.Body.String())
	}
	var cat map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &cat); err != nil {
		t.Fatal(err)
	}
	if cat["external_verified"] != false {
		t.Fatalf("%v", cat["external_verified"])
	}
	items, _ := cat["items"].([]any)
	if len(items) < 5 {
		t.Fatalf("items=%d", len(items))
	}

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/jws-profile-scaffold", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("jws status=%d", rr.Code)
	}
	var jws map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &jws); err != nil {
		t.Fatal(err)
	}
	if jws["invented_claims_forbidden"] != true || jws["external_verified"] != false {
		t.Fatalf("%v", jws)
	}
	if jws["algorithm_declared"] != "RS256" {
		t.Fatalf("%v", jws["algorithm_declared"])
	}

	closed := http.NewServeMux()
	adminapi.Mount(closed, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "x", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, &adminapi.Handler{
		Registry: adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:    adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		AuthMode: "injected",
	})
	rr = httptest.NewRecorder()
	closed.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/endpoint-catalog?environment=homologation", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
}
