// Package canonical produz a projeção versionada usada em request_hash (SHA-256).
package canonical

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

// Version identifies the canonical projection algorithm.
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
	IssuedAtUTC     string // RFC3339Nano UTC
	RequestedSeries string // empty if absent
	SellerTaxID     string
	SellerName      string
	CustomerTaxID   string // empty if absent
	CustomerName    string // empty if absent
	Lines           []Line
}

// Materialize builds the versioned canonical byte material for hashing.
func Materialize(in DocumentIntent) string {
	var b strings.Builder
	b.WriteString(Version)
	b.WriteByte('\n')
	writeField(&b, "scope_id", in.ScopeID)
	writeField(&b, "external_id", in.ExternalID)
	writeField(&b, "document_type", in.DocumentType)
	writeField(&b, "currency", in.Currency)
	writeField(&b, "issued_at", in.IssuedAtUTC)
	writeField(&b, "requested_series", in.RequestedSeries)
	writeField(&b, "seller_tax_id", in.SellerTaxID)
	writeField(&b, "seller_name", in.SellerName)
	writeField(&b, "customer_tax_id", in.CustomerTaxID)
	writeField(&b, "customer_name", in.CustomerName)

	lines := append([]Line(nil), in.Lines...)
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].LineID < lines[j].LineID
	})
	b.WriteString("lines_count=")
	b.WriteString(fmt.Sprintf("%d", len(lines)))
	b.WriteByte('\n')
	for _, ln := range lines {
		writeField(&b, "line.line_id", ln.LineID)
		writeField(&b, "line.description", ln.Description)
		writeField(&b, "line.quantity", ln.Quantity.FormatCanonical())
		writeField(&b, "line.unit_price", ln.UnitPrice.FormatCanonical())
		writeField(&b, "line.tax_code", ln.TaxCode)
	}
	return b.String()
}

// RequestHash returns the 32-byte SHA-256 of the canonical projection.
func RequestHash(in DocumentIntent) [HashSize]byte {
	sum := sha256.Sum256([]byte(Materialize(in)))
	return sum
}

func writeField(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(value)
	b.WriteByte('\n')
}
