package dbmigrate_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/migrations"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
)

func TestMigration0002EmptyDBReachesVersion2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	v, dirty, err := dbmigrate.Version(dbmigrate.DialectSQLite, path)
	if err != nil {
		t.Fatal(err)
	}
	if dirty || v != dbmigrate.ExpectedVersion {
		t.Fatalf("version=%d dirty=%v want %d", v, dirty, dbmigrate.ExpectedVersion)
	}
}

func TestMigration0002AbortsOnLegacyDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-docs.db")
	if err := migrateSQLiteTo(t, path, 1); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(context.Background(), db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = sqlDB.Exec(`
		INSERT INTO documents (
			id, scope_id, external_id, document_type, currency, issued_at,
			series_code, fiscal_seq, seller_tax_id, seller_name, created_at, sealed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"doc-legacy", "scope", "ext-1", "invoice", "AOA", now,
		"A", int64(1), "5000000000", "Seller", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	err = dbmigrate.Up(dbmigrate.DialectSQLite, path)
	if err == nil {
		t.Fatal("expected abort with legacy documents")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "check") && !strings.Contains(msg, "constraint") && !strings.Contains(msg, "abort") {
		t.Fatalf("unexpected error: %v", err)
	}
	v, dirty, verr := dbmigrate.Version(dbmigrate.DialectSQLite, path)
	if verr != nil {
		t.Fatal(verr)
	}
	if !dirty {
		t.Fatalf("expected dirty after aborted 0002, version=%d", v)
	}
	if v != 1 && v != 2 {
		t.Fatalf("version after failed 0002 = %d dirty=%v", v, dirty)
	}
}

func TestMigration0002AbortsOnLegacyIdempotencyWithoutDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-idem.db")
	if err := migrateSQLiteTo(t, path, 1); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(context.Background(), db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	hash := make([]byte, 32)
	_, err = sqlDB.Exec(`
		INSERT INTO idempotency_records (
			scope_id, idempotency_key, request_hash, document_id, state, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"scope", "11111111-1111-1111-1111-111111111111", hash, nil, "in_progress", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	var nDocs int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM documents`).Scan(&nDocs); err != nil || nDocs != 0 {
		t.Fatalf("docs=%d err=%v", nDocs, err)
	}
	_ = sqlDB.Close()

	err = dbmigrate.Up(dbmigrate.DialectSQLite, path)
	if err == nil {
		t.Fatal("expected abort with legacy idempotency_records only")
	}
}

func migrateSQLiteTo(t *testing.T, path string, version uint) error {
	t.Helper()
	src, err := iofs.New(migrations.FS, "sqlite")
	if err != nil {
		return err
	}
	dbURL := "sqlite://" + path + "?x-migrations-table=" + dbmigrate.MigrationsTable
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
