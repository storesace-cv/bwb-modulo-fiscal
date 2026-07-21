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
	// ErrValidation is the sentinel for typed request validation failures.
	ErrValidation = errors.New("persistence: validation failed")
)

// ValidationError is a typed field-level validation failure (not string-matched).
type ValidationError struct {
	Field   string
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "persistence: validation failed"
	}
	return fmt.Sprintf("persistence: validation %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}

func validationErr(field, code, message string) error {
	return &ValidationError{Field: field, Code: code, Message: message}
}

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
	CreatedAt     time.Time // persisted documents.created_at; identical on replay
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

// SetClock injects a clock for tests (created_at determinism).
func (s *Store) SetClock(now func() time.Time) {
	if now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
		return
	}
	s.now = now
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
		return SealRequest{}, validationErr("idempotency_key", "REQUIRED", "obrigatório")
	}
	if strings.TrimSpace(req.SeriesCode) == "" {
		return SealRequest{}, validationErr("series_code", "REQUIRED", "obrigatório")
	}
	if strings.TrimSpace(req.Intent.ScopeID) == "" {
		return SealRequest{}, validationErr("scope_id", "REQUIRED", "obrigatório")
	}
	if !hasNonWhitespace(req.Intent.ExternalID) {
		return SealRequest{}, validationErr("external_id", "REQUIRED", "obrigatório e non-empty")
	}
	if req.Intent.Currency != "AOA" {
		return SealRequest{}, validationErr("currency", "INVALID_ENUM", "deve ser AOA")
	}
	if req.Intent.DocumentType != "invoice" && req.Intent.DocumentType != "credit_note" {
		return SealRequest{}, validationErr("document_type", "INVALID_ENUM", "valor não permitido")
	}
	if len(req.Intent.Lines) == 0 {
		return SealRequest{}, validationErr("lines", "REQUIRED", "pelo menos uma linha")
	}
	if !hasNonWhitespace(req.Intent.SellerTaxID) {
		return SealRequest{}, validationErr("seller.tax_id", "REQUIRED", "obrigatório e non-empty")
	}
	if !hasNonWhitespace(req.Intent.SellerName) {
		return SealRequest{}, validationErr("seller.name", "REQUIRED", "obrigatório e non-empty")
	}
	if !hasNonWhitespace(req.Intent.IssuedTimezone) {
		return SealRequest{}, validationErr("issued_timezone", "REQUIRED", "obrigatório")
	}
	if req.Intent.IssuedOffsetMinutes < -840 || req.Intent.IssuedOffsetMinutes > 840 {
		return SealRequest{}, validationErr("issued_offset_minutes", "OUT_OF_RANGE", "fora de -840..840")
	}
	if !hasNonWhitespace(req.Intent.IssuedAtUTC) {
		return SealRequest{}, validationErr("issued_at", "REQUIRED", "obrigatório")
	}
	// IssuedAtUTC must already be UTC micro from fiscaltime (no silent re-normalization that drops TZ context).
	if _, err := time.Parse("2006-01-02T15:04:05.000000Z", req.Intent.IssuedAtUTC); err != nil {
		// also accept Format without forcing parse layout — try RFC3339Nano then truncate check
		t, err2 := time.Parse(time.RFC3339Nano, req.Intent.IssuedAtUTC)
		if err2 != nil {
			return SealRequest{}, validationErr("issued_at", "INVALID_FORMAT", "UTC micro inválido")
		}
		if t.UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.000000Z") != req.Intent.IssuedAtUTC {
			return SealRequest{}, validationErr("issued_at", "INVALID_FORMAT", "deve ser UTC com microssegundos canónicos")
		}
	}
	for i, ln := range req.Intent.Lines {
		prefix := fmt.Sprintf("lines[%d]", i)
		if !hasNonWhitespace(ln.LineID) {
			return SealRequest{}, validationErr(prefix+".line_id", "REQUIRED", "obrigatório e non-empty")
		}
		if !hasNonWhitespace(ln.Description) {
			return SealRequest{}, validationErr(prefix+".description", "REQUIRED", "obrigatório e non-empty")
		}
		if !hasNonWhitespace(ln.TaxCode) {
			return SealRequest{}, validationErr(prefix+".tax_code", "REQUIRED", "obrigatório e non-empty")
		}
	}
	out := req
	out.SeriesCode = strings.TrimSpace(req.SeriesCode)
	return out, nil
}

func hasNonWhitespace(s string) bool {
	return strings.TrimSpace(s) != ""
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
