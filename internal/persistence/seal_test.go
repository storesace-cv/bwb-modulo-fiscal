package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

func TestSealSQLiteSuite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "seal.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	store := persistence.NewStore(sqlDB, persistence.DialectSQLite)
	runSealSuite(t, ctx, store, sqlDB, false)
}

func TestSealPostgresSuite(t *testing.T) {
	dsn := os.Getenv("FISCAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("FISCAL_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	// Isolate suite data with unique scope prefix per run.
	store := persistence.NewStore(sqlDB, persistence.DialectPostgres)
	runSealSuite(t, ctx, store, sqlDB, true)
}

func runSealSuite(t *testing.T, ctx context.Context, store *persistence.Store, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	scope := fmt.Sprintf("scope-%s-%d", t.Name(), time.Now().UnixNano())

	t.Run("VS-T03_first_and_identical_replay", func(t *testing.T) {
		key := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
		req := sampleSealReq(scope, key, "ext-1", "A", "10.50")
		r1, err := store.SealInTx(ctx, req)
		if err != nil {
			t.Fatalf("first seal: %v", err)
		}
		if r1.IdempotentHit || r1.FiscalSeq != 1 {
			t.Fatalf("first result: %+v", r1)
		}
		r2, err := store.SealInTx(ctx, req)
		if err != nil {
			t.Fatalf("replay: %v", err)
		}
		if !r2.IdempotentHit || r2.DocumentID != r1.DocumentID || r2.FiscalSeq != r1.FiscalSeq {
			t.Fatalf("replay result: %+v want id=%s seq=%d", r2, r1.DocumentID, r1.FiscalSeq)
		}
		if r2.CreatedAt.IsZero() || !r1.CreatedAt.Equal(r2.CreatedAt) {
			t.Fatalf("CreatedAt replay: first=%v second=%v", r1.CreatedAt, r2.CreatedAt)
		}
		if r2.SubmissionID != r1.SubmissionID {
			t.Fatalf("SubmissionID replay mismatch")
		}
		assertExactCount(t, ctx, sqlDB, postgres, "documents", scope, 1)
		assertExactCount(t, ctx, sqlDB, postgres, "outbox_messages", scope, 1)
		assertExactCount(t, ctx, sqlDB, postgres, "ledger_events", scope, 1)
	})

	t.Run("VS-T04_hash_conflict", func(t *testing.T) {
		key := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2"
		req := sampleSealReq(scope, key, "ext-2", "A", "10.50")
		if _, err := store.SealInTx(ctx, req); err != nil {
			t.Fatalf("seal: %v", err)
		}
		bad := sampleSealReq(scope, key, "ext-2", "A", "11.00")
		_, err := store.SealInTx(ctx, bad)
		if !errors.Is(err, persistence.ErrIdempotencyConflict) {
			t.Fatalf("err = %v, want ErrIdempotencyConflict", err)
		}
		assertDocCountByExternal(t, ctx, sqlDB, postgres, scope, "ext-2", 1)
	})

	t.Run("series_code_changes_idempotency_hash", func(t *testing.T) {
		key := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaa10"
		req := sampleSealReq(scope, key, "ext-series-hash", "SA", "10.50")
		if _, err := store.SealInTx(ctx, req); err != nil {
			t.Fatalf("seal: %v", err)
		}
		other := sampleSealReq(scope, key, "ext-series-hash", "SB", "10.50")
		_, err := store.SealInTx(ctx, other)
		if !errors.Is(err, persistence.ErrIdempotencyConflict) {
			t.Fatalf("err = %v, want ErrIdempotencyConflict", err)
		}
		assertDocCountByExternal(t, ctx, sqlDB, postgres, scope, "ext-series-hash", 1)
	})

	t.Run("line_order_changes_idempotency_hash", func(t *testing.T) {
		key := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaa16"
		qty, _ := quantity.ParseCanonical("1")
		price, _ := money.ParseCanonical("10.50")
		req := sampleSealReq(scope, key, "ext-line-order", "LO", "10.50")
		req.Intent.Lines = []canonical.Line{
			{LineID: "L1", Description: "A", Quantity: qty, UnitPrice: price, TaxCode: "NOR"},
			{LineID: "L2", Description: "B", Quantity: qty, UnitPrice: price, TaxCode: "NOR"},
		}
		if _, err := store.SealInTx(ctx, req); err != nil {
			t.Fatalf("seal: %v", err)
		}
		reordered := sampleSealReq(scope, key, "ext-line-order", "LO", "10.50")
		reordered.Intent.Lines = []canonical.Line{
			{LineID: "L2", Description: "B", Quantity: qty, UnitPrice: price, TaxCode: "NOR"},
			{LineID: "L1", Description: "A", Quantity: qty, UnitPrice: price, TaxCode: "NOR"},
		}
		_, err := store.SealInTx(ctx, reordered)
		if !errors.Is(err, persistence.ErrIdempotencyConflict) {
			t.Fatalf("err = %v, want ErrIdempotencyConflict", err)
		}
		assertDocCountByExternal(t, ctx, sqlDB, postgres, scope, "ext-line-order", 1)
	})

	t.Run("issued_at_invalid_rejected", func(t *testing.T) {
		req := sampleSealReq(scope, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaa11", "ext-bad-date", "A", "1.00")
		req.Intent.IssuedAtUTC = "not-a-date"
		_, err := store.SealInTx(ctx, req)
		if err == nil {
			t.Fatal("expected invalid issued_at error")
		}
	})

	t.Run("issued_at_normalized_consistently", func(t *testing.T) {
		req := sampleSealReq(scope, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaa12", "ext-norm-date", "A", "1.00")
		req.Intent.IssuedAtUTC = "2026-07-21T11:00:00+01:00"
		r, err := store.SealInTx(ctx, req)
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		want := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
		q := `SELECT issued_at FROM ` + tbl(postgres, "documents") + ` WHERE id = ?`
		var got time.Time
		if postgres {
			if err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), r.DocumentID).Scan(&got); err != nil {
				t.Fatal(err)
			}
			got = got.UTC()
		} else {
			var raw string
			if err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), r.DocumentID).Scan(&raw); err != nil {
				t.Fatal(err)
			}
			got, err = time.Parse(time.RFC3339Nano, raw)
			if err != nil {
				t.Fatalf("parse sqlite issued_at %q: %v", raw, err)
			}
			got = got.UTC()
		}
		if !got.Equal(want) {
			t.Fatalf("stored issued_at = %v, want %v", got, want)
		}
	})

	t.Run("VS-T05_external_id_conflict", func(t *testing.T) {
		req1 := sampleSealReq(scope, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3", "ext-shared", "B", "1.00")
		if _, err := store.SealInTx(ctx, req1); err != nil {
			t.Fatalf("first: %v", err)
		}
		req2 := sampleSealReq(scope, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa4", "ext-shared", "B", "1.00")
		_, err := store.SealInTx(ctx, req2)
		if !errors.Is(err, persistence.ErrExternalIDConflict) {
			t.Fatalf("err = %v, want ErrExternalIDConflict", err)
		}
	})

	t.Run("VS-T05_concurrent_external_id", func(t *testing.T) {
		const n = 6
		errs := make([]error, n)
		results := make([]*persistence.SealResult, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			i := i
			go func() {
				defer wg.Done()
				req := sampleSealReq(scope, fmt.Sprintf("aaaaaaaa-aaaa-aaaa-aaaa-a%010d", 100+i), "ext-race", "R", "1.00")
				results[i], errs[i] = store.SealInTx(ctx, req)
			}()
		}
		wg.Wait()
		ok := 0
		conflicts := 0
		for i := 0; i < n; i++ {
			switch {
			case errs[i] == nil:
				ok++
			case errors.Is(errs[i], persistence.ErrExternalIDConflict):
				conflicts++
			default:
				t.Fatalf("goroutine %d: unexpected err %v", i, errs[i])
			}
		}
		if ok != 1 || conflicts != n-1 {
			t.Fatalf("ok=%d conflicts=%d, want 1 and %d", ok, conflicts, n-1)
		}
		assertDocCountByExternal(t, ctx, sqlDB, postgres, scope, "ext-race", 1)
	})

	t.Run("VS-T06_concurrency_same_series", func(t *testing.T) {
		const n = 8
		results := make([]*persistence.SealResult, n)
		errs := make([]error, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			i := i
			go func() {
				defer wg.Done()
				req := sampleSealReq(scope, fmt.Sprintf("cccccccc-cccc-cccc-cccc-c%010d", i), fmt.Sprintf("ext-c-%d", i), "C", "2.00")
				results[i], errs[i] = store.SealInTx(ctx, req)
			}()
		}
		wg.Wait()
		seqs := map[int64]struct{}{}
		for i := 0; i < n; i++ {
			if errs[i] != nil {
				t.Fatalf("goroutine %d: %v", i, errs[i])
			}
			if _, dup := seqs[results[i].FiscalSeq]; dup {
				t.Fatalf("duplicate fiscal_seq %d", results[i].FiscalSeq)
			}
			seqs[results[i].FiscalSeq] = struct{}{}
		}
		if len(seqs) != n {
			t.Fatalf("unique seqs = %d, want %d", len(seqs), n)
		}
	})

	t.Run("VS-T02_concurrency_same_idempotency_key", func(t *testing.T) {
		key := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1"
		const n = 6
		results := make([]*persistence.SealResult, n)
		errs := make([]error, n)
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 0; i < n; i++ {
			i := i
			go func() {
				defer wg.Done()
				req := sampleSealReq(scope, key, "ext-idem-conc", "D", "3.00")
				results[i], errs[i] = store.SealInTx(ctx, req)
			}()
		}
		wg.Wait()
		var winner *persistence.SealResult
		hits := 0
		for i := 0; i < n; i++ {
			if errs[i] != nil {
				t.Fatalf("goroutine %d: %v", i, errs[i])
			}
			if !results[i].IdempotentHit {
				if winner != nil && winner.DocumentID != results[i].DocumentID {
					t.Fatalf("multiple winners")
				}
				winner = results[i]
			} else {
				hits++
			}
		}
		if winner == nil {
			t.Fatal("no winner")
		}
		for i := 0; i < n; i++ {
			if results[i].DocumentID != winner.DocumentID || results[i].FiscalSeq != winner.FiscalSeq {
				t.Fatalf("mismatch %+v vs %+v", results[i], winner)
			}
		}
		assertDocCountByExternal(t, ctx, sqlDB, postgres, scope, "ext-idem-conc", 1)
		_ = hits
	})

	t.Run("VS-T07_validation_rejects_before_side_effects", func(t *testing.T) {
		series := "E"
		before := readSeriesLast(t, ctx, sqlDB, postgres, scope, series)
		req := sampleSealReq(scope, "dddddddd-dddd-dddd-dddd-ddddddddddd1", "ext-fail", series, "4.00")
		req.Intent.SellerName = "   "
		_, err := store.SealInTx(ctx, req)
		if !errors.Is(err, persistence.ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
		after := readSeriesLast(t, ctx, sqlDB, postgres, scope, series)
		if after != before {
			t.Fatalf("series counter changed on validation failure: before=%d after=%d", before, after)
		}
		assertDocCountByExternal(t, ctx, sqlDB, postgres, scope, "ext-fail", 0)
		assertIdempotencyAbsent(t, ctx, sqlDB, postgres, scope, req.IdempotencyKey)
	})

	t.Run("VS-T07_rollback_after_counter_on_trigger", func(t *testing.T) {
		const failExternal = "ext-fail-vst07"
		installTestFailTrigger(t, ctx, sqlDB, postgres, failExternal)
		t.Cleanup(func() { dropTestFailTrigger(t, ctx, sqlDB, postgres) })

		series := "E-TRIG"
		before := readSeriesLast(t, ctx, sqlDB, postgres, scope, series)
		req := sampleSealReq(scope, "dddddddd-dddd-dddd-dddd-ddddddddddd2", failExternal, series, "4.00")
		_, err := store.SealInTx(ctx, req)
		if err == nil {
			t.Fatal("expected trigger-induced failure")
		}
		after := readSeriesLast(t, ctx, sqlDB, postgres, scope, series)
		if after != before {
			t.Fatalf("series counter changed on rollback: before=%d after=%d", before, after)
		}
		assertDocCountByExternal(t, ctx, sqlDB, postgres, scope, failExternal, 0)
		assertIdempotencyAbsent(t, ctx, sqlDB, postgres, scope, req.IdempotencyKey)
		assertNoOrphanFiscalRows(t, ctx, sqlDB, postgres, scope, failExternal)
	})

	t.Run("independent_series_per_scope", func(t *testing.T) {
		scope2 := scope + "-other"
		r1, err := store.SealInTx(ctx, sampleSealReq(scope, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee1", "ext-ind-1", "F", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		r2, err := store.SealInTx(ctx, sampleSealReq(scope2, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeee2", "ext-ind-1", "F", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		if r1.FiscalSeq != 1 || r2.FiscalSeq != 1 {
			t.Fatalf("expected independent seq 1, got %d and %d", r1.FiscalSeq, r2.FiscalSeq)
		}
	})

	t.Run("immutability_after_seal", func(t *testing.T) {
		r, err := store.SealInTx(ctx, sampleSealReq(scope, "ffffffff-ffff-ffff-ffff-fffffffffff1", "ext-imm", "G", "1.00"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = sqlDB.ExecContext(ctx, rebind(postgres, `UPDATE `+tbl(postgres, "documents")+` SET seller_name = ? WHERE id = ?`), "X", r.DocumentID)
		if err == nil {
			t.Fatal("expected update blocked")
		}
	})
}

func sampleSealReq(scope, key, externalID, series, price string) persistence.SealRequest {
	qty, _ := quantity.ParseCanonical("1")
	amt, _ := money.ParseCanonical(price)
	return persistence.SealRequest{
		IdempotencyKey: key,
		SeriesCode:     series,
		Intent: canonical.DocumentIntent{
			ScopeID:      scope,
			ExternalID:   externalID,
			DocumentType: "invoice",
			Currency:     "AOA",
			IssuedAtUTC:  time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
			SellerTaxID:  "5000000000",
			SellerName:   "Seller SA",
			Lines: []canonical.Line{{
				LineID:      "L1",
				Description: "Item",
				Quantity:    qty,
				UnitPrice:   amt,
				TaxCode:     "NOR",
			}},
		},
	}
}

func tbl(postgres bool, name string) string {
	if postgres {
		return "fiscal." + name
	}
	return name
}

func assertExactCount(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, table, scope string, want int) {
	t.Helper()
	var n int
	var q string
	switch table {
	case "documents":
		q = `SELECT COUNT(*) FROM ` + tbl(postgres, "documents") + ` WHERE scope_id = ?`
	case "outbox_messages":
		q = `SELECT COUNT(*) FROM ` + tbl(postgres, "outbox_messages") + ` o
			JOIN ` + tbl(postgres, "documents") + ` d ON d.id = o.document_id WHERE d.scope_id = ?`
	case "ledger_events":
		q = `SELECT COUNT(*) FROM ` + tbl(postgres, "ledger_events") + ` e
			JOIN ` + tbl(postgres, "documents") + ` d ON d.id = e.document_id WHERE d.scope_id = ?`
	default:
		t.Fatalf("unknown table %s", table)
	}
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), scope).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("%s count = %d, want %d", table, n, want)
	}
}

func assertDocCountByExternal(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scope, external string, want int) {
	t.Helper()
	var n int
	q := `SELECT COUNT(*) FROM ` + tbl(postgres, "documents") + ` WHERE scope_id = ? AND external_id = ?`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), scope, external).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != want {
		t.Fatalf("doc count = %d, want %d", n, want)
	}
}

func assertIdempotencyAbsent(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scope, key string) {
	t.Helper()
	var n int
	q := `SELECT COUNT(*) FROM ` + tbl(postgres, "idempotency_records") + ` WHERE scope_id = ? AND idempotency_key = ?`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), scope, key).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("idempotency rows = %d, want 0 after rollback", n)
	}
}

func readSeriesLast(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scope, series string) int64 {
	t.Helper()
	var last sql.NullInt64
	q := `SELECT last_seq FROM ` + tbl(postgres, "series_counters") + ` WHERE scope_id = ? AND series_code = ?`
	err := sqlDB.QueryRowContext(ctx, rebind(postgres, q), scope, series).Scan(&last)
	if errors.Is(err, sql.ErrNoRows) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return last.Int64
}

func installTestFailTrigger(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, externalID string) {
	t.Helper()
	lit := strings.ReplaceAll(externalID, "'", "''")
	if postgres {
		fn := `
			CREATE OR REPLACE FUNCTION fiscal.test_fail_seal_vst07() RETURNS trigger AS $fn$
			BEGIN
				IF NEW.external_id = '` + lit + `' THEN
					RAISE EXCEPTION 'test-induced failure after series counter';
				END IF;
				RETURN NEW;
			END;
			$fn$ LANGUAGE plpgsql`
		if _, err := sqlDB.ExecContext(ctx, fn); err != nil {
			t.Fatalf("create function: %v", err)
		}
		_, err := sqlDB.ExecContext(ctx, `
			DROP TRIGGER IF EXISTS documents_test_fail_vst07 ON fiscal.documents;
			CREATE TRIGGER documents_test_fail_vst07
				BEFORE INSERT ON fiscal.documents
				FOR EACH ROW EXECUTE FUNCTION fiscal.test_fail_seal_vst07()`)
		if err != nil {
			t.Fatalf("create trigger: %v", err)
		}
		return
	}
	_, err := sqlDB.ExecContext(ctx, `
		CREATE TRIGGER documents_test_fail_vst07
		BEFORE INSERT ON documents
		FOR EACH ROW
		WHEN NEW.external_id = '`+lit+`'
		BEGIN
			SELECT RAISE(ABORT, 'test-induced failure after series counter');
		END`)
	if err != nil {
		t.Fatalf("create sqlite trigger: %v", err)
	}
}

func dropTestFailTrigger(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	if postgres {
		_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS documents_test_fail_vst07 ON fiscal.documents`)
		_, _ = sqlDB.ExecContext(ctx, `DROP FUNCTION IF EXISTS fiscal.test_fail_seal_vst07()`)
		return
	}
	_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS documents_test_fail_vst07`)
}

func assertNoOrphanFiscalRows(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scope, externalID string) {
	t.Helper()
	var ledgerN, outboxN int
	lq := `SELECT COUNT(*) FROM ` + tbl(postgres, "ledger_events") + ` e
		JOIN ` + tbl(postgres, "documents") + ` d ON d.id = e.document_id
		WHERE d.scope_id = ? AND d.external_id = ?`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, lq), scope, externalID).Scan(&ledgerN); err != nil {
		t.Fatal(err)
	}
	oq := `SELECT COUNT(*) FROM ` + tbl(postgres, "outbox_messages") + ` o
		JOIN ` + tbl(postgres, "documents") + ` d ON d.id = o.document_id
		WHERE d.scope_id = ? AND d.external_id = ?`
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, oq), scope, externalID).Scan(&outboxN); err != nil {
		t.Fatal(err)
	}
	if ledgerN != 0 || outboxN != 0 {
		t.Fatalf("orphan ledger=%d outbox=%d", ledgerN, outboxN)
	}
}
