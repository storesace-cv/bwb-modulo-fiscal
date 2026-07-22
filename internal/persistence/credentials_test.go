package persistence_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbtest"
)

func deliverCapture(dst *string) persistence.TokenSink {
	return func(token string) error {
		*dst = token
		return nil
	}
}

func mustIssue(t *testing.T, ctx context.Context, store *persistence.CredentialStore, p persistence.IssueParams) (*persistence.CredentialRecord, string) {
	t.Helper()
	var token string
	p.Deliver = deliverCapture(&token)
	rec, err := store.Issue(ctx, p)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if token == "" {
		t.Fatal("missing delivered token")
	}
	return rec, token
}

func mustRotate(t *testing.T, ctx context.Context, store *persistence.CredentialStore, p persistence.RotateParams) (*persistence.RotateOutcome, string) {
	t.Helper()
	var token string
	p.Deliver = deliverCapture(&token)
	out, err := store.Rotate(ctx, p)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if token == "" {
		t.Fatal("missing delivered token")
	}
	return out, token
}

func TestCredentialsSQLiteSuite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "creds.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	v, dirty, err := dbmigrate.Version(dbmigrate.DialectSQLite, path)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if dirty || v != 3 {
		t.Fatalf("version=%d dirty=%v want 3 false", v, dirty)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path, BusyTimeout: 5 * time.Second, MaxOpenConns: 1})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	store := persistence.NewCredentialStore(sqlDB, persistence.DialectSQLite)
	runCredentialsSuite(t, ctx, store, sqlDB, false)
}

func TestCredentialsPostgresSuite(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()
	ctx := context.Background()
	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	v, dirty, err := dbmigrate.Version(dbmigrate.DialectPostgres, dsn)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if dirty || v != 3 {
		t.Fatalf("version=%d dirty=%v want 3 false", v, dirty)
	}
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	store := persistence.NewCredentialStore(sqlDB, persistence.DialectPostgres)
	runCredentialsSuite(t, ctx, store, sqlDB, true)
}

func runCredentialsSuite(t *testing.T, ctx context.Context, store *persistence.CredentialStore, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	uid := fmt.Sprintf("%d", time.Now().UnixNano())

	t.Run("issue_rotate_revoke_with_audit", func(t *testing.T) {
		scopeID := "cred-scope-" + uid + "-irr"
		mustCreateScope(t, ctx, store, scopeID)

		issuedRec, issuedTok := mustIssue(t, ctx, store, persistence.IssueParams{
			ScopeID: scopeID, CreatedBy: "admin@test", RequestID: "req-issue-1",
		})
		assertTokenShape(t, issuedTok)
		assertStatus(t, ctx, sqlDB, postgres, scopeID, issuedRec.CredentialID, "active")
		assertAuditCount(t, ctx, sqlDB, postgres, scopeID, "credential.issue", 1)
		assertNoSecretInAudit(t, ctx, sqlDB, postgres, scopeID, issuedTok)

		rotated, rotatedTok := mustRotate(t, ctx, store, persistence.RotateParams{
			ScopeID: scopeID, CreatedBy: "admin@test",
			GraceUntil: time.Now().UTC().Add(time.Hour),
			RequestID:  "req-rotate-1",
		})
		_ = rotatedTok
		if rotated.PreviousID != issuedRec.CredentialID {
			t.Fatalf("previous=%s want %s", rotated.PreviousID, issuedRec.CredentialID)
		}
		if rotated.Credential.RotatedFrom == nil || *rotated.Credential.RotatedFrom != issuedRec.CredentialID {
			t.Fatalf("rotated_from=%v", rotated.Credential.RotatedFrom)
		}
		assertStatus(t, ctx, sqlDB, postgres, scopeID, issuedRec.CredentialID, "grace")
		assertStatus(t, ctx, sqlDB, postgres, scopeID, rotated.Credential.CredentialID, "active")
		assertAuditCount(t, ctx, sqlDB, postgres, scopeID, "credential.rotate", 1)

		revoked, err := store.Revoke(ctx, persistence.RevokeParams{
			ScopeID: scopeID, CredentialID: rotated.Credential.CredentialID,
			ReasonCode: "manual", RequestID: "req-revoke-1",
		})
		if err != nil {
			t.Fatalf("revoke: %v", err)
		}
		if revoked.Status != "revoked" || revoked.RevokedAt == nil {
			t.Fatalf("revoke meta: %+v", revoked)
		}
		assertStatus(t, ctx, sqlDB, postgres, scopeID, rotated.Credential.CredentialID, "revoked")
		assertAuditCount(t, ctx, sqlDB, postgres, scopeID, "credential.revoke", 1)
	})

	t.Run("one_active_and_one_grace_per_scope", func(t *testing.T) {
		scopeID := "cred-scope-" + uid + "-limits"
		mustCreateScope(t, ctx, store, scopeID)
		a, _ := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin"})
		_, err := store.Issue(ctx, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin", Deliver: func(string) error { return nil }})
		if !errors.Is(err, persistence.ErrCredentialConflict) {
			t.Fatalf("second active: %v", err)
		}
		_, err = store.Rotate(ctx, persistence.RotateParams{
			ScopeID: scopeID, CreatedBy: "admin", GraceUntil: time.Now().UTC().Add(time.Hour),
			Deliver: func(string) error { return nil },
		})
		if err != nil {
			t.Fatalf("rotate: %v", err)
		}
		// Force a second grace via SQL — must fail unique partial index.
		hash := make([]byte, 32)
		hash[0] = 0xab
		err = execSQL(ctx, sqlDB, postgres, `
			INSERT INTO `+tblCred(postgres, "api_credentials")+` (
				credential_id, scope_id, token_hash, status, grace_until, created_at, created_by
			) VALUES (?, ?, ?, 'grace', ?, ?, 'sql')`,
			"force-grace-"+uid, scopeID, hash, timeArg(postgres, time.Now().UTC().Add(time.Hour)), timeArg(postgres, time.Now().UTC()),
		)
		if err == nil {
			t.Fatal("expected second grace insert to fail")
		}
		_ = a
	})

	t.Run("rotate_same_scope_only", func(t *testing.T) {
		s1 := "cred-scope-" + uid + "-r1"
		s2 := "cred-scope-" + uid + "-r2"
		mustCreateScope(t, ctx, store, s1)
		mustCreateScope(t, ctx, store, s2)
		i1, _ := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: s1, CreatedBy: "admin"})
		i2, _ := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: s2, CreatedBy: "admin"})
		// Cross-scope rotated_from via SQL must fail composite FK.
		hash := make([]byte, 32)
		hash[1] = 0xcd
		err := execSQL(ctx, sqlDB, postgres, `
			INSERT INTO `+tblCred(postgres, "api_credentials")+` (
				credential_id, scope_id, token_hash, status, rotated_from, created_at, created_by
			) VALUES (?, ?, ?, 'active', ?, ?, 'sql')`,
			"cross-"+uid, s2, hash, i1.CredentialID, timeArg(postgres, time.Now().UTC()),
		)
		if err == nil {
			t.Fatal("expected cross-scope rotated_from to fail FK")
		}
		_ = i2
	})

	t.Run("self_reference_rejected", func(t *testing.T) {
		scopeID := "cred-scope-" + uid + "-self"
		mustCreateScope(t, ctx, store, scopeID)
		hash := make([]byte, 32)
		hash[2] = 0xef
		cid := "selfcred-" + uid
		err := execSQL(ctx, sqlDB, postgres, `
			INSERT INTO `+tblCred(postgres, "api_credentials")+` (
				credential_id, scope_id, token_hash, status, rotated_from, created_at, created_by
			) VALUES (?, ?, ?, 'active', ?, ?, 'sql')`,
			cid, scopeID, hash, cid, timeArg(postgres, time.Now().UTC()),
		)
		if err == nil {
			t.Fatal("expected self-reference CHECK to fail")
		}
	})

	t.Run("grace_until_must_be_future", func(t *testing.T) {
		scopeID := "cred-scope-" + uid + "-grace"
		mustCreateScope(t, ctx, store, scopeID)
		issuedRec, _ := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin"})
		fixed := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
		store.SetClock(func() time.Time { return fixed })
		t.Cleanup(func() { store.SetClock(nil) })

		assertRotateGraceRejected := func(t *testing.T, grace time.Time, wantCode string) {
			t.Helper()
			var credsBefore, auditBefore int
			mustScan(t, ctx, sqlDB, postgres,
				`SELECT COUNT(*) FROM `+tblCred(postgres, "api_credentials")+` WHERE scope_id = ?`,
				[]any{scopeID}, &credsBefore,
			)
			mustScan(t, ctx, sqlDB, postgres,
				`SELECT COUNT(*) FROM `+tblCred(postgres, "audit_events")+` WHERE scope_id = ?`,
				[]any{scopeID}, &auditBefore,
			)
			_, err := store.Rotate(ctx, persistence.RotateParams{
				ScopeID: scopeID, CreatedBy: "admin", GraceUntil: grace,
				Deliver: func(string) error { return nil },
			})
			var ve *persistence.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err=%v want ValidationError", err)
			}
			if ve.Field != "grace_until" || ve.Code != wantCode {
				t.Fatalf("validation = %+v want field=grace_until code=%s", ve, wantCode)
			}
			var credsAfter, auditAfter int
			mustScan(t, ctx, sqlDB, postgres,
				`SELECT COUNT(*) FROM `+tblCred(postgres, "api_credentials")+` WHERE scope_id = ?`,
				[]any{scopeID}, &credsAfter,
			)
			mustScan(t, ctx, sqlDB, postgres,
				`SELECT COUNT(*) FROM `+tblCred(postgres, "audit_events")+` WHERE scope_id = ?`,
				[]any{scopeID}, &auditAfter,
			)
			if credsAfter != credsBefore || auditAfter != auditBefore {
				t.Fatalf("mutation on validation failure creds %d→%d audit %d→%d",
					credsBefore, credsAfter, auditBefore, auditAfter)
			}
			assertStatus(t, ctx, sqlDB, postgres, scopeID, issuedRec.CredentialID, "active")
		}

		t.Run("past", func(t *testing.T) {
			assertRotateGraceRejected(t, fixed.Add(-time.Second), "not_future")
		})
		t.Run("equal_now", func(t *testing.T) {
			assertRotateGraceRejected(t, fixed, "not_future")
		})
		t.Run("future_ok", func(t *testing.T) {
			rotated, _ := mustRotate(t, ctx, store, persistence.RotateParams{
				ScopeID: scopeID, CreatedBy: "admin", GraceUntil: fixed.Add(time.Hour),
			})
			assertStatus(t, ctx, sqlDB, postgres, scopeID, issuedRec.CredentialID, "grace")
			assertStatus(t, ctx, sqlDB, postgres, scopeID, rotated.Credential.CredentialID, "active")
		})
	})

	t.Run("audit_failure_rolls_back_mutations", func(t *testing.T) {
		installAuditInsertReject(t, ctx, sqlDB, postgres)
		scopeID := "cred-scope-" + uid + "-rollback"
		mustCreateScope(t, ctx, store, scopeID)

		_, err := store.Issue(ctx, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin", Deliver: func(string) error { return nil }})
		if err == nil {
			t.Fatal("expected issue audit failure")
		}
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scopeID, 0, 0)

		// Seed an active credential without going through Issue (audit blocked).
		seedActiveCredential(t, ctx, sqlDB, postgres, scopeID, "seed-active-"+uid)
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scopeID, 1, 0)

		_, err = store.Rotate(ctx, persistence.RotateParams{
			ScopeID: scopeID, CreatedBy: "admin", GraceUntil: time.Now().UTC().Add(time.Hour),
			Deliver: func(string) error { return nil },
		})
		if err == nil {
			t.Fatal("expected rotate audit failure")
		}
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scopeID, 1, 0)
		assertStatus(t, ctx, sqlDB, postgres, scopeID, "seed-active-"+uid, "active")

		_, err = store.Revoke(ctx, persistence.RevokeParams{
			ScopeID: scopeID, CredentialID: "seed-active-" + uid, ReasonCode: "test",
		})
		if err == nil {
			t.Fatal("expected revoke audit failure")
		}
		assertScopeUnaffected(t, ctx, sqlDB, postgres, scopeID, 1, 0)
		assertStatus(t, ctx, sqlDB, postgres, scopeID, "seed-active-"+uid, "active")
	})

	t.Run("status_revoked_at_grace_until_constraints", func(t *testing.T) {
		scopeID := "cred-scope-" + uid + "-chk"
		mustCreateScope(t, ctx, store, scopeID)
		hash := make([]byte, 32)
		hash[3] = 0x11
		now := timeArg(postgres, time.Now().UTC())
		// revoked without revoked_at
		err := execSQL(ctx, sqlDB, postgres, `
			INSERT INTO `+tblCred(postgres, "api_credentials")+` (
				credential_id, scope_id, token_hash, status, created_at, created_by
			) VALUES (?, ?, ?, 'revoked', ?, 'sql')`,
			"rev-miss-"+uid, scopeID, hash, now,
		)
		if err == nil {
			t.Fatal("expected revoked without revoked_at to fail")
		}
		hash[4] = 0x22
		// grace without grace_until
		err = execSQL(ctx, sqlDB, postgres, `
			INSERT INTO `+tblCred(postgres, "api_credentials")+` (
				credential_id, scope_id, token_hash, status, created_at, created_by
			) VALUES (?, ?, ?, 'grace', ?, 'sql')`,
			"grace-miss-"+uid, scopeID, hash, now,
		)
		if err == nil {
			t.Fatal("expected grace without grace_until to fail")
		}
		hash[5] = 0x33
		// invalid status
		err = execSQL(ctx, sqlDB, postgres, `
			INSERT INTO `+tblCred(postgres, "api_credentials")+` (
				credential_id, scope_id, token_hash, status, created_at, created_by
			) VALUES (?, ?, ?, 'expired', ?, 'sql')`,
			"bad-status-"+uid, scopeID, hash, now,
		)
		if err == nil {
			t.Fatal("expected status expired to fail")
		}
	})

	t.Run("audit_events_immutable", func(t *testing.T) {
		scopeID := "cred-scope-" + uid + "-imm"
		mustCreateScope(t, ctx, store, scopeID)
		_, _ = mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin"})
		var eventID string
		mustScan(t, ctx, sqlDB, postgres,
			`SELECT event_id FROM `+tblCred(postgres, "audit_events")+` WHERE scope_id = ? LIMIT 1`,
			[]any{scopeID}, &eventID,
		)
		err := execSQL(ctx, sqlDB, postgres,
			`UPDATE `+tblCred(postgres, "audit_events")+` SET result = 'tampered' WHERE event_id = ?`, eventID)
		if err == nil {
			t.Fatal("expected audit update to fail")
		}
		err = execSQL(ctx, sqlDB, postgres,
			`DELETE FROM `+tblCred(postgres, "audit_events")+` WHERE event_id = ?`, eventID)
		if err == nil {
			t.Fatal("expected audit delete to fail")
		}
	})

	t.Run("scope_concurrency", func(t *testing.T) {
		if !postgres {
			// SQLite MaxOpenConns=1 serializes writers; still verify sequential rotates.
			scopeID := "cred-scope-" + uid + "-sqconc"
			mustCreateScope(t, ctx, store, scopeID)
			mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin"})
			for i := 0; i < 5; i++ {
				mustRotate(t, ctx, store, persistence.RotateParams{
					ScopeID: scopeID, CreatedBy: "admin", GraceUntil: time.Now().UTC().Add(time.Hour),
				})
			}
			var active, grace int
			mustScan(t, ctx, sqlDB, postgres,
				`SELECT COUNT(*) FROM `+tblCred(postgres, "api_credentials")+` WHERE scope_id = ? AND status = 'active'`,
				[]any{scopeID}, &active,
			)
			mustScan(t, ctx, sqlDB, postgres,
				`SELECT COUNT(*) FROM `+tblCred(postgres, "api_credentials")+` WHERE scope_id = ? AND status = 'grace'`,
				[]any{scopeID}, &grace,
			)
			if active != 1 || grace != 1 {
				t.Fatalf("active=%d grace=%d", active, grace)
			}
			return
		}

		scopeID := "cred-scope-" + uid + "-pgconc"
		mustCreateScope(t, ctx, store, scopeID)
		mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin"})
		var okCount atomic.Int64
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := store.Rotate(ctx, persistence.RotateParams{
					ScopeID: scopeID, CreatedBy: "admin", GraceUntil: time.Now().UTC().Add(time.Hour),
					Deliver: func(string) error { return nil },
				})
				if err == nil {
					okCount.Add(1)
				}
			}()
		}
		wg.Wait()
		if okCount.Load() < 1 {
			t.Fatal("expected at least one successful concurrent rotate")
		}
		var active, grace int
		mustScan(t, ctx, sqlDB, postgres,
			`SELECT COUNT(*) FROM `+tblCred(postgres, "api_credentials")+` WHERE scope_id = ? AND status = 'active'`,
			[]any{scopeID}, &active,
		)
		mustScan(t, ctx, sqlDB, postgres,
			`SELECT COUNT(*) FROM `+tblCred(postgres, "api_credentials")+` WHERE scope_id = ? AND status = 'grace'`,
			[]any{scopeID}, &grace,
		)
		if active != 1 || grace != 1 {
			t.Fatalf("after concurrency active=%d grace=%d ok=%d", active, grace, okCount.Load())
		}
	})

	t.Run("token_persists_sha256_only", func(t *testing.T) {
		scopeID := "cred-scope-" + uid + "-hash"
		mustCreateScope(t, ctx, store, scopeID)
		issuedRec, issuedTok := mustIssue(t, ctx, store, persistence.IssueParams{ScopeID: scopeID, CreatedBy: "admin"})
		want := persistence.HashCredentialToken(issuedTok)
		var got []byte
		mustScan(t, ctx, sqlDB, postgres,
			`SELECT token_hash FROM `+tblCred(postgres, "api_credentials")+` WHERE credential_id = ?`,
			[]any{issuedRec.CredentialID}, &got,
		)
		if len(got) != sha256.Size || string(got) != string(want) {
			t.Fatalf("token_hash mismatch len=%d", len(got))
		}
		var blob string
		row := sqlDB.QueryRowContext(ctx, rebind(postgres, `
			SELECT credential_id || COALESCE(status,'') || COALESCE(created_by,'')
			FROM `+tblCred(postgres, "api_credentials")+` WHERE credential_id = ?`),
			issuedRec.CredentialID,
		)
		if err := row.Scan(&blob); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(blob, issuedTok) || strings.Contains(blob, persistence.CredentialTokenPrefix) {
			t.Fatal("token material leaked into credential text columns")
		}
	})
}

func TestMigrationParityExpectedVersion3(t *testing.T) {
	if dbmigrate.ExpectedVersion != 3 {
		t.Fatalf("ExpectedVersion=%d", dbmigrate.ExpectedVersion)
	}
}

func TestPriorMigrationsImmutableLocally(t *testing.T) {
	root := repoRootCredentials(t)
	base := strings.TrimSpace(os.Getenv("MIGRATION_BASE_SHA"))
	if base == "" {
		base = "origin/main"
	}
	cmd := fmt.Sprintf("MIGRATION_BASE_SHA=%s bash scripts/check-migrations.sh", base)
	out, err := execShell(root, cmd)
	if err != nil {
		t.Fatalf("check-migrations: %v\n%s", err, out)
	}
	if !strings.Contains(out, "migration version parity ok") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func mustCreateScope(t *testing.T, ctx context.Context, store *persistence.CredentialStore, scopeID string) {
	t.Helper()
	_, err := store.CreateScope(ctx, persistence.CreateScopeParams{
		ScopeID:             scopeID,
		TaxpayerNIF:         "5000000000",
		IANATimezone:        "Africa/Luanda",
		SeriesEffectiveCode: "A",
		Environment:         "development",
	})
	if err != nil {
		t.Fatalf("create scope: %v", err)
	}
}

func assertTokenShape(t *testing.T, token string) {
	t.Helper()
	if !strings.HasPrefix(token, persistence.CredentialTokenPrefix) {
		prefix := token
		if len(prefix) > 12 {
			prefix = prefix[:12]
		}
		t.Fatalf("token prefix: %q", prefix)
	}
	rest := strings.TrimPrefix(token, persistence.CredentialTokenPrefix)
	if rest == "" {
		t.Fatal("empty token body")
	}
}

func assertStatus(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scopeID, credID, want string) {
	t.Helper()
	var got string
	mustScan(t, ctx, sqlDB, postgres,
		`SELECT status FROM `+tblCred(postgres, "api_credentials")+` WHERE credential_id = ? AND scope_id = ?`,
		[]any{credID, scopeID}, &got,
	)
	if got != want {
		t.Fatalf("status=%q want %q", got, want)
	}
}

func assertAuditCount(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scopeID, action string, want int) {
	t.Helper()
	var n int
	mustScan(t, ctx, sqlDB, postgres,
		`SELECT COUNT(*) FROM `+tblCred(postgres, "audit_events")+` WHERE scope_id = ? AND action = ?`,
		[]any{scopeID, action}, &n,
	)
	if n != want {
		t.Fatalf("audit %s count=%d want %d", action, n, want)
	}
}

func assertNoSecretInAudit(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scopeID, token string) {
	t.Helper()
	rows, err := sqlDB.QueryContext(ctx, rebind(postgres, `
		SELECT event_id, COALESCE(action,''), COALESCE(result,''), COALESCE(reason_code,''), COALESCE(request_id,'')
		FROM `+tblCred(postgres, "audit_events")+` WHERE scope_id = ?`), scopeID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a, b, c, d, e string
		if err := rows.Scan(&a, &b, &c, &d, &e); err != nil {
			t.Fatal(err)
		}
		joined := a + b + c + d + e
		if strings.Contains(joined, token) || strings.Contains(joined, "5000000000") {
			t.Fatal("audit leaked token or NIF")
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func tblCred(postgres bool, name string) string {
	if postgres {
		return "fiscal." + name
	}
	return name
}

func timeArg(postgres bool, tm time.Time) any {
	tm = tm.UTC().Truncate(time.Microsecond)
	if postgres {
		return tm
	}
	return tm.Format("2006-01-02T15:04:05.000000Z")
}

func execSQL(ctx context.Context, sqlDB *sql.DB, postgres bool, query string, args ...any) error {
	_, err := sqlDB.ExecContext(ctx, rebind(postgres, query), args...)
	return err
}

func mustScan(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, query string, args []any, dest ...any) {
	t.Helper()
	if err := sqlDB.QueryRowContext(ctx, rebind(postgres, query), args...).Scan(dest...); err != nil {
		t.Fatalf("scan: %v", err)
	}
}

func repoRootCredentials(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/persistence -> repo root
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func execShell(dir, cmd string) (string, error) {
	c := exec.Command("bash", "-lc", cmd)
	c.Dir = dir
	out, err := c.CombinedOutput()
	return string(out), err
}

func installAuditInsertReject(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	const triggerName = "audit_events_reject_insert_test"
	if postgres {
		_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+triggerName+` ON fiscal.audit_events`)
		must(t, execSQL(ctx, sqlDB, true, `
			CREATE TRIGGER `+triggerName+`
			BEFORE INSERT ON fiscal.audit_events
			FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation()`))
		t.Cleanup(func() {
			_, _ = sqlDB.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS `+triggerName+` ON fiscal.audit_events`)
		})
		return
	}
	_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+triggerName)
	must(t, execSQL(ctx, sqlDB, false, `
		CREATE TRIGGER `+triggerName+`
		BEFORE INSERT ON audit_events
		BEGIN
			SELECT RAISE(ABORT, 'audit_events insert rejected for test');
		END`))
	t.Cleanup(func() {
		_, _ = sqlDB.ExecContext(context.Background(), `DROP TRIGGER IF EXISTS `+triggerName)
	})
}

func dropAuditInsertReject(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool) {
	t.Helper()
	const triggerName = "audit_events_reject_insert_test"
	if postgres {
		_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+triggerName+` ON fiscal.audit_events`)
		return
	}
	_, _ = sqlDB.ExecContext(ctx, `DROP TRIGGER IF EXISTS `+triggerName)
}

func seedActiveCredential(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scopeID, credID string) {
	t.Helper()
	hash := make([]byte, 32)
	copy(hash, []byte(credID))
	must(t, execSQL(ctx, sqlDB, postgres, `
		INSERT INTO `+tblCred(postgres, "api_credentials")+` (
			credential_id, scope_id, token_hash, status, created_at, created_by
		) VALUES (?, ?, ?, 'active', ?, 'seed')`,
		credID, scopeID, hash, timeArg(postgres, time.Now().UTC()),
	))
}

func assertScopeUnaffected(t *testing.T, ctx context.Context, sqlDB *sql.DB, postgres bool, scopeID string, wantCreds, wantAudit int) {
	t.Helper()
	var creds, audit int
	mustScan(t, ctx, sqlDB, postgres,
		`SELECT COUNT(*) FROM `+tblCred(postgres, "api_credentials")+` WHERE scope_id = ?`,
		[]any{scopeID}, &creds,
	)
	mustScan(t, ctx, sqlDB, postgres,
		`SELECT COUNT(*) FROM `+tblCred(postgres, "audit_events")+` WHERE scope_id = ?`,
		[]any{scopeID}, &audit,
	)
	if creds != wantCreds || audit != wantAudit {
		t.Fatalf("scope %s creds=%d want %d audit=%d want %d", scopeID, creds, wantCreds, audit, wantAudit)
	}
}
