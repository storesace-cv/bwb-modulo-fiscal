package adminapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

type auditEventResp struct {
	EventID      string `json:"event_id"`
	OccurredAt   string `json:"occurred_at"`
	ActorSubject string `json:"actor_subject"`
	ActorRoles   string `json:"actor_roles"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Result       string `json:"result"`
	RequestID    string `json:"request_id,omitempty"`
}

type submissionSummaryResp struct {
	SubmissionID       string  `json:"submission_id"`
	DocumentID         string  `json:"document_id"`
	OutboxState        string  `json:"outbox_state"`
	QueueStatus        string  `json:"queue_status"`
	LedgerStatus       string  `json:"ledger_status,omitempty"`
	LatestOutcome      string  `json:"latest_outcome,omitempty"`
	Attempts           int64   `json:"attempts"`
	NextAttemptAt      *string `json:"next_attempt_at,omitempty"`
	AuthorityRequestID string  `json:"authority_request_id,omitempty"`
	SanitizedError     string  `json:"sanitized_error,omitempty"`
	OutboxUpdatedAt    string  `json:"outbox_updated_at"`
}

type secretMetadataResp struct {
	Kind           string  `json:"kind"`
	Environment    string  `json:"environment"`
	SubjectID      string  `json:"subject_id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Fingerprint    string  `json:"fingerprint,omitempty"`
	Version        int     `json:"version"`
	ExpiresAt      *string `json:"expires_at,omitempty"`
	LastVerifiedAt *string `json:"last_verified_at,omitempty"`
}

func (h *Handler) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	if h.Audit == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE", "Audit Unavailable")
		return
	}
	limit := adminops.ClampLimit(r.URL.Query().Get("limit"))
	events, err := h.Audit.ListRecent(r.Context(), limit)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	out := make([]auditEventResp, 0, len(events))
	for _, e := range events {
		out = append(out, auditEventResp{
			EventID: e.ID, OccurredAt: e.OccurredAt.UTC().Format(time.RFC3339Nano),
			ActorSubject: e.ActorSubject, ActorRoles: e.ActorRoles,
			Action: e.Action, ResourceType: e.ResourceType, ResourceID: e.ResourceID,
			Result: e.Result, RequestID: e.RequestID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) listOpsSubmissions(w http.ResponseWriter, r *http.Request) {
	if h.Ops == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_OPS_UNAVAILABLE", "Ops Unavailable")
		return
	}
	limit := adminops.ClampLimit(r.URL.Query().Get("limit"))
	rows, err := h.Ops.ListSubmissionSummariesFiltered(r.Context(), adminops.SubmissionFilter{
		Limit:       limit,
		QueueStatus: r.URL.Query().Get("queue_status"),
		OutboxState: r.URL.Query().Get("outbox_state"),
	})
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	out := make([]submissionSummaryResp, 0, len(rows))
	for _, row := range rows {
		item := submissionSummaryResp{
			SubmissionID: row.SubmissionID, DocumentID: row.DocumentID,
			OutboxState: row.OutboxState, QueueStatus: row.QueueStatus,
			LedgerStatus: row.LedgerStatus, LatestOutcome: row.LatestOutcome,
			Attempts: row.Attempts, AuthorityRequestID: row.AuthorityRequestID,
			SanitizedError:  row.SanitizedError,
			OutboxUpdatedAt: row.OutboxUpdatedAt.UTC().Format(time.RFC3339Nano),
		}
		if row.NextAttemptAt != nil {
			s := row.NextAttemptAt.UTC().Format(time.RFC3339Nano)
			item.NextAttemptAt = &s
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) getSecretRefMetadata(w http.ResponseWriter, r *http.Request) {
	if h.SecretsMeta == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "ADMIN_SECRETS_META_UNAVAILABLE", "Secrets Metadata Unavailable")
		return
	}
	q := r.URL.Query()
	ref := secretstore.Ref{
		Kind: q.Get("kind"), Environment: q.Get("environment"),
		SubjectID: q.Get("subject_id"), Name: q.Get("name"),
	}
	meta, err := h.SecretsMeta.Metadata(r.Context(), ref)
	if err != nil {
		if errors.Is(err, secretstore.ErrNotFound) {
			writeProblem(w, r, http.StatusNotFound, "ADMIN_NOT_FOUND", "Not Found")
			return
		}
		if errors.Is(err, secretstore.ErrValidation) {
			writeProblem(w, r, http.StatusUnprocessableEntity, "ADMIN_VALIDATION", "Unprocessable Entity")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "ADMIN_ERROR", "Internal Server Error")
		return
	}
	resp := secretMetadataResp{
		Kind: meta.Ref.Kind, Environment: meta.Environment, SubjectID: meta.Ref.SubjectID,
		Name: meta.Ref.Name, Status: meta.Status, Fingerprint: meta.Fingerprint, Version: meta.Version,
	}
	if meta.ExpiresAt != nil {
		s := meta.ExpiresAt.UTC().Format(time.RFC3339Nano)
		resp.ExpiresAt = &s
	}
	if meta.LastVerifiedAt != nil {
		s := meta.LastVerifiedAt.UTC().Format(time.RFC3339Nano)
		resp.LastVerifiedAt = &s
	}
	writeJSON(w, http.StatusOK, resp)
}
