package adminapi_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestSecAdmValidateOfflineNoPlaintext(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "secadm-offline.db")
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
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	h := &adminapi.Handler{
		Registry: reg, Audit: adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil),
		Ops: adminops.New(sqlDB, adminops.DialectSQLite), SecretsMeta: mem, SecAdm: gate,
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(9), Subject: pkix.Name{CommonName: "offline-api"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, IsCA: true, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyRef := secretstore.Ref{Kind: "producer_key", Environment: "homologation", SubjectID: "platform", Name: "k1"}
	certRef := secretstore.Ref{Kind: "certificate", Environment: "homologation", SubjectID: "platform", Name: "c1"}
	if _, err := mem.Put(ctx, keyRef, keyPEM, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := mem.Put(ctx, certRef, certPEM, nil); err != nil {
		t.Fatal(err)
	}
	prof, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: "homologation", DisplayName: "offline profile", Status: "draft",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := map[string]string{
		"key_kind": "producer_key", "key_environment": "homologation", "key_subject_id": "platform", "key_name": "k1",
		"cert_kind": "certificate", "cert_environment": "homologation", "cert_subject_id": "platform", "cert_name": "c1",
		"profile_id": prof.ID,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/v1/secadm/material/validate-offline", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	out := rr.Body.String()
	if strings.Contains(out, "BEGIN ") || strings.Contains(out, string(keyPEM)) {
		t.Fatal("plaintext leaked")
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["ok"] != true || resp["external_verified"] != false || resp["offline_validated_set"] != true {
		t.Fatalf("%v", resp)
	}
	updated, err := reg.GetAuthorityProfile(ctx, prof.ID)
	if err != nil || !updated.OfflineValidated || updated.ExternalVerified {
		t.Fatalf("%+v %v", updated, err)
	}
}
