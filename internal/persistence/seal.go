// Package persistence implementa a selagem co-transacional (SealInTx) do vertical slice.
package persistence

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
)

// Dialect identifies the SQL dialect and locking strategy.
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
)

var (
	// ErrIdempotencyConflict is returned when the same key is reused with a different request hash.
	ErrIdempotencyConflict = errors.New("persistence: idempotency key conflict")
	// ErrExternalIDConflict is returned when external_id already belongs to another document.
	ErrExternalIDConflict = errors.New("persistence: external_id conflict")
)

// SealRequest is the validated input for SealInTx.
type SealRequest struct {
	IdempotencyKey string
	SeriesCode     string
	Intent         canonical.DocumentIntent
}

// SealResult is the outcome of a successful seal (new or idempotent replay).
type SealResult struct {
	DocumentID    string
	FiscalSeq     int64
	SeriesCode    string
	ScopeID       string
	ExternalID    string
	SubmissionID  string
	IdempotentHit bool
}

// Store performs fiscal persistence operations.
type Store struct {
	db      *sql.DB
	dialect Dialect
	now     func() time.Time
}

// NewStore creates a Store. now defaults to time.Now UTC.
func NewStore(db *sql.DB, dialect Dialect) *Store {
	return &Store{
		db:      db,
		dialect: dialect,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// SealInTx seals a document in a single transaction: idempotency, series, document, ledger, outbox.
func (s *Store) SealInTx(ctx context.Context, req SealRequest) (*SealResult, error) {
	normalized, err := prepareSealRequest(req)
	if err != nil {
		return nil, err
	}
	hash := canonical.RequestHash(canonical.Projection{
		SeriesCode: normalized.SeriesCode,
		Intent:     normalized.Intent,
	})
	hashBytes := hash[:]

	switch s.dialect {
	case DialectPostgres:
		return s.sealPostgres(ctx, normalized, hashBytes)
	case DialectSQLite:
		return s.sealSQLite(ctx, normalized, hashBytes)
	default:
		return nil, fmt.Errorf("persistence: unknown dialect %q", s.dialect)
	}
}

func prepareSealRequest(req SealRequest) (SealRequest, error) {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return SealRequest{}, fmt.Errorf("persistence: empty idempotency key")
	}
	if strings.TrimSpace(req.SeriesCode) == "" {
		return SealRequest{}, fmt.Errorf("persistence: empty series code")
	}
	if strings.TrimSpace(req.Intent.ScopeID) == "" {
		return SealRequest{}, fmt.Errorf("persistence: empty scope_id")
	}
	if strings.TrimSpace(req.Intent.ExternalID) == "" {
		return SealRequest{}, fmt.Errorf("persistence: empty external_id")
	}
	if req.Intent.Currency != "AOA" {
		return SealRequest{}, fmt.Errorf("persistence: currency must be AOA")
	}
	if req.Intent.DocumentType != "invoice" && req.Intent.DocumentType != "credit_note" {
		return SealRequest{}, fmt.Errorf("persistence: invalid document_type")
	}
	if len(req.Intent.Lines) == 0 {
		return SealRequest{}, fmt.Errorf("persistence: lines required")
	}
	issued, err := canonical.NormalizeIssuedAtUTC(req.Intent.IssuedAtUTC)
	if err != nil {
		return SealRequest{}, fmt.Errorf("persistence: %w", err)
	}
	out := req
	out.SeriesCode = strings.TrimSpace(req.SeriesCode)
	out.Intent.IssuedAtUTC = issued
	return out, nil
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (s *Store) stamp() time.Time {
	return s.now().UTC()
}

// mapExternalIDConflict maps UNIQUE violations on documents(scope_id, external_id) only.
func mapExternalIDConflict(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "documents_scope_external_unique") {
		return ErrExternalIDConflict
	}
	// SQLite: UNIQUE constraint failed: documents.scope_id, documents.external_id
	if strings.Contains(msg, "unique constraint failed") &&
		strings.Contains(msg, "documents") &&
		strings.Contains(msg, "external_id") {
		return ErrExternalIDConflict
	}
	return err
}
