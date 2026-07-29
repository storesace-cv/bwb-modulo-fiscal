package adminapi_test

import (
	"context"
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

func TestAuthorityBindingValidationOwner(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bind.db")
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
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	h := &adminapi.Handler{
		Registry: reg, Audit: adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops: adminops.New(sqlDB, adminops.DialectSQLite), SecretsMeta: mem, AuthMode: "injected",
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	p, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: adminregistry.EnvHomologation, DisplayName: "bind",
		AllowedOperations:     []string{"registarFactura"},
		ProducerCredentialRef: "c", ProducerKeyRef: "k", CertificateRef: "cert",
	})
	if err != nil {
		t.Fatal(err)
	}
	trueVal := true
	if _, err := reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID: p.ID, SecretsReady: &trueVal,
	}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/binding-validation", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["valid"] != false || body["external_verified"] != false {
		t.Fatalf("%v", body)
	}

	// Conflict op rejected at registry
	_, err = reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: adminregistry.EnvHomologation, DisplayName: "bad",
		AllowedOperations: []string{"solicitarSerie"},
	})
	if err == nil {
		t.Fatal("expected conflict_open rejection")
	}
}
