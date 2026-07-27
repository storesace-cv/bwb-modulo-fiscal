package adminui

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminaudit"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
)

type submissionActionRow struct {
	adminops.SubmissionSummary
	IdemRetry  string
	IdemCancel string
	IdemReview string
}

type submissionsPage struct {
	pageBase
	Items       []submissionActionRow
	QueueStatus string
	FlashError  string
}

func (h *Handler) submissions(w http.ResponseWriter, r *http.Request) {
	queueStatus := strings.TrimSpace(r.URL.Query().Get("queue_status"))
	page := submissionsPage{
		pageBase:    h.baseWithCSRF(w, r, "Submissões", "Fila de submissões", "submissions"),
		QueueStatus: queueStatus,
		FlashError:  strings.TrimSpace(r.URL.Query().Get("err")),
	}
	if h.Ops != nil {
		items, err := h.Ops.ListSubmissionSummariesFiltered(r.Context(), adminops.SubmissionFilter{
			Limit: listLimit, QueueStatus: queueStatus,
		})
		if err != nil {
			h.recordUIAccess(r, "ui.ops.read", "ops_ui", "submissions", adminaudit.ResultError)
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
		page.Items = make([]submissionActionRow, 0, len(items))
		for _, it := range items {
			page.Items = append(page.Items, submissionActionRow{
				SubmissionSummary: it,
				IdemRetry:         newIdemKey(),
				IdemCancel:        newIdemKey(),
				IdemReview:        newIdemKey(),
			})
		}
	}
	h.recordUIAccess(r, "ui.ops.read", "ops_ui", "submissions", adminaudit.ResultSuccess)
	h.render(w, "submissions.html", page)
}

func (h *Handler) submissionActionForm(w http.ResponseWriter, r *http.Request) {
	if h.CSRF == nil || !h.requireCSRF(w, r) {
		return
	}
	subID := r.PathValue("submission_id")
	action := strings.TrimSpace(r.FormValue("action"))
	idem := strings.TrimSpace(r.FormValue("idempotency_key"))
	expectedRaw := strings.TrimSpace(r.FormValue("expected_updated_at"))
	expected, err := time.Parse(time.RFC3339Nano, expectedRaw)
	if err != nil {
		expected, err = time.Parse(time.RFC3339, expectedRaw)
	}
	redirectErr := func(code string) {
		http.Redirect(w, r, "/admin/ui/submissions?err="+code, http.StatusSeeOther)
	}
	if err != nil || subID == "" || action == "" || idem == "" {
		h.recordUIAccess(r, "ui.ops.action", "ops_submission", subID, adminaudit.ResultError)
		redirectErr("validation")
		return
	}
	if h.Ops == nil {
		redirectErr("unavailable")
		return
	}
	mode := h.AuthorityMode
	env := h.FiscalEnv
	if env == "" {
		env = h.EnvLabel
	}
	_, err = h.Ops.ApplyQueueAction(r.Context(), adminops.ActionInput{
		SubmissionID: subID, Action: action, IdempotencyKey: idem,
		ExpectedUpdatedAt: expected, AuthorityMode: mode, FiscalEnv: env,
	})
	if err != nil {
		h.recordUIAccess(r, "ui.ops.action", "ops_submission", subID, adminaudit.ResultError)
		switch {
		case errors.Is(err, adminops.ErrForbiddenEnv):
			redirectErr("forbidden_env")
		case errors.Is(err, adminops.ErrConflict):
			redirectErr("conflict")
		case errors.Is(err, adminops.ErrNotFound):
			redirectErr("not_found")
		case errors.Is(err, adminops.ErrValidation):
			redirectErr("validation")
		default:
			redirectErr("error")
		}
		return
	}
	h.recordUIAccess(r, "ui.ops.action."+action, "ops_submission", subID, adminaudit.ResultSuccess)
	http.Redirect(w, r, "/admin/ui/submissions", http.StatusSeeOther)
}

func newIdemKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "ui-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
