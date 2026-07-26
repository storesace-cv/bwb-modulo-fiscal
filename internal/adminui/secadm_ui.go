package adminui

import (
	"net/http"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

// SecretsMeta is optional AdminView for sanitized metadata only (never Reveal).
type secadmMetaPage struct {
	pageBase
	CSRFToken   string
	Error       string
	Kind        string
	Environment string
	SubjectID   string
	Name        string
	Status      string
	Fingerprint string
	Version     int
	HasResult   bool
}

func (h *Handler) secadmMetaForm(w http.ResponseWriter, r *http.Request) {
	tok := ""
	if h.CSRF != nil {
		tok, _ = h.CSRF.Issue(w)
	}
	h.render(w, "secadm_meta.html", secadmMetaPage{
		pageBase:  h.base(r, "SecAdm metadados", "Metadados de refs (owner)", "secadm"),
		CSRFToken: tok,
	})
}

func (h *Handler) secadmMetaLookup(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	tok, _ := h.CSRF.Issue(w)
	page := secadmMetaPage{
		pageBase:    h.base(r, "SecAdm metadados", "Metadados de refs (owner)", "secadm"),
		CSRFToken:   tok,
		Kind:        strings.TrimSpace(r.FormValue("kind")),
		Environment: strings.TrimSpace(r.FormValue("environment")),
		SubjectID:   strings.TrimSpace(r.FormValue("subject_id")),
		Name:        strings.TrimSpace(r.FormValue("name")),
	}
	if h.SecretsMeta == nil {
		page.Error = "SecretStore metadados indisponível"
		h.render(w, "secadm_meta.html", page)
		return
	}
	meta, err := h.SecretsMeta.Metadata(r.Context(), secretstore.Ref{
		Kind: page.Kind, Environment: page.Environment,
		SubjectID: page.SubjectID, Name: page.Name,
	})
	if err != nil {
		page.Error = "consulta falhou (validação ou erro interno)"
		h.render(w, "secadm_meta.html", page)
		return
	}
	page.HasResult = true
	page.Status = meta.Status
	page.Fingerprint = meta.Fingerprint
	page.Version = meta.Version
	// Never attach plaintext — Metadata type has none.
	h.recordUIAccess(r, "ui.secadm.metadata_read", "secret_ref",
		page.Kind+"/"+page.Environment+"/"+page.SubjectID+"/"+page.Name, adminaudit.ResultSuccess)
	h.render(w, "secadm_meta.html", page)
}

func htmlRequireOwnerSecAdm(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminauth.ClaimsFromContext(r.Context())
		if !ok || !adminauth.Allows(claims, adminauth.PermSecAdmWrite) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="pt"><body><h1>Proibido</h1><p>SecAdm: apenas owner.</p></body></html>`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
