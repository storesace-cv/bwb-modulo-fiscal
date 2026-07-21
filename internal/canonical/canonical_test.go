package canonical_test

import (
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

func sampleIntent() canonical.DocumentIntent {
	qty, _ := quantity.ParseCanonical("1.5")
	price, _ := money.ParseCanonical("10.50")
	return canonical.DocumentIntent{
		ScopeID:      "scope-a",
		ExternalID:   "ext-1",
		DocumentType: "invoice",
		Currency:     "AOA",
		IssuedAtUTC:  time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		SellerTaxID:  "5000000000",
		SellerName:   "Seller",
		Lines: []canonical.Line{{
			LineID:      "L2",
			Description: "B",
			Quantity:    qty,
			UnitPrice:   price,
			TaxCode:     "NOR",
		}, {
			LineID:      "L1",
			Description: "A",
			Quantity:    qty,
			UnitPrice:   price,
			TaxCode:     "NOR",
		}},
	}
}

func TestRequestHashStableAnd32Bytes(t *testing.T) {
	in := sampleIntent()
	h1 := canonical.RequestHash(in)
	h2 := canonical.RequestHash(in)
	if len(h1) != canonical.HashSize {
		t.Fatalf("hash len = %d", len(h1))
	}
	if h1 != h2 {
		t.Fatal("hash not stable")
	}
	// Line order in input must not affect hash (sorted by line_id).
	in.Lines[0], in.Lines[1] = in.Lines[1], in.Lines[0]
	h3 := canonical.RequestHash(in)
	if h1 != h3 {
		t.Fatal("hash depends on line order")
	}
}

func TestRequestHashDiffersOnSemanticChange(t *testing.T) {
	a := sampleIntent()
	b := sampleIntent()
	b.ExternalID = "ext-2"
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("expected different hashes")
	}
}

func TestMaterializeIncludesVersion(t *testing.T) {
	m := canonical.Materialize(sampleIntent())
	if m[:len(canonical.Version)] != canonical.Version {
		t.Fatalf("materialize missing version prefix: %q", m[:20])
	}
}
