package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fepath"
)

func (h *Handler) getAuthorityEndpointCatalog(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	env := r.URL.Query().Get("environment")
	rows, err := prep.EndpointCatalog(env)
	if err != nil {
		h.audit(r, claims, "authority.endpoint_catalog", "authority", env, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]any{
			"operation":    row.Operation,
			"environment":  row.Environment,
			"path_status":  row.PathStatus,
			"declared_url": row.DeclaredURL,
			"host":         row.Host,
			"conflict_id":  row.ConflictID,
			"source_note":  row.SourceNote,
			"bindable":     row.Bindable,
		})
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.endpoint_catalog", "authority", env, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"environment":       env,
		"conflict_open":     fepath.ConflictOpen,
		"external_verified": false,
		"items":             items,
	})
}

func (h *Handler) getAuthorityJWSProfileScaffold(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	s := prep.JWSProfileScaffoldDefault()
	_ = h.Audit.Record(r.Context(), claims, "authority.jws_scaffold", "authority", "jws", adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"algorithm_declared":        s.AlgorithmDeclared,
		"claims_status":             s.ClaimsStatus,
		"mechanism_id":              s.MechanismID,
		"saft_mechanism_separated":  s.SAFTMechanismSeparated,
		"external_verified":         false,
		"invented_claims_forbidden": s.InventedClaimsForbidden,
		"source_note":               s.SourceNote,
		"conflict_separation_note":  s.ConflictSeparationNote,
	})
}
