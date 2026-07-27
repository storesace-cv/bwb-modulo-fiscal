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

func TestTaxpayerFEEnrollmentAndNoNIFInAudit(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "admin-fe.db")
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
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-fe", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)

	const nif = "5000000199"
	createTP := httptest.NewRequest(http.MethodPost, "/admin/v1/taxpayers", bytes.NewBufferString(
		`{"nif":"`+nif+`","legal_name":"FE Demo","fe_enrollment":{"environment":"homologation","status":"active"}}`))
	createTP.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, createTP)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create=%d body=%s", rr.Code, rr.Body.String())
	}
	var tp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &tp); err != nil {
		t.Fatal(err)
	}
	tpID, _ := tp["taxpayer_id"].(string)
	feRaw, _ := tp["fe_enrollments"].([]any)
	if tpID == "" || len(feRaw) != 1 {
		t.Fatalf("body=%v", tp)
	}
	fe0, _ := feRaw[0].(map[string]any)
	if fe0["status"] != "active" || fe0["environment"] != "homologation" {
		t.Fatalf("fe=%v", fe0)
	}

	put := httptest.NewRequest(http.MethodPut, "/admin/v1/taxpayers/"+tpID+"/fe-enrollments", bytes.NewBufferString(
		`{"environment":"production","status":"pending"}`))
	put.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, put)
	if rr.Code != http.StatusOK {
		t.Fatalf("put fe=%d %s", rr.Code, rr.Body.String())
	}

	// Duplicate NIF → audit must not store NIF as resource_id.
	dup := httptest.NewRequest(http.MethodPost, "/admin/v1/taxpayers", bytes.NewBufferString(
		`{"nif":"`+nif+`","legal_name":"Dup"}`))
	dup.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, dup)
	if rr.Code != http.StatusConflict && rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("dup want conflict/422 got %d %s", rr.Code, rr.Body.String())
	}

	events, err := audit.ListRecent(ctx, 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		if strings.Contains(ev.ResourceID, nif) || strings.EqualFold(ev.ResourceID, nif) {
			t.Fatalf("NIF leaked in audit resource_id: %+v", ev)
		}
		if strings.Contains(ev.Action+ev.ResourceType+ev.Result+ev.RequestID, nif) {
			t.Fatalf("NIF leaked in audit fields: %+v", ev)
		}
	}
}
