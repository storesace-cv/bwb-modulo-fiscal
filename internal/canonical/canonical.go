// Package canonical produz a projeção versionada usada em request_hash (SHA-256).
package canonical

import (
	"crypto/sha256"
	"strconv"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

const (
	// VersionV1 is the frozen pre-DEC-TIME-001 algorithm (documented; golden-tested; not used for new seals).
	VersionV1 = "canonical_v1"
	// VersionV2 is the active algorithm (DEC-TIME-001): includes fiscal temporal context.
	VersionV2 = "canonical_v2"
	// Version is the algorithm used by RequestHash / Materialize for new seals.
	Version = VersionV2
	// HashSize is the SHA-256 digest length in bytes.
	HashSize = 32
)

// Format of MaterializeV1 (canonical_v1), UTF-8 — IMMUTABLE:
//
//  1. Literal VersionV1, then ASCII LF (0x0A).
//  2. Field pairs as LV records (decimal length, colon, UTF-8 bytes).
//  3. Keys in order: scope_id, external_id, document_type, currency, issued_at,
//     requested_series, series_code, seller_tax_id, seller_name, customer_tax_id,
//     customer_name, lines_count.
//  4. For each line in received order: line.line_id, line.description, line.quantity,
//     line.unit_price, line.tax_code.
//
// Format of MaterializeV2 (canonical_v2), UTF-8:
//
//  Same as v1 through issued_at, then additionally issued_timezone and
//  issued_offset_minutes (decimal string), then requested_series … lines as in v1.
//  issued_at MUST be UTC truncated to microseconds, formatted as RFC3339 with
//  exactly six fractional digits and Z (e.g. 2026-07-21T09:00:00.000000Z).

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
	ScopeID             string
	ExternalID          string
	DocumentType        string
	Currency            string
	IssuedAtUTC         string // UTC micro canonical string for hashing/persistence
	IssuedTimezone      string // IANA, e.g. Africa/Luanda
	IssuedOffsetMinutes int    // offset used at emission, minutes east of UTC
	RequestedSeries     string // empty if absent
	SellerTaxID         string
	SellerName          string
	CustomerTaxID       string // empty if absent
	CustomerName        string // empty if absent
	Lines               []Line
}

// Projection is the full versioned material for request_hash.
type Projection struct {
	SeriesCode string
	Intent     DocumentIntent
}

// MaterializeV1 builds canonical_v1 material (frozen; for golden / legacy docs only).
func MaterializeV1(p Projection) string {
	return materialize(VersionV1, p, false)
}

// MaterializeV2 builds canonical_v2 material (active).
func MaterializeV2(p Projection) string {
	return materialize(VersionV2, p, true)
}

// Materialize builds the active version material (v2).
func Materialize(p Projection) string {
	return MaterializeV2(p)
}

// RequestHashV1 returns SHA-256 of canonical_v1 material.
func RequestHashV1(p Projection) [HashSize]byte {
	return sha256.Sum256([]byte(MaterializeV1(p)))
}

// RequestHashV2 returns SHA-256 of canonical_v2 material.
func RequestHashV2(p Projection) [HashSize]byte {
	return sha256.Sum256([]byte(MaterializeV2(p)))
}

// RequestHash returns the active (v2) digest.
func RequestHash(p Projection) [HashSize]byte {
	return RequestHashV2(p)
}

func materialize(version string, p Projection, withTemporal bool) string {
	var b strings.Builder
	b.WriteString(version)
	b.WriteByte('\n')

	writeField(&b, "scope_id", p.Intent.ScopeID)
	writeField(&b, "external_id", p.Intent.ExternalID)
	writeField(&b, "document_type", p.Intent.DocumentType)
	writeField(&b, "currency", p.Intent.Currency)
	writeField(&b, "issued_at", p.Intent.IssuedAtUTC)
	if withTemporal {
		writeField(&b, "issued_timezone", p.Intent.IssuedTimezone)
		writeField(&b, "issued_offset_minutes", strconv.Itoa(p.Intent.IssuedOffsetMinutes))
	}
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

func writeField(b *strings.Builder, key, value string) {
	writeLV(b, key)
	writeLV(b, value)
}

func writeLV(b *strings.Builder, s string) {
	b.WriteString(strconv.Itoa(len(s)))
	b.WriteByte(':')
	b.WriteString(s)
}
