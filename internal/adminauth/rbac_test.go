package adminauth_test

import (
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

func TestRBACMatrix(t *testing.T) {
	cases := []struct {
		role adminauth.Role
		perm adminauth.Permission
		want bool
	}{
		{adminauth.RoleOwner, adminauth.PermSecAdmWrite, true},
		{adminauth.RoleOwner, adminauth.PermCadastroWrite, true},
		{adminauth.RoleAdmin, adminauth.PermCadastroWrite, true},
		{adminauth.RoleAdmin, adminauth.PermSecAdmWrite, false},
		{adminauth.RoleOperator, adminauth.PermOpsRead, true},
		{adminauth.RoleOperator, adminauth.PermCadastroWrite, false},
		{adminauth.RoleOperator, adminauth.PermSecAdmWrite, false},
		{adminauth.RoleAuditor, adminauth.PermAuditRead, true},
		{adminauth.RoleAuditor, adminauth.PermSecretMetaRead, true},
		{adminauth.RoleAuditor, adminauth.PermSecAdmWrite, false},
		{adminauth.RoleAuditor, adminauth.PermCadastroWrite, false},
	}
	for _, tc := range cases {
		c := adminauth.Claims{Subject: "s", Roles: []adminauth.Role{tc.role}}
		got := adminauth.Allows(c, tc.perm)
		if got != tc.want {
			t.Fatalf("role=%s perm=%s got=%v want=%v", tc.role, tc.perm, got, tc.want)
		}
	}
	// Fail-closed: empty subject / unknown perm / no secret reveal permission exists.
	if adminauth.Allows(adminauth.Claims{Roles: []adminauth.Role{adminauth.RoleOwner}}, adminauth.PermOpsRead) {
		t.Fatal("empty subject must deny")
	}
	if adminauth.Allows(adminauth.Claims{Subject: "s", Roles: []adminauth.Role{adminauth.RoleOwner}}, adminauth.Permission("secret.reveal")) {
		t.Fatal("secret.reveal must never be granted")
	}
}
