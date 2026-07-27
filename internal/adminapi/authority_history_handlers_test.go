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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminobs"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAuthorityProfileHistoryAppendOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "auth-history.db")
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
	h := &adminapi.Handler{
		Registry: reg,
		Audit:    audit,
		Ops:      adminops.New(sqlDB, adminops.DialectSQLite),
		Obs:      adminobs.New(nil, "fail_closed"),
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "auditor-1", Roles: []adminauth.Role{adminauth.RoleAuditor},
	}}, h)

	p, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: adminregistry.EnvHomologation, DisplayName: "hist",
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := adminauth.Claims{Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner}}
	if err := audit.Record(ctx, claims, "authority_profile.create", "authority_profile", p.ID, adminaudit.ResultSuccess, "req-1"); err != nil {
		t.Fatal(err)
	}
	if err := audit.Record(ctx, claims, "authority_profile.material_sync", "authority_profile", p.ID, adminaudit.ResultSuccess, "req-2"); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/history", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["append_only"] != true || body["external_verified"] != false {
		t.Fatalf("%v", body)
	}
	items, _ := body["items"].([]any)
	if len(items) < 2 {
		t.Fatalf("items=%v", items)
	}
	raw := rr.Body.String()
	if strings.Contains(raw, "BEGIN PRIVATE") || strings.Contains(strings.ToLower(raw), `"plaintext"`) {
		t.Fatal("plaintext leak in history")
	}
}
