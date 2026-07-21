package auth_test

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/auth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestCredentialStoreAuthenticatorPolicies(t *testing.T) {
	ctx := auth.ContextWithRequestID(context.Background(), "req_pol")
	path := filepath.Join(t.TempDir(), "auth.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, BusyTimeout: time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	store := persistence.NewCredentialStore(sqlDB, persistence.DialectSQLite)
	fixed := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	store.SetClock(func() time.Time { return fixed })

	create := func(id, env string) {
		t.Helper()
		if _, err := store.CreateScope(ctx, persistence.CreateScopeParams{
			ScopeID: id, TaxpayerNIF: "5000000000", IANATimezone: "Africa/Luanda",
			SeriesEffectiveCode: "A", Environment: env,
		}); err != nil {
			t.Fatal(err)
		}
	}
	issue := func(scope string, exp *time.Time) (string, string) {
		t.Helper()
		var tok string
		rec, err := store.Issue(ctx, persistence.IssueParams{
			ScopeID: scope, CreatedBy: "admin", ExpiresAt: exp,
			Deliver: func(token string) error { tok = token; return nil },
		})
		if err != nil {
			t.Fatal(err)
		}
		return rec.CredentialID, tok
	}
	authReq := func(tok string) *http.Request {
		r, _ := http.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set("Authorization", "Bearer "+tok)
		return r
	}

	create("hom", "homologation")
	create("dev", "development")
	_, tokHom := issue("hom", nil)
	_, tokDev := issue("dev", nil)

	a, err := auth.NewCredentialStoreAuthenticator(store, "homologation")
	if err != nil {
		t.Fatal(err)
	}
	a.SetClock(func() time.Time { return fixed })

	if b, err := a.Authenticate(ctx, authReq(tokHom)); err != nil || b.ScopeID != "hom" {
		t.Fatalf("hom ok: %#v %v", b, err)
	}
	if _, err := a.Authenticate(ctx, authReq(tokDev)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("env mismatch: %v", err)
	}
	if _, err := a.Authenticate(ctx, authReq("not-a-sandbox-token")); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("malformed: %v", err)
	}

	// grace accepted
	oldTok := tokHom
	var newTok string
	if _, err := store.Rotate(ctx, persistence.RotateParams{
		ScopeID: "hom", CreatedBy: "admin", GraceUntil: fixed.Add(time.Hour),
		Deliver: func(token string) error { newTok = token; return nil },
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Authenticate(ctx, authReq(newTok)); err != nil {
		t.Fatalf("new active: %v", err)
	}
	if _, err := a.Authenticate(ctx, authReq(oldTok)); err != nil {
		t.Fatalf("grace: %v", err)
	}

	// grace terminated
	a.SetClock(func() time.Time { return fixed.Add(2 * time.Hour) })
	if _, err := a.Authenticate(ctx, authReq(oldTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("grace ended: %v", err)
	}
	a.SetClock(func() time.Time { return fixed })

	// revoked
	authRec, err := store.VerifyCredentialTokenHash(ctx, persistence.HashCredentialToken(newTok))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Revoke(ctx, persistence.RevokeParams{ScopeID: "hom", CredentialID: authRec.CredentialID}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Authenticate(ctx, authReq(newTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("revoked: %v", err)
	}

	// expired
	past := fixed.Add(-time.Minute)
	create("hom-exp", "homologation")
	_, expTok := issue("hom-exp", &past)
	if _, err := a.Authenticate(ctx, authReq(expTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expired: %v", err)
	}
}
