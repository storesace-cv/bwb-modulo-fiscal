package adminapi_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"mime/multipart"
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

func TestSecAdmMaterialMultipartNoPlaintextLeak(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "secadm-material.db")
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
	audit := adminaudit.New(sqlDB, adminaudit.DialectSQLite, nil)
	h := &adminapi.Handler{
		Registry:    adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil),
		Audit:       audit,
		Ops:         adminops.New(sqlDB, adminops.DialectSQLite),
		SecretsMeta: mem,
		SecAdm:      gate,
	}
	mux := http.NewServeMux()
	adminapi.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{
		Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	marker := "MATERIAL-SECRET-MARKER-NOT-REAL"
	// embed marker only in unused padding? Better: use credential path for leak test
	_ = marker

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("kind", "producer_key")
	_ = w.WriteField("environment", "homologation")
	_ = w.WriteField("subject_id", "platform")
	_ = w.WriteField("name", "agt-key")
	_ = w.WriteField("encoding", "pem")
	part, err := w.CreateFormFile("material", "key.pem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(pemKey); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/secadm/material", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("put material %d %s", rr.Code, rr.Body.String())
	}
	out := rr.Body.String()
	if strings.Contains(out, "BEGIN ") || strings.Contains(out, string(pemKey)) {
		t.Fatal("PEM leaked in response")
	}
	var meta map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["format_note"] != "pem_private_key" || meta["fingerprint"] == "" {
		t.Fatalf("%v", meta)
	}

	// credential with marker — must not appear in response or audit resource
	var body2 bytes.Buffer
	w2 := multipart.NewWriter(&body2)
	_ = w2.WriteField("kind", "producer_credential")
	_ = w2.WriteField("environment", "homologation")
	_ = w2.WriteField("subject_id", "platform")
	_ = w2.WriteField("name", "agt-cred")
	_ = w2.WriteField("encoding", "credential")
	part2, _ := w2.CreateFormFile("material", "cred.txt")
	_, _ = part2.Write([]byte(marker))
	_ = w2.Close()
	req = httptest.NewRequest(http.MethodPost, "/admin/v1/secadm/material", &body2)
	req.Header.Set("Content-Type", w2.FormDataContentType())
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("cred %d %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), marker) {
		t.Fatal("credential leaked")
	}
	events, err := audit.ListRecent(ctx, 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if strings.Contains(e.ResourceID, marker) || strings.Contains(e.Action, marker) {
			t.Fatalf("audit leak: %+v", e)
		}
	}

	// oversized reject
	huge := bytes.Repeat([]byte("a"), secretstore.MaxCredentialBytes+1)
	var body3 bytes.Buffer
	w3 := multipart.NewWriter(&body3)
	_ = w3.WriteField("kind", "producer_credential")
	_ = w3.WriteField("environment", "homologation")
	_ = w3.WriteField("subject_id", "platform")
	_ = w3.WriteField("name", "too-big")
	_ = w3.WriteField("encoding", "credential")
	part3, _ := w3.CreateFormFile("material", "big.txt")
	_, _ = part3.Write(huge)
	_ = w3.Close()
	req = httptest.NewRequest(http.MethodPost, "/admin/v1/secadm/material", &body3)
	req.Header.Set("Content-Type", w3.FormDataContentType())
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("oversized want 422 got %d", rr.Code)
	}
}
