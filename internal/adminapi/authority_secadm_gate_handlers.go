package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

func (h *Handler) getAuthoritySecAdmGateStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	present := h.SecAdm != nil
	status := "absent"
	if present {
		status = "present"
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.secadm_gate_status", "authority", "secadm", adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"secadm_gate":       status,
		"secadm_gate_ready": present,
		"external_verified": false,
		"notes": []string{
			"present exige FISCAL_ADMIN_OWNER_SUBJECT no runtime (valor nunca exposto)",
			"≠ AGT; write-only material continua em /admin/v1/secadm/*",
			"admin auth fail_closed sem IdP → UI/API owner ainda indisponível em sandbox",
		},
	})
}
