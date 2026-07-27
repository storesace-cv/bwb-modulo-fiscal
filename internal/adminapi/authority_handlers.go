package adminapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

type createAuthorityProfileReq struct {
	Environment           string            `json:"environment"`
	TaxpayerID            string            `json:"taxpayer_id"`
	ScopeID               string            `json:"scope_id"`
	DisplayName           string            `json:"display_name"`
	Status                string            `json:"status"`
	AllowedOperations     []string          `json:"allowed_operations"`
	PendingExternal       map[string]string `json:"pending_external"`
	ProducerCredentialRef string            `json:"producer_credential_ref"`
	ProducerKeyRef        string            `json:"producer_key_ref"`
	CertificateRef        string            `json:"certificate_ref"`
	AlgorithmDeclared     string            `json:"algorithm_declared"`
	KeyIDSanitized        string            `json:"key_id_sanitized"`
	FingerprintSanitized  string            `json:"fingerprint_sanitized"`
	ExpiresAt             *string           `json:"expires_at"`
}

type updateAuthorityProfileReq struct {
	DisplayName           *string           `json:"display_name"`
	Status                *string           `json:"status"`
	AllowedOperations     *[]string         `json:"allowed_operations"`
	PendingExternal       map[string]string `json:"pending_external"`
	ProducerCredentialRef *string           `json:"producer_credential_ref"`
	ProducerKeyRef        *string           `json:"producer_key_ref"`
	CertificateRef        *string           `json:"certificate_ref"`
	AlgorithmDeclared     *string           `json:"algorithm_declared"`
	KeyIDSanitized        *string           `json:"key_id_sanitized"`
	FingerprintSanitized  *string           `json:"fingerprint_sanitized"`
	ExpiresAt             *string           `json:"expires_at"`
	ConfigReady           *bool             `json:"config_ready"`
	SecretsReady          *bool             `json:"secrets_ready"`
	OfflineValidated      *bool             `json:"offline_validated"`
}

type authorityProfileResp struct {
	ProfileID             string            `json:"profile_id"`
	Environment           string            `json:"environment"`
	TaxpayerID            string            `json:"taxpayer_id,omitempty"`
	ScopeID               string            `json:"scope_id,omitempty"`
	DisplayName           string            `json:"display_name"`
	Status                string            `json:"status"`
	AllowedOperations     []string          `json:"allowed_operations"`
	PendingExternal       map[string]string `json:"pending_external"`
	ProducerCredentialRef string            `json:"producer_credential_ref,omitempty"`
	ProducerKeyRef        string            `json:"producer_key_ref,omitempty"`
	CertificateRef        string            `json:"certificate_ref,omitempty"`
	AlgorithmDeclared     string            `json:"algorithm_declared,omitempty"`
	KeyIDSanitized        string            `json:"key_id_sanitized,omitempty"`
	FingerprintSanitized  string            `json:"fingerprint_sanitized,omitempty"`
	ExpiresAt             *string           `json:"expires_at,omitempty"`
	ConfigReady           bool              `json:"config_ready"`
	SecretsReady          bool              `json:"secrets_ready"`
	OfflineValidated      bool              `json:"offline_validated"`
	ExternalVerified      bool              `json:"external_verified"`
	CreatedAt             string            `json:"created_at"`
	UpdatedAt             string            `json:"updated_at"`
}

func authorityProfileToResp(p adminregistry.AuthorityProfile) authorityProfileResp {
	out := authorityProfileResp{
		ProfileID:             p.ID,
		Environment:           p.Environment,
		TaxpayerID:            p.TaxpayerID,
		ScopeID:               p.ScopeID,
		DisplayName:           p.DisplayName,
		Status:                p.Status,
		AllowedOperations:     p.AllowedOperations,
		PendingExternal:       p.PendingExternal,
		ProducerCredentialRef: p.ProducerCredentialRef,
		ProducerKeyRef:        p.ProducerKeyRef,
		CertificateRef:        p.CertificateRef,
		AlgorithmDeclared:     p.AlgorithmDeclared,
		KeyIDSanitized:        p.KeyIDSanitized,
		FingerprintSanitized:  p.FingerprintSanitized,
		ConfigReady:           p.ConfigReady,
		SecretsReady:          p.SecretsReady,
		OfflineValidated:      p.OfflineValidated,
		ExternalVerified:      false,
		CreatedAt:             p.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:             p.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if out.AllowedOperations == nil {
		out.AllowedOperations = []string{}
	}
	if out.PendingExternal == nil {
		out.PendingExternal = map[string]string{}
	}
	if p.ExpiresAt != nil {
		s := p.ExpiresAt.UTC().Format(time.RFC3339Nano)
		out.ExpiresAt = &s
	}
	return out
}

func (h *Handler) createAuthorityProfile(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	var req createAuthorityProfileReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "authority_profile.create", "authority_profile", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	var exp *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339Nano, *req.ExpiresAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, *req.ExpiresAt)
		}
		if err != nil {
			h.audit(r, claims, "authority_profile.create", "authority_profile", "-", adminaudit.ResultError)
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		t = t.UTC()
		exp = &t
	}
	out, err := h.Registry.CreateAuthorityProfile(r.Context(), adminregistry.CreateAuthorityProfileInput{
		Environment:           req.Environment,
		TaxpayerID:            req.TaxpayerID,
		ScopeID:               req.ScopeID,
		DisplayName:           req.DisplayName,
		Status:                req.Status,
		AllowedOperations:     req.AllowedOperations,
		PendingExternal:       req.PendingExternal,
		ProducerCredentialRef: req.ProducerCredentialRef,
		ProducerKeyRef:        req.ProducerKeyRef,
		CertificateRef:        req.CertificateRef,
		AlgorithmDeclared:     req.AlgorithmDeclared,
		KeyIDSanitized:        req.KeyIDSanitized,
		FingerprintSanitized:  req.FingerprintSanitized,
		ExpiresAt:             exp,
	})
	if err != nil {
		h.writeRegistryErr(w, r, claims, "authority_profile.create", "authority_profile", req.DisplayName, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "authority_profile.create", "authority_profile", out.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusCreated, authorityProfileToResp(out))
}

func (h *Handler) listAuthorityProfiles(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	list, err := h.Registry.ListAuthorityProfiles(r.Context())
	if err != nil {
		h.audit(r, claims, "authority_profile.list", "authority_profile", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	items := make([]authorityProfileResp, 0, len(list))
	for _, p := range list {
		items = append(items, authorityProfileToResp(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) getAuthorityProfile(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("profile_id")
	out, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		if errors.Is(err, adminregistry.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		h.audit(r, claims, "authority_profile.get", "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, authorityProfileToResp(out))
}

func (h *Handler) patchAuthorityProfile(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	id := r.PathValue("profile_id")
	var req updateAuthorityProfileReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "authority_profile.patch", "authority_profile", id, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	in := adminregistry.UpdateAuthorityProfileInput{ProfileID: id}
	if req.DisplayName != nil {
		in.DisplayName = *req.DisplayName
	}
	if req.Status != nil {
		in.Status = *req.Status
	}
	if req.AllowedOperations != nil {
		in.AllowedOperations = *req.AllowedOperations
	}
	if req.PendingExternal != nil {
		in.PendingExternal = req.PendingExternal
	}
	if req.ProducerCredentialRef != nil {
		in.ProducerCredentialRef = *req.ProducerCredentialRef
	}
	if req.ProducerKeyRef != nil {
		in.ProducerKeyRef = *req.ProducerKeyRef
	}
	if req.CertificateRef != nil {
		in.CertificateRef = *req.CertificateRef
	}
	if req.AlgorithmDeclared != nil {
		in.AlgorithmDeclared = *req.AlgorithmDeclared
	}
	if req.KeyIDSanitized != nil {
		in.KeyIDSanitized = *req.KeyIDSanitized
	}
	if req.FingerprintSanitized != nil {
		in.FingerprintSanitized = *req.FingerprintSanitized
	}
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339Nano, *req.ExpiresAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, *req.ExpiresAt)
		}
		if err != nil {
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		t = t.UTC()
		in.ExpiresAt = &t
	}
	in.ConfigReady = req.ConfigReady
	in.SecretsReady = req.SecretsReady
	in.OfflineValidated = req.OfflineValidated

	out, err := h.Registry.UpdateAuthorityProfile(r.Context(), in)
	if err != nil {
		h.writeRegistryErr(w, r, claims, "authority_profile.patch", "authority_profile", id, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "authority_profile.patch", "authority_profile", out.ID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, authorityProfileToResp(out))
}
