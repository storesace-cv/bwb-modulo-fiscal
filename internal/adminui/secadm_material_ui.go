package adminui

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

type secadmMaterialPage struct {
	pageBase
	CSRFToken   string
	Error       string
	FlashOK     string
	Kind        string
	Environment string
	SubjectID   string
	Name        string
	Encoding    string
	Status      string
	Fingerprint string
	Version     int
	FormatNote  string
	HasResult   bool
}

func (h *Handler) secadmMaterialForm(w http.ResponseWriter, r *http.Request) {
	tok := ""
	if h.CSRF != nil {
		tok, _ = h.CSRF.Issue(w)
	}
	h.render(w, "secadm_material.html", secadmMaterialPage{
		pageBase:    h.baseWithCSRF(w, r, "SecAdm material", "Importar / rotacionar material (owner)", "secadm"),
		CSRFToken:   tok,
		Environment: "homologation",
		Encoding:    "pem",
		Kind:        "certificate",
		SubjectID:   "platform",
	})
}

func (h *Handler) secadmMaterialSubmit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, int64(secretstore.MaxPKCS12Bytes+(64<<10)))
	if err := r.ParseMultipartForm(int64(secretstore.MaxPKCS12Bytes + (64 << 10))); err != nil {
		http.Error(w, "upload inválido", http.StatusBadRequest)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	if h.CSRF == nil || !h.CSRF.Validate(r, r.FormValue(csrfFieldName)) {
		http.Error(w, "CSRF inválido", http.StatusForbidden)
		return
	}
	tok, _ := h.CSRF.Issue(w)
	page := secadmMaterialPage{
		pageBase:    h.baseWithCSRF(w, r, "SecAdm material", "Importar / rotacionar material (owner)", "secadm"),
		CSRFToken:   tok,
		Kind:        strings.TrimSpace(r.FormValue("kind")),
		Environment: strings.TrimSpace(r.FormValue("environment")),
		SubjectID:   strings.TrimSpace(r.FormValue("subject_id")),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Encoding:    strings.TrimSpace(r.FormValue("encoding")),
	}
	if h.SecAdm == nil {
		page.Error = "SecAdm indisponível"
		h.render(w, "secadm_material.html", page)
		return
	}
	claims, ok := adminauth.ClaimsFromContext(r.Context())
	if !ok {
		page.Error = "sessão inválida"
		h.render(w, "secadm_material.html", page)
		return
	}

	password := []byte(r.FormValue("password"))
	defer secretstore.ZeroBytes(password)

	file, _, err := r.FormFile("material")
	if err != nil {
		page.Error = "ficheiro material obrigatório"
		h.render(w, "secadm_material.html", page)
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, int64(secretstore.MaxPKCS12Bytes)+1))
	if err != nil {
		page.Error = "leitura do ficheiro falhou"
		h.render(w, "secadm_material.html", page)
		return
	}
	prepared, err := secretstore.Prepare(secretstore.MaterialInput{
		Kind: page.Kind, Encoding: page.Encoding, Bytes: raw, Password: password,
	})
	secretstore.ZeroBytes(raw)
	if err != nil {
		page.Error = "material rejeitado (formato/limite/password)"
		h.render(w, "secadm_material.html", page)
		return
	}
	formatNote := prepared.FormatNote

	ref := secretstore.Ref{
		Kind: page.Kind, Environment: page.Environment,
		SubjectID: page.SubjectID, Name: page.Name,
	}
	actor := secadm.Actor{SubjectID: claims.Subject}
	action := strings.TrimSpace(r.FormValue("action"))
	var meta secretstore.Metadata
	switch action {
	case "rotate":
		res, err := h.SecAdm.Rotate(r.Context(), actor, ref, prepared.StorageBytes, nil)
		secretstore.ZeroBytes(prepared.StorageBytes)
		if err != nil {
			page.Error = "rotação falhou"
			h.render(w, "secadm_material.html", page)
			return
		}
		meta = res.Metadata
		h.recordUIAccess(r, "ui.secadm.material_rotate", "secret_ref", ref.Key(), adminaudit.ResultSuccess)
	default:
		res, err := h.SecAdm.Put(r.Context(), actor, ref, prepared.StorageBytes, nil)
		secretstore.ZeroBytes(prepared.StorageBytes)
		if err != nil {
			page.Error = "importação falhou"
			h.render(w, "secadm_material.html", page)
			return
		}
		meta = res.Metadata
		h.recordUIAccess(r, "ui.secadm.material_put", "secret_ref", ref.Key(), adminaudit.ResultSuccess)
	}
	page.HasResult = true
	page.FlashOK = "operação concluída — apenas metadados abaixo"
	page.Status = meta.Status
	page.Fingerprint = meta.Fingerprint
	page.Version = meta.Version
	page.FormatNote = formatNote
	h.render(w, "secadm_material.html", page)
}

func (h *Handler) secadmRevokeForm(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	tok, _ := h.CSRF.Issue(w)
	page := secadmMaterialPage{
		pageBase:    h.baseWithCSRF(w, r, "SecAdm material", "Revogar material (owner)", "secadm"),
		CSRFToken:   tok,
		Kind:        strings.TrimSpace(r.FormValue("kind")),
		Environment: strings.TrimSpace(r.FormValue("environment")),
		SubjectID:   strings.TrimSpace(r.FormValue("subject_id")),
		Name:        strings.TrimSpace(r.FormValue("name")),
	}
	if h.SecAdm == nil {
		page.Error = "SecAdm indisponível"
		h.render(w, "secadm_material.html", page)
		return
	}
	claims, ok := adminauth.ClaimsFromContext(r.Context())
	if !ok {
		page.Error = "sessão inválida"
		h.render(w, "secadm_material.html", page)
		return
	}
	ref := secretstore.Ref{
		Kind: page.Kind, Environment: page.Environment,
		SubjectID: page.SubjectID, Name: page.Name,
	}
	meta, err := h.SecAdm.Revoke(r.Context(), secadm.Actor{SubjectID: claims.Subject}, ref)
	if err != nil {
		page.Error = "revogação falhou"
		h.render(w, "secadm_material.html", page)
		return
	}
	h.recordUIAccess(r, "ui.secadm.material_revoke", "secret_ref", ref.Key(), adminaudit.ResultSuccess)
	page.HasResult = true
	page.FlashOK = "revogado"
	page.Status = meta.Status
	page.Fingerprint = meta.Fingerprint
	page.Version = meta.Version
	h.render(w, "secadm_material.html", page)
}

func (h *Handler) secadmValidateOfflineForm(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	tok, _ := h.CSRF.Issue(w)
	page := secadmMaterialPage{
		pageBase:  h.baseWithCSRF(w, r, "SecAdm material", "Validação offline (owner)", "secadm"),
		CSRFToken: tok,
	}
	if h.SecAdm == nil {
		page.Error = "SecAdm indisponível"
		h.render(w, "secadm_material.html", page)
		return
	}
	claims, ok := adminauth.ClaimsFromContext(r.Context())
	if !ok {
		page.Error = "sessão inválida"
		h.render(w, "secadm_material.html", page)
		return
	}
	keyRef := secretstore.Ref{
		Kind: strings.TrimSpace(r.FormValue("key_kind")), Environment: strings.TrimSpace(r.FormValue("key_environment")),
		SubjectID: strings.TrimSpace(r.FormValue("key_subject_id")), Name: strings.TrimSpace(r.FormValue("key_name")),
	}
	certRef := secretstore.Ref{
		Kind: strings.TrimSpace(r.FormValue("cert_kind")), Environment: strings.TrimSpace(r.FormValue("cert_environment")),
		SubjectID: strings.TrimSpace(r.FormValue("cert_subject_id")), Name: strings.TrimSpace(r.FormValue("cert_name")),
	}
	rep, err := h.SecAdm.ValidateOfflineRefs(r.Context(), secadm.Actor{SubjectID: claims.Subject}, keyRef, certRef, nil, time.Time{})
	if err != nil {
		page.Error = "validação offline falhou"
		h.render(w, "secadm_material.html", page)
		return
	}
	h.recordUIAccess(r, "ui.secadm.validate_offline", "secret_ref", keyRef.Key()+"+"+certRef.Key(), adminaudit.ResultSuccess)
	page.HasResult = true
	if rep.OK {
		page.FlashOK = "offline OK — fingerprint " + rep.FingerprintSHA256 + " (external_verified=false)"
	} else {
		page.Error = "offline NÃO OK (ver issues no audit/API); fingerprint " + rep.FingerprintSHA256
	}
	page.Status = "offline_check"
	page.Fingerprint = rep.FingerprintSHA256
	page.FormatNote = rep.AlgorithmNote
	h.render(w, "secadm_material.html", page)
}
