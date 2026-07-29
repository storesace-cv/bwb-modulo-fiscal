package adminapi_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestAuthoritySecretStoreStatusOwnerOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "ss-status.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 3)
	}
	b64 := base64.StdEncoding.EncodeToString(key)
	t.Setenv("FISCAL_SECRETSTORE_BACKEND", "sql")
	t.Setenv("FISCAL_SECRETSTORE_MASTER_KEY", b64)

	h := &adminapi.Handler{
		Registry: reg, Audit: adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		SecretsMeta: mem, AuthMode: "injected", FiscalEnv: "homologation",
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-ss", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/secretstore-status", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["external_verified"] != false {
		t.Fatal("external_verified")
	}
	if body["master_key_parse_ok"] != true {
		t.Fatalf("parse: %+v", body)
	}
	fp, _ := body["master_key_fingerprint"].(string)
	if fp == "" || strings.Contains(fp, b64) || strings.Contains(rr.Body.String(), b64) {
		t.Fatalf("fingerprint leak: %s", rr.Body.String())
	}

	adminMux := http.NewServeMux()
	adminapi.Mount(adminMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-1", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)
	rr = httptest.NewRecorder()
	adminMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/secretstore-status", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("non-owner want 403 got %d", rr.Code)
	}
}
