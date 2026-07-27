package persistence

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// SAFTGLEntriesQuery filters GL entries ledger records (scope/period).
type SAFTGLEntriesQuery struct {
	ScopeID      string
	IssuedFrom   saftao.Date
	IssuedTo     saftao.Date
	MaxDocuments int
}

// GLEntriesLedgerSource loads GLEntriesLedgerRecord views without inventing SAF-T fields.
type GLEntriesLedgerSource interface {
	ListGLEntriesForSAFT(ctx context.Context, q SAFTGLEntriesQuery) ([]saftao.GLEntriesLedgerRecord, error)
}

// ListGLEntriesForSAFT documents the schema gap: no accounting journal tables.
func (s *Store) ListGLEntriesForSAFT(ctx context.Context, q SAFTGLEntriesQuery) ([]saftao.GLEntriesLedgerRecord, error) {
	_ = s
	_ = ctx
	if err := validateSAFTGLEntriesQuery(q); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: gl entries ledger (GAP-SAFT-GLE-PERSIST: sem tabelas de diário contabilístico; não inventar schema)", ErrUnsupported)
}

// SyntheticGLEntriesLedger is an in-memory GLEntriesLedgerSource for tests/adapters.
type SyntheticGLEntriesLedger struct {
	Records []saftao.GLEntriesLedgerRecord
}

// ListGLEntriesForSAFT implements GLEntriesLedgerSource.
func (s SyntheticGLEntriesLedger) ListGLEntriesForSAFT(ctx context.Context, q SAFTGLEntriesQuery) ([]saftao.GLEntriesLedgerRecord, error) {
	_ = ctx
	if err := validateSAFTGLEntriesQuery(q); err != nil {
		return nil, err
	}
	maxDocs := q.MaxDocuments
	if maxDocs <= 0 {
		maxDocs = saftao.MaxTableEntries
	}
	var out []saftao.GLEntriesLedgerRecord
	for _, rec := range s.Records {
		if strings.TrimSpace(rec.ScopeID) != q.ScopeID {
			continue
		}
		if rec.TransactionAt.IsZero() {
			continue
		}
		d := rec.TransactionAt.UTC().Format("2006-01-02")
		if d < string(q.IssuedFrom) || d > string(q.IssuedTo) {
			continue
		}
		if len(out) >= maxDocs {
			return nil, fmt.Errorf("%w: MaxDocuments excedido", ErrValidation)
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].JournalID != out[j].JournalID {
			return out[i].JournalID < out[j].JournalID
		}
		return out[i].DocumentID < out[j].DocumentID
	})
	return out, nil
}

func validateSAFTGLEntriesQuery(q SAFTGLEntriesQuery) error {
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
