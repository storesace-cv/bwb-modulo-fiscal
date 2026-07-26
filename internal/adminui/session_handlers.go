package adminui

import (
	"net/http"
)

type loginPage struct {
	pageBase
	CSRFToken string
}

func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request) {
	tok := ""
	if h.CSRF != nil {
		tok, _ = h.CSRF.Issue(w)
	}
	h.render(w, "login.html", loginPage{
		pageBase: pageBase{
			Title: "Entrar", Heading: "Sessão backoffice", Nav: "login",
			EnvLabel: h.EnvLabel,
		},
		CSRFToken: tok,
	})
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	if h.Sessions == nil || h.TokenAuth == nil {
		http.Error(w, "sessão indisponível", http.StatusServiceUnavailable)
		return
	}
	claims, err := h.TokenAuth.Authenticate(r.Context(), r)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="pt"><body><h1>Não autenticado</h1><p>Bearer JWT inválido ou ausente.</p></body></html>`))
		return
	}
	if _, err := h.Sessions.Create(w, claims); err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	// Never echo JWT. Opaque session cookie only.
	http.Redirect(w, r, "/admin/ui/", http.StatusSeeOther)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	if h.Sessions != nil {
		h.Sessions.Destroy(w, r)
	}
	http.Redirect(w, r, "/admin/ui/login", http.StatusSeeOther)
}
