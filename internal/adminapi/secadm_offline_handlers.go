package adminapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

type offlineValidateReq struct {
	KeyKind          string `json:"key_kind"`
	KeyEnvironment   string `json:"key_environment"`
	KeySubjectID     string `json:"key_subject_id"`
	KeyName          string `json:"key_name"`
	CertKind         string `json:"cert_kind"`
	CertEnvironment  string `json:"cert_environment"`
	CertSubjectID    string `json:"cert_subject_id"`
	CertName         string `json:"cert_name"`
	ProfileID        string `json:"profile_id"`
	IntermediatesPEM string `json:"intermediates_pem"` // public CA chain only; optional
}

type offlineValidateResp struct {
	OK                  bool     `json:"ok"`
	PairMatch           bool     `json:"pair_match"`
	ChainOK             bool     `json:"chain_ok"`
	WithinValidity      bool     `json:"within_validity"`
	PurposeOK           bool     `json:"purpose_ok"`
	PurposeNote         string   `json:"purpose_note"`
	FingerprintSHA256   string   `json:"fingerprint_sha256"`
	NotBefore           string   `json:"not_before,omitempty"`
	NotAfter            string   `json:"not_after,omitempty"`
	KeyBits             int      `json:"key_bits,omitempty"`
	AlgorithmNote       string   `json:"algorithm_note,omitempty"`
	Issues              []string `json:"issues,omitempty"`
	OfflineValidatedSet bool     `json:"offline_validated_set"`
	ExternalVerified    bool     `json:"external_verified"` // always false
}

func (h *Handler) secadmValidateOffline(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	action := "secadm.material.validate_offline"
	if h.SecAdm == nil {
		h.audit(r, claims, action, "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SECADM_UNAVAILABLE", "SecAdm Unavailable")
		return
	}
	var req offlineValidateReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, action, "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	keyRef := secretstore.Ref{
		Kind: req.KeyKind, Environment: req.KeyEnvironment,
		SubjectID: req.KeySubjectID, Name: req.KeyName,
	}
	certRef := secretstore.Ref{
		Kind: req.CertKind, Environment: req.CertEnvironment,
		SubjectID: req.CertSubjectID, Name: req.CertName,
	}
	// Reject intermediates that look like private keys.
	inter := []byte(strings.TrimSpace(req.IntermediatesPEM))
	req.IntermediatesPEM = ""
	if secretstore.LooksLikePEM(inter) && bytesContainPrivate(inter) {
		secretstore.ZeroBytes(inter)
		h.audit(r, claims, action, "secret_ref", keyRef.Key(), adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}

	rep, err := h.SecAdm.ValidateOfflineRefs(r.Context(), secadm.Actor{SubjectID: claims.Subject}, keyRef, certRef, inter, time.Time{})
	secretstore.ZeroBytes(inter)
	if err != nil {
		h.writeSecAdmErr(w, r, claims, action, keyRef.Key()+"+"+certRef.Key(), err)
		return
	}
	resp := offlineValidateResp{
		OK: rep.OK, PairMatch: rep.PairMatch, ChainOK: rep.ChainOK,
		WithinValidity: rep.WithinValidity, PurposeOK: rep.PurposeOK, PurposeNote: rep.PurposeNote,
		FingerprintSHA256: rep.FingerprintSHA256, NotBefore: rep.NotBefore, NotAfter: rep.NotAfter,
		KeyBits: rep.KeyBits, AlgorithmNote: rep.AlgorithmNote, Issues: rep.Issues,
		ExternalVerified: false,
	}
	if rep.OK && strings.TrimSpace(req.ProfileID) != "" && h.Registry != nil {
		flag := true
		_, uerr := h.Registry.UpdateAuthorityProfile(r.Context(), adminregistry.UpdateAuthorityProfileInput{
			ProfileID:            req.ProfileID,
			OfflineValidated:     &flag,
			FingerprintSanitized: rep.FingerprintSHA256,
		})
		if uerr == nil {
			resp.OfflineValidatedSet = true
		}
	}
	_ = h.Audit.Record(r.Context(), claims, action, "secret_ref", keyRef.Key()+"+"+certRef.Key(), adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, resp)
}

func bytesContainPrivate(b []byte) bool {
	low := strings.ToLower(string(b))
	return strings.Contains(low, "private key") || strings.Contains(low, "begin private")
}
