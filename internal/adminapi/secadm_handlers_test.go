package adminapi_test

import (
	"bytes"
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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestSecAdmHTTPOwnerWriteOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "secadm-http.db")
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
	gate, err := secadm.NewGate("owner-1", mem)
	if err != nil {
		t.Fatal(err)
	}
	h := &adminapi.Handler{
		Registry:    adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:       adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:         adminops.New(sqlDB, adminops.DialectSQLite),
		SecretsMeta: mem,
		SecAdm:      gate,
	}

	ownerMux := http.NewServeMux()
	adminapi.Mount(ownerMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	body := `{"kind":"producer_credential","environment":"homologation","subject_id":"platform","name":"agt","plaintext":"synthetic-not-real"}`
	put := httptest.NewRequest(http.MethodPut, "/admin/v1/secadm/secret-refs", bytes.NewBufferString(body))
	put.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, put)
	if rr.Code != http.StatusOK {
		t.Fatalf("put status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "synthetic-not-real") {
		t.Fatal("plaintext leaked in put response")
	}
	var meta map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["status"] != secretstore.StatusPresent || meta["fingerprint"] == "" {
		t.Fatalf("%v", meta)
	}

	adminMux := http.NewServeMux()
	adminapi.Mount(adminMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-1", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)
	deny := httptest.NewRequest(http.MethodPut, "/admin/v1/secadm/secret-refs", bytes.NewBufferString(body))
	deny.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	adminMux.ServeHTTP(rr, deny)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("admin want 403 got %d", rr.Code)
	}

	// Owner role but wrong subject → gate deny
	wrongOwner := http.NewServeMux()
	adminapi.Mount(wrongOwner, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "other-owner", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)
	req2 := httptest.NewRequest(http.MethodPut, "/admin/v1/secadm/secret-refs", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	wrongOwner.ServeHTTP(rr, req2)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("wrong owner subject want 403 got %d body=%s", rr.Code, rr.Body.String())
	}

	rev := httptest.NewRequest(http.MethodPost, "/admin/v1/secadm/secret-refs/revoke", bytes.NewBufferString(
		`{"kind":"producer_credential","environment":"homologation","subject_id":"platform","name":"agt"}`))
	rev.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, rev)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke %d %s", rr.Code, rr.Body.String())
	}
}
