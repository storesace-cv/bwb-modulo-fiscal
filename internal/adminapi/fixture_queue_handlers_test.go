package adminapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/agttestkit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/feboundary"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fixtruntime"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestFixtureQueueAdmin_enqueue_process_list(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "fq.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	wbPath, cleanupWB, err := agttestkit.WriteSyntheticWorkbook(t.TempDir(), agttestkit.SyntheticOptions{Count: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanupWB()

	rt, err := fixtruntime.Open(wbPath, sqlDB, persistence.DialectSQLite, fixtruntime.Config{
		MockUser: "u", MockPassword: "p", WorkerInterval: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	ref := rt.Provider.List()[0].Ref

	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	authn := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}
	mux := http.NewServeMux()
	adminapi.Mount(mux, authn, &adminapi.Handler{
		Registry:            adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:               adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:                 adminops.New(sqlDB, adminops.DialectSQLite),
		SecretsMeta:         mem,
		AuthMode:            "injected",
		FiscalEnv:           "development",
		AGTTestWorkbookPath: wbPath,
		FixtureRuntime:      rt,
	})

	body, _ := json.Marshal(map[string]any{
		"operation":       feboundary.OpObterEstado,
		"identity_ref":    ref,
		"idempotency_key": "admin-idem-1",
		"payload": map[string]any{
			"obter_estado": map[string]string{
				"taxRegistrationNumber": "9100000000",
				"requestID":             "req-admin-1",
			},
		},
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/authority/fixture-submissions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("enqueue status=%d body=%s", rr.Code, rr.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id := created["id"].(string)

	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, httptest.NewRequest(http.MethodPost, "/admin/v1/authority/fixture-submissions/process-next", nil))
	if rr2.Code != http.StatusOK {
		t.Fatalf("process status=%d body=%s", rr2.Code, rr2.Body.String())
	}

	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/fixture-submissions/"+id, nil))
	if rr3.Code != http.StatusOK {
		t.Fatalf("get status=%d", rr3.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(rr3.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["state"] != feboundary.StateOK {
		t.Fatalf("state=%v", got["state"])
	}
	if got["agt_accepted"] != false {
		t.Fatal("must not claim AGT accepted")
	}

	rr4 := httptest.NewRecorder()
	mux.ServeHTTP(rr4, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/fixture-runtime", nil))
	if rr4.Code != http.StatusOK {
		t.Fatalf("runtime status=%d", rr4.Code)
	}
}

func TestFixtureQueueAdmin_unconfigured_returns_503(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "fq2.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	mem, _ := secretstore.NewMemorySimulator(nil)
	authn := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}
	mux := http.NewServeMux()
	adminapi.Mount(mux, authn, &adminapi.Handler{
		Registry:    adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:       adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops:         adminops.New(sqlDB, adminops.DialectSQLite),
		SecretsMeta: mem, AuthMode: "injected", FiscalEnv: "development",
	})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority/fixture-submissions", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rr.Code)
	}
}
