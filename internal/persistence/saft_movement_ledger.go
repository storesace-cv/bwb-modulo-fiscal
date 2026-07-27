package persistence

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// SAFTMovementQuery filters stock movement ledger records (scope/period).
type SAFTMovementQuery struct {
	ScopeID      string
	IssuedFrom   saftao.Date
	IssuedTo     saftao.Date
	MaxDocuments int
}

// MovementLedgerSource loads MovementLedgerRecord views without inventing SAF-T fields.
type MovementLedgerSource interface {
	ListMovementsForSAFT(ctx context.Context, q SAFTMovementQuery) ([]saftao.MovementLedgerRecord, error)
}

// ListMovementsForSAFT documents the schema gap: no stock-movement tables.
func (s *Store) ListMovementsForSAFT(ctx context.Context, q SAFTMovementQuery) ([]saftao.MovementLedgerRecord, error) {
	_ = s
	_ = ctx
	if err := validateSAFTMovementQuery(q); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: movement ledger (GAP-SAFT-MOV-PERSIST: sem tabelas de movimentos; não inventar schema)", ErrUnsupported)
}

// SyntheticMovementLedger is an in-memory MovementLedgerSource for tests/adapters.
type SyntheticMovementLedger struct {
	Records []saftao.MovementLedgerRecord
}

// ListMovementsForSAFT implements MovementLedgerSource.
func (s SyntheticMovementLedger) ListMovementsForSAFT(ctx context.Context, q SAFTMovementQuery) ([]saftao.MovementLedgerRecord, error) {
	_ = ctx
	if err := validateSAFTMovementQuery(q); err != nil {
		return nil, err
	}
	maxDocs := q.MaxDocuments
	if maxDocs <= 0 {
		maxDocs = saftao.MaxTableEntries
	}
	var out []saftao.MovementLedgerRecord
	for _, rec := range s.Records {
		if strings.TrimSpace(rec.ScopeID) != q.ScopeID {
			continue
		}
		if rec.MovementAt.IsZero() {
			continue
		}
		d := rec.MovementAt.UTC().Format("2006-01-02")
		if d < string(q.IssuedFrom) || d > string(q.IssuedTo) {
			continue
		}
		if len(out) >= maxDocs {
			return nil, fmt.Errorf("%w: MaxDocuments excedido", ErrValidation)
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DocumentID < out[j].DocumentID
	})
	return out, nil
}

func validateSAFTMovementQuery(q SAFTMovementQuery) error {
	if strings.TrimSpace(q.ScopeID) == "" {
		return fmt.Errorf("%w: ScopeID", ErrValidation)
	}
	if err := q.IssuedFrom.Validate(); err != nil {
		return fmt.Errorf("%w: IssuedFrom", ErrValidation)
	}
	if err := q.IssuedTo.Validate(); err != nil {
		return fmt.Errorf("%w: IssuedTo", ErrValidation)
	}
	if string(q.IssuedFrom) > string(q.IssuedTo) {
		return fmt.Errorf("%w: IssuedFrom>IssuedTo", ErrValidation)
	}
	return nil
}
