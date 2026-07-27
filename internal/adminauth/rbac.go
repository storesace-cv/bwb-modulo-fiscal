// RBAC permission matrix for Admin API plano A / SecAdm (RM-BO-004 / DEC-BO-002).
// MFA de operador fica adiado até IdP real; este pacote não implementa login.
package adminauth

import "strings"

// Permission is a coarse capability (never a secret ACL).
type Permission string

const (
	// PermCadastroWrite creates/updates non-secret cadastros (taxpayers, bindings config).
	PermCadastroWrite Permission = "cadastro.write"
	// PermCadastroRead reads cadastros.
	PermCadastroRead Permission = "cadastro.read"
	// PermOpsRead reads ops submissions / reconciliation summaries (no secret bodies).
	PermOpsRead Permission = "ops.read"
	// PermOpsWrite mutates ops queue actions (retry/cancel/manual_review); owner|admin only.
	PermOpsWrite Permission = "ops.write"
	// PermAuditRead reads admin audit events.
	PermAuditRead Permission = "audit.read"
	// PermSecretMetaRead reads sanitized secret-ref metadata only (no plaintext).
	PermSecretMetaRead Permission = "secret_meta.read"
	// PermSecAdmWrite provisions/rotates/revokes secrets (owner-only; plano B).
	PermSecAdmWrite Permission = "secadm.write"
)

// Allows reports whether any claim role grants permission.
// Fail-closed: unknown permission or empty roles → false.
// There is intentionally no PermSecretReveal — plaintext ACL does not exist.
func Allows(claims Claims, perm Permission) bool {
	if len(claims.Roles) == 0 || strings.TrimSpace(claims.Subject) == "" {
		return false
	}
	for _, role := range claims.Roles {
		if roleAllows(role, perm) {
			return true
		}
	}
	return false
}

func roleAllows(role Role, perm Permission) bool {
	switch role {
	case RoleOwner:
		// Owner: plano A completo + SecAdm write. Still no plaintext read ACL.
		switch perm {
		case PermCadastroWrite, PermCadastroRead, PermOpsRead, PermOpsWrite, PermAuditRead, PermSecretMetaRead, PermSecAdmWrite:
			return true
		default:
			return false
		}
	case RoleAdmin:
		switch perm {
		case PermCadastroWrite, PermCadastroRead, PermOpsRead, PermOpsWrite, PermAuditRead, PermSecretMetaRead:
			return true
		case PermSecAdmWrite:
			return false
		default:
			return false
		}
	case RoleOperator, RoleAuditor:
		switch perm {
		case PermCadastroRead, PermOpsRead, PermAuditRead, PermSecretMetaRead:
			return true
		case PermCadastroWrite, PermOpsWrite, PermSecAdmWrite:
			return false
		default:
			return false
		}
	default:
		return false
	}
}
