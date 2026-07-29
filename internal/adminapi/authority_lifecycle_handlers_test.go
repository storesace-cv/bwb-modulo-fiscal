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

func TestAuthorityMaterialLifecycle(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "lifecycle.db")
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
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	p, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: adminregistry.EnvHomologation, DisplayName: "lc",
		CertificateRef: "agt-cert", ProducerKeyRef: "agt-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	trueVal := true
	if _, err := reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID: p.ID, OfflineValidated: &trueVal, SecretsReady: &trueVal, ConfigReady: &trueVal,
	}); err != nil {
		t.Fatal(err)
	}

	ref := secretstore.Ref{
		Kind: secretstore.KindCertificate, Environment: secretstore.EnvHomologation,
		SubjectID: "platform", Name: "agt-cert",
	}
	if _, err := gate.Put(ctx, secadm.Actor{SubjectID: "owner-1"}, ref, []byte("cert-v1-material"), nil); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/authority-profiles/"+p.ID+"/material-lifecycle", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["external_verified"] != false {
		t.Fatalf("%v", body)
	}
	refs, _ := body["refs"].([]any)
	if len(refs) != 3 {
		t.Fatalf("refs=%v", refs)
	}

	rotateBody := `{
		"kind":"certificate","environment":"homologation","subject_id":"platform","name":"agt-cert",
		"plaintext":"cert-v2-rotated-material",
		"authority_profile_id":"` + p.ID + `"
	}`
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/secadm/secret-refs/rotate", strings.NewReader(rotateBody))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("rotate %d %s", rr.Code, rr.Body.String())
	}
	got, err := reg.GetAuthorityProfile(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OfflineValidated {
		t.Fatal("offline_validated must be cleared after rotate")
	}
	if got.SecretsReady || got.ExternalVerified {
		t.Fatalf("secrets_ready must clear after rotate: %+v", got)
	}
	if got.FingerprintSanitized == "" || !strings.HasPrefix(got.FingerprintSanitized, "sha256:") {
		t.Fatalf("fingerprint sync: %q", got.FingerprintSanitized)
	}
}
