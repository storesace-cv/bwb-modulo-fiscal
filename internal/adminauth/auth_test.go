package adminauth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

func TestRequireRoleAndFailClosed(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	authn := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "u1", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}
	h := adminauth.Middleware(authn)(adminauth.RequireAnyRole(adminauth.RoleAdmin)(okHandler))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("got %d", rr.Code)
	}

	h2 := adminauth.Middleware(adminauth.FailClosedAuthenticator{})(okHandler)
	rr = httptest.NewRecorder()
	h2.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rr.Code)
	}
}
