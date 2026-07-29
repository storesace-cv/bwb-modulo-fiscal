package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func (h *Handler) getAuthoritySecretStoreStatus(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	runtimeMode := ""
	if h.SecretsMeta != nil {
		if v, ok := h.SecretsMeta.(interface{ StorageMode() string }); ok {
			runtimeMode = v.StorageMode()
		}
	}
	st := secretstore.BuildVaultStatusFromEnv(h.FiscalEnv, runtimeMode)
	_ = h.Audit.Record(r.Context(), claims, "authority.secretstore_status", "authority", "secretstore", adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"fiscal_env":             st.FiscalEnv,
		"backend_declared":       st.BackendDeclared,
		"master_key_configured":  st.MasterKeyConfigured,
		"master_key_parse_ok":    st.MasterKeyParseOK,
		"master_key_fingerprint": st.MasterKeyFingerprint,
		"storage_mode_runtime":   st.StorageModeRuntime,
		"durable_required":       st.DurableRequired,
		"ready_for_homologation": st.ReadyForHomologation,
		"cipher_algorithm":       st.CipherAlgorithm,
		"external_verified":      false,
		"notes":                  st.Notes,
	})
}
