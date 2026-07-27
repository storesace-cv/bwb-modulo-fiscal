package adminui

import (
	"net/http"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

type auditPage struct {
	pageBase
	Items []adminaudit.Event
}

func (h *Handler) auditEvents(w http.ResponseWriter, r *http.Request) {
	page := auditPage{pageBase: h.baseWithCSRF(w, r, "Auditoria", "Auditoria admin", "audit")}
	h.recordUIAccess(r, "ui.audit.read", "audit_ui", "events", adminaudit.ResultSuccess)
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

func (h *Handler) recordUIAccess(r *http.Request, action, resourceType, resourceID, result string) {
	if h.Audit == nil {
		return
	}
	claims, ok := adminauth.ClaimsFromContext(r.Context())
	if !ok {
		return
	}
	_ = h.Audit.Record(r.Context(), claims, action, resourceType, resourceID, result, uiRequestID(r))
}

func uiRequestID(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Request-Id"))
}
