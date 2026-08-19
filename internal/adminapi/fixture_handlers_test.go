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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestAuthorityFixtureIdentitiesAndHubOwner(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "fix.db")
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
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/fixture-identities", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("unconfigured status=%d", rr.Code)
	}
	var unconfigured map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &unconfigured); err != nil {
		t.Fatal(err)
	}
	if unconfigured["workbook_configured"] != false {
		t.Fatalf("%v", unconfigured)
	}

	wbPath, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()
	mux2 := http.NewServeMux()
	adminapi.Mount(mux2, authn, &adminapi.Handler{
		Registry:            adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:               adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:                 adminops.New(sqlDB, adminops.DialectSQLite),
		SecretsMeta:         mem,
		AuthMode:            "injected",
		FiscalEnv:           "development",
		AGTTestWorkbookPath: wbPath,
	})
	rr2 := httptest.NewRecorder()
	mux2.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/fixture-identities", nil))
	if rr2.Code != http.StatusOK {
		t.Fatalf("configured status=%d body=%s", rr2.Code, rr2.Body.String())
	}
	var configured map[string]any
	if err := json.Unmarshal(rr2.Body.Bytes(), &configured); err != nil {
		t.Fatal(err)
	}
	if configured["workbook_configured"] != true {
		t.Fatalf("%v", configured)
	}
	if int(configured["count"].(float64)) != 2 {
		t.Fatalf("count=%v", configured["count"])
	}

	rr3 := httptest.NewRecorder()
	mux2.ServeHTTP(rr3, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/fixture-hub", nil))
	if rr3.Code != http.StatusOK {
		t.Fatalf("hub status=%d", rr3.Code)
	}
}
