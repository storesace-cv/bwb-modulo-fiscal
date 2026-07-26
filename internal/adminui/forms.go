package adminui

import (
	"net/http"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
)

func (h *Handler) newTaxpayerForm(w http.ResponseWriter, r *http.Request) {
	tok, err := h.CSRF.Issue(w)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "form_taxpayer.html", formPage{
		pageBase:  h.base(r, "Novo contribuinte", "Novo contribuinte", "taxpayers"),
		CSRFToken: tok,
	})
}

func (h *Handler) createTaxpayerForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	_, err := h.Registry.CreateTaxpayer(r.Context(), adminregistry.CreateTaxpayerInput{
		NIF: r.FormValue("nif"), LegalName: r.FormValue("legal_name"), Status: r.FormValue("status"),
	})
	if err != nil {
		h.formError(w, r, "form_taxpayer.html", "taxpayers", "Novo contribuinte", err)
		return
	}
	http.Redirect(w, r, "/admin/ui/taxpayers", http.StatusSeeOther)
}

func (h *Handler) patchTaxpayerForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	id := r.PathValue("taxpayer_id")
	_, err := h.Registry.UpdateTaxpayerStatus(r.Context(), id, r.FormValue("status"))
	if err != nil {
		http.Error(w, "falha ao actualizar", http.StatusUnprocessableEntity)
		return
	}
	http.Redirect(w, r, "/admin/ui/taxpayers", http.StatusSeeOther)
}

func (h *Handler) newEstablishmentForm(w http.ResponseWriter, r *http.Request) {
	tok, err := h.CSRF.Issue(w)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	tps, _ := h.Registry.ListTaxpayers(r.Context(), listLimit)
	h.render(w, "form_establishment.html", formPage{
		pageBase:  h.base(r, "Novo estabelecimento", "Novo estabelecimento", "establishments"),
		CSRFToken: tok,
		Taxpayers: tps,
	})
}

func (h *Handler) createEstablishmentForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	_, err := h.Registry.CreateEstablishment(r.Context(), adminregistry.CreateEstablishmentInput{
		TaxpayerID: r.FormValue("taxpayer_id"), Code: r.FormValue("code"),
		Name: r.FormValue("name"), Status: r.FormValue("status"),
	})
	if err != nil {
		h.formError(w, r, "form_establishment.html", "establishments", "Novo estabelecimento", err)
		return
	}
	http.Redirect(w, r, "/admin/ui/establishments", http.StatusSeeOther)
}

func (h *Handler) patchEstablishmentForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	id := r.PathValue("establishment_id")
	_, err := h.Registry.UpdateEstablishmentStatus(r.Context(), id, r.FormValue("status"))
	if err != nil {
		http.Error(w, "falha ao actualizar", http.StatusUnprocessableEntity)
		return
	}
	http.Redirect(w, r, "/admin/ui/establishments", http.StatusSeeOther)
}

func (h *Handler) newBindingForm(w http.ResponseWriter, r *http.Request) {
	tok, err := h.CSRF.Issue(w)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	tps, _ := h.Registry.ListTaxpayers(r.Context(), listLimit)
	ests, _ := h.Registry.ListEstablishments(r.Context(), "", listLimit)
	h.render(w, "form_binding.html", formPage{
		pageBase:       h.base(r, "Novo binding", "Novo scope binding", "bindings"),
		CSRFToken:      tok,
		Taxpayers:      tps,
		Establishments: ests,
	})
}

func (h *Handler) createBindingForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	_, err := h.Registry.CreateScopeBinding(r.Context(), adminregistry.CreateScopeBindingInput{
		ScopeID: r.FormValue("scope_id"), TaxpayerID: r.FormValue("taxpayer_id"),
		EstablishmentID: r.FormValue("establishment_id"), Environment: r.FormValue("environment"),
		IANATimezone: r.FormValue("iana_timezone"), SeriesEffectiveCode: r.FormValue("series_effective_code"),
		Status: r.FormValue("status"),
	})
	if err != nil {
		h.formError(w, r, "form_binding.html", "bindings", "Novo scope binding", err)
		return
	}
	http.Redirect(w, r, "/admin/ui/bindings", http.StatusSeeOther)
}

func (h *Handler) patchBindingForm(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	scopeID := r.PathValue("scope_id")
	_, err := h.Registry.UpdateScopeConfig(r.Context(), adminregistry.UpdateScopeConfigInput{
		ScopeID: scopeID, Environment: r.FormValue("environment"),
		IANATimezone: r.FormValue("iana_timezone"), SeriesEffectiveCode: r.FormValue("series_effective_code"),
		Status: r.FormValue("status"),
	})
	if err != nil {
		http.Error(w, "falha ao actualizar config", http.StatusUnprocessableEntity)
		return
	}
	http.Redirect(w, r, "/admin/ui/bindings", http.StatusSeeOther)
}

func (h *Handler) editBindingForm(w http.ResponseWriter, r *http.Request) {
	scopeID := r.PathValue("scope_id")
	bind, err := h.Registry.GetScopeBinding(r.Context(), scopeID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tok, err := h.CSRF.Issue(w)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "form_binding_edit.html", formPage{
		pageBase:  h.base(r, "Editar binding", "Configuração não secreta", "bindings"),
		CSRFToken: tok,
		Binding:   &bind,
	})
}

type formPage struct {
	pageBase
	CSRFToken      string
	Error          string
	Taxpayers      []adminregistry.Taxpayer
	Establishments []adminregistry.Establishment
	Binding        *adminregistry.ScopeBinding
}

func (h *Handler) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	if h.CSRF == nil {
		http.Error(w, "csrf indisponível", http.StatusServiceUnavailable)
		return false
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "pedido inválido", http.StatusBadRequest)
		return false
	}
	if !h.CSRF.Validate(r, r.FormValue(csrfFieldName)) {
		http.Error(w, "CSRF inválido", http.StatusForbidden)
		return false
	}
	return true
}

func (h *Handler) formError(w http.ResponseWriter, r *http.Request, tmpl, nav, heading string, err error) {
	tok, _ := h.CSRF.Issue(w)
	msg := "validação falhou"
	if err != nil {
		msg = err.Error()
		// strip package prefix for UI
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = strings.TrimSpace(msg[i+2:])
		}
	}
	tps, _ := h.Registry.ListTaxpayers(r.Context(), listLimit)
	ests, _ := h.Registry.ListEstablishments(r.Context(), "", listLimit)
	page := formPage{
		pageBase:  h.base(r, heading, heading, nav),
		CSRFToken: tok, Error: msg, Taxpayers: tps, Establishments: ests,
	}
	h.render(w, tmpl, page)
}

func htmlRequireWrite(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := adminauth.ClaimsFromContext(r.Context())
		if !ok || !adminauth.Allows(claims, adminauth.PermCadastroWrite) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="pt"><body><h1>Proibido</h1><p>Sem permissão cadastro.write.</p></body></html>`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
