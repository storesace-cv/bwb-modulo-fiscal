package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminobs"
)

// getAuthorityReadiness returns the canonical 4-flag checklist (RM-AGTPREP-007).
// external_verified is always false. No secrets in response.
func (h *Handler) getAuthorityReadiness(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("profile_id")
	p, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		h.writeRegistryErr(w, r, claims, "authority.readiness", "authority_profile", id, err)
		return
	}
	ext := false
	checklistComplete := p.ConfigReady && p.SecretsReady && p.OfflineValidated && !ext
	_ = h.Audit.Record(r.Context(), claims, "authority.readiness", "authority_profile", id, adminaudit.ResultSuccess, requestID(r))
	if h.Obs != nil {
		if checklistComplete {
			h.Obs.Inc(adminobs.RouteAuthority, r.Method, adminobs.OutcomeOK)
		} else {
			h.Obs.Inc(adminobs.RouteAuthority, r.Method, adminobs.OutcomeValidation)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profile_id":         p.ID,
		"environment":        p.Environment,
		"status":             p.Status,
		"config_ready":       p.ConfigReady,
		"secrets_ready":      p.SecretsReady,
		"offline_validated":  p.OfflineValidated,
		"external_verified":  false,
		"checklist_complete": checklistComplete,
		"rbac": map[string]any{
			"read":  "ops.read",
			"write": "secadm.write + owner subject allowlist",
		},
		"notes": []string{
			"external_verified=false até probe AGT real (GAP-006)",
			"métricas sem labels sensíveis (route=authority)",
			"auditoria append-only em authority.readiness",
		},
	})
}
