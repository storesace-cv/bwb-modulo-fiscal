package persistence

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// ErrUnsupported indicates a capability not backed by the current schema
// (e.g. SAF-T payment ledger tables). Callers should use adapters/fixtures.
var ErrUnsupported = errors.New("persistence: unsupported")

// SAFTPaymentQuery filters payment ledger records for SAF-T loading (scope/period).
type SAFTPaymentQuery struct {
	ScopeID      string
	IssuedFrom   saftao.Date // inclusive calendar date on TransactionAt
	IssuedTo     saftao.Date
	MaxDocuments int
}

// PaymentLedgerSource loads PaymentLedgerRecord views without inventing SAF-T fields.
type PaymentLedgerSource interface {
	ListPaymentsForSAFT(ctx context.Context, q SAFTPaymentQuery) ([]saftao.PaymentLedgerRecord, error)
}

// ListPaymentsForSAFT documents the schema gap: documents table only stores
// invoice|credit_note. No payment/receipt tables exist — do not invent schema here.
// Use SyntheticPaymentLedger (tests/adapters) until a governed migration exists.
func (s *Store) ListPaymentsForSAFT(ctx context.Context, q SAFTPaymentQuery) ([]saftao.PaymentLedgerRecord, error) {
	_ = s
	_ = ctx
	if err := validateSAFTPaymentQuery(q); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("%w: payments ledger (GAP-SAFT-PAY-PERSIST: sem tabelas de recibos; não inventar schema)", ErrUnsupported)
}

// SyntheticPaymentLedger is an in-memory PaymentLedgerSource for tests/adapters.
// Filters by scope/period with fail-closed caps; never invents enrichment fields.
type SyntheticPaymentLedger struct {
	Records []saftao.PaymentLedgerRecord
}

// ListPaymentsForSAFT implements PaymentLedgerSource.
func (s SyntheticPaymentLedger) ListPaymentsForSAFT(ctx context.Context, q SAFTPaymentQuery) ([]saftao.PaymentLedgerRecord, error) {
	_ = ctx
	if err := validateSAFTPaymentQuery(q); err != nil {
		return nil, err
	}
	maxDocs := q.MaxDocuments
	if maxDocs <= 0 {
		maxDocs = saftao.MaxTableEntries
	}
	var out []saftao.PaymentLedgerRecord
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
		if out[i].DocumentID != out[j].DocumentID {
			return out[i].DocumentID < out[j].DocumentID
		}
		return out[i].TransactionAt.Before(out[j].TransactionAt)
	})
	return out, nil
}

func validateSAFTPaymentQuery(q SAFTPaymentQuery) error {
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
