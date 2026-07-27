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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

// RM-AGTPREP-013: API authority — RBAC, no plaintext, external_verified=false.
func TestAuthorityAPISurfaceSecurity(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "auth-api-sec.db")
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
	gate, err := secadm.NewGate("owner-1", mem)
	if err != nil {
		t.Fatal(err)
	}
	h := &adminapi.Handler{
		Registry: reg,
		Audit:    adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:      adminops.New(sqlDB, adminops.DialectSQLite),
		Obs:      adminobs.New(nil, "fail_closed"),
		SecAdm:   gate,
	}

	p, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: adminregistry.EnvHomologation, DisplayName: "sec-api",
		CertificateRef: "agt-cert",
	})
	if err != nil {
		t.Fatal(err)
	}

	opsMux := http.NewServeMux()
	adminapi.Mount(opsMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "ops-1", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, h)

	// Operator may read readiness/history; must not create profiles or lifecycle (owner).
	rr := httptest.NewRecorder()
	opsMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/readiness", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("ops readiness %d", rr.Code)
	}
	assertNoSecretJSON(t, rr.Body.Bytes())
	var ready map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &ready); err != nil {
		t.Fatal(err)
	}
	if ready["external_verified"] != false {
		t.Fatalf("%v", ready)
	}

	rr = httptest.NewRecorder()
	opsMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/history", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("ops history %d", rr.Code)
	}
	assertNoSecretJSON(t, rr.Body.Bytes())

	rr = httptest.NewRecorder()
	opsMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/material-lifecycle", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("ops lifecycle want 403 got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/authority-profiles", strings.NewReader(
		`{"environment":"homologation","display_name":"denied"}`))
	req.Header.Set("Content-Type", "application/json")
	opsMux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("ops create want 403 got %d", rr.Code)
	}

	ownerMux := http.NewServeMux()
	adminapi.Mount(ownerMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)
	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/material-lifecycle", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("owner lifecycle %d %s", rr.Code, rr.Body.String())
	}
	assertNoSecretJSON(t, rr.Body.Bytes())
	var lc map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &lc); err != nil {
		t.Fatal(err)
	}
	if lc["external_verified"] != false {
		t.Fatalf("%v", lc)
	}
}

func assertNoSecretJSON(t *testing.T, raw []byte) {
	t.Helper()
	s := string(raw)
	low := strings.ToLower(s)
	for _, bad := range []string{"-----begin", `"plaintext"`, "private key", `"password"`} {
		if strings.Contains(low, bad) {
			t.Fatalf("possible secret leak containing %q", bad)
		}
	}
}
