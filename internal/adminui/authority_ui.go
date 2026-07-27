package adminui

import (
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

type authorityProfilesPage struct {
	pageBase
	Profiles []authorityProfileView
}

type authorityProfileFormPage struct {
	pageBase
	CSRFToken string
	Error     string
	Profile   *authorityProfileView
	KnownOps  []string
}

type authorityProfileView struct {
	ID                   string
	Environment          string
	TaxpayerID           string
	ScopeID              string
	DisplayName          string
	Status               string
	AllowedOps           string
	AllowedOpsList       []string
	PendingExternalLines string
	CredentialRef        string
	KeyRef               string
	CertRef              string
	Algorithm            string
	KeyID                string
	Fingerprint          string
	ExpiresAt            string
	ConfigReady          bool
	SecretsReady         bool
	OfflineValidated     bool
	ExternalVerified     bool
	CreatedAt            string
	UpdatedAt            string
}

func knownAuthorityOpsOrdered() []string {
	return []string{
		"registarFactura", "solicitarSerie", "listarSeries", "obterEstado", "listarFacturas",
	}
}

func viewAuthorityProfile(p adminregistry.AuthorityProfile) authorityProfileView {
	v := authorityProfileView{
		ID:               p.ID,
		Environment:      p.Environment,
		TaxpayerID:       p.TaxpayerID,
		ScopeID:          p.ScopeID,
		DisplayName:      p.DisplayName,
		Status:           p.Status,
		AllowedOps:       strings.Join(p.AllowedOperations, ", "),
		AllowedOpsList:   append([]string(nil), p.AllowedOperations...),
		CredentialRef:    p.ProducerCredentialRef,
		KeyRef:           p.ProducerKeyRef,
		CertRef:          p.CertificateRef,
		Algorithm:        p.AlgorithmDeclared,
		KeyID:            p.KeyIDSanitized,
		Fingerprint:      p.FingerprintSanitized,
		ConfigReady:      p.ConfigReady,
		SecretsReady:     p.SecretsReady,
		OfflineValidated: p.OfflineValidated,
		ExternalVerified: false, // never surface true until AGT real
		CreatedAt:        sanitizeTS(p.CreatedAt),
		UpdatedAt:        sanitizeTS(p.UpdatedAt),
	}
	if p.ExpiresAt != nil {
		v.ExpiresAt = sanitizeTS(*p.ExpiresAt)
	}
	if len(p.PendingExternal) > 0 {
		var lines []string
		for k, val := range p.PendingExternal {
			lines = append(lines, k+"="+val)
		}
		v.PendingExternalLines = strings.Join(lines, "\n")
	}
	return v
}

func sanitizeTS(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (h *Handler) authorityProfiles(w http.ResponseWriter, r *http.Request) {
	list, err := h.Registry.ListAuthorityProfiles(r.Context())
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	views := make([]authorityProfileView, 0, len(list))
	for _, p := range list {
		views = append(views, viewAuthorityProfile(p))
	}
	h.recordUIAccess(r, "ui.authority.list", "authority_profile", "list", adminaudit.ResultSuccess)
	h.render(w, "authority_profiles.html", authorityProfilesPage{
		pageBase: h.baseWithCSRF(w, r, "Perfis autoridade", "Perfis de autoridade (metadados)", "authority"),
		Profiles: views,
	})
}

func (h *Handler) newAuthorityProfileForm(w http.ResponseWriter, r *http.Request) {
	pb := h.baseWithCSRF(w, r, "Novo perfil autoridade", "Novo perfil de autoridade", "authority")
	h.render(w, "form_authority_profile.html", authorityProfileFormPage{
		pageBase:  pb,
		CSRFToken: pb.CSRFToken,
		KnownOps:  knownAuthorityOpsOrdered(),
	})
}

func (h *Handler) createAuthorityProfileForm(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	in, err := parseCreateAuthorityForm(r)
	if err != nil {
		h.authorityFormError(w, r, "form_authority_profile.html", "Novo perfil de autoridade", nil, err)
		return
	}
	created, err := h.Registry.CreateAuthorityProfile(r.Context(), in)
	if err != nil {
		h.authorityFormError(w, r, "form_authority_profile.html", "Novo perfil de autoridade", nil, err)
		return
	}
	h.recordUIAccess(r, "ui.authority.create", "authority_profile", created.ID, adminaudit.ResultSuccess)
	http.Redirect(w, r, "/admin/ui/authority-profiles", http.StatusSeeOther)
}

func (h *Handler) editAuthorityProfileForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	p, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	pb := h.baseWithCSRF(w, r, "Editar perfil autoridade", "Editar perfil (metadados)", "authority")
	v := viewAuthorityProfile(p)
	h.render(w, "form_authority_profile_edit.html", authorityProfileFormPage{
		pageBase:  pb,
		CSRFToken: pb.CSRFToken,
		Profile:   &v,
		KnownOps:  knownAuthorityOpsOrdered(),
	})
}

func (h *Handler) patchAuthorityProfileForm(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	id := r.PathValue("profile_id")
	in, err := parseUpdateAuthorityForm(r, id)
	if err != nil {
		cur, _ := h.Registry.GetAuthorityProfile(r.Context(), id)
		var v *authorityProfileView
		if cur.ID != "" {
			vv := viewAuthorityProfile(cur)
			v = &vv
		}
		h.authorityFormError(w, r, "form_authority_profile_edit.html", "Editar perfil (metadados)", v, err)
		return
	}
	updated, err := h.Registry.UpdateAuthorityProfile(r.Context(), in)
	if err != nil {
		vv := viewAuthorityProfile(adminregistry.AuthorityProfile{ID: id})
		if cur, gerr := h.Registry.GetAuthorityProfile(r.Context(), id); gerr == nil {
			vv = viewAuthorityProfile(cur)
		}
		h.authorityFormError(w, r, "form_authority_profile_edit.html", "Editar perfil (metadados)", &vv, err)
		return
	}
	h.recordUIAccess(r, "ui.authority.update", "authority_profile", updated.ID, adminaudit.ResultSuccess)
	http.Redirect(w, r, "/admin/ui/authority-profiles", http.StatusSeeOther)
}

func (h *Handler) authorityFormError(w http.ResponseWriter, r *http.Request, tmpl, heading string, profile *authorityProfileView, err error) {
	pb := h.baseWithCSRF(w, r, heading, heading, "authority")
	msg := "validação falhou"
	if err != nil {
		msg = err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = strings.TrimSpace(msg[i+2:])
		}
	}
	h.render(w, tmpl, authorityProfileFormPage{
		pageBase:  pb,
		CSRFToken: pb.CSRFToken,
		Error:     msg,
		Profile:   profile,
		KnownOps:  knownAuthorityOpsOrdered(),
	})
}

func parseCreateAuthorityForm(r *http.Request) (adminregistry.CreateAuthorityProfileInput, error) {
	ops := r.Form["allowed_operations"]
	pe, err := parsePendingExternalLines(r.FormValue("pending_external"))
	if err != nil {
		return adminregistry.CreateAuthorityProfileInput{}, err
	}
	exp, err := parseOptionalExpires(r.FormValue("expires_at"))
	if err != nil {
		return adminregistry.CreateAuthorityProfileInput{}, err
	}
	return adminregistry.CreateAuthorityProfileInput{
		Environment:           r.FormValue("environment"),
		TaxpayerID:            r.FormValue("taxpayer_id"),
		ScopeID:               r.FormValue("scope_id"),
		DisplayName:           r.FormValue("display_name"),
		Status:                r.FormValue("status"),
		AllowedOperations:     ops,
		PendingExternal:       pe,
		ProducerCredentialRef: r.FormValue("producer_credential_ref"),
		ProducerKeyRef:        r.FormValue("producer_key_ref"),
		CertificateRef:        r.FormValue("certificate_ref"),
		AlgorithmDeclared:     r.FormValue("algorithm_declared"),
		KeyIDSanitized:        r.FormValue("key_id_sanitized"),
		FingerprintSanitized:  r.FormValue("fingerprint_sanitized"),
		ExpiresAt:             exp,
	}, nil
}

func parseUpdateAuthorityForm(r *http.Request, profileID string) (adminregistry.UpdateAuthorityProfileInput, error) {
	ops := r.Form["allowed_operations"]
	if ops == nil {
		ops = []string{} // explicit clear vs omit — send empty slice to replace
	}
	pe, err := parsePendingExternalLines(r.FormValue("pending_external"))
	if err != nil {
		return adminregistry.UpdateAuthorityProfileInput{}, err
	}
	exp, err := parseOptionalExpires(r.FormValue("expires_at"))
	if err != nil {
		return adminregistry.UpdateAuthorityProfileInput{}, err
	}
	cfg := formBool(r, "config_ready")
	sec := formBool(r, "secrets_ready")
	off := formBool(r, "offline_validated")
	return adminregistry.UpdateAuthorityProfileInput{
		ProfileID:             profileID,
		DisplayName:           r.FormValue("display_name"),
		Status:                r.FormValue("status"),
		AllowedOperations:     ops,
		PendingExternal:       pe,
		ProducerCredentialRef: r.FormValue("producer_credential_ref"),
		ProducerKeyRef:        r.FormValue("producer_key_ref"),
		CertificateRef:        r.FormValue("certificate_ref"),
		AlgorithmDeclared:     r.FormValue("algorithm_declared"),
		KeyIDSanitized:        r.FormValue("key_id_sanitized"),
		FingerprintSanitized:  r.FormValue("fingerprint_sanitized"),
		ExpiresAt:             exp,
		ConfigReady:           &cfg,
		SecretsReady:          &sec,
		OfflineValidated:      &off,
	}, nil
}

func formBool(r *http.Request, name string) bool {
	v := strings.TrimSpace(strings.ToLower(r.FormValue(name)))
	return v == "on" || v == "true" || v == "1" || v == "yes"
}

func parsePendingExternalLines(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, errPendingExternalFormat
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}

func parseOptionalExpires(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, errExpiresFormat
	}
	t = t.UTC()
	return &t, nil
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

const (
	errPendingExternalFormat simpleError = "pending_external: use linhas key=value"
	errExpiresFormat         simpleError = "expires_at: use RFC3339 (UTC)"
)

func (h *Handler) authorityReadiness(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	p, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v := viewAuthorityProfile(p)
	complete := v.ConfigReady && v.SecretsReady && v.OfflineValidated && !v.ExternalVerified
	h.recordUIAccess(r, "ui.authority.readiness", "authority_profile", id, adminaudit.ResultSuccess)
	h.render(w, "authority_readiness.html", authorityReadinessPage{
		pageBase: h.baseWithCSRF(w, r, "Readiness autoridade", "Checklist readiness", "authority"),
		Profile:  v,
		Complete: complete,
	})
}

type authorityReadinessPage struct {
	pageBase
	Profile  authorityProfileView
	Complete bool
}

type authorityHistoryPage struct {
	pageBase
	Profile authorityProfileView
	Events  []authorityHistoryEvent
}

type authorityHistoryEvent struct {
	OccurredAt   string
	ActorSubject string
	Action       string
	Result       string
}

func (h *Handler) authorityHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	p, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v := viewAuthorityProfile(p)
	var events []authorityHistoryEvent
	if h.Audit != nil {
		list, err := h.Audit.ListByResource(r.Context(), "authority_profile", id, 100)
		if err == nil {
			events = make([]authorityHistoryEvent, 0, len(list))
			for _, e := range list {
				events = append(events, authorityHistoryEvent{
					OccurredAt:   sanitizeTS(e.OccurredAt),
					ActorSubject: e.ActorSubject,
					Action:       e.Action,
					Result:       e.Result,
				})
			}
		}
	}
	h.recordUIAccess(r, "ui.authority.history", "authority_profile", id, adminaudit.ResultSuccess)
	h.render(w, "authority_history.html", authorityHistoryPage{
		pageBase: h.baseWithCSRF(w, r, "Histórico autoridade", "Histórico append-only", "authority"),
		Profile:  v,
		Events:   events,
	})
}
