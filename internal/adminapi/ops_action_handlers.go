package adminapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
)

type opsActionReq struct {
	Action            string `json:"action"`
	ExpectedUpdatedAt string `json:"expected_updated_at"`
}

type opsActionResp struct {
	SubmissionID     string `json:"submission_id"`
	QueueStatus      string `json:"queue_status"`
	OutboxState      string `json:"outbox_state"`
	OpsDisposition   string `json:"ops_disposition,omitempty"`
	IdempotentReplay bool   `json:"idempotent_replay"`
	UpdatedAt        string `json:"updated_at"`
}

func (h *Handler) applyOpsSubmissionAction(w http.ResponseWriter, r *http.Request) {
	claims, _ := adminauth.ClaimsFromContext(r.Context())
	subID := r.PathValue("submission_id")
	actionName := "ops.queue_action"
	idem := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	var req opsActionReq
	if err := decodeJSON(r, &req); err != nil {
		h.audit(r, claims, actionName, "ops_submission", subID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	expected, err := parseRFC3339(req.ExpectedUpdatedAt)
	if err != nil || idem == "" {
		h.audit(r, claims, actionName, "ops_submission", subID, adminaudit.ResultError)
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
		return
	}
	if h.Ops == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_OPS_UNAVAILABLE", "Ops Unavailable")
		return
	}
	out, err := h.Ops.ApplyQueueAction(r.Context(), adminops.ActionInput{
		SubmissionID: subID, Action: req.Action, IdempotencyKey: idem,
		ExpectedUpdatedAt: expected, AuthorityMode: h.AuthorityMode, FiscalEnv: h.FiscalEnv,
	})
	if err != nil {
		h.writeOpsErr(w, r, claims, actionName, subID, err)
		return
	}
	_ = h.Audit.Record(r.Context(), claims, actionName+"."+req.Action, "ops_submission", subID, adminaudit.ResultSuccess, requestID(r))
	writeJSON(w, http.StatusOK, opsActionResp{
		SubmissionID: out.SubmissionID, QueueStatus: out.QueueStatus, OutboxState: out.OutboxState,
		OpsDisposition: out.Disposition, IdempotentReplay: out.IdempotentReplay,
		UpdatedAt: out.UpdatedAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) writeOpsErr(w http.ResponseWriter, r *http.Request, claims adminauth.Claims, action, subID string, err error) {
	h.audit(r, claims, action, "ops_submission", subID, adminaudit.ResultError)
	switch {
	case errors.Is(err, adminops.ErrValidation):
		writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
	case errors.Is(err, adminops.ErrConflict):
		writeProblem(w, r, http.StatusConflict, "ADMIN_CONFLICT", "Conflict")
	case errors.Is(err, adminops.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
	case errors.Is(err, adminops.ErrForbiddenEnv):
		writeProblem(w, r, http.StatusForbidden, "ADMIN_FORBIDDEN_ENV", "Forbidden")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
	}
}

func parseRFC3339(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("empty time")
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	return time.Parse(time.RFC3339, raw)
}
