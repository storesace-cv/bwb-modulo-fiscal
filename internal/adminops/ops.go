// Package adminops provides read-only ops visibility for Admin API (RM-BO-003 / RM-BO-015).
// Never returns secret bodies, credentials, JWS, NIF, or full fiscal document payloads.
package adminops

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Dialect selects SQL dialect.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// Queue statuses exposed to ops UI/API (derived; do not invent AGT codes).
const (
	QueuePending      = "pending"
	QueueProcessing   = "processing"
	QueueAccepted     = "accepted"
	QueueRejected     = "rejected"
	QueueRetry        = "retry"
	QueueManualReview = "manual_review"
)

// SubmissionSummary is a sanitized ops row (no document body / secrets / JWS).
type SubmissionSummary struct {
	SubmissionID       string
	DocumentID         string
	OutboxState        string
	QueueStatus        string // derived for ops panel
	LedgerStatus       string
	LatestOutcome      string
	Attempts           int64
	NextAttemptAt      *time.Time // outbox available_at when pending/retry
	AuthorityRequestID string     // latest response correlation; never a secret
	SanitizedError     string     // allowlisted outcome/code only
	OutboxUpdatedAt    time.Time
}

// Store reads ops tables without mutation.
type Store struct {
	db      *sql.DB
	dialect Dialect
}

// New returns an ops Store.
func New(db *sql.DB, dialect Dialect) *Store {
	return &Store{db: db, dialect: dialect}
}

// ListSubmissionSummaries returns recent outbox submissions with sanitized queue metadata (RM-BO-015).
func (s *Store) ListSubmissionSummaries(ctx context.Context, limit int) ([]SubmissionSummary, error) {
	return s.ListSubmissionSummariesFiltered(ctx, SubmissionFilter{Limit: limit})
}

// SubmissionFilter is a read filter for the ops queue panel.
type SubmissionFilter struct {
	Limit       int
	QueueStatus string // optional; empty = all
	OutboxState string // optional exact outbox state
}

// ListSubmissionSummariesFiltered lists submissions with optional queue/outbox filters.
func (s *Store) ListSubmissionSummariesFiltered(ctx context.Context, f SubmissionFilter) ([]SubmissionSummary, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	t := s.t
	ph := s.p
	q := `
SELECT o.submission_id, o.document_id, o.state, o.available_at, o.updated_at,
  COALESCE((
    SELECT le.to_status FROM ` + t("ledger_events") + ` le
    WHERE le.document_id = o.document_id
    ORDER BY le.seq DESC LIMIT 1
  ), ''),
  COALESCE((
    SELECT COALESCE(r.outcome, '') FROM ` + t("authority_attempts") + ` a
    LEFT JOIN ` + t("authority_responses") + ` r ON r.attempt_id = a.id
    WHERE a.submission_id = o.submission_id
    ORDER BY a.attempt_no DESC LIMIT 1
  ), ''),
  COALESCE((
    SELECT COALESCE(r.authority_request_id, '') FROM ` + t("authority_attempts") + ` a
    LEFT JOIN ` + t("authority_responses") + ` r ON r.attempt_id = a.id
    WHERE a.submission_id = o.submission_id
    ORDER BY a.attempt_no DESC LIMIT 1
  ), ''),
  COALESCE((
    SELECT COUNT(*) FROM ` + t("authority_attempts") + ` a
    WHERE a.submission_id = o.submission_id
  ), 0)
FROM ` + t("outbox_messages") + ` o`
	args := make([]any, 0, 2)
	where := make([]string, 0, 1)
	if st := strings.TrimSpace(f.OutboxState); st != "" {
		where = append(where, "o.state = "+ph(len(args)+1))
		args = append(args, st)
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += `
ORDER BY o.updated_at DESC
LIMIT ` + ph(len(args)+1)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("adminops: list submissions: %w", err)
	}
	defer rows.Close()
	out := make([]SubmissionSummary, 0)
	wantQueue := strings.TrimSpace(f.QueueStatus)
	for rows.Next() {
		var row SubmissionSummary
		var available, updated any
		if err := rows.Scan(
			&row.SubmissionID, &row.DocumentID, &row.OutboxState, &available, &updated,
			&row.LedgerStatus, &row.LatestOutcome, &row.AuthorityRequestID, &row.Attempts,
		); err != nil {
			return nil, err
		}
		row.OutboxUpdatedAt, err = parseTime(updated)
		if err != nil {
			return nil, err
		}
		if at, err := parseTime(available); err == nil && !at.IsZero() {
			row.NextAttemptAt = &at
		}
		row.AuthorityRequestID = sanitizeRequestID(row.AuthorityRequestID)
		row.SanitizedError = SanitizeOpsError(row.LatestOutcome, row.OutboxState)
		row.QueueStatus = DeriveQueueStatus(row.OutboxState, row.LatestOutcome, row.Attempts)
		if wantQueue != "" && row.QueueStatus != wantQueue {
			continue
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// DeriveQueueStatus maps outbox/outcome/attempts to ops queue status (fail-closed allowlist).
func DeriveQueueStatus(outboxState, outcome string, attempts int64) string {
	switch strings.TrimSpace(outboxState) {
	case "in_flight":
		return QueueProcessing
	case "dead":
		return QueueManualReview
	case "succeeded":
		switch strings.TrimSpace(outcome) {
		case "authority_accepted":
			return QueueAccepted
		case "authority_rejected":
			return QueueRejected
		default:
			return QueueManualReview
		}
	case "pending":
		if attempts > 0 {
			return QueueRetry
		}
		return QueuePending
	default:
		return QueueManualReview
	}
}

// SanitizeOpsError returns an allowlisted error token — never payloads, JWS, NIF, or free text.
func SanitizeOpsError(outcome, outboxState string) string {
	outcome = strings.TrimSpace(outcome)
	switch outcome {
	case "authority_rejected", "authority_outcome_unknown", "retried_unavailable":
		return outcome
	}
	if strings.TrimSpace(outboxState) == "dead" {
		return "outbox_dead"
	}
	return ""
}

func sanitizeRequestID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	// Reject values that look like secrets/PEM/JWS fragments.
	low := strings.ToLower(id)
	for _, banned := range []string{"-----begin", "eyj", "bearer ", "password", "nif=", "postgres://"} {
		if strings.Contains(low, banned) {
			return ""
		}
	}
	if len(id) > 128 {
		return id[:128]
	}
	return id
}

func (s *Store) t(name string) string {
	if s.dialect == DialectPostgres {
		return "fiscal." + name
	}
	return name
}

func (s *Store) p(n int) string {
	if s.dialect == DialectPostgres {
		return "$" + strconv.Itoa(n)
	}
	return "?"
}

func parseTime(v any) (time.Time, error) {
	switch x := v.(type) {
	case time.Time:
		return x.UTC(), nil
	case string:
		if x == "" {
			return time.Time{}, nil
		}
		if t, err := time.Parse(time.RFC3339Nano, x); err == nil {
			return t.UTC(), nil
		}
		return time.Parse(time.RFC3339, x)
	case []byte:
		return parseTime(string(x))
	default:
		return time.Time{}, fmt.Errorf("adminops: time tipo %T", v)
	}
}

// ClampLimit parses limit query with defaults.
func ClampLimit(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 50
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}
