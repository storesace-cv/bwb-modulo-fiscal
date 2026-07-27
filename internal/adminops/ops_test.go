package adminops_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminops"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestDeriveQueueStatus(t *testing.T) {
	cases := []struct {
		state, outcome string
		attempts       int64
		want           string
	}{
		{"pending", "", 0, adminops.QueuePending},
		{"pending", "retried_unavailable", 2, adminops.QueueRetry},
		{"in_flight", "", 1, adminops.QueueProcessing},
		{"succeeded", "authority_accepted", 1, adminops.QueueAccepted},
		{"succeeded", "authority_rejected", 1, adminops.QueueRejected},
		{"succeeded", "authority_outcome_unknown", 1, adminops.QueueManualReview},
		{"dead", "", 3, adminops.QueueManualReview},
	}
	for _, tc := range cases {
		got := adminops.DeriveQueueStatus(tc.state, tc.outcome, tc.attempts)
		if got != tc.want {
			t.Fatalf("%s/%s/%d → %s want %s", tc.state, tc.outcome, tc.attempts, got, tc.want)
		}
	}
	if got := adminops.DeriveQueueStatusWithDisposition("dead", "", 1, adminops.DispositionCancelled); got != adminops.QueueCancelled {
		t.Fatalf("cancelled disposition: %s", got)
	}
}

func TestSanitizeOpsError(t *testing.T) {
	if got := adminops.SanitizeOpsError("authority_rejected", "succeeded", ""); got != "authority_rejected" {
		t.Fatalf("got %q", got)
	}
	if got := adminops.SanitizeOpsError("-----BEGIN PRIVATE KEY-----", "pending", ""); got != "" {
		t.Fatalf("must drop free text: %q", got)
	}
	if got := adminops.SanitizeOpsError("", "dead", ""); got != "outbox_dead" {
		t.Fatalf("dead: %q", got)
	}
	if got := adminops.SanitizeOpsError("", "dead", adminops.DispositionCancelled); got != "ops_cancelled" {
		t.Fatalf("cancelled: %q", got)
	}
}

func TestGuardSimulatorNotProduction(t *testing.T) {
	if err := adminops.GuardSimulatorNotProduction("simulator", "production"); !errors.Is(err, adminops.ErrForbiddenEnv) {
		t.Fatalf("want forbidden, got %v", err)
	}
	if err := adminops.GuardSimulatorNotProduction("simulator", "development"); err != nil {
		t.Fatal(err)
	}
}

func TestApplyQueueActionRetryCancelIdempotency(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ops-actions.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	store := adminops.New(sqlDB, adminops.DialectSQLite)

	now := time.Now().UTC().Truncate(time.Microsecond)
	mustInsertOutbox(t, ctx, sqlDB, "doc-act", "sub-act", "pending", now)

	_, err = store.ApplyQueueAction(ctx, adminops.ActionInput{
		SubmissionID: "sub-act", Action: adminops.ActionRetry, IdempotencyKey: "idem-1",
		ExpectedUpdatedAt: now, AuthorityMode: "simulator", FiscalEnv: "production",
	})
	if !errors.Is(err, adminops.ErrForbiddenEnv) {
		t.Fatalf("prod+sim want forbidden, got %v", err)
	}

	res, err := store.ApplyQueueAction(ctx, adminops.ActionInput{
		SubmissionID: "sub-act", Action: adminops.ActionManualReview, IdempotencyKey: "idem-mr",
		ExpectedUpdatedAt: now, AuthorityMode: "simulator", FiscalEnv: "development", Now: now.Add(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.QueueStatus != adminops.QueueManualReview || res.OutboxState != "dead" {
		t.Fatalf("%+v", res)
	}

	replay, err := store.ApplyQueueAction(ctx, adminops.ActionInput{
		SubmissionID: "sub-act", Action: adminops.ActionManualReview, IdempotencyKey: "idem-mr",
		ExpectedUpdatedAt: now, AuthorityMode: "simulator", FiscalEnv: "development",
	})
	if err != nil || !replay.IdempotentReplay {
		t.Fatalf("replay: %+v err=%v", replay, err)
	}

	_, err = store.ApplyQueueAction(ctx, adminops.ActionInput{
		SubmissionID: "sub-act", Action: adminops.ActionRetry, IdempotencyKey: "idem-retry",
		ExpectedUpdatedAt: res.UpdatedAt, AuthorityMode: "simulator", FiscalEnv: "development",
		Now: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := store.ListSubmissionSummariesFiltered(ctx, adminops.SubmissionFilter{Limit: 10, QueueStatus: adminops.QueuePending})
	if err != nil || len(rows) != 1 {
		t.Fatalf("list pending: %v %#v", err, rows)
	}

	cur := rows[0]
	cancelled, err := store.ApplyQueueAction(ctx, adminops.ActionInput{
		SubmissionID: "sub-act", Action: adminops.ActionCancel, IdempotencyKey: "idem-cancel",
		ExpectedUpdatedAt: cur.OutboxUpdatedAt, AuthorityMode: "simulator", FiscalEnv: "homologation",
		Now: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.QueueStatus != adminops.QueueCancelled {
		t.Fatalf("%+v", cancelled)
	}
	_, err = store.ApplyQueueAction(ctx, adminops.ActionInput{
		SubmissionID: "sub-act", Action: adminops.ActionRetry, IdempotencyKey: "idem-reopen",
		ExpectedUpdatedAt: cancelled.UpdatedAt, AuthorityMode: "simulator", FiscalEnv: "development",
	})
	if !errors.Is(err, adminops.ErrValidation) {
		t.Fatalf("cancelled terminal want validation, got %v", err)
	}
}

func mustInsertOutbox(t *testing.T, ctx context.Context, sqlDB *sql.DB, docID, subID, state string, now time.Time) {
	t.Helper()
	ts := now.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
	_, err := sqlDB.ExecContext(ctx, `
		INSERT INTO documents (
			id, scope_id, external_id, document_type, currency, issued_at,
			issued_timezone, issued_offset_minutes,
			series_code, fiscal_seq, seller_tax_id, seller_name, created_at, sealed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		docID, "scope-ops", "ext-"+docID, "invoice", "AOA", ts,
		"Africa/Luanda", 60, "A", int64(1), "5000000000", "Seller", ts, ts,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqlDB.ExecContext(ctx, `
		INSERT INTO outbox_messages (
			id, document_id, message_type, submission_id, state, available_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"ob-"+subID, docID, "authority_submission", subID, state, ts, ts, ts,
	)
	if err != nil {
		t.Fatal(err)
	}
}
