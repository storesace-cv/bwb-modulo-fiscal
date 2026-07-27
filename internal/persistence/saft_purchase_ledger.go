package persistence

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// SAFTPurchaseQuery filters purchase ledger records for SAF-T loading (scope/period).
type SAFTPurchaseQuery struct {
	ScopeID      string
	IssuedFrom   saftao.Date
	IssuedTo     saftao.Date
	MaxDocuments int
}

// PurchaseLedgerSource loads PurchaseLedgerRecord views without inventing SAF-T fields.
type PurchaseLedgerSource interface {
	ListPurchasesForSAFT(ctx context.Context, q SAFTPurchaseQuery) ([]saftao.PurchaseLedgerRecord, error)
}

// ListPurchasesForSAFT documents the schema gap: no purchase-invoice tables.
// Use SyntheticPurchaseLedger until a governed migration exists — do not invent schema.
func (s *Store) ListPurchasesForSAFT(ctx context.Context, q SAFTPurchaseQuery) ([]saftao.PurchaseLedgerRecord, error) {
	_ = s
	_ = ctx
	if err := validateSAFTPurchaseQuery(q); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: purchase ledger (GAP-SAFT-PUR-PERSIST: sem tabelas de compras; não inventar schema)", ErrUnsupported)
}

// SyntheticPurchaseLedger is an in-memory PurchaseLedgerSource for tests/adapters.
type SyntheticPurchaseLedger struct {
	Records []saftao.PurchaseLedgerRecord
}

// ListPurchasesForSAFT implements PurchaseLedgerSource.
func (s SyntheticPurchaseLedger) ListPurchasesForSAFT(ctx context.Context, q SAFTPurchaseQuery) ([]saftao.PurchaseLedgerRecord, error) {
	_ = ctx
	if err := validateSAFTPurchaseQuery(q); err != nil {
		return nil, err
	}
	maxDocs := q.MaxDocuments
	if maxDocs <= 0 {
		maxDocs = saftao.MaxTableEntries
	}
	var out []saftao.PurchaseLedgerRecord
	for _, rec := range s.Records {
		if strings.TrimSpace(rec.ScopeID) != q.ScopeID {
			continue
		}
		if rec.IssuedAt.IsZero() {
			continue
		}
		d := rec.IssuedAt.UTC().Format("2006-01-02")
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

func validateSAFTPurchaseQuery(q SAFTPurchaseQuery) error {
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
