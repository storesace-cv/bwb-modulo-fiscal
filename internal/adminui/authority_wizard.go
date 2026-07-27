package adminui

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

type authorityWizardPage struct {
	pageBase
	CSRFToken string
	Error     string
	Step      int
	KnownOps  []string
	Profile   *authorityProfileView
}

func (h *Handler) authorityWizardStart(w http.ResponseWriter, r *http.Request) {
	tok := ""
	if h.CSRF != nil {
		tok, _ = h.CSRF.Issue(w)
	}
	h.render(w, "authority_wizard.html", authorityWizardPage{
		pageBase:  h.baseWithCSRF(w, r, "Wizard autoridade", "Wizard preparação AGT (owner)", "authority"),
		CSRFToken: tok,
		Step:      1,
		KnownOps:  knownAuthorityOpsOrdered(),
	})
}

func (h *Handler) authorityWizardStep1(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	in, err := parseCreateAuthorityForm(r)
	if err != nil {
		h.wizardError(w, r, 1, nil, err)
		return
	}
	// Wizard always starts as draft — never active, never external_verified.
	in.Status = adminregistry.AuthorityStatusDraft
	created, err := h.Registry.CreateAuthorityProfile(r.Context(), in)
	if err != nil {
		h.wizardError(w, r, 1, nil, err)
		return
	}
	h.recordUIAccess(r, "ui.authority.wizard_step1", "authority_profile", created.ID, adminaudit.ResultSuccess)
	http.Redirect(w, r, "/admin/ui/authority-profiles/"+created.ID+"/wizard?step=2", http.StatusSeeOther)
}

func (h *Handler) authorityWizardContinue(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("profile_id")
	p, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	step := wizardStep(r)
	if step < 2 {
		step = 2
	}
	if step > 3 {
		step = 3
	}
	tok := ""
	if h.CSRF != nil {
		tok, _ = h.CSRF.Issue(w)
	}
	v := viewAuthorityProfile(p)
	h.render(w, "authority_wizard.html", authorityWizardPage{
		pageBase:  h.baseWithCSRF(w, r, "Wizard autoridade", "Wizard preparação AGT (owner)", "authority"),
		CSRFToken: tok,
		Step:      step,
		KnownOps:  knownAuthorityOpsOrdered(),
		Profile:   &v,
	})
}

func (h *Handler) authorityWizardStep2(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	id := r.PathValue("profile_id")
	cur, err := h.Registry.GetAuthorityProfile(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	vv := viewAuthorityProfile(cur)
	_, err = h.Registry.UpdateAuthorityProfile(r.Context(), adminregistry.UpdateAuthorityProfileInput{
		ProfileID:             id,
		DisplayName:           cur.DisplayName,
		Status:                adminregistry.AuthorityStatusDraft, // never escalate via wizard
		ProducerCredentialRef: r.FormValue("producer_credential_ref"),
		ProducerKeyRef:        r.FormValue("producer_key_ref"),
		CertificateRef:        r.FormValue("certificate_ref"),
		AlgorithmDeclared:     r.FormValue("algorithm_declared"),
		KeyIDSanitized:        r.FormValue("key_id_sanitized"),
		FingerprintSanitized:  r.FormValue("fingerprint_sanitized"),
	})
	if err != nil {
		h.wizardError(w, r, 2, &vv, err)
		return
	}
	h.recordUIAccess(r, "ui.authority.wizard_step2", "authority_profile", id, adminaudit.ResultSuccess)
	http.Redirect(w, r, "/admin/ui/authority-profiles/"+id+"/wizard?step=3", http.StatusSeeOther)
}

func (h *Handler) authorityWizardStep3Ack(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	id := r.PathValue("profile_id")
	if _, err := h.Registry.GetAuthorityProfile(r.Context(), id); err != nil {
		http.NotFound(w, r)
		return
	}
	// Step 3 only acknowledges checklist links; does not activate or set external_verified.
	cfg := true
	_, err := h.Registry.UpdateAuthorityProfile(r.Context(), adminregistry.UpdateAuthorityProfileInput{
		ProfileID:   id,
		Status:      adminregistry.AuthorityStatusDraft,
		ConfigReady: &cfg,
	})
	if err != nil {
		cur, _ := h.Registry.GetAuthorityProfile(r.Context(), id)
		vv := viewAuthorityProfile(cur)
		h.wizardError(w, r, 3, &vv, err)
		return
	}
	h.recordUIAccess(r, "ui.authority.wizard_step3", "authority_profile", id, adminaudit.ResultSuccess)
	http.Redirect(w, r, "/admin/ui/authority-profiles/"+id+"/readiness", http.StatusSeeOther)
}

func wizardStep(r *http.Request) int {
	n, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("step")))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func (h *Handler) wizardError(w http.ResponseWriter, r *http.Request, step int, profile *authorityProfileView, err error) {
	tok := ""
	if h.CSRF != nil {
		tok, _ = h.CSRF.Issue(w)
	}
	msg := "validação falhou"
	if err != nil {
		msg = err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = strings.TrimSpace(msg[i+2:])
		}
	}
	h.render(w, "authority_wizard.html", authorityWizardPage{
		pageBase:  h.base(r, "Wizard autoridade", "Wizard preparação AGT (owner)", "authority"),
		CSRFToken: tok,
		Error:     msg,
		Step:      step,
		KnownOps:  knownAuthorityOpsOrdered(),
		Profile:   profile,
	})
}
