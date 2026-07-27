package prep

import (
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

// AlertSeverity is a sanitized readiness alert level (never secrets).
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityBlocking AlertSeverity = "blocking"
)

// ReadinessAlert is a non-secret operational warning for AuthorityProfile prep.
type ReadinessAlert struct {
	Code     string        `json:"code"`
	Severity AlertSeverity `json:"severity"`
	Message  string        `json:"message"`
}

// BuildReadinessAlerts returns sanitized alerts for checklist/expiry/refs.
// external_verified is never treated as true. now may be zero → UTC now.
func BuildReadinessAlerts(p adminregistry.AuthorityProfile, now time.Time) []ReadinessAlert {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	var out []ReadinessAlert
	if !p.ConfigReady {
		out = append(out, ReadinessAlert{
			Code: "config_not_ready", Severity: SeverityBlocking,
			Message: "config_ready=false — completar metadados do perfil",
		})
	}
	if !p.SecretsReady {
		out = append(out, ReadinessAlert{
			Code: "secrets_not_ready", Severity: SeverityBlocking,
			Message: "secrets_ready=false — importar material SecAdm e sync",
		})
	}
	if !p.OfflineValidated {
		out = append(out, ReadinessAlert{
			Code: "offline_not_validated", Severity: SeverityBlocking,
			Message: "offline_validated=false — correr validação offline chave-cert",
		})
	}
	out = append(out, ReadinessAlert{
		Code: "external_verified_false", Severity: SeverityInfo,
		Message: "external_verified permanece false (≠ AGT real / GAP-006)",
	})
	if strings.TrimSpace(p.CertificateRef) == "" || strings.TrimSpace(p.ProducerKeyRef) == "" {
		out = append(out, ReadinessAlert{
			Code: "refs_incomplete", Severity: SeverityWarning,
			Message: "refs lógicas de chave/certificado incompletas",
		})
	}
	if p.ExpiresAt != nil {
		exp := p.ExpiresAt.UTC()
		if !exp.After(now) {
			out = append(out, ReadinessAlert{
				Code: "certificate_expired", Severity: SeverityBlocking,
				Message: "validade do certificado (metadado) já passou — rotacionar",
			})
		} else if exp.Sub(now) <= 30*24*time.Hour {
			out = append(out, ReadinessAlert{
				Code: "certificate_expiring_soon", Severity: SeverityWarning,
				Message: "validade do certificado (metadado) ≤ 30 dias — planear rotação",
			})
		}
	}
	if p.Status == adminregistry.AuthorityStatusActive &&
		(!p.ConfigReady || !p.SecretsReady || !p.OfflineValidated) {
		out = append(out, ReadinessAlert{
			Code: "active_inconsistent", Severity: SeverityBlocking,
			Message: "status=active sem readiness local completa (estado inconsistente)",
		})
	}
	return out
}
