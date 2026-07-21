package canonical_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/canonical"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaltime"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
)

// Immutable golden input (DEC-TIME-001). Changing this vector or MaterializeV1 breaks certification baselines.
func goldenIntentBase() canonical.DocumentIntent {
	qty, _ := quantity.ParseCanonical("1.5")
	price, _ := money.ParseCanonical("10.50")
	return canonical.DocumentIntent{
		ScopeID:      "scope-a",
		ExternalID:   "ext-1",
		DocumentType: "invoice",
		Currency:     "AOA",
		SellerTaxID:  "5000000000",
		SellerName:   "Seller",
		Lines: []canonical.Line{{
			LineID: "L2", Description: "B", Quantity: qty, UnitPrice: price, TaxCode: "NOR",
		}, {
			LineID: "L1", Description: "A", Quantity: qty, UnitPrice: price, TaxCode: "NOR",
		}},
	}
}

func sampleProjectionV2(series string) canonical.Projection {
	in := goldenIntentBase()
	in.IssuedAtUTC = "2026-07-21T09:00:00.000000Z"
	in.IssuedTimezone = fiscaltime.AfricaLuanda
	in.IssuedOffsetMinutes = 60
	return canonical.Projection{SeriesCode: series, Intent: in}
}

func TestGoldenCanonicalV1Immutable(t *testing.T) {
	in := goldenIntentBase()
	// Historical v1 issued_at string (RFC3339Nano without trailing fractional zeros).
	in.IssuedAtUTC = "2026-07-21T01:00:00Z"
	p := canonical.Projection{SeriesCode: "A", Intent: in}

	wantMaterialPrefix := "canonical_v1\n"
	got := canonical.MaterializeV1(p)
	if !strings.HasPrefix(got, wantMaterialPrefix) {
		t.Fatalf("v1 material prefix = %q", got[:min(20, len(got))])
	}
	if strings.Contains(got, "issued_timezone") || strings.Contains(got, "issued_offset_minutes") {
		t.Fatal("canonical_v1 must not include fiscal timezone fields")
	}

	sum := sha256.Sum256([]byte(got))
	wantHash := "951cf245f794182e5d32012190f66b9e89aa0ac818f5c5fb6a59c2169e885d75"
	if hex.EncodeToString(sum[:]) != wantHash {
		t.Fatalf("canonical_v1 SHA-256 changed:\n got %s\nwant %s\nmaterial=%q", hex.EncodeToString(sum[:]), wantHash, got)
	}
	if canonical.RequestHashV1(p) != sum {
		t.Fatal("RequestHashV1 mismatch")
	}
	// Active Version must not alter the frozen v1 digest.
	if canonical.Version != canonical.VersionV2 {
		t.Fatalf("active Version = %q", canonical.Version)
	}
}

func TestGoldenCanonicalV2(t *testing.T) {
	p := sampleProjectionV2("A")
	got := canonical.MaterializeV2(p)
	if !strings.HasPrefix(got, "canonical_v2\n") {
		t.Fatalf("prefix: %q", got[:min(20, len(got))])
	}
	if !strings.Contains(got, "15:issued_timezone13:Africa/Luanda") {
		t.Fatalf("missing timezone LV: %q", got)
	}
	if !strings.Contains(got, "21:issued_offset_minutes2:60") {
		t.Fatalf("missing offset LV: %q", got)
	}
	sum := sha256.Sum256([]byte(got))
	wantHash := "d1df2d828af48dfbdc6a55002f708afe5feab819294d16a54719575a28c85d9d"
	if hex.EncodeToString(sum[:]) != wantHash {
		t.Fatalf("canonical_v2 SHA-256:\n got %s\nwant %s\nmaterial=%q", hex.EncodeToString(sum[:]), wantHash, got)
	}
	if canonical.RequestHash(p) != sum || canonical.RequestHashV2(p) != sum {
		t.Fatal("active RequestHash must equal RequestHashV2 golden")
	}
}

func TestRequestHashStableAnd32Bytes(t *testing.T) {
	in := sampleProjectionV2("A")
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
	a := sampleProjectionV2("A")
	b := sampleProjectionV2("A")
	b.Intent.Lines[0], b.Intent.Lines[1] = b.Intent.Lines[1], b.Intent.Lines[0]
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("same lines in different order must produce different hashes")
	}
}

func TestRequestHashIncludesSeriesCode(t *testing.T) {
	a := sampleProjectionV2("A")
	b := sampleProjectionV2("B")
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("series_code must affect request hash")
	}
}

func TestRequestHashDiffersOnSemanticChange(t *testing.T) {
	a := sampleProjectionV2("A")
	b := sampleProjectionV2("A")
	b.Intent.ExternalID = "ext-2"
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("expected different hashes")
	}
}

func TestRequestHashIncludesTemporalContext(t *testing.T) {
	a := sampleProjectionV2("A")
	b := sampleProjectionV2("A")
	b.Intent.IssuedOffsetMinutes = 120
	if canonical.RequestHash(a) == canonical.RequestHash(b) {
		t.Fatal("issued_offset_minutes must affect v2 hash")
	}
}

func TestMaterializeIncludesVersionAndSeries(t *testing.T) {
	m := canonical.Materialize(sampleProjectionV2("SER1"))
	if !strings.HasPrefix(m, canonical.Version+"\n") {
		t.Fatalf("materialize missing version prefix: %q", m[:min(40, len(m))])
	}
	if !strings.Contains(m, "11:series_code4:SER1") {
		t.Fatalf("missing series_code LV field in materialize: %q", m)
	}
}

func TestMaterializeUnambiguousWithSpecialValues(t *testing.T) {
	qty, _ := quantity.ParseCanonical("1")
	price, _ := money.ParseCanonical("1.00")
	base := sampleProjectionV2("A")
	base.Intent.Lines = []canonical.Line{{
		LineID: "L1", Description: "plain", Quantity: qty, UnitPrice: price, TaxCode: "NOR",
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
	a := sampleProjectionV2("A")
	b := sampleProjectionV2("A")
	b.Intent.ScopeID = "scope"
	b.Intent.ExternalID = "-aext-1"

	c := sampleProjectionV2("A")
	c.Intent.SellerName = "X\nlines_count=0\n"

	ma, mb, mc := canonical.Materialize(a), canonical.Materialize(b), canonical.Materialize(c)
	if ma == mb || ma == mc || mb == mc {
		t.Fatal("structurally different projections must not share material")
	}
}

func TestMaterializeAndHashDeterministic(t *testing.T) {
	in := sampleProjectionV2("A")
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
