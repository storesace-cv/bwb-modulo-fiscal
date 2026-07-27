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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAuthorityProfileHTTP_OwnerWriteOpsRead(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "auth-api.db")
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
	h := &adminapi.Handler{Registry: reg, Audit: audit, AuthMode: "fail_closed"}

	ownerMux := http.NewServeMux()
	adminapi.Mount(ownerMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	body := `{"environment":"homologation","display_name":"HML","allowed_operations":["registarFactura"],"algorithm_declared":"RS256","pending_external":{"note":"C-FE-001"}}`
	rr := httptest.NewRecorder()
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/authority-profiles", bytes.NewBufferString(body)))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create %d %s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created["external_verified"] != false {
		t.Fatalf("%v", created)
	}
	id, _ := created["profile_id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}
	low := strings.ToLower(rr.Body.String())
	for _, bad := range []string{"begin rsa", "password", "authorization"} {
		if strings.Contains(low, bad) {
			t.Fatalf("leak %q", bad)
		}
	}

	opsMux := http.NewServeMux()
	adminapi.Mount(opsMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "ops-1", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}, h)
	rr = httptest.NewRecorder()
	opsMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+id, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("ops get %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	opsMux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/authority-profiles", bytes.NewBufferString(body)))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("ops create want 403 got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	patch := `{"status":"validated","config_ready":true}`
	ownerMux.ServeHTTP(rr, httptest.NewRequest(http.MethodPatch, "/admin/v1/authority-profiles/"+id, bytes.NewBufferString(patch)))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"config_ready":true`) {
		t.Fatalf("patch %d %s", rr.Code, rr.Body.String())
	}
}
