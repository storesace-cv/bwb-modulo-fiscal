package adminui

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// saftStatusPage is a read-only structural status view (≠ AGT / export de produção).
// Must not render NIF, tokens, or full fiscal document payloads.
type saftStatusPage struct {
	pageBase
	SourceID        string
	SchemaVersion   string
	CatalogStatus   string
	Certified       bool
	TargetNamespace string
	DocumentGroups  []string
	PersistenceGaps []string
	Disclaimer      string
}

func (h *Handler) saftStatus(w http.ResponseWriter, r *http.Request) {
	meta := saftao.Meta()
	groups := saftao.AllDocumentGroups()
	groupNames := make([]string, 0, len(groups))
	for _, g := range groups {
		groupNames = append(groupNames, string(g))
	}
	page := saftStatusPage{
		pageBase:        h.baseWithCSRF(w, r, "SAF-T AO", "SAF-T (AO) — estado estrutural", "saft"),
		SourceID:        meta.SourceID,
		SchemaVersion:   meta.SchemaVersion,
		CatalogStatus:   meta.Status,
		Certified:       meta.Certified,
		TargetNamespace: meta.TargetNamespace,
		DocumentGroups:  groupNames,
		PersistenceGaps: []string{
			"GAP-SAFT-PAY-PERSIST — recibos/pagamentos",
			"GAP-SAFT-PUR-PERSIST — compras",
			"GAP-SAFT-MOV-PERSIST — movimentos de stock",
			"GAP-SAFT-WRK-PERSIST — documentos de trabalho",
			"GAP-SAFT-GLE-PERSIST — diário contabilístico",
		},
		Disclaimer: "Estrutura XSD ≠ conformidade AGT. Sem download de XML fiscal nesta página. Fonte pending_validation.",
	}
	h.recordUIAccess(r, "ui.saft.read", "saft_ui", "status", adminaudit.ResultSuccess)
	h.render(w, "saft.html", page)
}
