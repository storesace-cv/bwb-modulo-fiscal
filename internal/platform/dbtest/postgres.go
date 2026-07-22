// Package dbtest provides disposable PostgreSQL databases for integration tests.
package dbtest

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

	_ "github.com/jackc/pgx/v5/stdlib"
)

// isolatedTempDBName matches only disposable test databases we create.
var isolatedTempDBName = regexp.MustCompile(`^bwb_fiscal_test_[0-9]+$`)

// OpenIsolatedPostgres creates a disposable DB named bwb_fiscal_test_<nano>,
// derived from FISCAL_TEST_DATABASE_URL. It never migrates or mutates the
// bootstrap database named in that URL. Cleanup drops the temp DB.
func OpenIsolatedPostgres(t *testing.T) (dsn string, cleanup func()) {
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
	if !isolatedTempDBName.MatchString(name) {
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
	if !isIdentifiedPostgresTestDB(dbname) {
		return "", fmt.Errorf("database %q is not identified as a test database", dbname)
	}
	u.Path = "/postgres"
	return u.String(), nil
}

func isIdentifiedPostgresTestDB(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	if n == "fiscal" {
		return true
	}
	return strings.Contains(n, "test")
}

func rewritePostgresDBName(base, name string) (string, error) {
	if !isolatedTempDBName.MatchString(name) {
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
