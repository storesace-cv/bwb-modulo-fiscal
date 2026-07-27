package persistence

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// SAFTWorkingQuery filters working-document ledger records (scope/period).
type SAFTWorkingQuery struct {
	ScopeID      string
	IssuedFrom   saftao.Date
	IssuedTo     saftao.Date
	MaxDocuments int
}

// WorkingLedgerSource loads WorkingLedgerRecord views without inventing SAF-T fields.
type WorkingLedgerSource interface {
	ListWorkingForSAFT(ctx context.Context, q SAFTWorkingQuery) ([]saftao.WorkingLedgerRecord, error)
}

// ListWorkingForSAFT documents the schema gap: no working-document tables.
func (s *Store) ListWorkingForSAFT(ctx context.Context, q SAFTWorkingQuery) ([]saftao.WorkingLedgerRecord, error) {
	_ = s
	_ = ctx
	if err := validateSAFTWorkingQuery(q); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: working ledger (GAP-SAFT-WRK-PERSIST: sem tabelas de documentos de trabalho; não inventar schema)", ErrUnsupported)
}

// SyntheticWorkingLedger is an in-memory WorkingLedgerSource for tests/adapters.
type SyntheticWorkingLedger struct {
	Records []saftao.WorkingLedgerRecord
}

// ListWorkingForSAFT implements WorkingLedgerSource.
func (s SyntheticWorkingLedger) ListWorkingForSAFT(ctx context.Context, q SAFTWorkingQuery) ([]saftao.WorkingLedgerRecord, error) {
	_ = ctx
	if err := validateSAFTWorkingQuery(q); err != nil {
		return nil, err
	}
	maxDocs := q.MaxDocuments
	if maxDocs <= 0 {
		maxDocs = saftao.MaxTableEntries
	}
	var out []saftao.WorkingLedgerRecord
	for _, rec := range s.Records {
		if strings.TrimSpace(rec.ScopeID) != q.ScopeID {
			continue
		}
		if rec.WorkAt.IsZero() {
			continue
		}
		d := rec.WorkAt.UTC().Format("2006-01-02")
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

func validateSAFTWorkingQuery(q SAFTWorkingQuery) error {
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
