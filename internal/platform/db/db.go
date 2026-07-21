// Package db abre conexões PostgreSQL e SQLite conforme DEC-STACK-001.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const (
	DriverPostgres = "postgres"
	DriverSQLite   = "sqlite"

	// DefaultSearchPath is applied via pgx RuntimeParams on every pool connection.
	// Application SQL must still use fiscal.* qualifiers explicitly.
	DefaultSearchPath = "fiscal,public"

	defaultBusyTimeout = 5 * time.Second
)

// PostgresConfig configures a cloud PostgreSQL connection.
type PostgresConfig struct {
	URL        string
	SearchPath string // default DefaultSearchPath; set on every connection via RuntimeParams
}

// SQLiteConfig configures an Edge SQLite connection.
type SQLiteConfig struct {
	Path         string
	BusyTimeout  time.Duration // default 5s
	MaxOpenConns int           // must be 0 (default) or 1; Edge is always a single open connection
}

// OpenPostgres opens pgx via database/sql with search_path on all pool connections.
func OpenPostgres(ctx context.Context, cfg PostgresConfig) (*sql.DB, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("db: empty postgres URL")
	}
	searchPath := cfg.SearchPath
	if searchPath == "" {
		searchPath = DefaultSearchPath
	}

	pgCfg, err := pgx.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("db: parse postgres config: %w", err)
	}
	if pgCfg.RuntimeParams == nil {
		pgCfg.RuntimeParams = make(map[string]string)
	}
	pgCfg.RuntimeParams["search_path"] = searchPath

	db := stdlib.OpenDB(*pgCfg)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db: ping postgres: %w", err)
	}
	return db, nil
}

// OpenSQLite opens modernc SQLite with WAL, foreign_keys, busy_timeout and MaxOpenConns(1).
func OpenSQLite(ctx context.Context, cfg SQLiteConfig) (*sql.DB, error) {
	if strings.TrimSpace(cfg.Path) == "" {
		return nil, fmt.Errorf("db: empty sqlite path")
	}
	if cfg.MaxOpenConns != 0 && cfg.MaxOpenConns != 1 {
		return nil, fmt.Errorf("db: MaxOpenConns must be 0 or 1 for Edge SQLite, got %d", cfg.MaxOpenConns)
	}
	busy := cfg.BusyTimeout
	if busy <= 0 {
		busy = defaultBusyTimeout
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("db: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db: ping sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		fmt.Sprintf("PRAGMA busy_timeout = %d", busy.Milliseconds()),
	}
	for _, p := range pragmas {
		if _, err := db.ExecContext(ctx, p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("db: %s: %w", p, err)
		}
	}
	return db, nil
}
