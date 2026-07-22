package persistence

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// CredentialTokenPrefix is the stable public prefix of sandbox API tokens.
	CredentialTokenPrefix  = "bwb_sbox_"
	credentialTokenEntropy = 32
	// CredentialTokenExactLen is len(prefix) + RawURLEncoding(32 bytes) without padding.
	CredentialTokenExactLen = len(CredentialTokenPrefix) + 43

	credentialStatusActive  = "active"
	credentialStatusGrace   = "grace"
	credentialStatusRevoked = "revoked"

	scopeStatusActive = "active"

	auditActionIssue  = "credential.issue"
	auditActionRotate = "credential.rotate"
	auditActionRevoke = "credential.revoke"
	auditResultOK     = "success"

	// Auth audit actions (best-effort; outside admin mutation tx).
	AuditActionAuthReject        = "auth.reject"
	AuditActionAuthAccept        = "auth.accept"
	AuditActionAuthScopeMismatch = "auth.scope_mismatch"
	auditResultFailure           = "failure"
)

var (
	// ErrScopeNotFound is returned when the target scope does not exist.
	ErrScopeNotFound = errors.New("persistence: scope not found")
	// ErrCredentialNotFound is returned when the credential is missing in the scope.
	ErrCredentialNotFound = errors.New("persistence: credential not found")
	// ErrNoActiveCredential is returned when rotate requires an active credential.
	ErrNoActiveCredential = errors.New("persistence: no active credential")
	// ErrCredentialConflict is returned on unique active/grace or hash collisions.
	ErrCredentialConflict = errors.New("persistence: credential conflict")
	// ErrInvalidCredentialState is returned for illegal status transitions.
	ErrInvalidCredentialState = errors.New("persistence: invalid credential state")
)

// ScopeRecord is a persisted fiscal scope (sandbox binding unit).
type ScopeRecord struct {
	ScopeID             string
	TaxpayerNIF         string
	IANATimezone        string
	SeriesEffectiveCode string
	Environment         string
	Status              string
	CreatedAt           time.Time
}

// CreateScopeParams creates a scope row.
type CreateScopeParams struct {
	ScopeID             string
	TaxpayerNIF         string
	IANATimezone        string
	SeriesEffectiveCode string
	Environment         string // homologation | development
}

// CredentialRecord is metadata for an API credential (never includes token or hash).
type CredentialRecord struct {
	CredentialID string
	ScopeID      string
	Status       string
	ExpiresAt    *time.Time
	GraceUntil   *time.Time
	RotatedFrom  *string
	RevokedAt    *time.Time
	CreatedAt    time.Time
	CreatedBy    string
}

// TokenSink receives plaintext once inside the admin transaction.
// Error causes full rollback of credential mutation and audit.
type TokenSink func(token string) error

// IssueParams issues a new active credential for a scope.
type IssueParams struct {
	ScopeID   string
	CreatedBy string
	ExpiresAt *time.Time
	RequestID string // optional correlation; never a secret
	Deliver   TokenSink
}

// RotateParams rotates the active credential; previous active becomes grace.
type RotateParams struct {
	ScopeID    string
	CreatedBy  string
	GraceUntil time.Time
	ExpiresAt  *time.Time
	RequestID  string
	Deliver    TokenSink
}

// RotateOutcome is metadata after a successful rotate (no plaintext token).
type RotateOutcome struct {
	Credential CredentialRecord
	PreviousID string
}

// CredentialAuthRecord is safe metadata for authenticators (no hash/token).
type CredentialAuthRecord struct {
	CredentialID        string
	ScopeID             string
	Status              string
	ExpiresAt           *time.Time
	GraceUntil          *time.Time
	RevokedAt           *time.Time
	ScopeStatus         string
	ScopeEnvironment    string
	TaxpayerNIF         string
	IANATimezone        string
	SeriesEffectiveCode string
}

// AuthAuditEvent is a best-effort auth audit row (never includes secrets).
type AuthAuditEvent struct {
	Action       string
	Result       string
	ReasonCode   string
	RequestID    string
	CredentialID string
	ScopeID      string
}

// RevokeParams revokes a credential in a scope (active or grace).
type RevokeParams struct {
	ScopeID      string
	CredentialID string
	RequestID    string
	ReasonCode   string
}

// CredentialStore manages scopes, API credentials and co-transactional audit events.
type CredentialStore struct {
	db      *sql.DB
	dialect Dialect
	now     func() time.Time
}

type auditEvent struct {
	EventID      string
	OccurredAt   time.Time
	CredentialID *string
	ScopeID      *string
	Action       string
	Result       string
	ReasonCode   *string
	RequestID    *string
}

// NewCredentialStore creates a CredentialStore. now defaults to time.Now UTC.
func NewCredentialStore(db *sql.DB, dialect Dialect) *CredentialStore {
	return &CredentialStore{
		db:      db,
		dialect: dialect,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// SetClock injects a clock for tests.
func (s *CredentialStore) SetClock(now func() time.Time) {
	if now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
		return
	}
	s.now = now
}

// CreateScope inserts a scope. Does not write audit (scope bootstrap is separate from credential mutations).
func (s *CredentialStore) CreateScope(ctx context.Context, p CreateScopeParams) (*ScopeRecord, error) {
	p.ScopeID = strings.TrimSpace(p.ScopeID)
	p.TaxpayerNIF = strings.TrimSpace(p.TaxpayerNIF)
	p.IANATimezone = strings.TrimSpace(p.IANATimezone)
	p.SeriesEffectiveCode = strings.TrimSpace(p.SeriesEffectiveCode)
	p.Environment = strings.TrimSpace(p.Environment)
	if p.ScopeID == "" {
		return nil, validationErr("scope_id", "required", "scope_id is required")
	}
	if p.TaxpayerNIF == "" {
		return nil, validationErr("taxpayer_nif", "required", "taxpayer_nif is required")
	}
	if p.IANATimezone == "" {
		return nil, validationErr("iana_timezone", "required", "iana_timezone is required")
	}
	if p.SeriesEffectiveCode == "" {
		return nil, validationErr("series_effective_code", "required", "series_effective_code is required")
	}
	if p.Environment != "homologation" && p.Environment != "development" {
		return nil, validationErr("environment", "invalid", "environment must be homologation or development")
	}

	postgres := s.dialect == DialectPostgres
	t := tablePrefix(postgres)
	now := s.stamp().UTC().Truncate(time.Microsecond)
	var nowArg any = now
	if !postgres {
		nowArg = formatUTCMicro(now)
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+t("scopes")+` (
			scope_id, taxpayer_nif, iana_timezone, series_effective_code,
			environment, status, created_at
		) VALUES (`+placeholders(postgres, 7)+`)`,
		p.ScopeID, p.TaxpayerNIF, p.IANATimezone, p.SeriesEffectiveCode,
		p.Environment, scopeStatusActive, nowArg,
	)
	if err != nil {
		return nil, fmt.Errorf("persistence: create scope: %w", err)
	}
	return &ScopeRecord{
		ScopeID:             p.ScopeID,
		TaxpayerNIF:         p.TaxpayerNIF,
		IANATimezone:        p.IANATimezone,
		SeriesEffectiveCode: p.SeriesEffectiveCode,
		Environment:         p.Environment,
		Status:              scopeStatusActive,
		CreatedAt:           now,
	}, nil
}

// Issue creates an active credential. Token plaintext is delivered only via Deliver inside the tx.
func (s *CredentialStore) Issue(ctx context.Context, p IssueParams) (*CredentialRecord, error) {
	p.ScopeID = strings.TrimSpace(p.ScopeID)
	p.CreatedBy = strings.TrimSpace(p.CreatedBy)
	p.RequestID = strings.TrimSpace(p.RequestID)
	if p.ScopeID == "" {
		return nil, validationErr("scope_id", "required", "scope_id is required")
	}
	if p.CreatedBy == "" {
		return nil, validationErr("created_by", "required", "created_by is required")
	}
	if p.Deliver == nil {
		return nil, validationErr("deliver", "required", "token sink is required")
	}

	credID, err := newID()
	if err != nil {
		return nil, fmt.Errorf("persistence: credential id: %w", err)
	}

	var out *CredentialRecord
	err = s.withScopeTx(ctx, p.ScopeID, func(q querier, postgres bool, now time.Time, nowArg any) error {
		activeID, err := s.findStatusID(ctx, q, postgres, p.ScopeID, credentialStatusActive)
		if err != nil {
			return err
		}
		if activeID != "" {
			return ErrCredentialConflict
		}

		token, hash, err := generateCredentialToken()
		if err != nil {
			return err
		}
		rec, err := s.insertCredential(ctx, q, postgres, insertCredentialArgs{
			CredentialID: credID,
			ScopeID:      p.ScopeID,
			TokenHash:    hash,
			Status:       credentialStatusActive,
			ExpiresAt:    p.ExpiresAt,
			CreatedAt:    now,
			CreatedBy:    p.CreatedBy,
			NowArg:       nowArg,
		})
		if err != nil {
			return err
		}
		if err := p.Deliver(token); err != nil {
			return fmt.Errorf("persistence: token deliver: %w", err)
		}
		if err := s.writeAudit(ctx, q, postgres, auditEvent{
			OccurredAt:   now,
			CredentialID: &credID,
			ScopeID:      &p.ScopeID,
			Action:       auditActionIssue,
			Result:       auditResultOK,
			RequestID:    nullStr(p.RequestID),
		}, nowArg); err != nil {
			return err
		}
		out = rec
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Rotate issues a new active credential and moves the previous active to grace.
// Token plaintext is delivered only via Deliver inside the tx.
func (s *CredentialStore) Rotate(ctx context.Context, p RotateParams) (*RotateOutcome, error) {
	p.ScopeID = strings.TrimSpace(p.ScopeID)
	p.CreatedBy = strings.TrimSpace(p.CreatedBy)
	p.RequestID = strings.TrimSpace(p.RequestID)
	if p.ScopeID == "" {
		return nil, validationErr("scope_id", "required", "scope_id is required")
	}
	if p.CreatedBy == "" {
		return nil, validationErr("created_by", "required", "created_by is required")
	}
	if p.Deliver == nil {
		return nil, validationErr("deliver", "required", "token sink is required")
	}
	now := s.stamp().UTC().Truncate(time.Microsecond)
	graceUntil := p.GraceUntil.UTC().Truncate(time.Microsecond)
	if graceUntil.IsZero() {
		return nil, validationErr("grace_until", "required", "grace_until is required")
	}
	if !graceUntil.After(now) {
		return nil, validationErr("grace_until", "not_future", "grace_until must be in the future")
	}

	credID, err := newID()
	if err != nil {
		return nil, fmt.Errorf("persistence: credential id: %w", err)
	}

	var out *RotateOutcome
	err = s.withScopeTx(ctx, p.ScopeID, func(q querier, postgres bool, now time.Time, nowArg any) error {
		t := tablePrefix(postgres)
		activeID, err := s.findStatusID(ctx, q, postgres, p.ScopeID, credentialStatusActive)
		if err != nil {
			return err
		}
		if activeID == "" {
			return ErrNoActiveCredential
		}

		prevGrace, err := s.findStatusID(ctx, q, postgres, p.ScopeID, credentialStatusGrace)
		if err != nil {
			return err
		}
		if prevGrace != "" {
			if err := s.markRevoked(ctx, q, postgres, p.ScopeID, prevGrace, nowArg); err != nil {
				return err
			}
		}

		var graceUntilArg any = graceUntil
		if !postgres {
			graceUntilArg = formatUTCMicro(graceUntil)
		}
		_, err = q.ExecContext(ctx, `
			UPDATE `+t("api_credentials")+` SET
				status = `+ph(postgres, 1)+`,
				grace_until = `+ph(postgres, 2)+`,
				revoked_at = NULL
			WHERE credential_id = `+ph(postgres, 3)+`
			  AND scope_id = `+ph(postgres, 4)+`
			  AND status = `+ph(postgres, 5),
			credentialStatusGrace, graceUntilArg, activeID, p.ScopeID, credentialStatusActive,
		)
		if err != nil {
			return fmt.Errorf("persistence: demote active to grace: %w", err)
		}

		token, hash, err := generateCredentialToken()
		if err != nil {
			return err
		}
		rec, err := s.insertCredential(ctx, q, postgres, insertCredentialArgs{
			CredentialID: credID,
			ScopeID:      p.ScopeID,
			TokenHash:    hash,
			Status:       credentialStatusActive,
			ExpiresAt:    p.ExpiresAt,
			RotatedFrom:  &activeID,
			CreatedAt:    now,
			CreatedBy:    p.CreatedBy,
			NowArg:       nowArg,
		})
		if err != nil {
			return err
		}
		if err := p.Deliver(token); err != nil {
			return fmt.Errorf("persistence: token deliver: %w", err)
		}
		if err := s.writeAudit(ctx, q, postgres, auditEvent{
			OccurredAt:   now,
			CredentialID: &credID,
			ScopeID:      &p.ScopeID,
			Action:       auditActionRotate,
			Result:       auditResultOK,
			RequestID:    nullStr(p.RequestID),
		}, nowArg); err != nil {
			return err
		}
		out = &RotateOutcome{Credential: *rec, PreviousID: activeID}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Revoke marks a credential revoked in its scope.
func (s *CredentialStore) Revoke(ctx context.Context, p RevokeParams) (*CredentialRecord, error) {
	p.ScopeID = strings.TrimSpace(p.ScopeID)
	p.CredentialID = strings.TrimSpace(p.CredentialID)
	p.RequestID = strings.TrimSpace(p.RequestID)
	p.ReasonCode = strings.TrimSpace(p.ReasonCode)
	if p.ScopeID == "" {
		return nil, validationErr("scope_id", "required", "scope_id is required")
	}
	if p.CredentialID == "" {
		return nil, validationErr("credential_id", "required", "credential_id is required")
	}

	var out *CredentialRecord
	err := s.withScopeTx(ctx, p.ScopeID, func(q querier, postgres bool, now time.Time, nowArg any) error {
		status, err := s.credentialStatus(ctx, q, postgres, p.ScopeID, p.CredentialID)
		if err != nil {
			return err
		}
		if status == "" {
			return ErrCredentialNotFound
		}
		if status == credentialStatusRevoked {
			return ErrInvalidCredentialState
		}
		if err := s.markRevoked(ctx, q, postgres, p.ScopeID, p.CredentialID, nowArg); err != nil {
			return err
		}
		if err := s.writeAudit(ctx, q, postgres, auditEvent{
			OccurredAt:   now,
			CredentialID: &p.CredentialID,
			ScopeID:      &p.ScopeID,
			Action:       auditActionRevoke,
			Result:       auditResultOK,
			ReasonCode:   nullStr(p.ReasonCode),
			RequestID:    nullStr(p.RequestID),
		}, nowArg); err != nil {
			return err
		}
		rec, err := s.getCredential(ctx, q, postgres, p.ScopeID, p.CredentialID)
		if err != nil {
			return err
		}
		out = rec
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetCredential returns credential metadata (no token/hash).
func (s *CredentialStore) GetCredential(ctx context.Context, scopeID, credentialID string) (*CredentialRecord, error) {
	postgres := s.dialect == DialectPostgres
	return s.getCredential(ctx, &dbQuerier{db: s.db}, postgres, scopeID, credentialID)
}

// GetScope returns a scope row.
func (s *CredentialStore) GetScope(ctx context.Context, scopeID string) (*ScopeRecord, error) {
	scopeID = strings.TrimSpace(scopeID)
	if scopeID == "" {
		return nil, ErrScopeNotFound
	}
	postgres := s.dialect == DialectPostgres
	t := tablePrefix(postgres)
	var (
		nif, tz, series, env, status string
		createdAt                    time.Time
	)
	if postgres {
		err := s.db.QueryRowContext(ctx, `
			SELECT taxpayer_nif, iana_timezone, series_effective_code, environment, status, created_at
			FROM `+t("scopes")+` WHERE scope_id = $1`, scopeID,
		).Scan(&nif, &tz, &series, &env, &status, &createdAt)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScopeNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("persistence: get scope: %w", err)
		}
	} else {
		var createdAtStr string
		err := s.db.QueryRowContext(ctx, `
			SELECT taxpayer_nif, iana_timezone, series_effective_code, environment, status, created_at
			FROM `+t("scopes")+` WHERE scope_id = ?`, scopeID,
		).Scan(&nif, &tz, &series, &env, &status, &createdAtStr)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrScopeNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("persistence: get scope: %w", err)
		}
		createdAt, err = parseUTCMicro(createdAtStr)
		if err != nil {
			return nil, err
		}
	}
	return &ScopeRecord{
		ScopeID:             scopeID,
		TaxpayerNIF:         nif,
		IANATimezone:        tz,
		SeriesEffectiveCode: series,
		Environment:         env,
		Status:              status,
		CreatedAt:           createdAt.UTC(),
	}, nil
}

// VerifyCredentialTokenHash looks up by hash equality, compares with subtle.ConstantTimeCompare
// inside this package, and returns a safe record without hash/token.
func (s *CredentialStore) VerifyCredentialTokenHash(ctx context.Context, computedHash []byte) (*CredentialAuthRecord, error) {
	if len(computedHash) != sha256.Size {
		return nil, ErrCredentialNotFound
	}
	postgres := s.dialect == DialectPostgres
	t := tablePrefix(postgres)

	var (
		storedHash                                     []byte
		credID, scopeID, status, scopeStatus, scopeEnv string
		nif, tz, series                                string
		expiresAt, graceUntil, revokedAt               sql.NullTime
	)

	if postgres {
		err := s.db.QueryRowContext(ctx, `
			SELECT c.token_hash, c.credential_id, c.scope_id, c.status,
			       c.expires_at, c.grace_until, c.revoked_at,
			       s.status, s.environment, s.taxpayer_nif, s.iana_timezone, s.series_effective_code
			FROM `+t("api_credentials")+` c
			INNER JOIN `+t("scopes")+` s ON s.scope_id = c.scope_id
			WHERE c.token_hash = $1`, computedHash,
		).Scan(&storedHash, &credID, &scopeID, &status,
			&expiresAt, &graceUntil, &revokedAt,
			&scopeStatus, &scopeEnv, &nif, &tz, &series)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("persistence: verify credential: %w", err)
		}
	} else {
		var expiresStr, graceStr, revokedStr sql.NullString
		err := s.db.QueryRowContext(ctx, `
			SELECT c.token_hash, c.credential_id, c.scope_id, c.status,
			       c.expires_at, c.grace_until, c.revoked_at,
			       s.status, s.environment, s.taxpayer_nif, s.iana_timezone, s.series_effective_code
			FROM `+t("api_credentials")+` c
			INNER JOIN `+t("scopes")+` s ON s.scope_id = c.scope_id
			WHERE c.token_hash = ?`, computedHash,
		).Scan(&storedHash, &credID, &scopeID, &status,
			&expiresStr, &graceStr, &revokedStr,
			&scopeStatus, &scopeEnv, &nif, &tz, &series)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("persistence: verify credential: %w", err)
		}
		if expiresStr.Valid {
			tm, err := parseUTCMicro(expiresStr.String)
			if err != nil {
				return nil, fmt.Errorf("persistence: verify credential: %w", err)
			}
			expiresAt = sql.NullTime{Time: tm, Valid: true}
		}
		if graceStr.Valid {
			tm, err := parseUTCMicro(graceStr.String)
			if err != nil {
				return nil, fmt.Errorf("persistence: verify credential: %w", err)
			}
			graceUntil = sql.NullTime{Time: tm, Valid: true}
		}
		if revokedStr.Valid {
			tm, err := parseUTCMicro(revokedStr.String)
			if err != nil {
				return nil, fmt.Errorf("persistence: verify credential: %w", err)
			}
			revokedAt = sql.NullTime{Time: tm, Valid: true}
		}
	}

	if len(storedHash) != sha256.Size || subtle.ConstantTimeCompare(storedHash, computedHash) != 1 {
		// Constant-time path after a row was found; treat as not found to callers.
		return nil, ErrCredentialNotFound
	}

	rec := &CredentialAuthRecord{
		CredentialID:        credID,
		ScopeID:             scopeID,
		Status:              status,
		ScopeStatus:         scopeStatus,
		ScopeEnvironment:    scopeEnv,
		TaxpayerNIF:         nif,
		IANATimezone:        tz,
		SeriesEffectiveCode: series,
	}
	if expiresAt.Valid {
		tm := expiresAt.Time.UTC()
		rec.ExpiresAt = &tm
	}
	if graceUntil.Valid {
		tm := graceUntil.Time.UTC()
		rec.GraceUntil = &tm
	}
	if revokedAt.Valid {
		tm := revokedAt.Time.UTC()
		rec.RevokedAt = &tm
	}
	return rec, nil
}

// RecordAuthAudit inserts an auth audit event best-effort (never logs secrets).
func (s *CredentialStore) RecordAuthAudit(ctx context.Context, ev AuthAuditEvent) error {
	postgres := s.dialect == DialectPostgres
	now := s.stamp().UTC().Truncate(time.Microsecond)
	var nowArg any = now
	if !postgres {
		nowArg = formatUTCMicro(now)
	}
	result := strings.TrimSpace(ev.Result)
	if result == "" {
		result = auditResultOK
	}
	return s.writeAudit(ctx, &dbQuerier{db: s.db}, postgres, auditEvent{
		OccurredAt:   now,
		CredentialID: nullStr(strings.TrimSpace(ev.CredentialID)),
		ScopeID:      nullStr(strings.TrimSpace(ev.ScopeID)),
		Action:       strings.TrimSpace(ev.Action),
		Result:       result,
		ReasonCode:   nullStr(strings.TrimSpace(ev.ReasonCode)),
		RequestID:    nullStr(strings.TrimSpace(ev.RequestID)),
	}, nowArg)
}

type dbQuerier struct {
	db *sql.DB
}

func (q *dbQuerier) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return q.db.ExecContext(ctx, query, args...)
}

func (q *dbQuerier) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return q.db.QueryRowContext(ctx, query, args...)
}

func (s *CredentialStore) stamp() time.Time {
	return s.now().UTC()
}

type scopeTxFn func(q querier, postgres bool, now time.Time, nowArg any) error

func (s *CredentialStore) withScopeTx(ctx context.Context, scopeID string, fn scopeTxFn) error {
	if s.dialect == DialectPostgres {
		return s.withScopeTxPostgres(ctx, scopeID, fn)
	}
	return s.withScopeTxSQLite(ctx, scopeID, fn)
}

func (s *CredentialStore) withScopeTxPostgres(ctx context.Context, scopeID string, fn scopeTxFn) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persistence: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	q := &pgQuerier{tx: tx}
	var locked string
	err = q.QueryRowContext(ctx,
		`SELECT scope_id FROM fiscal.scopes WHERE scope_id = $1 FOR UPDATE`,
		scopeID,
	).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrScopeNotFound
	}
	if err != nil {
		return fmt.Errorf("persistence: lock scope: %w", err)
	}

	now := s.stamp().UTC().Truncate(time.Microsecond)
	if err := fn(q, true, now, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persistence: commit: %w", err)
	}
	committed = true
	return nil
}

func (s *CredentialStore) withScopeTxSQLite(ctx context.Context, scopeID string, fn scopeTxFn) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("persistence: conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("persistence: BEGIN IMMEDIATE: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	q := &sqliteQuerier{conn: conn}
	var locked string
	err = q.QueryRowContext(ctx,
		`SELECT scope_id FROM scopes WHERE scope_id = ?`,
		scopeID,
	).Scan(&locked)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrScopeNotFound
	}
	if err != nil {
		return fmt.Errorf("persistence: lock scope: %w", err)
	}

	now := s.stamp().UTC().Truncate(time.Microsecond)
	nowArg := formatUTCMicro(now)
	if err := fn(q, false, now, nowArg); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("persistence: commit: %w", err)
	}
	committed = true
	return nil
}

type insertCredentialArgs struct {
	CredentialID string
	ScopeID      string
	TokenHash    []byte
	Status       string
	ExpiresAt    *time.Time
	RotatedFrom  *string
	CreatedAt    time.Time
	CreatedBy    string
	NowArg       any
}

func (s *CredentialStore) insertCredential(ctx context.Context, q querier, postgres bool, a insertCredentialArgs) (*CredentialRecord, error) {
	t := tablePrefix(postgres)
	var expires any
	if a.ExpiresAt != nil {
		exp := a.ExpiresAt.UTC().Truncate(time.Microsecond)
		if postgres {
			expires = exp
		} else {
			expires = formatUTCMicro(exp)
		}
	}
	var rotated any
	if a.RotatedFrom != nil {
		rotated = *a.RotatedFrom
	}

	_, err := q.ExecContext(ctx, `
		INSERT INTO `+t("api_credentials")+` (
			credential_id, scope_id, token_hash, status,
			expires_at, grace_until, rotated_from, revoked_at,
			created_at, created_by
		) VALUES (`+placeholders(postgres, 10)+`)`,
		a.CredentialID, a.ScopeID, a.TokenHash, a.Status,
		expires, nil, rotated, nil,
		a.NowArg, a.CreatedBy,
	)
	if err != nil {
		if mapped := mapCredentialConflict(err); mapped != nil {
			return nil, mapped
		}
		return nil, fmt.Errorf("persistence: insert credential: %w", err)
	}
	return &CredentialRecord{
		CredentialID: a.CredentialID,
		ScopeID:      a.ScopeID,
		Status:       a.Status,
		ExpiresAt:    a.ExpiresAt,
		RotatedFrom:  a.RotatedFrom,
		CreatedAt:    a.CreatedAt,
		CreatedBy:    a.CreatedBy,
	}, nil
}

func (s *CredentialStore) markRevoked(ctx context.Context, q querier, postgres bool, scopeID, credentialID string, nowArg any) error {
	t := tablePrefix(postgres)
	res, err := q.ExecContext(ctx, `
		UPDATE `+t("api_credentials")+` SET
			status = `+ph(postgres, 1)+`,
			revoked_at = `+ph(postgres, 2)+`,
			grace_until = NULL
		WHERE credential_id = `+ph(postgres, 3)+`
		  AND scope_id = `+ph(postgres, 4)+`
		  AND status <> `+ph(postgres, 5),
		credentialStatusRevoked, nowArg, credentialID, scopeID, credentialStatusRevoked,
	)
	if err != nil {
		return fmt.Errorf("persistence: revoke credential: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("persistence: revoke rows: %w", err)
	}
	if n == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

func (s *CredentialStore) writeAudit(ctx context.Context, q querier, postgres bool, ev auditEvent, nowArg any) error {
	eventID, err := newID()
	if err != nil {
		return fmt.Errorf("persistence: audit id: %w", err)
	}
	ev.EventID = eventID
	t := tablePrefix(postgres)
	var occurred any = nowArg
	if postgres {
		occurred = ev.OccurredAt
	}
	_, err = q.ExecContext(ctx, `
		INSERT INTO `+t("audit_events")+` (
			event_id, occurred_at, credential_id, scope_id,
			action, result, reason_code, request_id
		) VALUES (`+placeholders(postgres, 8)+`)`,
		ev.EventID, occurred, nullStrPtr(ev.CredentialID), nullStrPtr(ev.ScopeID),
		ev.Action, ev.Result, nullStrPtr(ev.ReasonCode), nullStrPtr(ev.RequestID),
	)
	if err != nil {
		return fmt.Errorf("persistence: insert audit: %w", err)
	}
	return nil
}

func (s *CredentialStore) findStatusID(ctx context.Context, q querier, postgres bool, scopeID, status string) (string, error) {
	t := tablePrefix(postgres)
	var id string
	err := q.QueryRowContext(ctx,
		`SELECT credential_id FROM `+t("api_credentials")+`
		 WHERE scope_id = `+ph(postgres, 1)+` AND status = `+ph(postgres, 2),
		scopeID, status,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("persistence: find credential by status: %w", err)
	}
	return id, nil
}

func (s *CredentialStore) credentialStatus(ctx context.Context, q querier, postgres bool, scopeID, credentialID string) (string, error) {
	t := tablePrefix(postgres)
	var status string
	err := q.QueryRowContext(ctx,
		`SELECT status FROM `+t("api_credentials")+`
		 WHERE credential_id = `+ph(postgres, 1)+` AND scope_id = `+ph(postgres, 2),
		credentialID, scopeID,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("persistence: credential status: %w", err)
	}
	return status, nil
}

func (s *CredentialStore) getCredential(ctx context.Context, q querier, postgres bool, scopeID, credentialID string) (*CredentialRecord, error) {
	t := tablePrefix(postgres)
	var (
		status, createdBy string
		createdAt         time.Time
		expiresAt         sql.NullTime
		graceUntil        sql.NullTime
		revokedAt         sql.NullTime
		rotatedFrom       sql.NullString
	)

	if postgres {
		err := q.QueryRowContext(ctx, `
			SELECT status, expires_at, grace_until, rotated_from, revoked_at, created_at, created_by
			FROM `+t("api_credentials")+`
			WHERE credential_id = $1 AND scope_id = $2`,
			credentialID, scopeID,
		).Scan(&status, &expiresAt, &graceUntil, &rotatedFrom, &revokedAt, &createdAt, &createdBy)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("persistence: get credential: %w", err)
		}
	} else {
		var createdAtStr string
		var expiresStr, graceStr, revokedStr sql.NullString
		err := q.QueryRowContext(ctx, `
			SELECT status, expires_at, grace_until, rotated_from, revoked_at, created_at, created_by
			FROM `+t("api_credentials")+`
			WHERE credential_id = ? AND scope_id = ?`,
			credentialID, scopeID,
		).Scan(&status, &expiresStr, &graceStr, &rotatedFrom, &revokedStr, &createdAtStr, &createdBy)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredentialNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("persistence: get credential: %w", err)
		}
		createdAt, err = parseUTCMicro(createdAtStr)
		if err != nil {
			return nil, err
		}
		if expiresStr.Valid {
			t, err := parseUTCMicro(expiresStr.String)
			if err != nil {
				return nil, err
			}
			expiresAt = sql.NullTime{Time: t, Valid: true}
		}
		if graceStr.Valid {
			t, err := parseUTCMicro(graceStr.String)
			if err != nil {
				return nil, err
			}
			graceUntil = sql.NullTime{Time: t, Valid: true}
		}
		if revokedStr.Valid {
			t, err := parseUTCMicro(revokedStr.String)
			if err != nil {
				return nil, err
			}
			revokedAt = sql.NullTime{Time: t, Valid: true}
		}
	}

	rec := &CredentialRecord{
		CredentialID: credentialID,
		ScopeID:      scopeID,
		Status:       status,
		CreatedAt:    createdAt.UTC(),
		CreatedBy:    createdBy,
	}
	if expiresAt.Valid {
		t := expiresAt.Time.UTC()
		rec.ExpiresAt = &t
	}
	if graceUntil.Valid {
		t := graceUntil.Time.UTC()
		rec.GraceUntil = &t
	}
	if revokedAt.Valid {
		t := revokedAt.Time.UTC()
		rec.RevokedAt = &t
	}
	if rotatedFrom.Valid {
		v := rotatedFrom.String
		rec.RotatedFrom = &v
	}
	return rec, nil
}

func generateCredentialToken() (plaintext string, hash []byte, err error) {
	var raw [credentialTokenEntropy]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", nil, fmt.Errorf("persistence: token entropy: %w", err)
	}
	token := CredentialTokenPrefix + base64.RawURLEncoding.EncodeToString(raw[:])
	sum := sha256.Sum256([]byte(token))
	out := make([]byte, sha256.Size)
	copy(out, sum[:])
	return token, out, nil
}

func mapCredentialConflict(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "api_credentials_one_active_per_scope") ||
		strings.Contains(msg, "api_credentials_one_grace_per_scope") ||
		strings.Contains(msg, "api_credentials_token_hash_unique") {
		return ErrCredentialConflict
	}
	if strings.Contains(msg, "unique") && strings.Contains(msg, "api_credentials") {
		return ErrCredentialConflict
	}
	return nil
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullStrPtr(p *string) any {
	if p == nil || *p == "" {
		return nil
	}
	return *p
}

func parseUTCMicro(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", s)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return time.Time{}, fmt.Errorf("persistence: parse time: %w", err)
		}
	}
	return t.UTC(), nil
}

// HashCredentialToken returns SHA-256 of the full token (for auth lookup; S2+).
func HashCredentialToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	out := make([]byte, sha256.Size)
	copy(out, sum[:])
	return out
}
