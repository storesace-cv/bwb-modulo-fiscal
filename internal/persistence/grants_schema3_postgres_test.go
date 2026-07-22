package persistence_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbtest"
)

// TestPostgresSchema3OperationalGrants applies deploy/postgres/grants-schema3-runtime-admin.sql
// and asserts positive/negative privileges for fiscal_admin and fiscal_runtime.
func TestPostgresSchema3OperationalGrants(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()
	ctx := context.Background()

	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	owner, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatalf("open owner: %v", err)
	}
	defer owner.Close()

	grantsPath := filepath.Join(repoRootFromPersistenceTest(t), "deploy", "postgres", "grants-schema3-runtime-admin.sql")
	raw, err := os.ReadFile(grantsPath)
	if err != nil {
		t.Fatalf("read grants sql: %v", err)
	}
	if _, err := owner.ExecContext(ctx, string(raw)); err != nil {
		t.Fatalf("apply grants: %v", err)
	}

	if _, err := owner.ExecContext(ctx, `GRANT fiscal_admin TO CURRENT_USER`); err != nil {
		t.Fatalf("grant fiscal_admin to test user: %v", err)
	}
	if _, err := owner.ExecContext(ctx, `GRANT fiscal_runtime TO CURRENT_USER`); err != nil {
		t.Fatalf("grant fiscal_runtime to test user: %v", err)
	}

	must(t, execSQL(ctx, owner, true, `
		INSERT INTO fiscal.scopes (
			scope_id, taxpayer_nif, iana_timezone, series_effective_code, environment, status, created_at
		) VALUES (?, ?, ?, ?, ?, 'active', ?)`,
		"scope-grants-synth", "5000000000", "Africa/Luanda", "A", "homologation",
		timeArg(true, time.Now().UTC()),
	))
	hash := make([]byte, 32)
	copy(hash, []byte("grants-synth-token-hash-pad!!!!"))
	must(t, execSQL(ctx, owner, true, `
		INSERT INTO fiscal.api_credentials (
			credential_id, scope_id, token_hash, status, created_at, created_by
		) VALUES (?, ?, ?, 'active', ?, 'grants-test')`,
		"cred-grants-synth", "scope-grants-synth", hash, timeArg(true, time.Now().UTC()),
	))
	must(t, execSQL(ctx, owner, true, `
		INSERT INTO fiscal.audit_events (
			event_id, occurred_at, credential_id, scope_id, action, result
		) VALUES (?, ?, ?, ?, 'grants_probe', 'ok')`,
		"audit-grants-synth", timeArg(true, time.Now().UTC()), "cred-grants-synth", "scope-grants-synth",
	))

	t.Run("admin_update_allowed_columns", func(t *testing.T) {
		err := withRole(t, ctx, owner, "fiscal_admin", func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
				UPDATE fiscal.api_credentials
				SET status = 'revoked', revoked_at = $1, grace_until = NULL
				WHERE credential_id = $2`, time.Now().UTC(), "cred-grants-synth")
			return err
		})
		if err != nil {
			t.Fatalf("allowed column update failed: %v", err)
		}
	})

	t.Run("admin_update_forbidden_columns", func(t *testing.T) {
		forbidden := []struct {
			name string
			sql  string
			args []any
		}{
			{"token_hash", `UPDATE fiscal.api_credentials SET token_hash = $1 WHERE credential_id = $2`, []any{make([]byte, 32), "cred-grants-synth"}},
			{"credential_id", `UPDATE fiscal.api_credentials SET credential_id = $1 WHERE credential_id = $2`, []any{"cred-hijack", "cred-grants-synth"}},
			{"scope_id", `UPDATE fiscal.api_credentials SET scope_id = $1 WHERE credential_id = $2`, []any{"scope-grants-synth", "cred-grants-synth"}},
			{"rotated_from", `UPDATE fiscal.api_credentials SET rotated_from = $1 WHERE credential_id = $2`, []any{"cred-other", "cred-grants-synth"}},
			{"expires_at", `UPDATE fiscal.api_credentials SET expires_at = $1 WHERE credential_id = $2`, []any{time.Now().UTC(), "cred-grants-synth"}},
			{"created_at", `UPDATE fiscal.api_credentials SET created_at = $1 WHERE credential_id = $2`, []any{time.Now().UTC(), "cred-grants-synth"}},
			{"created_by", `UPDATE fiscal.api_credentials SET created_by = $1 WHERE credential_id = $2`, []any{"evil", "cred-grants-synth"}},
		}
		for _, tc := range forbidden {
			t.Run(tc.name, func(t *testing.T) {
				err := withRole(t, ctx, owner, "fiscal_admin", func(tx *sql.Tx) error {
					_, err := tx.ExecContext(ctx, tc.sql, tc.args...)
					return err
				})
				if err == nil {
					t.Fatalf("expected permission denied updating %s", tc.name)
				}
				if !isPrivilegeDenied(err) {
					t.Fatalf("want privilege denied for %s, got %v", tc.name, err)
				}
			})
		}
	})

	t.Run("runtime_cannot_update_credentials_or_scopes", func(t *testing.T) {
		err := withRole(t, ctx, owner, "fiscal_runtime", func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
				UPDATE fiscal.api_credentials SET status = 'revoked' WHERE credential_id = $1`,
				"cred-grants-synth")
			return err
		})
		if err == nil || !isPrivilegeDenied(err) {
			t.Fatalf("runtime update credentials: want privilege denied, got %v", err)
		}
		err = withRole(t, ctx, owner, "fiscal_runtime", func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
				UPDATE fiscal.scopes SET status = 'inactive' WHERE scope_id = $1`,
				"scope-grants-synth")
			return err
		})
		if err == nil || !isPrivilegeDenied(err) {
			t.Fatalf("runtime update scopes: want privilege denied, got %v", err)
		}
	})

	t.Run("admin_and_runtime_cannot_mutate_audit", func(t *testing.T) {
		for _, role := range []string{"fiscal_admin", "fiscal_runtime"} {
			err := withRole(t, ctx, owner, role, func(tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					UPDATE fiscal.audit_events SET result = 'tamper' WHERE event_id = $1`,
					"audit-grants-synth")
				return err
			})
			if err == nil || !isPrivilegeDenied(err) {
				t.Fatalf("%s update audit: want privilege denied, got %v", role, err)
			}
			err = withRole(t, ctx, owner, role, func(tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `
					DELETE FROM fiscal.audit_events WHERE event_id = $1`,
					"audit-grants-synth")
				return err
			})
			if err == nil || !isPrivilegeDenied(err) {
				t.Fatalf("%s delete audit: want privilege denied, got %v", role, err)
			}
		}
	})

	t.Run("no_generic_default_privileges_for_runtime_admin", func(t *testing.T) {
		var n int
		err := owner.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM pg_default_acl d
			JOIN pg_namespace n ON n.oid = d.defaclnamespace
			WHERE n.nspname = 'fiscal'
			  AND (
			    d.defaclacl::text LIKE '%fiscal_runtime%'
			    OR d.defaclacl::text LIKE '%fiscal_admin%'
			  )`).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("unexpected default privileges involving runtime/admin: count=%d", n)
		}
	})

	t.Run("admin_select_insert_credentials_ok", func(t *testing.T) {
		err := withRole(t, ctx, owner, "fiscal_admin", func(tx *sql.Tx) error {
			var status string
			if err := tx.QueryRowContext(ctx, `
				SELECT status FROM fiscal.api_credentials WHERE credential_id = $1`,
				"cred-grants-synth").Scan(&status); err != nil {
				return err
			}
			hash2 := make([]byte, 32)
			copy(hash2, []byte("grants-synth-token-hash-pad2222"))
			_, err := tx.ExecContext(ctx, `
				INSERT INTO fiscal.api_credentials (
					credential_id, scope_id, token_hash, status, grace_until, created_at, created_by
				) VALUES ($1, $2, $3, 'grace', $4, $5, 'grants-test')`,
				"cred-grants-synth-2", "scope-grants-synth", hash2,
				time.Now().UTC().Add(time.Hour), time.Now().UTC())
			return err
		})
		if err != nil {
			t.Fatalf("admin select/insert: %v", err)
		}
	})
}

func withRole(t *testing.T, ctx context.Context, owner *sql.DB, role string, fn func(*sql.Tx) error) error {
	t.Helper()
	tx, err := owner.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SET LOCAL ROLE `+role); err != nil {
		t.Fatalf("set role %s: %v", role, err)
	}
	return fn(tx)
}

func isPrivilegeDenied(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "permission denied") || strings.Contains(msg, "insufficient privilege")
}

func repoRootFromPersistenceTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Clean(filepath.Join(wd, "../.."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root not found from %s: %v", wd, err)
	}
	return root
}
