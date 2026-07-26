// Package adminops provides read-only ops visibility for Admin API (RM-BO-003).
// Never returns secret bodies, credentials, or full fiscal document payloads.
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

// SubmissionSummary is a sanitized ops row (no document body / secrets).
type SubmissionSummary struct {
	SubmissionID    string
	DocumentID      string
	OutboxState     string
	LedgerStatus    string // latest ledger to_status; empty if none
	LatestOutcome   string // authority outcome; empty if none
	OutboxUpdatedAt time.Time
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

// ListSubmissionSummaries returns recent outbox submissions with ledger/outcome metadata.
func (s *Store) ListSubmissionSummaries(ctx context.Context, limit int) ([]SubmissionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	t := s.t
	ph := s.p
	q := `
SELECT o.submission_id, o.document_id, o.state, o.updated_at,
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
  ), '')
FROM ` + t("outbox_messages") + ` o
ORDER BY o.updated_at DESC
LIMIT ` + ph(1)
	rows, err := s.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("adminops: list submissions: %w", err)
	}
	defer rows.Close()
	out := make([]SubmissionSummary, 0)
	for rows.Next() {
		var row SubmissionSummary
		var updated any
		if err := rows.Scan(
			&row.SubmissionID, &row.DocumentID, &row.OutboxState, &updated,
			&row.LedgerStatus, &row.LatestOutcome,
		); err != nil {
			return nil, err
		}
		row.OutboxUpdatedAt, err = parseTime(updated)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
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
		return time.Parse(time.RFC3339Nano, x)
	case []byte:
		return time.Parse(time.RFC3339Nano, string(x))
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
