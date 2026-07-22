package dbmigrate_test

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbtest"
	"github.com/storesace-cv/bwb-modulo-fiscal/migrations"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
)

func TestMigration0002PostgresEmptyReachesVersion2(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()

	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatal(err)
	}
	v, dirty, err := dbmigrate.Version(dbmigrate.DialectPostgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if dirty || v != dbmigrate.ExpectedVersion {
		t.Fatalf("version=%d dirty=%v want %d false", v, dirty, dbmigrate.ExpectedVersion)
	}
}

func TestMigration0002PostgresAbortsOnLegacyDocuments(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()

	if err := migratePostgresTo(t, dsn, 1); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenPostgres(context.Background(), db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	_, err = sqlDB.Exec(`
		INSERT INTO fiscal.documents (
			id, scope_id, external_id, document_type, currency, issued_at,
			series_code, fiscal_seq, seller_tax_id, seller_name, created_at, sealed_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		"doc-legacy", "scope", "ext-1", "invoice", "AOA", now,
		"A", int64(1), "5000000000", "Seller", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	err = dbmigrate.Up(dbmigrate.DialectPostgres, dsn)
	if err == nil {
		t.Fatal("expected abort with legacy documents")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "aborted") && !strings.Contains(msg, "legacy") {
		t.Fatalf("unexpected error: %v", err)
	}
	v, dirty, verr := dbmigrate.Version(dbmigrate.DialectPostgres, dsn)
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

func TestMigration0002PostgresAbortsOnLegacyIdempotencyWithoutDocuments(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()

	if err := migratePostgresTo(t, dsn, 1); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenPostgres(context.Background(), db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	hash := make([]byte, 32)
	_, err = sqlDB.Exec(`
		INSERT INTO fiscal.idempotency_records (
			scope_id, idempotency_key, request_hash, document_id, state, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		"scope", "11111111-1111-1111-1111-111111111111", hash, nil, "in_progress", now, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	var nDocs int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM fiscal.documents`).Scan(&nDocs); err != nil || nDocs != 0 {
		t.Fatalf("docs=%d err=%v", nDocs, err)
	}
	_ = sqlDB.Close()

	err = dbmigrate.Up(dbmigrate.DialectPostgres, dsn)
	if err == nil {
		t.Fatal("expected abort with legacy idempotency_records only")
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "aborted") && !strings.Contains(msg, "legacy") {
		t.Fatalf("unexpected error: %v", err)
	}
	v, dirty, verr := dbmigrate.Version(dbmigrate.DialectPostgres, dsn)
	if verr != nil {
		t.Fatal(verr)
	}
	if !dirty {
		t.Fatalf("expected dirty after aborted 0002, version=%d", v)
	}
}

func migratePostgresTo(t *testing.T, dsn string, version uint) error {
	t.Helper()
	src, err := iofs.New(migrations.FS, "postgres")
	if err != nil {
		return err
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	switch u.Scheme {
	case "postgres", "postgresql", "pgx", "pgx5":
		u.Scheme = "pgx5"
	default:
		return fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	q := u.Query()
	q.Set("x-migrations-table", `"public"."`+dbmigrate.MigrationsTable+`"`)
	q.Set("x-migrations-table-quoted", "true")
	u.RawQuery = q.Encode()
	m, err := migrate.NewWithSourceInstance("iofs", src, u.String())
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Migrate(version); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
