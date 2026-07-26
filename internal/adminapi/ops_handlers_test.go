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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestAdminOpsVisibilityAndSecretMetadata(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "admin-ops.db")
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
	ref := secretstore.Ref{
		Kind: "producer_credential", Environment: secretstore.EnvHomologation,
		SubjectID: "platform", Name: "basic",
	}
	if _, err := mem.Put(ctx, ref, []byte("not-a-real-secret"), nil); err != nil {
		t.Fatal(err)
	}

	h := &adminapi.Handler{
		Registry:    adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:       adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:         adminops.New(sqlDB, adminops.DialectSQLite),
		SecretsMeta: mem,
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "ops-1", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, h)

	listSub := httptest.NewRequest(http.MethodGet, "/admin/v1/ops/submissions?limit=10", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, listSub)
	if rr.Code != http.StatusOK {
		t.Fatalf("ops submissions %d %s", rr.Code, rr.Body.String())
	}
	var subs map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &subs); err != nil {
		t.Fatal(err)
	}
	if _, ok := subs["items"]; !ok {
		t.Fatalf("missing items: %v", subs)
	}
	if strings.Contains(rr.Body.String(), "not-a-real-secret") {
		t.Fatal("secret material leaked in ops list")
	}

	metaReq := httptest.NewRequest(http.MethodGet,
		"/admin/v1/secret-refs/metadata?kind=producer_credential&environment=homologation&subject_id=platform&name=basic", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, metaReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("metadata %d %s", rr.Code, rr.Body.String())
	}
	var meta map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["status"] != secretstore.StatusPresent {
		t.Fatalf("status=%v", meta["status"])
	}
	fp, _ := meta["fingerprint"].(string)
	if fp == "" || strings.Contains(fp, "not-a-real-secret") {
		t.Fatalf("bad fingerprint %v", meta["fingerprint"])
	}
	if strings.Contains(rr.Body.String(), "not-a-real-secret") {
		t.Fatal("plaintext leaked in metadata response")
	}

	auditReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-events?limit=5", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, auditReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("audit %d", rr.Code)
	}
}
