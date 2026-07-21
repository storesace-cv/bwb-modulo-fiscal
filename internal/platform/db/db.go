// Package db abre conexões PostgreSQL e SQLite conforme DEC-STACK-001.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const (
	DriverPostgres = "postgres"
	DriverSQLite   = "sqlite"

	// DefaultSearchPath pins session resolution; application SQL still uses fiscal.* qualifiers.
	DefaultSearchPath = "fiscal,public"

	defaultBusyTimeout = 5 * time.Second
)

// PostgresConfig configures a cloud PostgreSQL connection.
type PostgresConfig struct {
	URL        string
	SearchPath string // default DefaultSearchPath
}

// SQLiteConfig configures an Edge SQLite connection.
type SQLiteConfig struct {
	Path         string
	BusyTimeout  time.Duration // default 5s
	MaxOpenConns int           // Edge must be 1
}

// OpenPostgres opens pgx via database/sql and sets search_path explicitly.
func OpenPostgres(ctx context.Context, cfg PostgresConfig) (*sql.DB, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("db: empty postgres URL")
	}
	searchPath := cfg.SearchPath
	if searchPath == "" {
		searchPath = DefaultSearchPath
	}
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("db: open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db: ping postgres: %w", err)
	}
	if _, err := db.ExecContext(ctx, "SELECT set_config('search_path', $1, false)", searchPath); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("db: set search_path: %w", err)
	}
	return db, nil
}

// OpenSQLite opens modernc SQLite with WAL, foreign_keys, busy_timeout and MaxOpenConns for Edge.
func OpenSQLite(ctx context.Context, cfg SQLiteConfig) (*sql.DB, error) {
	if strings.TrimSpace(cfg.Path) == "" {
		return nil, fmt.Errorf("db: empty sqlite path")
	}
	busy := cfg.BusyTimeout
	if busy <= 0 {
		busy = defaultBusyTimeout
	}
	maxOpen := cfg.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 1
	}

	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("db: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(maxOpen)

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
