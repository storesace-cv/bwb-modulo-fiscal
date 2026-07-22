package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbtest"
)

func TestPublicAPIDoesNotExposeTokenHash(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(persistence.CredentialAuthRecord{}),
		reflect.TypeOf(persistence.CredentialRecord{}),
		reflect.TypeOf(persistence.ScopeRecord{}),
		reflect.TypeOf(persistence.AuthAuditEvent{}),
		reflect.TypeOf(persistence.RotateOutcome{}),
	} {
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			if strings.Contains(name, "hash") || strings.Contains(name, "token") {
				t.Fatalf("%s exposes field %s", typ.Name(), typ.Field(i).Name)
			}
		}
	}
}

func TestTokenSinkRollbackAndVerifySQLite(t *testing.T) {
	store, sqlDB := openCredSQLite(t)
	defer sqlDB.Close()
	runTokenSinkSuite(t, store, sqlDB, false)
}

func TestTokenSinkRollbackAndVerifyPostgres(t *testing.T) {
	store, sqlDB := openCredPostgres(t)
	defer sqlDB.Close()
	runTokenSinkSuite(t, store, sqlDB, true)
}

func runTokenSinkSuite(t *testing.T, store *persistence.CredentialStore, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	ctx := context.Background()
	scopeID := "sink-scope"
	mustCreateScope(t, ctx, store, scopeID)

	t.Run("deliver_failure_rolls_back", func(t *testing.T) {
		_, err := store.Issue(ctx, persistence.IssueParams{
			ScopeID: scopeID, CreatedBy: "admin",
			Deliver: func(string) error { return errors.New("sink failed") },
		})
		if err == nil {
			t.Fatal("expected sink failure")
		}
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scopeID, 0, 0)
	})

	t.Run("rotate_deliver_failure_keeps_active", func(t *testing.T) {
		rec, tok := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin"})
		_, err := store.Rotate(ctx, persistence.RotateParams{
			ScopeID: scopeID, CreatedBy: "admin", GraceUntil: time.Now().UTC().Add(time.Hour),
			Deliver: func(string) error { return errors.New("sink failed") },
		})
		if err == nil {
			t.Fatal("expected rotate sink failure")
		}
		assertStatus(t, ctx, sqlDB, postgres, scopeID, rec.CredentialID, "active")
		got, err := store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(tok))
		if err != nil || got.CredentialID != rec.CredentialID {
			t.Fatalf("previous token must still verify: %v %#v", err, got)
		}
	})

	t.Run("success_verify", func(t *testing.T) {
		scope2 := "sink-scope-ok"
		mustCreateScope(t, ctx, store, scope2)
		rec, tok := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scope2, CreatedBy: "admin"})
		got, err := store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(tok))
		if err != nil {
			t.Fatal(err)
		}
		if got.CredentialID != rec.CredentialID || got.TaxpayerNIF == "" {
			t.Fatalf("%#v", got)
		}
		if _, err := store.VerifyCredentialTokenHash(ctx, make([]byte, 32)); !errors.Is(err, persistence.ErrCredentialNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("audit_after_deliver_rolls_back_delivered_not_verifiable", func(t *testing.T) {
		installAuditInsertReject(t, ctx, sqlDB, postgres)
		scope := "sink-audit-after"
		mustCreateScope(t, ctx, store, scope)
		var delivered string
		_, err := store.Issue(ctx, persistence.IssueParams{
			ScopeID: scope, CreatedBy: "admin",
			Deliver: func(token string) error {
				delivered = token
				return nil
			},
		})
		if err == nil {
			t.Fatal("expected audit failure after deliver")
		}
		if delivered == "" {
			t.Fatal("token was delivered to sink before audit failure")
		}
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scope, 0, 0)
		_, err = store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(delivered))
		if !errors.Is(err, persistence.ErrCredentialNotFound) {
			t.Fatalf("delivered token must not verify after rollback: %v", err)
		}
		assertNoSecretInAudit(t, ctx, sqlDB, postgres, scope, delivered)
		dropAuditInsertReject(t, ctx, sqlDB, postgres)
	})
}

func TestCredentialStorePublicAPIHasNoCommitSubstitution(t *testing.T) {
	typ := reflect.TypeOf(persistence.CredentialStore{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		if strings.Contains(name, "commit") {
			t.Fatalf("CredentialStore exposes commit-related field %s", typ.Field(i).Name)
		}
	}
	ptr := reflect.TypeOf((*persistence.CredentialStore)(nil))
	for i := 0; i < ptr.NumMethod(); i++ {
		name := ptr.Method(i).Name
		lower := strings.ToLower(name)
		if strings.Contains(lower, "commithook") ||
			strings.Contains(lower, "setcommit") ||
			strings.Contains(lower, "replacecommit") ||
			(strings.Contains(lower, "commit") && strings.Contains(lower, "hook")) {
			t.Fatalf("CredentialStore exports commit substitution method %s", name)
		}
	}
}

func TestCredentialStoreAuthEnvironmentMismatchSQLite(t *testing.T) {
	store, sqlDB := openCredSQLite(t)
	defer sqlDB.Close()
	ctx := context.Background()
	_, err := store.CreateScope(ctx, persistence.CreateScopeParams{
		ScopeID: "env-dev", TaxpayerNIF: "5000000000", IANATimezone: "Africa/Luanda",
		SeriesEffectiveCode: "A", Environment: "development",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, tok := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: "env-dev", CreatedBy: "admin"})
	rec, err := store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(tok))
	if err != nil {
		t.Fatal(err)
	}
	if rec.ScopeEnvironment != "development" {
		t.Fatalf("%q", rec.ScopeEnvironment)
	}
}

func openCredSQLite(t *testing.T) (*persistence.CredentialStore, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "sink.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, BusyTimeout: time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	return persistence.NewCredentialStore(sqlDB, persistence.DialectSQLite), sqlDB
}

func openCredPostgres(t *testing.T) (*persistence.CredentialStore, *sql.DB) {
	t.Helper()
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	return persistence.NewCredentialStore(sqlDB, persistence.DialectPostgres), sqlDB
}
