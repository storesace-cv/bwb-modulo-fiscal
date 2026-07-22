package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
)

// Engine-level deferred constraints force a real COMMIT failure after Deliver,
// without any CredentialStore commit substitution hooks.

func TestCommitFailureAfterDeliverPostgres(t *testing.T) {
	store, sqlDB := openCredPostgres(t)
	defer sqlDB.Close()
	runCommitFailureAfterDeliver(t, store, sqlDB, true)
}

func TestCommitFailureAfterDeliverSQLite(t *testing.T) {
	store, sqlDB := openCredSQLite(t)
	defer sqlDB.Close()
	runCommitFailureAfterDeliver(t, store, sqlDB, false)
}

func runCommitFailureAfterDeliver(t *testing.T, store *persistence.CredentialStore, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	ctx := context.Background()

	t.Run("issue", func(t *testing.T) {
		scope := "commit-fail-issue"
		mustCreateScope(t, ctx, store, scope)
		installDeferredCommitFailure(t, ctx, sqlDB, postgres)
		t.Cleanup(func() { dropDeferredCommitFailure(t, context.Background(), sqlDB, postgres) })

		var delivered string
		_, err := store.Issue(ctx, persistence.IssueParams{
			ScopeID: scope, CreatedBy: "admin",
			Deliver: func(token string) error {
				delivered = token
				return nil
			},
		})
		if err == nil {
			t.Fatal("expected Issue commit failure")
		}
		if delivered == "" {
			t.Fatal("token must have been delivered before commit failure")
		}
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scope, 0, 0)
		_, err = store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(delivered))
		if !errors.Is(err, persistence.ErrCredentialNotFound) {
			t.Fatalf("delivered token must not verify: %v", err)
		}
	})

	t.Run("rotate", func(t *testing.T) {
		scope := "commit-fail-rotate"
		mustCreateScope(t, ctx, store, scope)
		prev, prevTok := mustIssue(t, ctx, store, persistence.IssueParams{
			ScopeID: scope, CreatedBy: "admin",
		})
		assertStatus(t, ctx, sqlDB, postgres, scope, prev.CredentialID, "active")

		installDeferredCommitFailure(t, ctx, sqlDB, postgres)
		t.Cleanup(func() { dropDeferredCommitFailure(t, context.Background(), sqlDB, postgres) })

		var delivered string
		_, err := store.Rotate(ctx, persistence.RotateParams{
			ScopeID: scope, CreatedBy: "admin", GraceUntil: time.Now().UTC().Add(time.Hour),
			Deliver: func(token string) error {
				delivered = token
				return nil
			},
		})
		if err == nil {
			t.Fatal("expected Rotate commit failure")
		}
		if delivered == "" {
			t.Fatal("token must have been delivered before commit failure")
		}
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scope, 1, 1) // prior issue audit remains
		assertStatus(t, ctx, sqlDB, postgres, scope, prev.CredentialID, "active")
		got, err := store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(prevTok))
		if err != nil || got.CredentialID != prev.CredentialID {
			t.Fatalf("previous active token must still verify: %v %#v", err, got)
		}
		_, err = store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(delivered))
		if !errors.Is(err, persistence.ErrCredentialNotFound) {
			t.Fatalf("delivered rotate token must not verify: %v", err)
		}
	})
}

func installDeferredCommitFailure(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	if postgres {
		must(t, execSQL(ctx, sqlDB, true, `
			CREATE OR REPLACE FUNCTION fiscal.test_deferred_commit_fail()
			RETURNS trigger
			LANGUAGE plpgsql
			AS $$
			BEGIN
				RAISE EXCEPTION 'test deferred commit failure';
			END;
			$$`))
		_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS test_deferred_commit_fail ON fiscal.api_credentials`)
		must(t, execSQL(ctx, sqlDB, true, `
			CREATE CONSTRAINT TRIGGER test_deferred_commit_fail
			AFTER INSERT ON fiscal.api_credentials
			DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW
			EXECUTE FUNCTION fiscal.test_deferred_commit_fail()`))
		return
	}

	// SQLite: AFTER INSERT trigger inserts a deferred FK violation checked only at COMMIT.
	must(t, execSQL(ctx, sqlDB, false, `
		CREATE TABLE IF NOT EXISTS commit_fail_parent (
			id TEXT PRIMARY KEY
		)`))
	must(t, execSQL(ctx, sqlDB, false, `
		CREATE TABLE IF NOT EXISTS commit_fail_child (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id TEXT NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES commit_fail_parent(id) DEFERRABLE INITIALLY DEFERRED
		)`))
	_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS api_credentials_deferred_commit_fail`)
	must(t, execSQL(ctx, sqlDB, false, `
		CREATE TRIGGER api_credentials_deferred_commit_fail
		AFTER INSERT ON api_credentials
		BEGIN
			INSERT INTO commit_fail_child(parent_id) VALUES (NEW.credential_id);
		END`))
}

func dropDeferredCommitFailure(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	if postgres {
		_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS test_deferred_commit_fail ON fiscal.api_credentials`)
		_, _ = sqlDB.ExecContext(ctx, `DROP FUNCTION IF EXISTS fiscal.test_deferred_commit_fail()`)
		return
	}
	_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS api_credentials_deferred_commit_fail`)
	_, _ = sqlDB.ExecContext(ctx, `DROP TABLE IF EXISTS commit_fail_child`)
	_, _ = sqlDB.ExecContext(ctx, `DROP TABLE IF EXISTS commit_fail_parent`)
}
