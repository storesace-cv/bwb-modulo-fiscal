package adminapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func (h *Handler) getAuthorityMaterialLifecycle(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("profile_id")
	action := "authority_profile.material_lifecycle"
	if h.SecAdm == nil {
		h.audit(r, claims, action, "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SECADM_UNAVAILABLE", "SecAdm Unavailable")
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
	actor := secadm.Actor{SubjectID: claims.Subject}
	lc, err := prep.BuildMaterialLifecycle(p, func(ref secretstore.Ref) (secretstore.Metadata, error) {
		return h.SecAdm.Metadata(r.Context(), actor, ref)
	})
	if err != nil {
		h.writeSecAdmErr(w, r, claims, action, id, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, action, "authority_profile", id, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, lifecycleResp(lc))
}

type materialLifecycleResp struct {
	ProfileID        string                 `json:"profile_id"`
	Environment      string                 `json:"environment"`
	Status           string                 `json:"status"`
	OfflineValidated bool                   `json:"offline_validated"`
	SecretsReady     bool                   `json:"secrets_ready"`
	ExternalVerified bool                   `json:"external_verified"`
	Note             string                 `json:"note"`
	Refs             []materialLifecycleRef `json:"refs"`
}

type materialLifecycleRef struct {
	Role        string  `json:"role"`
	RefName     string  `json:"ref_name"`
	Kind        string  `json:"kind"`
	Environment string  `json:"environment"`
	SubjectID   string  `json:"subject_id"`
	Status      string  `json:"status"`
	Version     int     `json:"version"`
	Fingerprint string  `json:"fingerprint,omitempty"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

func lifecycleResp(lc prep.MaterialLifecycle) materialLifecycleResp {
	out := materialLifecycleResp{
		ProfileID: lc.ProfileID, Environment: lc.Environment, Status: lc.Status,
		OfflineValidated: lc.OfflineValidated, SecretsReady: lc.SecretsReady,
		ExternalVerified: false, Note: lc.Note,
		Refs: make([]materialLifecycleRef, 0, len(lc.Refs)),
	}
	for _, r := range lc.Refs {
		item := materialLifecycleRef{
			Role: string(r.Role), RefName: r.RefName, Kind: r.Kind,
			Environment: r.Environment, SubjectID: r.SubjectID,
			Status: r.Status, Version: r.Version, Fingerprint: r.Fingerprint,
		}
		if r.ExpiresAt != nil {
			s := r.ExpiresAt.UTC().Format(time.RFC3339Nano)
			item.ExpiresAt = &s
		}
		out.Refs = append(out.Refs, item)
	}
	return out
}

// syncProfileAfterMaterialChange invalidates offline validation and may refresh cert metadata.
func (h *Handler) syncProfileAfterMaterialChange(r *http.Request, claims adminauth.Claims, profileID string, kind, name string, meta secretstore.Metadata, revoked bool) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" || h.Registry == nil {
		return
	}
	p, err := h.Registry.GetAuthorityProfile(r.Context(), profileID)
	if err != nil {
		return
	}
	in := prep.ProfilePatchAfterMaterialChange(p, kind, name, meta, revoked)
	if _, err := h.Registry.UpdateAuthorityProfile(r.Context(), in); err != nil {
		_ = h.Audit.Record(r.Context(), claims, "authority_profile.material_sync", "authority_profile", profileID, adminaudit.ResultError, requestID(r))
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "authority_profile.material_sync", "authority_profile", profileID, adminaudit.ResultSuccess, requestID(r))
}
