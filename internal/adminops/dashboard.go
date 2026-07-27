package adminops

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// AlertSeverity is a sanitized ops alert level (never secrets / NIF / payloads).
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityBlocking AlertSeverity = "blocking"
)

// OpsAlert is a non-secret operational warning for the ops dashboard (RM-BO-017).
type OpsAlert struct {
	Code     string
	Severity AlertSeverity
	Message  string
}

// QueueCounts are derived queue buckets (allowlist; no document bodies).
type QueueCounts struct {
	Pending      int64
	Processing   int64
	Accepted     int64
	Rejected     int64
	Retry        int64
	ManualReview int64
	Cancelled    int64
	Total        int64
}

// OpsDashboard is the sanitized ops home snapshot.
type OpsDashboard struct {
	Counts QueueCounts
	Alerts []OpsAlert
}

// LoadOpsDashboard aggregates queue counts and builds alerts (RM-BO-017).
// Caps scan at 10_000 rows fail-closed (ops visibility, not warehouse analytics).
func (s *Store) LoadOpsDashboard(ctx context.Context) (OpsDashboard, error) {
	rows, err := s.scanSubmissionSummaries(ctx, SubmissionFilter{Limit: 10000}, 10000, 0)
	if err != nil {
		return OpsDashboard{}, err
	}
	var c QueueCounts
	for _, row := range rows {
		c.Total++
		switch row.QueueStatus {
		case QueuePending:
			c.Pending++
		case QueueProcessing:
			c.Processing++
		case QueueAccepted:
			c.Accepted++
		case QueueRejected:
			c.Rejected++
		case QueueRetry:
			c.Retry++
		case QueueManualReview:
			c.ManualReview++
		case QueueCancelled:
			c.Cancelled++
		}
	}
	return OpsDashboard{Counts: c, Alerts: BuildOpsAlerts(c)}, nil
}

// BuildOpsAlerts returns allowlisted alerts from queue counts (no IDs/payloads).
func BuildOpsAlerts(c QueueCounts) []OpsAlert {
	var out []OpsAlert
	if c.ManualReview > 0 {
		sev := SeverityWarning
		if c.ManualReview >= 10 {
			sev = SeverityBlocking
		}
		out = append(out, OpsAlert{
			Code: "ops_manual_review_backlog", Severity: sev,
			Message: fmt.Sprintf("fila manual_review=%d — rever na UI ops", c.ManualReview),
		})
	}
	if c.Retry > 0 {
		sev := SeverityInfo
		if c.Retry >= 20 {
			sev = SeverityWarning
		}
		out = append(out, OpsAlert{
			Code: "ops_retry_backlog", Severity: sev,
			Message: fmt.Sprintf("fila retry=%d — monitorizar tentativas", c.Retry),
		})
	}
	if c.Processing > 0 {
		out = append(out, OpsAlert{
			Code: "ops_processing_inflight", Severity: SeverityInfo,
			Message: fmt.Sprintf("fila processing=%d — submissões in_flight", c.Processing),
		})
	}
	if c.Total == 0 {
		out = append(out, OpsAlert{
			Code: "ops_queue_empty", Severity: SeverityInfo,
			Message: "outbox sem submissões no horizonte de dashboard",
		})
	}
	return out
}

// ClampPage returns 1-based page (default 1).
func ClampPage(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 1
	}
	if n > 10000 {
		return 10000
	}
	return n
}
