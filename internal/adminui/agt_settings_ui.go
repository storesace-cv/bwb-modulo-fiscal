package adminui

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

type agtSettingsPage struct {
	pageBase
	Environment   string
	Catalog       []prep.EndpointCatalogEntry
	JWS           prep.JWSProfileScaffold
	Connectivity  prep.ConnectivityStatus
	Profiles      []agtProfileHubRow
	SecretRefs    []secretstore.Metadata
	StorageMode   string
	CatalogErr    string
	AuthorityMode string
	FiscalEnv     string
	ProbeFlash    string
	ProbeError    string
}

type agtProfileHubRow struct {
	adminregistry.AuthorityProfile
	BindingsValid bool
	IssueCount    int
}

func (h *Handler) agtSettingsHub(w http.ResponseWriter, r *http.Request) {
	env := r.URL.Query().Get("environment")
	if env == "" {
		env = adminregistry.EnvHomologation
	}
	mode := h.AuthorityMode
	if mode == "" {
		mode = "simulator"
	}
	var lastAt *time.Time
	lastResult, lastMode := "", ""
	if h.Audit != nil {
		if evs, err := h.Audit.ListByResource(r.Context(), "authority", mode, 20); err == nil {
			for _, e := range evs {
				if e.Action == "authority.probe_config" {
					t := e.OccurredAt.UTC()
					lastAt = &t
					lastResult = e.Result
					lastMode = e.ResourceID
					break
				}
			}
		}
	}
	page := agtSettingsPage{
		pageBase:      h.baseWithCSRF(w, r, "Preparação AGT", "Preparação AGT (owner-only)", "agt"),
		Environment:   env,
		JWS:           prep.JWSProfileScaffoldDefault(),
		Connectivity:  prep.BuildConnectivityStatus(h.FiscalEnv, mode, lastResult, lastMode, lastAt),
		AuthorityMode: mode,
		FiscalEnv:     h.FiscalEnv,
		ProbeFlash:    strings.TrimSpace(r.URL.Query().Get("probe_ok")),
		ProbeError:    strings.TrimSpace(r.URL.Query().Get("probe_err")),
	}
	rows, err := prep.EndpointCatalog(env)
	if err != nil {
		page.CatalogErr = "Ambiente inválido (use homologation|production)."
	} else {
		page.Catalog = rows
	}
	if h.Registry != nil {
		list, err := h.Registry.ListAuthorityProfiles(r.Context())
		if err == nil {
			for _, p := range list {
				if p.Environment != env {
					continue
				}
				row := agtProfileHubRow{AuthorityProfile: p, BindingsValid: true}
				if h.SecretsMeta != nil {
					bv, berr := prep.ValidateProfileBindings(r.Context(), p, func(ref secretstore.Ref) (secretstore.Metadata, error) {
						return h.SecretsMeta.Metadata(r.Context(), ref)
					})
					if berr == nil {
						row.BindingsValid = bv.Valid
						row.IssueCount = len(bv.Issues)
					} else {
						row.BindingsValid = false
						row.IssueCount = 1
					}
				}
				page.Profiles = append(page.Profiles, row)
			}
		}
	}
	if h.SecretsMeta != nil {
		if v, ok := h.SecretsMeta.(interface{ StorageMode() string }); ok {
			page.StorageMode = v.StorageMode()
		}
		if refs, err := h.SecretsMeta.ListMetadata(r.Context(), env); err == nil {
			page.SecretRefs = refs
		}
	}
	h.recordUIAccess(r, "ui.agt_settings.read", "authority", env, adminaudit.ResultSuccess)
	h.render(w, "agt_settings.html", page)
}

func (h *Handler) agtSettingsProbe(w http.ResponseWriter, r *http.Request) {
	if !h.requireCSRF(w, r) {
		return
	}
	env := strings.TrimSpace(r.FormValue("environment"))
	if env == "" {
		env = adminregistry.EnvHomologation
	}
	redirect := "/admin/ui/agt-settings?environment=" + url.QueryEscape(env)
	mode := h.AuthorityMode
	if mode == "" {
		mode = "simulator"
	}
	claims, ok := adminauth.ClaimsFromContext(r.Context())
	if !ok {
		http.Redirect(w, r, redirect+"&probe_err="+url.QueryEscape("sessão inválida"), http.StatusSeeOther)
		return
	}
	action := "authority.probe_config"
	if err := prep.FailClosedProduction(h.FiscalEnv, mode); err != nil {
		if h.Audit != nil {
			_ = h.Audit.Record(r.Context(), claims, action, "authority", mode, adminaudit.ResultDenied, uiRequestID(r))
		}
		http.Redirect(w, r, redirect+"&probe_err="+url.QueryEscape("fail-closed (prod+simulator ou modo AGT reservado)"), http.StatusSeeOther)
		return
	}
	client := simulator.New(simulator.OutcomeAccept)
	rep, err := prep.ProbeSimulator(r.Context(), mode, client)
	if err != nil {
		if h.Audit != nil {
			_ = h.Audit.Record(r.Context(), claims, action, "authority", mode, adminaudit.ResultError, uiRequestID(r))
		}
		http.Redirect(w, r, redirect+"&probe_err="+url.QueryEscape("probe recusado (só simulator; ≠ AGT)"), http.StatusSeeOther)
		return
	}
	result := adminaudit.ResultSuccess
	msg := "probe_ok=true · outcome=" + rep.Outcome + " · external_verified=false"
	if !rep.OK {
		result = adminaudit.ResultError
		msg = "probe_ok=false · simulator inalcançável (≠ AGT)"
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), claims, action, "authority", mode, result, uiRequestID(r))
	}
	if result == adminaudit.ResultSuccess {
		http.Redirect(w, r, redirect+"&probe_ok="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, redirect+"&probe_err="+url.QueryEscape(msg), http.StatusSeeOther)
}
