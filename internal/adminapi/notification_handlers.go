package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
)

// POST /admin/v1/ops/notifications/test — owner-only; sends to configured admin address only.
func (h *Handler) postNotificationTest(w http.ResponseWriter, r *http.Request) {
	claims, ok := adminauth.ClaimsFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "ADMIN_UNAUTHORIZED", "Unauthorized")
		return
	}
	if h.Mailer == nil || !h.Mailer.Configured() {
		if h.Audit != nil {
			_ = h.Audit.Record(r.Context(), claims, "notification.test", "smtp", "admin_notification", adminaudit.ResultError, requestID(r))
		}
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SMTP_NOT_CONFIGURED", "SMTP Not Configured")
		return
	}
	st, err := h.Mailer.SendAdminTest(r.Context(), requestID(r))
	result := adminaudit.ResultSuccess
	if err != nil || st.Status != "sent" {
		result = adminaudit.ResultError
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), claims, "notification.test", "smtp", "admin_notification", result, requestID(r))
	}
	if err != nil || st.Status != "sent" {
		writeJSON(w, http.StatusBadGateway, st)
		return
	}
	writeJSON(w, http.StatusOK, st)
}
