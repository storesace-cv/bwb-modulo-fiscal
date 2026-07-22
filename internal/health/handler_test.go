package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/health"
)

func TestHandlerGETReturnsValidJSON(t *testing.T) {
	h := health.NewHandler("0.1.0", "2f96fe45c0d8ad3cb2e21d8755f2988eb4a43dfd", "AO-PKG-1")
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var body health.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if body.Status != health.StatusOK {
		t.Fatalf("status = %q, want %q", body.Status, health.StatusOK)
	}
	if body.Version != "0.1.0" {
		t.Fatalf("version = %q", body.Version)
	}
	if body.Revision != "2f96fe45c0d8ad3cb2e21d8755f2988eb4a43dfd" {
		t.Fatalf("revision = %q", body.Revision)
	}
	if body.FiscalPackage != "AO-PKG-1" {
		t.Fatalf("fiscalPackage = %q", body.FiscalPackage)
	}
}

func TestHandlerRejectsNonGET(t *testing.T) {
	h := health.NewHandler("0.1.0", "dev", "AO-PKG-1")
	req := httptest.NewRequest(http.MethodPost, "/v1/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", rec.Header().Get("Allow"))
	}
}
