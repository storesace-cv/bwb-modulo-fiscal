// Package adminui serves the M7 backoffice (RM-ARCH-005 / RM-UI-001).
//
// Server-rendered HTML (Go html/template + embed) in the fiscal-api monolith —
// aligned with DEC-STACK-001. No SPA, no secrets in the browser.
// Auth uses adminauth.Authenticator (OIDC/JWT contract); production fail-closed.
package adminui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminobs"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

//go:embed templates/*.html static/*
var embedded embed.FS

const (
	cookieName = "fiscal_admin_ui_session"
	listLimit  = 50
)

// Handler serves /admin/ui pages.
type Handler struct {
	Registry     *adminregistry.Registry
	Ops          *adminops.Store
	Audit        *adminaudit.Store
	SecretsMeta  secretstore.AdminView // Metadata only — never Reveal
	SecAdm       *secadm.Gate          // owner subject gate for material write
	Sessions     *SessionStore
	TokenAuth    adminauth.Authenticator // Bearer validator for session mint (oidc_jwt)
	Obs          *adminobs.Observer
	EnvLabel     string
	CSRF         *CSRFStore
	CookieSecure bool
}

// New builds a Handler. Ops/Audit may be nil (pages return empty/unavailable).
func New(reg *adminregistry.Registry, envLabel string) (*Handler, error) {
	if reg == nil {
		return nil, fmt.Errorf("adminui: registry nil")
	}
	if strings.TrimSpace(envLabel) == "" {
		envLabel = "unknown"
	}
	secure := envLabel != "development"
	return &Handler{
		Registry:     reg,
		EnvLabel:     envLabel,
		CSRF:         NewCSRFStore(nil),
		Sessions:     NewSessionStore(nil, secure),
		CookieSecure: secure,
	}, nil
}

// Mount registers UI routes. Static CSS is public; pages require auth + cadastro.read.
func Mount(mux *http.ServeMux, authn adminauth.Authenticator, h *Handler) {
	if h == nil {
		return
	}
	staticFS, err := fs.Sub(embedded, "static")
	if err != nil {
		panic("adminui: static embed: " + err.Error())
	}
	obs := h.Obs
	if obs == nil {
		obs = adminobs.New(nil, "fail_closed")
	}
	authn = adminobs.ObservingAuthenticator{Inner: authn, Obs: obs}
	obsPublic := func(next http.Handler) http.Handler {
		return obs.Middleware(securityHeaders(next))
	}
	obsAuth := func(next http.Handler) http.Handler {
		return obs.Middleware(securityHeaders(htmlAuthMiddleware(authn)(adminobs.CaptureClaims(next))))
	}

	mux.Handle("GET /admin/ui/static/", obsPublic(http.StripPrefix("/admin/ui/static/", http.FileServer(http.FS(staticFS)))))

	// Public auth endpoints (no session required).
	mux.Handle("GET /admin/ui/login", obsPublic(http.HandlerFunc(h.loginPage)))
	mux.Handle("POST /admin/ui/auth/session", obsPublic(http.HandlerFunc(h.createSession)))

	read := adminauth.RequirePermission(adminauth.PermCadastroRead)
	wrapRead := func(next http.Handler) http.Handler {
		return obsAuth(read(next))
	}
	wrapWrite := func(next http.Handler) http.Handler {
		return obsAuth(htmlRequireWrite(next))
	}
	wrapPerm := func(perm adminauth.Permission, next http.Handler) http.Handler {
		return obsAuth(htmlRequirePerm(perm, next))
	}
	mux.Handle("GET /admin/ui/", wrapRead(http.HandlerFunc(h.dashboard)))
	mux.Handle("GET /admin/ui/taxpayers", wrapRead(http.HandlerFunc(h.taxpayers)))
	mux.Handle("GET /admin/ui/establishments", wrapRead(http.HandlerFunc(h.establishments)))
	mux.Handle("GET /admin/ui/bindings", wrapRead(http.HandlerFunc(h.bindings)))
	mux.Handle("GET /admin/ui/submissions", wrapPerm(adminauth.PermOpsRead, http.HandlerFunc(h.submissions)))
	mux.Handle("GET /admin/ui/saft", wrapPerm(adminauth.PermOpsRead, http.HandlerFunc(h.saftStatus)))
	mux.Handle("GET /admin/ui/audit", wrapPerm(adminauth.PermAuditRead, http.HandlerFunc(h.auditEvents)))
	wrapOwner := func(next http.Handler) http.Handler {
		return obsAuth(htmlRequireOwnerSecAdm(next))
	}
	mux.Handle("GET /admin/ui/secadm/metadata", wrapOwner(http.HandlerFunc(h.secadmMetaForm)))
	mux.Handle("POST /admin/ui/secadm/metadata", wrapOwner(http.HandlerFunc(h.secadmMetaLookup)))
	mux.Handle("GET /admin/ui/secadm/material", wrapOwner(http.HandlerFunc(h.secadmMaterialForm)))
	mux.Handle("POST /admin/ui/secadm/material", wrapOwner(http.HandlerFunc(h.secadmMaterialSubmit)))
	mux.Handle("POST /admin/ui/secadm/material/revoke", wrapOwner(http.HandlerFunc(h.secadmRevokeForm)))
	mux.Handle("GET /admin/ui/authority-profiles", wrapOwner(http.HandlerFunc(h.authorityProfiles)))
	mux.Handle("GET /admin/ui/authority-profiles/new", wrapOwner(http.HandlerFunc(h.newAuthorityProfileForm)))
	mux.Handle("POST /admin/ui/authority-profiles", wrapOwner(http.HandlerFunc(h.createAuthorityProfileForm)))
	mux.Handle("GET /admin/ui/authority-profiles/{profile_id}/edit", wrapOwner(http.HandlerFunc(h.editAuthorityProfileForm)))
	mux.Handle("POST /admin/ui/authority-profiles/{profile_id}", wrapOwner(http.HandlerFunc(h.patchAuthorityProfileForm)))
	mux.Handle("POST /admin/ui/auth/logout", obsAuth(http.HandlerFunc(h.logout)))

	mux.Handle("GET /admin/ui/taxpayers/new", wrapWrite(http.HandlerFunc(h.newTaxpayerForm)))
	mux.Handle("POST /admin/ui/taxpayers", wrapWrite(http.HandlerFunc(h.createTaxpayerForm)))
	mux.Handle("POST /admin/ui/taxpayers/{taxpayer_id}/status", wrapWrite(http.HandlerFunc(h.patchTaxpayerForm)))
	mux.Handle("GET /admin/ui/establishments/new", wrapWrite(http.HandlerFunc(h.newEstablishmentForm)))
	mux.Handle("POST /admin/ui/establishments", wrapWrite(http.HandlerFunc(h.createEstablishmentForm)))
	mux.Handle("POST /admin/ui/establishments/{establishment_id}/status", wrapWrite(http.HandlerFunc(h.patchEstablishmentForm)))
	mux.Handle("GET /admin/ui/bindings/new", wrapWrite(http.HandlerFunc(h.newBindingForm)))
	mux.Handle("POST /admin/ui/bindings", wrapWrite(http.HandlerFunc(h.createBindingForm)))
	mux.Handle("GET /admin/ui/bindings/{scope_id}/edit", wrapWrite(http.HandlerFunc(h.editBindingForm)))
	mux.Handle("POST /admin/ui/bindings/{scope_id}", wrapWrite(http.HandlerFunc(h.patchBindingForm)))
}

type pageBase struct {
	Title      string
	Heading    string
	Nav        string
	EnvLabel   string
	Subject    string
	RolesLabel string
	Flash      string
	CanWrite   bool
	IsOwner    bool
	CSRFToken  string
}

type dashboardPage struct {
	pageBase
	TaxpayerCount      int
	EstablishmentCount int
	BindingCount       int
	Taxpayers          []adminregistry.Taxpayer
	Establishments     []adminregistry.Establishment
	Bindings           []adminregistry.ScopeBinding
}

type taxpayersPage struct {
	pageBase
	Taxpayers []adminregistry.Taxpayer
}

type establishmentsPage struct {
	pageBase
	Establishments []adminregistry.Establishment
}

type bindingsPage struct {
	pageBase
	Bindings []adminregistry.ScopeBinding
}

func (h *Handler) base(r *http.Request, title, heading, nav string) pageBase {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	roles := make([]string, 0, len(claims.Roles))
	for _, role := range claims.Roles {
		roles = append(roles, string(role))
	}
	return pageBase{
		Title: title, Heading: heading, Nav: nav,
		EnvLabel: h.EnvLabel, Subject: claims.Subject, RolesLabel: strings.Join(roles, ", "),
		CanWrite: adminauth.Allows(claims, adminauth.PermCadastroWrite),
		IsOwner:  adminauth.Allows(claims, adminauth.PermSecAdmWrite),
	}
}

func (h *Handler) baseWithCSRF(w http.ResponseWriter, r *http.Request, title, heading, nav string) pageBase {
	b := h.base(r, title, heading, nav)
	if h.CSRF != nil {
		tok, err := h.CSRF.Issue(w)
		if err == nil {
			b.CSRFToken = tok
		}
	}
	return b
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/ui/" && r.URL.Path != "/admin/ui" {
		http.NotFound(w, r)
		return
	}
	tps, err := h.Registry.ListTaxpayers(r.Context(), listLimit)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	ests, err := h.Registry.ListEstablishments(r.Context(), "", listLimit)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	binds, err := h.Registry.ListScopeBindings(r.Context(), listLimit)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "dashboard.html", dashboardPage{
		pageBase:           h.baseWithCSRF(w, r, "Painel", "Painel operacional", "dashboard"),
		TaxpayerCount:      len(tps),
		EstablishmentCount: len(ests),
		BindingCount:       len(binds),
		Taxpayers:          tps,
		Establishments:     ests,
		Bindings:           binds,
	})
}

func (h *Handler) taxpayers(w http.ResponseWriter, r *http.Request) {
	tps, err := h.Registry.ListTaxpayers(r.Context(), listLimit)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "taxpayers.html", taxpayersPage{
		pageBase:  h.baseWithCSRF(w, r, "Contribuintes", "Contribuintes", "taxpayers"),
		Taxpayers: tps,
	})
}

func (h *Handler) establishments(w http.ResponseWriter, r *http.Request) {
	ests, err := h.Registry.ListEstablishments(r.Context(), "", listLimit)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "establishments.html", establishmentsPage{
		pageBase:       h.baseWithCSRF(w, r, "Estabelecimentos", "Estabelecimentos", "establishments"),
		Establishments: ests,
	})
}

func (h *Handler) bindings(w http.ResponseWriter, r *http.Request) {
	binds, err := h.Registry.ListScopeBindings(r.Context(), listLimit)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	h.render(w, "bindings.html", bindingsPage{
		pageBase: h.baseWithCSRF(w, r, "Bindings", "Scope bindings", "bindings"),
		Bindings: binds,
	})
}

func (h *Handler) render(w http.ResponseWriter, pageFile string, data any) {
	t, err := template.ParseFS(embedded,
		"templates/layout.html",
		"templates/partials.html",
		"templates/"+pageFile,
	)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; style-src 'self'; img-src 'self'; font-src 'self'; "+
				"frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
