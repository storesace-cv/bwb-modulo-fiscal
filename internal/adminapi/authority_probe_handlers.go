package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
)

func (h *Handler) probeAuthorityConfig(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	action := "authority.probe_config"
	mode := h.AuthorityMode
	if mode == "" {
		mode = "simulator"
	}
	if err := prep.FailClosedProduction(h.FiscalEnv, mode); err != nil {
		h.audit(r, claims, action, "authority", mode, adminaudit.ResultDenied)
		writeProblem(w, r, http.StatusForbidden, "ADMIN_AUTHORITY_FAIL_CLOSED", "Forbidden")
		return
	}
	client := simulator.New(simulator.OutcomeAccept)
	rep, err := prep.ProbeSimulator(r.Context(), mode, client)
	if err != nil {
		h.audit(r, claims, action, "authority", mode, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	_ = h.Audit.Record(r.Context(), claims, action, "authority", mode, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                   rep.OK,
		"mode":                 rep.Mode,
		"simulator_reachable":  rep.SimulatorReachable,
		"outcome":              rep.Outcome,
		"authority_request_id": rep.AuthorityRequestID,
		"external_verified":    false,
		"notes":                rep.Notes,
		"probed_at":            rep.ProbedAt,
	})
}
