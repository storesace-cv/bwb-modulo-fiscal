// Package dbmigrate aplica migrations embutidas (forward-only) via golang-migrate.
package dbmigrate

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/storesace-cv/bwb-modulo-fiscal/migrations"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
)

const (
	// MigrationsTable is the control table name (PostgreSQL: public.bwb_schema_migrations).
	MigrationsTable = "bwb_schema_migrations"
	// ExpectedVersion is the latest forward migration version shipped in this binary.
	ExpectedVersion = uint(3)
)

// Dialect selects which embedded migration set to apply.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

// Up applies all pending forward migrations for the given DSN. Idempotent at head.
// Opens its own database connection (does not share caller's *sql.DB).
func Up(dialect Dialect, dsn string) error {
	m, err := newMigrate(dialect, dsn)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("dbmigrate: up: %w", err)
	}
	return nil
}

// Version returns the current migration version and dirty flag.
func Version(dialect Dialect, dsn string) (version uint, dirty bool, err error) {
	m, err := newMigrate(dialect, dsn)
	if err != nil {
		return 0, false, err
	}
	defer func() { _, _ = m.Close() }()
	v, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("dbmigrate: version: %w", err)
	}
	return v, dirty, nil
}

func newMigrate(dialect Dialect, dsn string) (*migrate.Migrate, error) {
	root, dbURL, err := sourceAndURL(dialect, dsn)
	if err != nil {
		return nil, err
	}
	src, err := iofs.New(migrations.FS, root)
	if err != nil {
		return nil, fmt.Errorf("dbmigrate: iofs: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, dbURL)
	if err != nil {
		return nil, fmt.Errorf("dbmigrate: new: %w", err)
	}
	return m, nil
}

func sourceAndURL(dialect Dialect, dsn string) (root string, databaseURL string, err error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return "", "", fmt.Errorf("dbmigrate: empty DSN")
	}
	switch dialect {
	case DialectPostgres:
		u, err := url.Parse(dsn)
		if err != nil {
			return "", "", fmt.Errorf("dbmigrate: parse postgres DSN: %w", err)
		}
		switch u.Scheme {
		case "postgres", "postgresql", "pgx", "pgx5":
			u.Scheme = "pgx5"
		default:
			return "", "", fmt.Errorf("dbmigrate: unsupported postgres scheme %q", u.Scheme)
		}
		q := u.Query()
		// Explicit public control table — do not rely on session search_path.
		q.Set("x-migrations-table", `"public"."`+MigrationsTable+`"`)
		q.Set("x-migrations-table-quoted", "true")
		u.RawQuery = q.Encode()
		return "postgres", u.String(), nil
	case DialectSQLite:
		path := dsn
		if strings.Contains(dsn, "://") {
			u, err := url.Parse(dsn)
			if err != nil {
				return "", "", fmt.Errorf("dbmigrate: parse sqlite DSN: %w", err)
			}
			path = strings.TrimPrefix(u.Path, "/")
			if u.Host != "" && u.Host != "." {
				path = u.Host + "/" + path
			}
			if path == "" {
				path = u.Opaque
			}
		}
		q := url.Values{}
		q.Set("x-migrations-table", MigrationsTable)
		return "sqlite", "sqlite://" + path + "?" + q.Encode(), nil
	default:
		return "", "", fmt.Errorf("dbmigrate: unknown dialect %q", dialect)
	}
}
