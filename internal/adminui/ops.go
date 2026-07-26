package adminui

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
)

type submissionsPage struct {
	pageBase
	Items []adminops.SubmissionSummary
}

type auditPage struct {
	pageBase
	Items []adminaudit.Event
}

func (h *Handler) submissions(w http.ResponseWriter, r *http.Request) {
	page := submissionsPage{pageBase: h.base(r, "Submissões", "Submissões e reconciliação", "submissions")}
	if h.Ops != nil {
		items, err := h.Ops.ListSubmissionSummaries(r.Context(), listLimit)
		if err != nil {
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
		page.Items = items
	}
	h.render(w, "submissions.html", page)
}

func (h *Handler) auditEvents(w http.ResponseWriter, r *http.Request) {
	page := auditPage{pageBase: h.base(r, "Auditoria", "Auditoria admin", "audit")}
	if h.Audit != nil {
		items, err := h.Audit.ListRecent(r.Context(), listLimit)
		if err != nil {
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
		page.Items = items
	}
	h.render(w, "audit.html", page)
}
