package canonical_test

import (
	"bytes"
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
}

func TestRequestHashPreservesLineOrder(t *testing.T) {
	a := sampleProjection("A")
	b := sampleProjection("A")
	b.Intent.Lines[0], b.Intent.Lines[1] = b.Intent.Lines[1], b.Intent.Lines[0]
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("same lines in different order must produce different hashes")
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
	if !strings.HasPrefix(m, canonical.Version+"\n") {
		t.Fatalf("materialize missing version prefix: %q", m[:min(40, len(m))])
	}
	// length-prefixed: 11:series_code4:SER1
	if !strings.Contains(m, "11:series_code4:SER1") {
		t.Fatalf("missing series_code LV field in materialize: %q", m)
	}
}

func TestMaterializeUnambiguousWithSpecialValues(t *testing.T) {
	qty, _ := quantity.ParseCanonical("1")
	price, _ := money.ParseCanonical("1.00")
	base := sampleProjection("A")
	base.Intent.Lines = []canonical.Line{{
		LineID:      "L1",
		Description: "plain",
		Quantity:    qty,
		UnitPrice:   price,
		TaxCode:     "NOR",
	}}

	withNL := base
	withNL.Intent.SellerName = "Seller\nName"
	withEq := base
	withEq.Intent.SellerName = "Seller=Name"
	withUnicode := base
	withUnicode.Intent.SellerName = "Seller café 🇵🇹"
	withEmbedded := base
	withEmbedded.Intent.SellerName = "11:fake_key0:"

	materials := []string{
		canonical.Materialize(base),
		canonical.Materialize(withNL),
		canonical.Materialize(withEq),
		canonical.Materialize(withUnicode),
		canonical.Materialize(withEmbedded),
	}
	for i := 0; i < len(materials); i++ {
		for j := i + 1; j < len(materials); j++ {
			if materials[i] == materials[j] {
				t.Fatalf("materials %d and %d collided", i, j)
			}
		}
	}
	if !bytes.Contains([]byte(materials[1]), []byte("Seller\nName")) {
		t.Fatal("newline value must appear verbatim inside LV payload")
	}
	if !bytes.Contains([]byte(materials[2]), []byte("Seller=Name")) {
		t.Fatal("equals value must appear verbatim inside LV payload")
	}
	if !bytes.Contains([]byte(materials[3]), []byte("Seller café 🇵🇹")) {
		t.Fatal("unicode value must appear verbatim inside LV payload")
	}
}

func TestMaterializeStructurallyDifferentNeverCollide(t *testing.T) {
	a := sampleProjection("A")
	// Split what could look like key=value framing in the old format.
	b := sampleProjection("A")
	b.Intent.ScopeID = "scope"
	b.Intent.ExternalID = "-aext-1" // different structure from scope-a + ext-1

	c := sampleProjection("A")
	c.Intent.SellerName = "X\nlines_count=0\n" // must not inject extra fields

	ma, mb, mc := canonical.Materialize(a), canonical.Materialize(b), canonical.Materialize(c)
	if ma == mb || ma == mc || mb == mc {
		t.Fatal("structurally different projections must not share material")
	}
}

func TestMaterializeAndHashDeterministic(t *testing.T) {
	in := sampleProjection("A")
	in.Intent.SellerName = "line1\nline2=x 漢字"
	m1 := canonical.Materialize(in)
	m2 := canonical.Materialize(in)
	if m1 != m2 {
		t.Fatal("Materialize not byte-identical across runs")
	}
	if canonical.RequestHash(in) != canonical.RequestHash(in) {
		t.Fatal("RequestHash not identical across runs")
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
