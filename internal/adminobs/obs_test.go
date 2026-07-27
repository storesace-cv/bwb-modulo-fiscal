package adminobs_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminobs"
)

func TestObserverSanitizedLogsAndMetrics(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	obs := adminobs.New(log, "oidc_jwt")

	authn := adminobs.ObservingAuthenticator{
		Inner: adminauth.StaticAuthenticator{Claims: adminauth.Claims{
			Subject: "user-secret-sub", Roles: []adminauth.Role{adminauth.RoleOperator},
		}},
		Obs: obs,
	}
	h := obs.Middleware(adminauth.Middleware(authn)(adminobs.CaptureClaims(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if adminobs.RequestIDFromContext(r.Context()) == "" {
			t.Fatal("missing request_id in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/v1/ops/submissions", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token-value")
	req.Header.Set("Cookie", "fiscal_admin_session=cookie-secret")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	if rr.Header().Get("X-Request-Id") == "" {
		t.Fatal("missing response request id")
	}
	out := buf.String()
	for _, banned := range []string{
		"super-secret-token-value", "cookie-secret", "Bearer ",
		"user-secret-sub", "Authorization", "fiscal_admin_session=",
	} {
		if strings.Contains(out, banned) {
			t.Fatalf("log leaked %q: %s", banned, out)
		}
	}
	if !strings.Contains(out, `"route_class":"ops"`) || !strings.Contains(out, `"roles":"operator"`) {
		t.Fatalf("log missing fields: %s", out)
	}

	snap := obs.Snapshot()
	if snap.AuthOK < 1 || snap.AuthMode != "oidc_jwt" {
		t.Fatalf("%+v", snap)
	}
	found := false
	for _, s := range snap.Series {
		if s.RouteClass == "ops" && s.Outcome == "ok" && s.Count >= 1 {
			found = true
		}
		if strings.Contains(s.RouteClass, "/") || strings.Contains(s.RouteClass, "taxpayer") {
			// taxpayers is ok as class name; path with id is not
		}
	}
	if !found {
		t.Fatalf("series: %+v", snap.Series)
	}
}

func TestHealthReadyMetricsHandlers(t *testing.T) {
	obs := adminobs.New(nil, "fail_closed")
	mux := http.NewServeMux()
	mux.Handle("GET /admin/v1/health", obs.Middleware(adminobs.HealthHandler("1.0", "rev")))
	mux.Handle("GET /admin/v1/ready", obs.Middleware(adminobs.ReadyHandler(adminobs.ReadyDeps{
		AuthMode: "fail_closed", Version: "1.0", Revision: "rev",
	})))
	mux.Handle("GET /admin/v1/ops/metrics", obs.Middleware(adminauth.Middleware(adminauth.StaticAuthenticator{
		Claims: adminauth.Claims{Subject: "op", Roles: []adminauth.Role{adminauth.RoleOperator}},
	})(adminauth.RequirePermission(adminauth.PermOpsRead)(adminobs.MetricsHandler(obs)))))

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health %d", rr.Code)
	}
	var health map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &health)
	if health["status"] != "ok" || health["component"] != "admin" {
		t.Fatalf("%v", health)
	}

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/ready", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("ready without DB want 503 got %d %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, banned := range []string{"postgres://", "password", "Bearer", "nif"} {
		if strings.Contains(strings.ToLower(body), banned) {
			t.Fatalf("ready leaked %q", banned)
		}
	}

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/admin/v1/ops/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics %d", rr.Code)
	}
}

func TestClassifyPathBounded(t *testing.T) {
	if adminobs.ClassifyPath("/admin/v1/taxpayers/abc-123") != adminobs.RouteTaxpayers {
		t.Fatal("taxpayer id must collapse")
	}
	if adminobs.ClassifyPath("/admin/v1/secadm/secret-refs") != adminobs.RouteSecAdm {
		t.Fatal("secadm separate")
	}
	if adminobs.ClassifyPath("/admin/ui/taxpayers") != adminobs.RouteUI {
		t.Fatal("ui")
	}
}
