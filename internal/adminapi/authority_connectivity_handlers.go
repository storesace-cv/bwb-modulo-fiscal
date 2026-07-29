package adminapi

import (
	"net/http"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
)

func (h *Handler) getAuthorityConnectivityStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	mode := h.AuthorityMode
	if mode == "" {
		mode = "simulator"
	}
	var lastAt *time.Time
	lastResult, lastMode := "", ""
	if h.Audit != nil {
		evs, err := h.Audit.ListByResource(r.Context(), "authority", mode, 20)
		if err == nil {
			for _, e := range evs {
				if e.Action == "authority.probe_config" {
					t := e.OccurredAt.UTC()
					lastAt = &t
					lastResult = e.Result
					lastMode = e.ResourceID
					break
				}
			}
		}
	}
	st := prep.BuildConnectivityStatus(h.FiscalEnv, mode, lastResult, lastMode, lastAt)
	_ = h.Audit.Record(r.Context(), claims, "authority.connectivity_status", "authority", mode, adminaudit.ResultSuccess, requestID(r))
	resp := map[string]any{
		"fiscal_env":        st.FiscalEnv,
		"authority_mode":    st.AuthorityMode,
		"status":            st.Status,
		"external_verified": false,
		"simulator_allowed": st.SimulatorAllowed,
		"last_probe_result": st.LastProbeResult,
		"last_probe_mode":   st.LastProbeMode,
		"notes":             st.Notes,
	}
	if st.LastProbeAt != nil {
		resp["last_probe_at"] = st.LastProbeAt.UTC().Format(time.RFC3339Nano)
	}
	writeJSON(w, http.StatusOK, resp)
}
