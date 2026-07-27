package adminapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

// writeSecretReq carries a secret for Put/Rotate. Plaintext is never logged or returned.
type writeSecretReq struct {
	Kind               string  `json:"kind"`
	Environment        string  `json:"environment"`
	SubjectID          string  `json:"subject_id"`
	Name               string  `json:"name"`
	Plaintext          string  `json:"plaintext"`
	ExpiresAt          *string `json:"expires_at"`
	AuthorityProfileID string  `json:"authority_profile_id"`
}

type revokeSecretReq struct {
	Kind               string `json:"kind"`
	Environment        string `json:"environment"`
	SubjectID          string `json:"subject_id"`
	Name               string `json:"name"`
	AuthorityProfileID string `json:"authority_profile_id"`
}

func (h *Handler) secadmPut(w http.ResponseWriter, r *http.Request) {
	h.secadmWrite(w, r, "secadm.put", func(ctxActor secadm.Actor, ref secretstore.Ref, plain []byte, exp *time.Time) (secretstore.Metadata, error) {
		res, err := h.SecAdm.Put(r.Context(), ctxActor, ref, plain, exp)
		return res.Metadata, err
	})
}

func (h *Handler) secadmRotate(w http.ResponseWriter, r *http.Request) {
	h.secadmWrite(w, r, "secadm.rotate", func(ctxActor secadm.Actor, ref secretstore.Ref, plain []byte, exp *time.Time) (secretstore.Metadata, error) {
		res, err := h.SecAdm.Rotate(r.Context(), ctxActor, ref, plain, exp)
		return res.Metadata, err
	})
}

func (h *Handler) secadmWrite(
	w http.ResponseWriter,
	r *http.Request,
	action string,
	fn func(secadm.Actor, secretstore.Ref, []byte, *time.Time) (secretstore.Metadata, error),
) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	if h.SecAdm == nil {
		h.audit(r, claims, action, "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SECADM_UNAVAILABLE", "SecAdm Unavailable")
		return
	}
	var req writeSecretReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, action, "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	ref := secretstore.Ref{
		Kind: req.Kind, Environment: req.Environment,
		SubjectID: req.SubjectID, Name: req.Name,
	}
	plain := []byte(req.Plaintext)
	req.Plaintext = "" // drop from heap-visible struct ASAP
	if len(plain) == 0 {
		h.audit(r, claims, action, "secret_ref", ref.Key(), adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	exp, err := parseOptionalRFC3339(req.ExpiresAt)
	if err != nil {
		h.audit(r, claims, action, "secret_ref", ref.Key(), adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	meta, err := fn(secadm.Actor{SubjectID: claims.Subject}, ref, plain, exp)
	zeroBytes(plain)
	if err != nil {
		h.writeSecAdmErr(w, r, claims, action, ref.Key(), err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, action, "secret_ref", ref.Key(), adminaudit.ResultSuccess, requestID(r))
	h.syncProfileAfterMaterialChange(r, claims, req.AuthorityProfileID, ref.Kind, ref.Name, meta, false)
	writeJSON(w, http.StatusOK, metadataResp(meta))
}

func (h *Handler) secadmRevoke(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	if h.SecAdm == nil {
		h.audit(r, claims, "secadm.revoke", "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SECADM_UNAVAILABLE", "SecAdm Unavailable")
		return
	}
	var req revokeSecretReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, "secadm.revoke", "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	ref := secretstore.Ref{
		Kind: req.Kind, Environment: req.Environment,
		SubjectID: req.SubjectID, Name: req.Name,
	}
	meta, err := h.SecAdm.Revoke(r.Context(), secadm.Actor{SubjectID: claims.Subject}, ref)
	if err != nil {
		h.writeSecAdmErr(w, r, claims, "secadm.revoke", ref.Key(), err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, "secadm.revoke", "secret_ref", ref.Key(), adminaudit.ResultSuccess, requestID(r))
	h.syncProfileAfterMaterialChange(r, claims, req.AuthorityProfileID, ref.Kind, ref.Name, meta, true)
	writeJSON(w, http.StatusOK, metadataResp(meta))
}

func (h *Handler) writeSecAdmErr(w http.ResponseWriter, r *http.Request, claims adminauth.Claims, action, resID string, err error) {
	switch {
	case errors.Is(err, secadm.ErrUnauthorized):
		h.audit(r, claims, action, "secret_ref", resID, adminaudit.ResultDenied)
		writeProblem(w, r, http.StatusForbidden, "ADMIN_FORBIDDEN", "Forbidden")
	case errors.Is(err, secretstore.ErrValidation), errors.Is(err, secadm.ErrValidation):
		h.audit(r, claims, action, "secret_ref", resID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
	case errors.Is(err, secretstore.ErrNotFound):
		h.audit(r, claims, action, "secret_ref", resID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
	case errors.Is(err, secretstore.ErrEnvIsolation):
		h.audit(r, claims, action, "secret_ref", resID, adminaudit.ResultDenied)
		writeProblem(w, r, http.StatusForbidden, "ADMIN_ENV_ISOLATION", "Forbidden")
	default:
		h.audit(r, claims, action, "secret_ref", resID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
	}
}

func metadataResp(meta secretstore.Metadata) secretMetadataResp {
	resp := secretMetadataResp{
		Kind: meta.Ref.Kind, Environment: meta.Environment, SubjectID: meta.Ref.SubjectID,
		Name: meta.Ref.Name, Status: meta.Status, Fingerprint: meta.Fingerprint, Version: meta.Version,
	}
	if meta.ExpiresAt != nil {
		s := meta.ExpiresAt.UTC().Format(time.RFC3339Nano)
		resp.ExpiresAt = &s
	}
	if meta.LastVerifiedAt != nil {
		s := meta.LastVerifiedAt.UTC().Format(time.RFC3339Nano)
		resp.LastVerifiedAt = &s
	}
	return resp
}

func parseOptionalRFC3339(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
	}
	if err != nil {
		return nil, err
	}
	u := t.UTC()
	return &u, nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
