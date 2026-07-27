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
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestDocumentAvailabilityPolicyAPI(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "doc-policy.db")
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
		Subject: "admin-doc", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)

	tp, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{NIF: "5000000299", LegalName: "Doc Co"})
	if err != nil {
		t.Fatal(err)
	}
	est, err := reg.CreateEstablishment(ctx, adminregistry.CreateEstablishmentInput{
		TaxpayerID: tp.ID, Code: "L1", Name: "Loja",
	})
	if err != nil {
		t.Fatal(err)
	}

	get := httptest.NewRequest(http.MethodGet, "/admin/v1/establishments/"+est.ID+"/document-availability?environment=homologation", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, get)
	if rr.Code != http.StatusOK {
		t.Fatalf("get avail=%d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["fe_aderiu"] != false {
		t.Fatalf("fe_aderiu=%v", body["fe_aderiu"])
	}
	types, _ := body["types"].([]any)
	if len(types) < 2 {
		t.Fatalf("types=%d", len(types))
	}

	putG := httptest.NewRequest(http.MethodPut, "/admin/v1/establishments/"+est.ID+"/document-groups", bytes.NewBufferString(
		`{"environment":"homologation","grupo":"vendas","active":false}`))
	putG.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, putG)
	if rr.Code != http.StatusOK {
		t.Fatalf("put group=%d %s", rr.Code, rr.Body.String())
	}

	get = httptest.NewRequest(http.MethodGet, "/admin/v1/establishments/"+est.ID+"/document-availability?environment=homologation", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, get)
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	types, _ = body["types"].([]any)
	for _, raw := range types {
		row, _ := raw.(map[string]any)
		if row["grupo"] == "vendas" && row["available"] == true {
			t.Fatalf("vendas still available after group off: %v", row)
		}
	}

	_, err = reg.UpsertFEEnrollment(ctx, adminregistry.UpsertFEEnrollmentInput{
		TaxpayerID: tp.ID, Environment: adminregistry.EnvHomologation, Status: adminregistry.FEEnrollmentActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	// re-enable group
	_ = reg.UpsertDocGroupConfig(ctx, adminregistry.UpsertDocGroupInput{
		EstablishmentID: est.ID, Environment: adminregistry.EnvHomologation, Grupo: "vendas", Active: true,
	})
	get = httptest.NewRequest(http.MethodGet, "/admin/v1/establishments/"+est.ID+"/document-availability?environment=homologation", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, get)
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["fe_aderiu"] != true {
		t.Fatalf("fe_aderiu after active=%v", body["fe_aderiu"])
	}
	foundFT, foundGR := false, false
	for _, raw := range body["types"].([]any) {
		row := raw.(map[string]any)
		switch row["codigo_canonico"] {
		case doctype.CanonicalFT:
			foundFT = true
			if row["available"] != true {
				t.Fatalf("FT should be available with FE: %v", row)
			}
		case "bwb.ao.movimentacao.gr":
			foundGR = true
			if row["available"] == true {
				t.Fatalf("GR SAF-T-only must be unavailable when FE aderiu: %v", row)
			}
		}
	}
	if !foundFT || !foundGR {
		t.Fatal("missing FT/GR in response")
	}

	bad := httptest.NewRequest(http.MethodPut, "/admin/v1/establishments/"+est.ID+"/document-types", bytes.NewBufferString(
		`{"environment":"homologation","codigo_canonico":"invented.code","active":true}`))
	bad.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, bad)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invented code want 422 got %d", rr.Code)
	}
}
