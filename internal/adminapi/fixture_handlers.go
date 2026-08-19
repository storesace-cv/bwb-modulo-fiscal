package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
)

func (h *Handler) getAuthorityFixtureIdentities(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	configured, refs, err := prep.FixtureIdentityCatalog(h.AGTTestWorkbookPath)
	if err != nil {
		h.audit(r, claims, "authority.fixture_identities", "authority", "workbook", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	items := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		items = append(items, map[string]any{
			"ref":       ref.Ref,
			"algorithm": ref.Algorithm,
			"rsa_bits":  ref.RSABits,
			"role":      ref.Role,
		})
	}
	_ = h.Audit.Record(r.Context(), claims, "authority.fixture_identities", "authority", "workbook", adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"workbook_configured": configured,
		"count":               len(items),
		"identities":          items,
		"external_verified":   false,
		"mock_only":           true,
		"note":                "sanitized refs only; ≠ AGT acceptance; wire JWS blocked (C-FE-JWS-TYP-001)",
	})
}

func (h *Handler) getAuthorityFixtureHub(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	view := prep.FixtureHubView()
	_ = h.Audit.Record(r.Context(), claims, "authority.fixture_hub", "authority", string(view.Kind), adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":              view.Kind,
		"transport_allowed": view.TransportAllowed,
		"external_verified": view.ExternalVerified,
		"slots":             view.Slots,
		"note":              view.Note,
		"mock_only":         true,
	})
}
