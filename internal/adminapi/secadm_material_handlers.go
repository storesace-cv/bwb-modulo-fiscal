package adminapi

import (
	"io"
	"net/http"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

const maxMaterialMultipart = secretstore.MaxPKCS12Bytes + (64 << 10) // payload + form overhead

// secadmPutMaterial imports PEM/PKCS#12/credential via multipart (RM-AGTPREP-004).
// Password field is ephemeral and never audited/stored/returned.
func (h *Handler) secadmPutMaterial(w http.ResponseWriter, r *http.Request) {
	h.secadmMaterialWrite(w, r, "secadm.material.put", false)
}

func (h *Handler) secadmRotateMaterial(w http.ResponseWriter, r *http.Request) {
	h.secadmMaterialWrite(w, r, "secadm.material.rotate", true)
}

func (h *Handler) secadmMaterialWrite(w http.ResponseWriter, r *http.Request, action string, rotate bool) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	if h.SecAdm == nil {
		h.audit(r, claims, action, "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SECADM_UNAVAILABLE", "SecAdm Unavailable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxMaterialMultipart))
	if err := r.ParseMultipartForm(int64(maxMaterialMultipart)); err != nil {
		h.audit(r, claims, action, "secret_ref", "-", adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	ref := secretstore.Ref{
		Kind: r.FormValue("kind"), Environment: r.FormValue("environment"),
		SubjectID: r.FormValue("subject_id"), Name: r.FormValue("name"),
	}
	encoding := strings.TrimSpace(r.FormValue("encoding"))
	password := []byte(r.FormValue("password"))
	defer secretstore.ZeroBytes(password)

	file, _, err := r.FormFile("material")
	if err != nil {
		h.audit(r, claims, action, "secret_ref", ref.Key(), adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, int64(secretstore.MaxPKCS12Bytes)+1))
	if err != nil {
		h.audit(r, claims, action, "secret_ref", ref.Key(), adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	prepared, err := secretstore.Prepare(secretstore.MaterialInput{
		Kind: ref.Kind, Encoding: encoding, Bytes: raw, Password: password,
	})
	secretstore.ZeroBytes(raw)
	if err != nil {
		h.audit(r, claims, action, "secret_ref", ref.Key(), adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	exp, err := parseOptionalRFC3339(optionalStringPtr(r.FormValue("expires_at")))
	if err != nil {
		secretstore.ZeroBytes(prepared.StorageBytes)
		h.audit(r, claims, action, "secret_ref", ref.Key(), adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}

	actor := secadm.Actor{SubjectID: claims.Subject}
	var meta secretstore.Metadata
	if rotate {
		res, err := h.SecAdm.Rotate(r.Context(), actor, ref, prepared.StorageBytes, exp)
		secretstore.ZeroBytes(prepared.StorageBytes)
		if err != nil {
			h.writeSecAdmErr(w, r, claims, action, ref.Key(), err)
			return
		}
		meta = res.Metadata
	} else {
		res, err := h.SecAdm.Put(r.Context(), actor, ref, prepared.StorageBytes, exp)
		secretstore.ZeroBytes(prepared.StorageBytes)
		if err != nil {
			h.writeSecAdmErr(w, r, claims, action, ref.Key(), err)
			return
		}
		meta = res.Metadata
	}
	_ = h.Audit.Record(r.Context(), claims, action, "secret_ref", ref.Key(), adminaudit.ResultSuccess, requestID(r))
	base := metadataResp(meta)
	writeJSON(w, http.StatusOK, materialMetaResp{
		Kind: base.Kind, Environment: base.Environment, SubjectID: base.SubjectID,
		Name: base.Name, Status: base.Status, Fingerprint: base.Fingerprint, Version: base.Version,
		ExpiresAt: base.ExpiresAt, LastVerifiedAt: base.LastVerifiedAt,
		Encoding: prepared.Encoding, FormatNote: prepared.FormatNote,
	})
}

type materialMetaResp struct {
	Kind           string  `json:"kind"`
	Environment    string  `json:"environment"`
	SubjectID      string  `json:"subject_id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Fingerprint    string  `json:"fingerprint,omitempty"`
	Version        int     `json:"version"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
	LastVerifiedAt *string `json:"last_verified_at,omitempty"`
	Encoding       string  `json:"encoding"`
	FormatNote     string  `json:"format_note"`
}

func optionalStringPtr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}