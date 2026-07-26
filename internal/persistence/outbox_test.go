package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/simulator"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaljws"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbtest"
)

func TestOutboxSimulatorSQLite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "outbox.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runOutboxSimulatorSuite(t, ctx, persistence.NewStore(sqlDB, persistence.DialectSQLite), sqlDB, false)
}

func TestOutboxSimulatorPostgres(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()
	ctx := context.Background()
	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runOutboxSimulatorSuite(t, ctx, persistence.NewStore(sqlDB, persistence.DialectPostgres), sqlDB, true)
}

func runOutboxSimulatorSuite(t *testing.T, ctx context.Context, store *persistence.Store, sqlDB *sql.DB, postgres bool) {
	t.Helper()

	t.Run("VS-T10_accept_and_reject", func(t *testing.T) {
		scope := fmt.Sprintf("outbox-t10-%d", time.Now().UnixNano())
		sim := simulator.New(simulator.OutcomeAccept)
		r, err := store.SealInTx(ctx, sampleSealReq(scope, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa1", "ext-acc", "S", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		sim.Script(r.SubmissionID, simulator.OutcomeAccept)
		out, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcome != "authority_accepted" || out.OutboxState != "succeeded" {
			t.Fatalf("%+v", out)
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, r.DocumentID, "authority_accepted")
		assertAttemptResponseCount(t, ctx, sqlDB, postgres, r.SubmissionID, 1, 1)

		r2, err := store.SealInTx(ctx, sampleSealReq(scope, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa2", "ext-rej", "S", "2.00"))
		if err != nil {
			t.Fatal(err)
		}
		sim.Script(r2.SubmissionID, simulator.OutcomeReject)
		out, err = store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcome != "authority_rejected" {
			t.Fatalf("%+v", out)
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, r2.DocumentID, "authority_rejected")
		// number not reused: next seal on same series continues seq
		r3, err := store.SealInTx(ctx, sampleSealReq(scope, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaa3", "ext-next", "S", "3.00"))
		if err != nil {
			t.Fatal(err)
		}
		if r3.FiscalSeq != 3 {
			t.Fatalf("seq=%d want 3 (reject must not free number)", r3.FiscalSeq)
		}
		// Drain leftover outbox so peer subtests are isolated.
		sim.Script(r3.SubmissionID, simulator.OutcomeAccept)
		if _, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("VS-T08_unavailable_retries_no_accept", func(t *testing.T) {
		scope := fmt.Sprintf("outbox-t08-%d", time.Now().UnixNano())
		sim := simulator.New(simulator.OutcomeUnavailable)
		r, err := store.SealInTx(ctx, sampleSealReq(scope, "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbb1", "ext-unavail", "U", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		out, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcome != "retried_unavailable" || out.OutboxState != "pending" {
			t.Fatalf("%+v", out)
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, r.DocumentID, "sealed_locally")
		assertAttemptResponseCount(t, ctx, sqlDB, postgres, r.SubmissionID, 0, 0)
		assertOutboxState(t, ctx, sqlDB, postgres, r.SubmissionID, "pending")

		// Force available_at to past for immediate retry.
		forceOutboxAvailable(t, ctx, sqlDB, postgres, r.SubmissionID)
		sim.Script(r.SubmissionID, simulator.OutcomeAccept)
		out, err = store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcome != "authority_accepted" {
			t.Fatalf("%+v", out)
		}
		if sim.CallCount(r.SubmissionID) < 2 {
			t.Fatalf("calls=%d want >=2", sim.CallCount(r.SubmissionID))
		}
	})

	t.Run("VS-T11_duplicate_process_idempotent", func(t *testing.T) {
		scope := fmt.Sprintf("outbox-t11-%d", time.Now().UnixNano())
		sim := simulator.New(simulator.OutcomeAccept)
		r, err := store.SealInTx(ctx, sampleSealReq(scope, "cccccccc-cccc-4ccc-8ccc-ccccccccccc1", "ext-dup", "D", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		sim.Script(r.SubmissionID, simulator.OutcomeAccept)
		if _, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{}); err != nil {
			t.Fatal(err)
		}
		attemptsBefore, _ := countAttemptsResponses(t, ctx, sqlDB, postgres, r.SubmissionID)
		// Force pending again illegally then process — terminal ledger should short-circuit.
		forceOutboxState(t, ctx, sqlDB, postgres, r.SubmissionID, "pending")
		out, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if out.OutboxState != "succeeded" || out.Outcome != "authority_accepted" {
			t.Fatalf("%+v", out)
		}
		attemptsAfter, responsesAfter := countAttemptsResponses(t, ctx, sqlDB, postgres, r.SubmissionID)
		if attemptsAfter != attemptsBefore || responsesAfter != attemptsBefore {
			t.Fatalf("dedup failed attempts %d→%d responses=%d", attemptsBefore, attemptsAfter, responsesAfter)
		}
	})

	t.Run("VS-T12_reclaim_inflight_and_unknown", func(t *testing.T) {
		scope := fmt.Sprintf("outbox-t12-%d", time.Now().UnixNano())
		sim := simulator.New(simulator.OutcomeAccept)

		rUnknown, err := store.SealInTx(ctx, sampleSealReq(scope, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeee1", "ext-unk", "K", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		sim.Script(rUnknown.SubmissionID, simulator.OutcomeUnknown)
		out, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcome != "authority_outcome_unknown" || out.OutboxState != "succeeded" {
			t.Fatalf("unknown: %+v", out)
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, rUnknown.DocumentID, "authority_outcome_unknown")

		rCrash, err := store.SealInTx(ctx, sampleSealReq(scope, "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeee2", "ext-crash", "K", "2.00"))
		if err != nil {
			t.Fatal(err)
		}
		forceStaleInFlight(t, ctx, sqlDB, postgres, rCrash.SubmissionID)
		sim.Script(rCrash.SubmissionID, simulator.OutcomeAccept)
		out, err = store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if out.Outcome != "authority_accepted" || out.OutboxState != "succeeded" {
			t.Fatalf("reclaim: %+v", out)
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, rCrash.DocumentID, "authority_accepted")
		assertAttemptResponseCount(t, ctx, sqlDB, postgres, rCrash.SubmissionID, 1, 1)
	})

	t.Run("VS-T09_slow_simulator_authority_processing", func(t *testing.T) {
		scope := fmt.Sprintf("outbox-t09-%d", time.Now().UnixNano())
		inner := simulator.New(simulator.OutcomeAccept)
		entered := make(chan struct{})
		release := make(chan struct{})
		slow := &blockingAuthority{
			inner:   inner,
			entered: entered,
			release: release,
		}
		r, err := store.SealInTx(ctx, sampleSealReq(scope, "ffffffff-ffff-4fff-8fff-fffffffffff1", "ext-slow", "L", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		inner.Script(r.SubmissionID, simulator.OutcomeAccept)

		errCh := make(chan error, 1)
		go func() {
			_, err := store.ProcessNextAuthoritySubmission(ctx, slow, persistence.ProcessOpts{})
			errCh <- err
		}()

		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for slow Submit")
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, r.DocumentID, "authority_processing")
		assertOutboxState(t, ctx, sqlDB, postgres, r.SubmissionID, "in_flight")

		close(release)
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for process finish")
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, r.DocumentID, "authority_accepted")
	})

	t.Run("JWS_ephemeral_verified_by_simulator", func(t *testing.T) {
		scope := fmt.Sprintf("outbox-jws-%d", time.Now().UnixNano())
		signer, err := fiscaljws.NewEphemeral(fiscaljws.DefaultRSABits)
		if err != nil {
			t.Fatal(err)
		}
		sim := simulator.New(simulator.OutcomeAccept)
		sim.VerifyPublic = signer.PublicKey()
		r, err := store.SealInTx(ctx, sampleSealReq(scope, "dddddddd-dddd-4ddd-8ddd-ddddddddddd1", "ext-jws", "J", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		out, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{Signer: signer})
		if err != nil {
			t.Fatal(err)
		}
		if !out.JWSAttached || out.Outcome != "authority_accepted" {
			t.Fatalf("%+v", out)
		}
		assertLatestLedger(t, ctx, sqlDB, postgres, r.DocumentID, "authority_accepted")
	})

	t.Run("empty_outbox", func(t *testing.T) {
		sim := simulator.New(simulator.OutcomeAccept)
		_, err := store.ProcessNextAuthoritySubmission(ctx, sim, persistence.ProcessOpts{})
		if !errors.Is(err, persistence.ErrOutboxEmpty) {
			t.Fatalf("err=%v", err)
		}
	})
}

func assertLatestLedger(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, docID, want string) {
	t.Helper()
	var got string
	q := `SELECT to_status FROM ` + tbl(postgres, "ledger_events") + ` WHERE document_id = ? ORDER BY seq DESC LIMIT 1`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), docID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ledger=%q want %q", got, want)
	}
}

func assertOutboxState(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, submissionID, want string) {
	t.Helper()
	var got string
	q := `SELECT state FROM ` + tbl(postgres, "outbox_messages") + ` WHERE submission_id = ?`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), submissionID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("outbox state=%q want %q", got, want)
	}
}

func assertAttemptResponseCount(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, submissionID string, wantAttempts, wantResponses int) {
	t.Helper()
	a, r := countAttemptsResponses(t, ctx, sqlDB, postgres, submissionID)
	if a != wantAttempts || r != wantResponses {
		t.Fatalf("attempts=%d responses=%d want %d/%d", a, r, wantAttempts, wantResponses)
	}
}

func countAttemptsResponses(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, submissionID string) (attempts, responses int) {
	t.Helper()
	aq := `SELECT COUNT(*) FROM ` + tbl(postgres, "authority_attempts") + ` WHERE submission_id = ?`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, aq), submissionID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	rq := `SELECT COUNT(*) FROM ` + tbl(postgres, "authority_responses") + ` r
		JOIN ` + tbl(postgres, "authority_attempts") + ` a ON a.id = r.attempt_id
		WHERE a.submission_id = ?`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, rq), submissionID).Scan(&responses); err != nil {
		t.Fatal(err)
	}
	return attempts, responses
}

func forceOutboxAvailable(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, submissionID string) {
	t.Helper()
	past := time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05.000000Z07:00")
	q := `UPDATE ` + tbl(postgres, "outbox_messages") + ` SET available_at = ?, state = 'pending' WHERE submission_id = ?`
	if _, err := sqlDB.ExecContext(ctx, rebind(postgres, q), past, submissionID); err != nil {
		t.Fatal(err)
	}
}

func forceOutboxState(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, submissionID, state string) {
	t.Helper()
	past := time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05.000000Z07:00")
	q := `UPDATE ` + tbl(postgres, "outbox_messages") + ` SET state = ?, available_at = ? WHERE submission_id = ?`
	if _, err := sqlDB.ExecContext(ctx, rebind(postgres, q), state, past, submissionID); err != nil {
		t.Fatal(err)
	}
}

func forceStaleInFlight(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, submissionID string) {
	t.Helper()
	past := time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05.000000Z07:00")
	q := `UPDATE ` + tbl(postgres, "outbox_messages") + ` SET state = 'in_flight', updated_at = ?, available_at = ? WHERE submission_id = ?`
	if _, err := sqlDB.ExecContext(ctx, rebind(postgres, q), past, past, submissionID); err != nil {
		t.Fatal(err)
	}
}

// blockingAuthority holds Submit until release is closed (VS-T09).
type blockingAuthority struct {
	inner   *simulator.Client
	entered chan struct{}
	release chan struct{}
	once    bool
}

func (b *blockingAuthority) Submit(ctx context.Context, req simulator.Request) (simulator.Result, error) {
	if !b.once {
		b.once = true
		close(b.entered)
		select {
		case <-b.release:
		case <-ctx.Done():
			return simulator.Result{}, ctx.Err()
		}
	}
	return b.inner.Submit(ctx, req)
}
