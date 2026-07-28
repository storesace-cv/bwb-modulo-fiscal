package landing_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/landing"
)

func TestRootLandingOK(t *testing.T) {
	h := landing.NewHandler()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content-type %q", ct)
	}
	body := rr.Body.String()
	for _, needle := range []string{"/v1/health", "/admin/ui/", "404 na raiz", "homologação oficial AGT"} {
		if !strings.Contains(body, needle) {
			t.Fatalf("missing %q", needle)
		}
	}
	for _, bad := range []string{"password", "Bearer ", "NIF", "BEGIN RSA"} {
		if strings.Contains(body, bad) {
			t.Fatalf("must not contain %q", bad)
		}
	}
	if !landing.MentionsAvailabilityConfusion() {
		t.Fatal("disclaimer helper")
	}
}

func TestRootLandingMethodNotAllowed(t *testing.T) {
	h := landing.NewHandler()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestMuxExactRoot(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", landing.NewHandler())
	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "sandbox") {
		t.Fatalf("root: %d %q", rr.Code, rr.Body.String())
	}

	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/v1/health", nil))
	if rr2.Code != http.StatusOK || !strings.Contains(rr2.Body.String(), `"ok"`) {
		t.Fatalf("health shadowed: %d %q", rr2.Code, rr2.Body.String())
	}

	rr3 := httptest.NewRecorder()
	mux.ServeHTTP(rr3, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rr3.Code != http.StatusNotFound {
		t.Fatalf("unknown path want 404 got %d", rr3.Code)
	}
}
