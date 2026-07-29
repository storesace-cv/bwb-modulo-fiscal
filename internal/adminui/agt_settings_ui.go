package adminui

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

type agtSettingsPage struct {
	pageBase
	Environment   string
	Catalog       []prep.EndpointCatalogEntry
	JWS           prep.JWSProfileScaffold
	Profiles      []adminregistry.AuthorityProfile
	SecretRefs    []secretstore.Metadata
	StorageMode   string
	CatalogErr    string
	AuthorityMode string
	FiscalEnv     string
}

func (h *Handler) agtSettingsHub(w http.ResponseWriter, r *http.Request) {
	env := r.URL.Query().Get("environment")
	if env == "" {
		env = adminregistry.EnvHomologation
	}
	page := agtSettingsPage{
		pageBase:      h.baseWithCSRF(w, r, "Preparação AGT", "Preparação AGT (owner-only)", "agt"),
		Environment:   env,
		JWS:           prep.JWSProfileScaffoldDefault(),
		AuthorityMode: h.AuthorityMode,
		FiscalEnv:     h.FiscalEnv,
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
				if p.Environment == env {
					page.Profiles = append(page.Profiles, p)
				}
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
