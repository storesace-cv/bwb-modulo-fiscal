package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// SAFTSalesQuery filters sealed sales documents for SAF-T ledger loading.
// Does not invent Hash/ProductCode/tax rules — enrichment remains caller's responsibility.
type SAFTSalesQuery struct {
	ScopeID string
	// IssuedFrom/IssuedTo are inclusive calendar dates (YYYY-MM-DD) compared on issued_at date part.
	IssuedFrom saftao.Date
	IssuedTo   saftao.Date
	// MaxDocuments fail-closed cap (0 ⇒ saftao.MaxTableEntries).
	MaxDocuments int
}

// ListSealedSalesForSAFT loads documents + lines into saftao.SalesLedgerRecord (deterministic order).
// Hash/SourceID/CustomerID/ProductCode/UnitOfMeasure/TaxPercentage are left empty → MapSalesLedgerToExport omissions.
func (s *Store) ListSealedSalesForSAFT(ctx context.Context, q SAFTSalesQuery) ([]saftao.SalesLedgerRecord, error) {
	if strings.TrimSpace(q.ScopeID) == "" {
		return nil, fmt.Errorf("%w: ScopeID", ErrValidation)
	}
	if err := q.IssuedFrom.Validate(); err != nil {
		return nil, fmt.Errorf("%w: IssuedFrom", ErrValidation)
	}
	if err := q.IssuedTo.Validate(); err != nil {
		return nil, fmt.Errorf("%w: IssuedTo", ErrValidation)
	}
	if string(q.IssuedFrom) > string(q.IssuedTo) {
		return nil, fmt.Errorf("%w: IssuedFrom>IssuedTo", ErrValidation)
	}
	maxDocs := q.MaxDocuments
	if maxDocs <= 0 {
		maxDocs = saftao.MaxTableEntries
	}

	postgres := s.dialect == DialectPostgres
	t := tablePrefix(postgres)
	fromBound := string(q.IssuedFrom) + "T00:00:00Z"
	toBound := string(q.IssuedTo) + "T23:59:59.999999999Z"

	docSQL := `
		SELECT id, scope_id, external_id, document_type, series_code, fiscal_seq,
			issued_at, sealed_at, seller_tax_id, seller_name,
			COALESCE(customer_tax_id, ''), COALESCE(customer_name, '')
		FROM ` + t("documents") + `
		WHERE scope_id = ` + ph(postgres, 1) + `
		  AND issued_at >= ` + ph(postgres, 2) + `
		  AND issued_at <= ` + ph(postgres, 3) + `
		ORDER BY issued_at ASC, series_code ASC, fiscal_seq ASC, id ASC`

	rows, err := s.db.QueryContext(ctx, docSQL, q.ScopeID, fromBound, toBound)
	if err != nil {
		return nil, fmt.Errorf("persistence: list saft docs: %w", err)
	}
	defer rows.Close()

	var out []saftao.SalesLedgerRecord
	for rows.Next() {
		if len(out) >= maxDocs {
			return nil, fmt.Errorf("%w: MaxDocuments excedido", ErrValidation)
		}
		var rec saftao.SalesLedgerRecord
		var issuedAt, sealedAt string
		var externalID string
		if err := rows.Scan(
			&rec.DocumentID, &rec.ScopeID, &externalID, &rec.DocumentType,
			&rec.SeriesCode, &rec.FiscalSeq,
			&issuedAt, &sealedAt, &rec.SellerTaxID, &rec.SellerName,
			&rec.CustomerTaxID, &rec.CustomerName,
		); err != nil {
			return nil, fmt.Errorf("persistence: scan saft doc: %w", err)
		}
		rec.IssuedAt, err = parseStoredTime(issuedAt)
		if err != nil {
			return nil, fmt.Errorf("persistence: issued_at: %w", err)
		}
		rec.SealedAt, err = parseStoredTime(sealedAt)
		if err != nil {
			return nil, fmt.Errorf("persistence: sealed_at: %w", err)
		}
		_ = externalID // available for adapters; not a SAF-T CustomerID
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		lines, err := s.loadSAFTLines(ctx, postgres, t, out[i].DocumentID)
		if err != nil {
			return nil, err
		}
		out[i].Lines = lines
	}
	return out, nil
}

func (s *Store) loadSAFTLines(ctx context.Context, postgres bool, t func(string) string, docID string) ([]saftao.SalesLedgerLine, error) {
	q := `
		SELECT line_no, description, quantity_scaled, unit_price_cents, tax_code
		FROM ` + t("document_lines") + `
		WHERE document_id = ` + ph(postgres, 1) + `
		ORDER BY line_no ASC`
	rows, err := s.db.QueryContext(ctx, q, docID)
	if err != nil {
		return nil, fmt.Errorf("persistence: list saft lines: %w", err)
	}
	defer rows.Close()
	var lines []saftao.SalesLedgerLine
	for rows.Next() {
		if len(lines) >= saftao.MaxLinesPerDocument {
			return nil, fmt.Errorf("%w: MaxLinesPerDocument", ErrValidation)
		}
		var ln saftao.SalesLedgerLine
		if err := rows.Scan(&ln.LineNo, &ln.Description, &ln.QuantityScaled, &ln.UnitPriceCents, &ln.TaxCode); err != nil {
			return nil, fmt.Errorf("persistence: scan saft line: %w", err)
		}
		lines = append(lines, ln)
	}
	return lines, rows.Err()
}

func parseStoredTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if tm, err := time.Parse(layout, s); err == nil {
			return tm.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("instante inválido %q", s)
}

// Ensure sql package available for future ErrNoRows handling without unused import noise.
var _ = sql.ErrNoRows
