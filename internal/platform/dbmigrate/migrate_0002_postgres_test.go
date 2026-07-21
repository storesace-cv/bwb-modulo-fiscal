package dbmigrate_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/migrations"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// isolatedPostgresTempDBName matches only disposable test databases we create.
var isolatedPostgresTempDBName = regexp.MustCompile(`^bwb_fiscal_test_[0-9]+$`)

func TestMigration0002PostgresEmptyReachesVersion2(t *testing.T) {
	dsn, cleanup := createIsolatedPostgresTestDB(t)
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
	dsn, cleanup := createIsolatedPostgresTestDB(t)
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
	dsn, cleanup := createIsolatedPostgresTestDB(t)
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

// createIsolatedPostgresTestDB creates a disposable DB named bwb_fiscal_test_<nano>.
// It never migrates or mutates the original FISCAL_TEST_DATABASE_URL database.
func createIsolatedPostgresTestDB(t *testing.T) (dsn string, cleanup func()) {
	t.Helper()
	base := strings.TrimSpace(os.Getenv("FISCAL_TEST_DATABASE_URL"))
	if base == "" {
		t.Skip("FISCAL_TEST_DATABASE_URL not set")
	}
	adminDSN, err := postgresAdminDSN(base)
	if err != nil {
		t.Fatalf("refusing unsafe test DSN: %v", err)
	}
	name := fmt.Sprintf("bwb_fiscal_test_%d", time.Now().UnixNano())
	if !isolatedPostgresTempDBName.MatchString(name) {
		t.Fatalf("generated name %q rejected by safety pattern", name)
	}

	ctx := context.Background()
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.PingContext(ctx); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE `+pqQuoteIdent(name)); err != nil {
		_ = admin.Close()
		t.Fatalf("CREATE DATABASE: %v", err)
	}

	isolated, err := rewritePostgresDBName(base, name)
	if err != nil {
		_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS `+pqQuoteIdent(name)+` WITH (FORCE)`)
		_ = admin.Close()
		t.Fatal(err)
	}

	cleanup = func() {
		_, _ = admin.ExecContext(context.Background(), `DROP DATABASE IF EXISTS `+pqQuoteIdent(name)+` WITH (FORCE)`)
		_ = admin.Close()
	}
	return isolated, cleanup
}

func postgresAdminDSN(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	host := strings.ToLower(u.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return "", fmt.Errorf("host %q is not a local test host", host)
	}
	dbname := strings.TrimPrefix(u.Path, "/")
	if i := strings.IndexByte(dbname, '/'); i >= 0 {
		dbname = dbname[:i]
	}
	// Require the bootstrap URL to look like a test harness DB (CI uses "fiscal").
	if !isIdentifiedPostgresTestDB(dbname) {
		return "", fmt.Errorf("database %q is not identified as a test database", dbname)
	}
	// Connect to maintenance DB only — never issue CREATE/DROP against the shared app DB name.
	u.Path = "/postgres"
	return u.String(), nil
}

func isIdentifiedPostgresTestDB(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if n == "fiscal" { // CI service container default for FISCAL_TEST_DATABASE_URL
		return true
	}
	return strings.Contains(n, "test")
}

func rewritePostgresDBName(base, name string) (string, error) {
	if !isolatedPostgresTempDBName.MatchString(name) {
		return "", fmt.Errorf("refusing non-isolated database name %q", name)
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = "/" + name
	return u.String(), nil
}

func pqQuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
