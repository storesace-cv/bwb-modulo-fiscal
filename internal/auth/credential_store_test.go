package auth_test

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/auth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

type countingVerifier struct {
	verifyCalls atomic.Int64
	auditCalls  atomic.Int64
	verifyErr   error
	rec         *persistence.CredentialAuthRecord
	lastAudit   persistence.AuthAuditEvent
}

func (c *countingVerifier) VerifyCredentialTokenHash(ctx context.Context, computedHash []byte) (*persistence.CredentialAuthRecord, error) {
	c.verifyCalls.Add(1)
	if c.verifyErr != nil {
		return nil, c.verifyErr
	}
	if c.rec == nil {
		return nil, persistence.ErrCredentialNotFound
	}
	return c.rec, nil
}

func (c *countingVerifier) RecordAuthAudit(ctx context.Context, ev persistence.AuthAuditEvent) error {
	c.auditCalls.Add(1)
	c.lastAudit = ev
	return nil
}

func validFormatToken() string {
	return persistence.CredentialTokenPrefix + strings.Repeat("A", 43)
}

func authReq(tok string) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	return r
}

func TestValidSandboxTokenFormatRejectsUnicodeWithoutDB(t *testing.T) {
	v := &countingVerifier{}
	a, err := auth.NewCredentialStoreAuthenticator(v, "homologation")
	if err != nil {
		t.Fatal(err)
	}
	ctx := auth.ContextWithRequestID(context.Background(), "req_fmt")

	// Multibyte letter keeps exact byte length but is not Base64URL ASCII.
	unicodeBody := strings.Repeat("A", 41) + "é" // é = 2 bytes
	unicodeTok := persistence.CredentialTokenPrefix + unicodeBody
	if len(unicodeTok) != persistence.CredentialTokenExactLen {
		t.Fatalf("fixture length=%d want %d", len(unicodeTok), persistence.CredentialTokenExactLen)
	}
	if _, err := a.Authenticate(ctx, authReq(unicodeTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("unicode: %v", err)
	}
	if v.verifyCalls.Load() != 0 {
		t.Fatalf("unicode token must not hit verifier, calls=%d", v.verifyCalls.Load())
	}

	plusTok := persistence.CredentialTokenPrefix + strings.Repeat("A", 42) + "+"
	if _, err := a.Authenticate(ctx, authReq(plusTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("plus: %v", err)
	}
	slashTok := persistence.CredentialTokenPrefix + strings.Repeat("A", 42) + "/"
	if _, err := a.Authenticate(ctx, authReq(slashTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("slash: %v", err)
	}
	if v.verifyCalls.Load() != 0 {
		t.Fatalf("invalid alphabet must not hit verifier, calls=%d", v.verifyCalls.Load())
	}

	if _, err := a.Authenticate(ctx, authReq(validFormatToken())); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("well-formed unknown: %v", err)
	}
	if v.verifyCalls.Load() != 1 {
		t.Fatalf("well-formed token must hit verifier once, calls=%d", v.verifyCalls.Load())
	}
}

func TestCredentialVerifierErrorMapsToErrInternal(t *testing.T) {
	v := &countingVerifier{verifyErr: errors.New("db unavailable")}
	a, err := auth.NewCredentialStoreAuthenticator(v, "homologation")
	if err != nil {
		t.Fatal(err)
	}
	ctx := auth.ContextWithRequestID(context.Background(), "req_int")
	_, err = a.Authenticate(ctx, authReq(validFormatToken()))
	if !errors.Is(err, auth.ErrInternal) {
		t.Fatalf("err=%v", err)
	}
	if v.auditCalls.Load() != 0 {
		t.Fatal("verifier infrastructure failure must not write reject audit as success path")
	}
}

func TestCredentialStoreAuthenticatorClosedDBAndCanceledContext(t *testing.T) {
	ctx := auth.ContextWithRequestID(context.Background(), "req_db")
	path := filepath.Join(t.TempDir(), "auth-closed.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, BusyTimeout: time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	store := persistence.NewCredentialStore(sqlDB, persistence.DialectSQLite)
	a, err := auth.NewCredentialStoreAuthenticator(store, "homologation")
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()
	_, err = a.Authenticate(ctx, authReq(validFormatToken()))
	if !errors.Is(err, auth.ErrInternal) {
		t.Fatalf("closed db: %v", err)
	}

	path2 := filepath.Join(t.TempDir(), "auth-cancel.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path2); err != nil {
		t.Fatal(err)
	}
	sqlDB2, err := db.OpenSQLite(context.Background(), db.SQLiteConfig{Path: path2, BusyTimeout: time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB2.Close()
	store2 := persistence.NewCredentialStore(sqlDB2, persistence.DialectSQLite)
	a2, err := auth.NewCredentialStoreAuthenticator(store2, "homologation")
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(auth.ContextWithRequestID(context.Background(), "req_cancel"))
	cancel()
	_, err = a2.Authenticate(canceled, authReq(validFormatToken()))
	if !errors.Is(err, auth.ErrInternal) {
		t.Fatalf("canceled: %v", err)
	}
}

func TestAuthRejectAuditOmitsToken(t *testing.T) {
	v := &countingVerifier{}
	a, err := auth.NewCredentialStoreAuthenticator(v, "homologation")
	if err != nil {
		t.Fatal(err)
	}
	tok := validFormatToken()
	ctx := auth.ContextWithRequestID(context.Background(), "req_audit")
	_, _ = a.Authenticate(ctx, authReq(tok))
	joined := v.lastAudit.Action + v.lastAudit.Result + v.lastAudit.ReasonCode + v.lastAudit.RequestID +
		v.lastAudit.CredentialID + v.lastAudit.ScopeID
	if strings.Contains(joined, tok) || strings.Contains(joined, persistence.CredentialTokenPrefix) {
		t.Fatal("audit event leaked token material")
	}
}

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

	a.SetClock(func() time.Time { return fixed.Add(2 * time.Hour) })
	if _, err := a.Authenticate(ctx, authReq(oldTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("grace ended: %v", err)
	}
	a.SetClock(func() time.Time { return fixed })

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

	past := fixed.Add(-time.Minute)
	create("hom-exp", "homologation")
	_, expTok := issue("hom-exp", &past)
	if _, err := a.Authenticate(ctx, authReq(expTok)); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("expired: %v", err)
	}
}
