package canonical_test

import (
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

func sampleProjection(series string) canonical.Projection {
	qty, _ := quantity.ParseCanonical("1.5")
	price, _ := money.ParseCanonical("10.50")
	return canonical.Projection{
		SeriesCode: series,
		Intent: canonical.DocumentIntent{
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
		},
	}
}

func TestRequestHashStableAnd32Bytes(t *testing.T) {
	in := sampleProjection("A")
	h1 := canonical.RequestHash(in)
	h2 := canonical.RequestHash(in)
	if len(h1) != canonical.HashSize {
		t.Fatalf("hash len = %d", len(h1))
	}
	if h1 != h2 {
		t.Fatal("hash not stable")
	}
	in.Intent.Lines[0], in.Intent.Lines[1] = in.Intent.Lines[1], in.Intent.Lines[0]
	h3 := canonical.RequestHash(in)
	if h1 != h3 {
		t.Fatal("hash depends on line order")
	}
}

func TestRequestHashIncludesSeriesCode(t *testing.T) {
	a := sampleProjection("A")
	b := sampleProjection("B")
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("series_code must affect request hash")
	}
}

func TestRequestHashDiffersOnSemanticChange(t *testing.T) {
	a := sampleProjection("A")
	b := sampleProjection("A")
	b.Intent.ExternalID = "ext-2"
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("expected different hashes")
	}
}

func TestMaterializeIncludesVersionAndSeries(t *testing.T) {
	m := canonical.Materialize(sampleProjection("SER1"))
	if m[:len(canonical.Version)] != canonical.Version {
		t.Fatalf("materialize missing version prefix: %q", m[:20])
	}
	if !strings.Contains(m, "series_code=SER1\n") {
		t.Fatalf("missing series_code field in materialize")
	}
}

func TestNormalizeIssuedAtUTC(t *testing.T) {
	raw := "2026-07-21T10:00:00+01:00"
	got, err := canonical.NormalizeIssuedAtUTC(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if _, err := canonical.NormalizeIssuedAtUTC("not-a-date"); err == nil {
		t.Fatal("expected invalid issued_at error")
	}
	again, err := canonical.NormalizeIssuedAtUTC(got)
	if err != nil || again != got {
		t.Fatalf("renormalize: %q %v", again, err)
	}
}
