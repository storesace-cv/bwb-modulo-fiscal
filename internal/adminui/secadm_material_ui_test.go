package adminui_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminui"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestAdminUISecAdmMaterialOwnerOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "adminui-material.db")
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
	gate, err := secadm.NewGate("owner-ui", mem)
	if err != nil {
		t.Fatal(err)
	}
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)
	h, err := adminui.New(reg, "development")
	if err != nil {
		t.Fatal(err)
	}
	h.SecAdm = gate
	h.SecretsMeta = mem

	mux := http.NewServeMux()
	adminui.Mount(mux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-ui", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}, h)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/secadm/material", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "efémera") || strings.Contains(rr.Body.String(), "-----BEGIN") {
		t.Fatalf("form body: %s", rr.Body.String())
	}
	csrf := csrfFromSetCookie(rr)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("csrf_token", csrf)
	_ = w.WriteField("action", "put")
	_ = w.WriteField("kind", "producer_key")
	_ = w.WriteField("encoding", "pem")
	_ = w.WriteField("environment", "homologation")
	_ = w.WriteField("subject_id", "platform")
	_ = w.WriteField("name", "ui-key")
	part, err := w.CreateFormFile("material", "key.pem")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(pemKey)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/admin/ui/secadm/material", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "fiscal_admin_csrf", Value: csrf})
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("put %d %s", rr.Code, rr.Body.String())
	}
	out := rr.Body.String()
	if strings.Contains(out, "BEGIN ") || strings.Contains(out, string(pemKey)) {
		t.Fatal("PEM leaked in HTML")
	}
	if !strings.Contains(out, "Fingerprint") || !strings.Contains(out, "pem_private_key") {
		t.Fatalf("missing meta: %s", out)
	}

	adminMux := http.NewServeMux()
	adminui.Mount(adminMux, adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-ui", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}, h)
	rr = httptest.NewRecorder()
	adminMux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/ui/secadm/material", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("admin want 403 got %d", rr.Code)
	}
}
