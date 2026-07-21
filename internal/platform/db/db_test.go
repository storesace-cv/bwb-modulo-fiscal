package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
)

func TestOpenSQLiteRejectsMaxOpenConnsAboveOne(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "edge.db")
	_, err := db.OpenSQLite(ctx, db.SQLiteConfig{
		Path:         path,
		BusyTimeout:  time.Second,
		MaxOpenConns: 2,
	})
	if err == nil {
		t.Fatal("expected error for MaxOpenConns > 1")
	}
}

func TestOpenSQLiteForcesMaxOpenConnsOne(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "edge.db")
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, MaxOpenConns: 0})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", sqlDB.Stats().MaxOpenConnections)
	}
}

func TestOpenPostgresSearchPathOnAllPoolConns(t *testing.T) {
	dsn := os.Getenv("FISCAL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("FISCAL_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()

	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(4)

	const n = 3
	conns := make([]*struct {
		release func()
		path    string
	}, n)

	for i := 0; i < n; i++ {
		conn, err := sqlDB.Conn(ctx)
		if err != nil {
			t.Fatalf("Conn[%d]: %v", i, err)
		}
		var path string
		if err := conn.QueryRowContext(ctx, `SELECT current_setting('search_path')`).Scan(&path); err != nil {
			_ = conn.Close()
			t.Fatalf("search_path[%d]: %v", i, err)
		}
		conns[i] = &struct {
			release func()
			path    string
		}{release: func() { _ = conn.Close() }, path: path}
	}
	defer func() {
		for _, c := range conns {
			if c != nil {
				c.release()
			}
		}
	}()

	want := db.DefaultSearchPath
	for i, c := range conns {
		got := normalizeSearchPath(c.path)
		if got != want {
			t.Fatalf("conn[%d] search_path = %q (normalized %q), want %q", i, c.path, got, want)
		}
	}
}

func normalizeSearchPath(s string) string {
	// Postgres may render "fiscal, public"; RuntimeParams use "fiscal,public".
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}
