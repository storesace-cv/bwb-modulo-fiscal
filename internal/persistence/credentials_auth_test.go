package persistence_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
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

func TestTokenSinkRollbackAndVerify(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "sink.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, BusyTimeout: time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	store := persistence.NewCredentialStore(sqlDB, persistence.DialectSQLite)
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
		assertScopeUnaffected(t, ctx, sqlDB, false, scopeID, 0, 0)
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
		assertStatus(t, ctx, sqlDB, false, scopeID, rec.CredentialID, "active")
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
}

func TestCredentialStoreAuthEnvironmentMismatchSQLite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "env.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, BusyTimeout: time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	store := persistence.NewCredentialStore(sqlDB, persistence.DialectSQLite)
	_, err = store.CreateScope(ctx, persistence.CreateScopeParams{
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
