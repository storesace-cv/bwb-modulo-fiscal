// Package canonical produz a projeção versionada usada em request_hash (SHA-256).
package canonical

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

// Version identifies the canonical projection algorithm.
//
// Format of Materialize (canonical_v1), UTF-8:
//
//  1. The literal Version token, then ASCII LF (0x0A).
//  2. A fixed sequence of field pairs. Each pair is two length-value (LV) records:
//     KEY then VALUE. An LV record is decimal ASCII length (no leading zeros except
//     for zero itself), ASCII colon (0x3A), then exactly that many UTF-8 bytes.
//     Values and keys may contain newlines, '=', spaces, or any other UTF-8 bytes;
//     length makes the framing unambiguous (no delimiter scanning of payloads).
//  3. Document field keys, in this exact order:
//     scope_id, external_id, document_type, currency, issued_at, requested_series,
//     series_code, seller_tax_id, seller_name, customer_tax_id, customer_name,
//     lines_count.
//  4. For each line in Intent.Lines order (received order; never sorted by LineID),
//     five field pairs: line.line_id, line.description, line.quantity,
//     line.unit_price, line.tax_code. Quantity and unit_price use FormatCanonical.
//
// RequestHash is SHA-256 over the UTF-8 bytes of Materialize.
const Version = "canonical_v1"

// HashSize is the SHA-256 digest length in bytes.
const HashSize = 32

// Line is one document line in the validated projection.
type Line struct {
	LineID      string
	Description string
	Quantity    quantity.Qty
	UnitPrice   money.Amount
	TaxCode     string
}

// DocumentIntent is the validated semantic projection (not raw JSON).
type DocumentIntent struct {
	ScopeID         string
	ExternalID      string
	DocumentType    string
	Currency        string
	IssuedAtUTC     string // RFC3339Nano UTC (normalized)
	RequestedSeries string // empty if absent
	SellerTaxID     string
	SellerName      string
	CustomerTaxID   string // empty if absent
	CustomerName    string // empty if absent
	Lines           []Line
}

// Projection is the full versioned material for request_hash.
// SeriesCode is the effective numbering series (distinct from RequestedSeries).
type Projection struct {
	SeriesCode string
	Intent     DocumentIntent
}

// NormalizeIssuedAtUTC parses issued_at and returns RFC3339Nano in UTC.
func NormalizeIssuedAtUTC(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("canonical: empty issued_at")
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return "", fmt.Errorf("canonical: invalid issued_at %q: %w", raw, err)
		}
	}
	return t.UTC().Format(time.RFC3339Nano), nil
}

// Materialize builds the versioned canonical byte material for hashing.
// Line order matches Intent.Lines exactly (same order persistence uses for line_no).
func Materialize(p Projection) string {
	var b strings.Builder
	b.WriteString(Version)
	b.WriteByte('\n')

	writeField(&b, "scope_id", p.Intent.ScopeID)
	writeField(&b, "external_id", p.Intent.ExternalID)
	writeField(&b, "document_type", p.Intent.DocumentType)
	writeField(&b, "currency", p.Intent.Currency)
	writeField(&b, "issued_at", p.Intent.IssuedAtUTC)
	writeField(&b, "requested_series", p.Intent.RequestedSeries)
	writeField(&b, "series_code", p.SeriesCode)
	writeField(&b, "seller_tax_id", p.Intent.SellerTaxID)
	writeField(&b, "seller_name", p.Intent.SellerName)
	writeField(&b, "customer_tax_id", p.Intent.CustomerTaxID)
	writeField(&b, "customer_name", p.Intent.CustomerName)
	writeField(&b, "lines_count", strconv.Itoa(len(p.Intent.Lines)))

	for _, ln := range p.Intent.Lines {
		writeField(&b, "line.line_id", ln.LineID)
		writeField(&b, "line.description", ln.Description)
		writeField(&b, "line.quantity", ln.Quantity.FormatCanonical())
		writeField(&b, "line.unit_price", ln.UnitPrice.FormatCanonical())
		writeField(&b, "line.tax_code", ln.TaxCode)
	}
	return b.String()
}

// RequestHash returns the 32-byte SHA-256 of the canonical projection.
func RequestHash(p Projection) [HashSize]byte {
	sum := sha256.Sum256([]byte(Materialize(p)))
	return sum
}

func writeField(b *strings.Builder, key, value string) {
	writeLV(b, key)
	writeLV(b, value)
}

func writeLV(b *strings.Builder, s string) {
	b.WriteString(strconv.Itoa(len(s)))
	b.WriteByte(':')
	b.WriteString(s)
}
