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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminobs"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAuthorityReadinessChecklist(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "readiness.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	obs := adminobs.New(nil, "fail_closed")
	h := &adminapi.Handler{
		Registry: reg,
		Audit:    adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:      adminops.New(sqlDB, adminops.DialectSQLite),
		Obs:      obs,
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "ops-1", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, h)

	p, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: "homologation", DisplayName: "ready-check", Status: "draft",
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg, sec, off := true, true, true
	if _, err := reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID: p.ID, ConfigReady: &cfg, SecretsReady: &sec, OfflineValidated: &off,
	}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/readiness", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["config_ready"] != true || body["secrets_ready"] != true || body["offline_validated"] != true {
		t.Fatalf("%v", body)
	}
	if body["external_verified"] != false || body["checklist_complete"] != true {
		t.Fatalf("%v", body)
	}
	snap := obs.Snapshot()
	found := false
	for _, s := range snap.Series {
		if s.RouteClass == string(adminobs.RouteAuthority) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected authority metric series: %+v", snap)
	}
}
