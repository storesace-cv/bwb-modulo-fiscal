package adminapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func (h *Handler) getAuthorityBindingValidation(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("profile_id")
	action := "authority_profile.binding_validation"
	if h.SecretsMeta == nil {
		h.audit(r, claims, action, "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SECRETS_META_UNAVAILABLE", "Secrets Metadata Unavailable")
		return
	}
	p, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		h.audit(r, claims, action, "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	v, err := prep.ValidateProfileBindings(r.Context(), p, h.secretLookup(r.Context()))
	if err != nil {
		h.audit(r, claims, action, "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	_ = h.Audit.Record(r.Context(), claims, action, "authority_profile", id, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, map[string]any{
		"profile_id":        v.ProfileID,
		"environment":       v.Environment,
		"status":            v.Status,
		"secrets_ready":     v.SecretsReady,
		"external_verified": false,
		"valid":             v.Valid,
		"issues":            v.Issues,
		"ops_path_statuses": v.OpsPathStatuses,
		"note":              v.Note,
	})
}

func (h *Handler) secretLookup(ctx context.Context) prep.MetadataLookup {
	return func(ref secretstore.Ref) (secretstore.Metadata, error) {
		return h.SecretsMeta.Metadata(ctx, ref)
	}
}

func (h *Handler) assertProfileBindings(ctx context.Context, p adminregistry.AuthorityProfile) error {
	lookup := func(ref secretstore.Ref) (secretstore.Metadata, error) {
		return secretstore.Metadata{Ref: ref, Status: secretstore.StatusAbsent, Environment: ref.Environment}, nil
	}
	if h.SecretsMeta != nil {
		lookup = h.secretLookup(ctx)
	} else if p.SecretsReady || p.Status == adminregistry.AuthorityStatusActive {
		return fmt.Errorf("%w: SecretStore indisponível para validar bindings", adminregistry.ErrValidation)
	}
	return prep.AssertBindingsForWrite(ctx, p, lookup)
}

// previewAuthorityPatch merges update input onto current profile (no persistence).
func previewAuthorityPatch(cur adminregistry.AuthorityProfile, in adminregistry.UpdateAuthorityProfileInput) adminregistry.AuthorityProfile {
	out := cur
	if in.DisplayName != "" {
		out.DisplayName = in.DisplayName
	}
	if in.Status != "" {
		out.Status = in.Status
	}
	if in.AllowedOperations != nil {
		out.AllowedOperations = in.AllowedOperations
	}
	if in.PendingExternal != nil {
		out.PendingExternal = in.PendingExternal
	}
	if in.ProducerCredentialRef != "" {
		out.ProducerCredentialRef = in.ProducerCredentialRef
	}
	if in.ProducerKeyRef != "" {
		out.ProducerKeyRef = in.ProducerKeyRef
	}
	if in.CertificateRef != "" {
		out.CertificateRef = in.CertificateRef
	}
	if in.AlgorithmDeclared != "" {
		out.AlgorithmDeclared = in.AlgorithmDeclared
	}
	if in.ConfigReady != nil {
		out.ConfigReady = *in.ConfigReady
	}
	if in.SecretsReady != nil {
		out.SecretsReady = *in.SecretsReady
	}
	if in.OfflineValidated != nil {
		out.OfflineValidated = *in.OfflineValidated
	}
	out.ExternalVerified = false
	return out
}
