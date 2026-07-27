package adminapi

import (
	"net/http"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/notify/smtp"
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

type alertDigestResp struct {
	smtp.DeliveryStatus
	AlertCount int      `json:"alert_count"`
	AlertCodes []string `json:"alert_codes"`
}

// POST /admin/v1/ops/notifications/alerts-digest — owner-only; ops queue alerts only (RM-OPS-009).
func (h *Handler) postNotificationAlertsDigest(w http.ResponseWriter, r *http.Request) {
	claims, ok := adminauth.ClaimsFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "ADMIN_UNAUTHORIZED", "Unauthorized")
		return
	}
	if h.Mailer == nil || !h.Mailer.Configured() {
		if h.Audit != nil {
			_ = h.Audit.Record(r.Context(), claims, "notification.alerts_digest", "smtp", "admin_notification", adminaudit.ResultError, requestID(r))
		}
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SMTP_NOT_CONFIGURED", "SMTP Not Configured")
		return
	}
	load := h.OpsDashboardFn
	if load == nil && h.Ops != nil {
		load = h.Ops.LoadOpsDashboard
	}
	if load == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_OPS_UNAVAILABLE", "Ops Store Unavailable")
		return
	}
	dash, err := load(r.Context())
	if err != nil {
		if h.Audit != nil {
			_ = h.Audit.Record(r.Context(), claims, "notification.alerts_digest", "smtp", "admin_notification", adminaudit.ResultError, requestID(r))
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_OPS_DASHBOARD_FAILED", "Ops Dashboard Failed")
		return
	}
	lines := adminops.AlertDigestLines(dash.Alerts)
	codes := adminops.AlertCodes(dash.Alerts)
	st, sendErr := h.Mailer.SendAdminAlertDigest(r.Context(), requestID(r), lines)
	result := adminaudit.ResultSuccess
	if sendErr != nil || st.Status != "sent" {
		result = adminaudit.ResultError
	}
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), claims, "notification.alerts_digest", "smtp", "admin_notification", result, requestID(r))
	}
	resp := alertDigestResp{DeliveryStatus: st, AlertCount: len(codes), AlertCodes: codes}
	if codes == nil {
		resp.AlertCodes = []string{}
	}
	if sendErr != nil || st.Status != "sent" {
		writeJSON(w, http.StatusBadGateway, resp)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
