package adminapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAdminCadastrosHappyPathAndRBAC(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "admin-api.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	audit := adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil)
	h := &adminapi.Handler{Registry: reg, Audit: audit}

	adminAuth := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-1", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminAuth, h)

	createTP := httptest.NewRequest(http.MethodPost, "/admin/v1/taxpayers", bytes.NewBufferString(
		`{"nif":"5000000099","legal_name":"Demo Lda"}`))
	createTP.Header.Set("Content-Type", "application/json")
	createTP.Header.Set("X-Request-Id", "req-1")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, createTP)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create taxpayer status=%d body=%s", rr.Code, rr.Body.String())
	}
	var tp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &tp); err != nil {
		t.Fatal(err)
	}
	tpID, _ := tp["taxpayer_id"].(string)
	if tpID == "" {
		t.Fatal("missing taxpayer_id")
	}

	createEst := httptest.NewRequest(http.MethodPost, "/admin/v1/establishments", bytes.NewBufferString(
		`{"taxpayer_id":"`+tpID+`","code":"L1","name":"Loja"}`))
	createEst.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, createEst)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create est status=%d body=%s", rr.Code, rr.Body.String())
	}
	var est map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &est)
	estID, _ := est["establishment_id"].(string)

	createBind := httptest.NewRequest(http.MethodPost, "/admin/v1/scope-bindings", bytes.NewBufferString(
		`{"scope_id":"scope-admin-1","taxpayer_id":"`+tpID+`","establishment_id":"`+estID+`","environment":"homologation","iana_timezone":"Africa/Luanda","series_effective_code":"A"}`))
	createBind.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, createBind)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create bind status=%d body=%s", rr.Code, rr.Body.String())
	}

	getTP := httptest.NewRequest(http.MethodGet, "/admin/v1/taxpayers/"+tpID, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, getTP)
	if rr.Code != http.StatusOK {
		t.Fatalf("get taxpayer %d", rr.Code)
	}

	n, err := audit.CountForTests(ctx)
	if err != nil || n < 3 {
		t.Fatalf("audit count=%d err=%v", n, err)
	}

	// Auditor cannot create.
	auditorMux := http.NewServeMux()
	adminapi.Mount(auditorMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "aud-1", Roles: []adminauth.Role{adminauth.RoleAuditor},
	}}, h)
	deny := httptest.NewRequest(http.MethodPost, "/admin/v1/taxpayers", bytes.NewBufferString(
		`{"nif":"5000000098","legal_name":"X"}`))
	deny.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	auditorMux.ServeHTTP(rr, deny)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("auditor create want 403 got %d", rr.Code)
	}

	// Fail-closed authenticator → 401
	closedMux := http.NewServeMux()
	adminapi.Mount(closedMux, adminauth.FailClosedAuthenticator{}, h)
	unauth := httptest.NewRequest(http.MethodGet, "/admin/v1/taxpayers/"+tpID, nil)
	rr = httptest.NewRecorder()
	closedMux.ServeHTTP(rr, unauth)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rr.Code)
	}
}
