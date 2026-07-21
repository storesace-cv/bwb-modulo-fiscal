package persistence_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestSQLiteFoundation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fiscal.db")

	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	v, dirty, err := dbmigrate.Version(dbmigrate.DialectSQLite, path)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if dirty || v != dbmigrate.ExpectedVersion {
		t.Fatalf("version=%d dirty=%v, want %d false", v, dirty, dbmigrate.ExpectedVersion)
	}

	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, BusyTimeout: 5 * time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	assertSQLitePragmas(t, ctx, sqlDB)
	assertSchemaBehavior(t, ctx, sqlDB, false)
}

func TestPostgresFoundation(t *testing.T) {
	dsn := os.Getenv("FISCAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("FISCAL_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	v, dirty, err := dbmigrate.Version(dbmigrate.DialectPostgres, dsn)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if dirty || v != dbmigrate.ExpectedVersion {
		t.Fatalf("version=%d dirty=%v, want %d false", v, dirty, dbmigrate.ExpectedVersion)
	}

	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	var tableSchema string
	err = sqlDB.QueryRowContext(ctx, `
		SELECT table_schema FROM information_schema.tables
		WHERE table_name = $1`, dbmigrate.MigrationsTable).Scan(&tableSchema)
	if err != nil {
		t.Fatalf("lookup migrations table: %v", err)
	}
	if tableSchema != "public" {
		t.Fatalf("migrations table schema = %q, want public", tableSchema)
	}

	var fiscalExists bool
	err = sqlDB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'fiscal')`).Scan(&fiscalExists)
	if err != nil || !fiscalExists {
		t.Fatalf("fiscal schema missing: %v", err)
	}

	assertSchemaBehavior(t, ctx, sqlDB, true)
}

func assertSQLitePragmas(t *testing.T, ctx context.Context, sqlDB *sql.DB) {
	t.Helper()
	var fk, journal string
	var busy int
	if err := sqlDB.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != "1" {
		t.Fatalf("foreign_keys = %q, want 1", fk)
	}
	if err := sqlDB.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journal)
	}
	if err := sqlDB.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatal(err)
	}
	if busy < 1000 {
		t.Fatalf("busy_timeout = %d, want >= 1000", busy)
	}
	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", sqlDB.Stats().MaxOpenConnections)
	}

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("BEGIN IMMEDIATE: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		t.Fatalf("ROLLBACK: %v", err)
	}
}

func assertSchemaBehavior(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	tbl := func(name string) string {
		if postgres {
			return "fiscal." + name
		}
		return name
	}
	exec := func(query string, args ...any) error {
		_, err := sqlDB.ExecContext(ctx, rebind(postgres, query), args...)
		return err
	}
	queryRow := func(query string, args ...any) *sql.Row {
		return sqlDB.QueryRowContext(ctx, rebind(postgres, query), args...)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	hash := make([]byte, canonical.HashSize)
	for i := range hash {
		hash[i] = byte(i + 1)
	}
	docID := "doc-1"

	must(t, exec(`
		INSERT INTO `+tbl("documents")+` (
			id, scope_id, external_id, document_type, currency, issued_at,
			series_code, fiscal_seq, seller_tax_id, seller_name, created_at, sealed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		docID, "scope", "ext-1", "invoice", "AOA", now,
		"A", int64(1), "5000000000", "Seller", now, now,
	))
	must(t, exec(`
		INSERT INTO `+tbl("document_lines")+` (
			document_id, line_no, line_id, description, quantity_scaled, unit_price_cents, tax_code
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		docID, 1, "L1", "Item", int64(10000), int64(1050), "NOR",
	))
	must(t, exec(`
		INSERT INTO `+tbl("ledger_events")+` (
			id, document_id, seq, event_type, from_status, to_status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"ev-1", docID, int64(1), "status_transition", nil, "sealed_locally", now,
	))
	must(t, exec(`
		INSERT INTO `+tbl("outbox_messages")+` (
			id, document_id, message_type, submission_id, state, available_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"ob-1", docID, "authority_submission", "sub-1", "pending", now, now, now,
	))
	must(t, exec(`
		INSERT INTO `+tbl("idempotency_records")+` (
			scope_id, idempotency_key, request_hash, document_id, state, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"scope", "11111111-1111-1111-1111-111111111111", hash, docID, "completed", now, now,
	))

	if err := exec(`
		INSERT INTO `+tbl("documents")+` (
			id, scope_id, external_id, document_type, currency, issued_at,
			series_code, fiscal_seq, seller_tax_id, seller_name, created_at, sealed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"doc-bad-cur", "scope", "ext-bad-cur", "invoice", "USD", now, "A", int64(2), "5000000000", "Seller", now, now,
	); err == nil {
		t.Fatal("expected currency check failure")
	}
	if err := exec(`
		INSERT INTO `+tbl("documents")+` (
			id, scope_id, external_id, document_type, currency, issued_at,
			series_code, fiscal_seq, seller_tax_id, seller_name, created_at, sealed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"doc-bad-seller", "scope", "ext-bad-seller", "invoice", "AOA", now, "A", int64(3), " ", "Seller", now, now,
	); err == nil {
		t.Fatal("expected seller_tax_id nonempty check failure")
	}
	if err := exec(`
		INSERT INTO `+tbl("idempotency_records")+` (
			scope_id, idempotency_key, request_hash, document_id, state, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"scope", "22222222-2222-2222-2222-222222222222", []byte{1, 2, 3}, nil, "in_progress", now, now,
	); err == nil {
		t.Fatal("expected hash length check failure")
	}
	if err := exec(`
		INSERT INTO `+tbl("ledger_events")+` (
			id, document_id, seq, event_type, from_status, to_status, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"ev-bad", docID, int64(0), "status_transition", nil, "sealed_locally", now,
	); err == nil {
		t.Fatal("expected seq positive check failure")
	}
	if err := exec(`
		INSERT INTO `+tbl("outbox_messages")+` (
			id, document_id, message_type, submission_id, state, available_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"ob-bad", docID, "other", "sub-bad", "pending", now, now, now,
	); err == nil {
		t.Fatal("expected message_type check failure")
	}

	must(t, exec(`
		INSERT INTO `+tbl("documents")+` (
			id, scope_id, external_id, document_type, currency, issued_at,
			series_code, fiscal_seq, seller_tax_id, seller_name, created_at, sealed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"doc-cn", "scope", "ext-cn", "credit_note", "AOA", now, "A", int64(9), "5000000000", "Seller", now, now,
	))

	if err := exec(`UPDATE `+tbl("documents")+` SET seller_name = ? WHERE id = ?`, "X", docID); err == nil {
		t.Fatal("expected documents UPDATE blocked")
	}
	if err := exec(`DELETE FROM `+tbl("documents")+` WHERE id = ?`, docID); err == nil {
		t.Fatal("expected documents DELETE blocked")
	}
	if err := exec(`UPDATE `+tbl("document_lines")+` SET description = ? WHERE document_id = ?`, "X", docID); err == nil {
		t.Fatal("expected document_lines UPDATE blocked")
	}
	if err := exec(`DELETE FROM `+tbl("document_lines")+` WHERE document_id = ?`, docID); err == nil {
		t.Fatal("expected document_lines DELETE blocked")
	}
	if err := exec(`UPDATE `+tbl("ledger_events")+` SET event_type = ? WHERE id = ?`, "X", "ev-1"); err == nil {
		t.Fatal("expected ledger_events UPDATE blocked")
	}
	if err := exec(`DELETE FROM `+tbl("ledger_events")+` WHERE id = ?`, "ev-1"); err == nil {
		t.Fatal("expected ledger_events DELETE blocked")
	}

	var status string
	if err := queryRow(`
		SELECT to_status FROM `+tbl("ledger_events")+`
		WHERE document_id = ? ORDER BY seq DESC LIMIT 1`, docID,
	).Scan(&status); err != nil || status != "sealed_locally" {
		t.Fatalf("current status = %q err=%v", status, err)
	}
}

func rebind(postgres bool, query string) string {
	if !postgres {
		return query
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(fmt.Sprintf("%d", n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
