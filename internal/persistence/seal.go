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
	// FailBeforeCommit, if non-nil and returning an error, aborts before COMMIT (VS-T07).
	FailBeforeCommit func() error
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
	if err := validateSealRequest(req); err != nil {
		return nil, err
	}
	hash := canonical.RequestHash(req.Intent)
	hashBytes := hash[:]

	switch s.dialect {
	case DialectPostgres:
		return s.sealPostgres(ctx, req, hashBytes)
	case DialectSQLite:
		return s.sealSQLite(ctx, req, hashBytes)
	default:
		return nil, fmt.Errorf("persistence: unknown dialect %q", s.dialect)
	}
}

func validateSealRequest(req SealRequest) error {
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return fmt.Errorf("persistence: empty idempotency key")
	}
	if strings.TrimSpace(req.SeriesCode) == "" {
		return fmt.Errorf("persistence: empty series code")
	}
	if strings.TrimSpace(req.Intent.ScopeID) == "" {
		return fmt.Errorf("persistence: empty scope_id")
	}
	if strings.TrimSpace(req.Intent.ExternalID) == "" {
		return fmt.Errorf("persistence: empty external_id")
	}
	if req.Intent.Currency != "AOA" {
		return fmt.Errorf("persistence: currency must be AOA")
	}
	if req.Intent.DocumentType != "invoice" && req.Intent.DocumentType != "credit_note" {
		return fmt.Errorf("persistence: invalid document_type")
	}
	if len(req.Intent.Lines) == 0 {
		return fmt.Errorf("persistence: lines required")
	}
	return nil
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	// UUID-like hex without inventing fiscal meaning.
	return hex.EncodeToString(b[:]), nil
}

func (s *Store) stamp() time.Time {
	return s.now().UTC()
}

func (s *Store) stampText() string {
	return s.stamp().Format(time.RFC3339Nano)
}
