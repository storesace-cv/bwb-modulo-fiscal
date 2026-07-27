package adminauth_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

func TestRBACMatrix(t *testing.T) {
	roles := []adminauth.Role{
		adminauth.RoleOwner, adminauth.RoleAdmin, adminauth.RoleOperator, adminauth.RoleAuditor,
	}
	perms := []adminauth.Permission{
		adminauth.PermCadastroWrite, adminauth.PermCadastroRead,
		adminauth.PermOpsRead, adminauth.PermOpsWrite, adminauth.PermAuditRead,
		adminauth.PermSecretMetaRead, adminauth.PermSecAdmWrite,
	}
	want := map[adminauth.Role]map[adminauth.Permission]bool{
		adminauth.RoleOwner: {
			adminauth.PermCadastroWrite: true, adminauth.PermCadastroRead: true,
			adminauth.PermOpsRead: true, adminauth.PermOpsWrite: true,
			adminauth.PermAuditRead: true, adminauth.PermSecretMetaRead: true, adminauth.PermSecAdmWrite: true,
		},
		adminauth.RoleAdmin: {
			adminauth.PermCadastroWrite: true, adminauth.PermCadastroRead: true,
			adminauth.PermOpsRead: true, adminauth.PermOpsWrite: true,
			adminauth.PermAuditRead: true, adminauth.PermSecretMetaRead: true, adminauth.PermSecAdmWrite: false,
		},
		adminauth.RoleOperator: {
			adminauth.PermCadastroWrite: false, adminauth.PermCadastroRead: true,
			adminauth.PermOpsRead: true, adminauth.PermOpsWrite: false,
			adminauth.PermAuditRead: true, adminauth.PermSecretMetaRead: true, adminauth.PermSecAdmWrite: false,
		},
		adminauth.RoleAuditor: {
			adminauth.PermCadastroWrite: false, adminauth.PermCadastroRead: true,
			adminauth.PermOpsRead: true, adminauth.PermOpsWrite: false,
			adminauth.PermAuditRead: true, adminauth.PermSecretMetaRead: true, adminauth.PermSecAdmWrite: false,
		},
	}
	for _, role := range roles {
		for _, perm := range perms {
			role, perm := role, perm
			t.Run(fmt.Sprintf("%s/%s", role, perm), func(t *testing.T) {
				c := adminauth.Claims{Subject: "s", Roles: []adminauth.Role{role}}
				got := adminauth.Allows(c, perm)
				if got != want[role][perm] {
					t.Errorf("got=%v want=%v", got, want[role][perm])
				}
			})
		}
	}
	t.Run("empty_subject_deny", func(t *testing.T) {
		if adminauth.Allows(adminauth.Claims{Roles: []adminauth.Role{adminauth.RoleOwner}}, adminauth.PermOpsRead) {
			t.Error("empty subject must deny")
		}
	})
	t.Run("no_secret_reveal", func(t *testing.T) {
		if adminauth.Allows(adminauth.Claims{Subject: "s", Roles: []adminauth.Role{adminauth.RoleOwner}}, adminauth.Permission("secret.reveal")) {
			t.Error("secret.reveal must never be granted")
		}
	})
}

func TestRequirePermissionEnforcesMatrix(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	authn := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "op", Roles: []adminauth.Role{adminauth.RoleOperator},
	}}
	h := adminauth.Middleware(authn)(adminauth.RequirePermission(adminauth.PermCadastroWrite)(okHandler))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", nil))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("operator write want 403 got %d", rr.Code)
	}

	hRead := adminauth.Middleware(authn)(adminauth.RequirePermission(adminauth.PermOpsRead)(okHandler))
	rr = httptest.NewRecorder()
	hRead.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("operator ops.read want 204 got %d", rr.Code)
	}
}
