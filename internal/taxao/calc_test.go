package taxao_test

import (
	"errors"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/money"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/quantity"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/taxao"
)

func mustQty(t *testing.T, s string) quantity.Qty {
	t.Helper()
	q, err := quantity.ParseCanonical(s)
	if err != nil {
		t.Fatalf("qty %q: %v", s, err)
	}
	return q
}

func mustMoney(t *testing.T, s string) money.Amount {
	t.Helper()
	a, err := money.ParseCanonical(s)
	if err != nil {
		t.Fatalf("money %q: %v", s, err)
	}
	return a
}

func TestAO_TAX_001_saft_fixture_single_line_nor(t *testing.T) {
	t.Parallel()
	q := mustQty(t, "1")
	price := mustMoney(t, "100.00")
	lr, err := taxao.CalculateLine(taxao.LineInput{Quantity: q, UnitPrice: price, TaxCode: "NOR"})
	if err != nil {
		t.Fatal(err)
	}
	if lr.NetCents != 10000 || lr.TaxCents != 1400 || lr.GrossCents != 11400 {
		t.Fatalf("line totals: net=%d tax=%d gross=%d", lr.NetCents, lr.TaxCents, lr.GrossCents)
	}
	if taxao.FormatCents(lr.GrossCents) != "114.00" {
		t.Fatalf("gross format: %s", taxao.FormatCents(lr.GrossCents))
	}
}

func TestAO_TAX_001_line_nor_standard(t *testing.T) {
	t.Parallel()
	q := mustQty(t, "1")
	price := mustMoney(t, "10.50")
	lr, err := taxao.CalculateLine(taxao.LineInput{Quantity: q, UnitPrice: price, TaxCode: "NOR"})
	if err != nil {
		t.Fatal(err)
	}
	if lr.NetCents != 1050 || lr.TaxCents != 147 || lr.GrossCents != 1197 {
		t.Fatalf("got net=%d tax=%d gross=%d", lr.NetCents, lr.TaxCents, lr.GrossCents)
	}
}

func TestAO_TAX_001_ise_zero_tax(t *testing.T) {
	t.Parallel()
	q := mustQty(t, "2")
	price := mustMoney(t, "25.00")
	lr, err := taxao.CalculateLine(taxao.LineInput{Quantity: q, UnitPrice: price, TaxCode: "ISE"})
	if err != nil {
		t.Fatal(err)
	}
	if lr.TaxCents != 0 || lr.GrossCents != lr.NetCents || lr.NetCents != 5000 {
		t.Fatalf("ISE: net=%d tax=%d gross=%d", lr.NetCents, lr.TaxCents, lr.GrossCents)
	}
}

func TestAO_TAX_001_red_rate(t *testing.T) {
	t.Parallel()
	q := mustQty(t, "1")
	price := mustMoney(t, "100.00")
	lr, err := taxao.CalculateLine(taxao.LineInput{Quantity: q, UnitPrice: price, TaxCode: "RED"})
	if err != nil {
		t.Fatal(err)
	}
	if lr.TaxCents != 500 || lr.GrossCents != 10500 {
		t.Fatalf("RED: tax=%d gross=%d", lr.TaxCents, lr.GrossCents)
	}
}

func TestAO_TAX_001_fractional_quantity_rounding(t *testing.T) {
	t.Parallel()
	q := mustQty(t, "1.5")
	price := mustMoney(t, "10.00")
	lr, err := taxao.CalculateLine(taxao.LineInput{Quantity: q, UnitPrice: price, TaxCode: "NOR"})
	if err != nil {
		t.Fatal(err)
	}
	if lr.NetCents != 1500 {
		t.Fatalf("net=%d want 1500", lr.NetCents)
	}
	if lr.TaxCents != 210 {
		t.Fatalf("tax=%d want 210", lr.TaxCents)
	}
}

func TestAO_TAX_001_tax_half_up_edge(t *testing.T) {
	t.Parallel()
	// 4 cents net @ 14% → 0.56 cents → rounds to 1 cent tax.
	net := int64(4)
	tax, err := taxao.LineTaxCents(net, 1400)
	if err != nil {
		t.Fatal(err)
	}
	if tax != 1 {
		t.Fatalf("tax=%d want 1", tax)
	}
}

func TestAO_TAX_001_document_multi_line_totals(t *testing.T) {
	t.Parallel()
	doc, err := taxao.CalculateDocument([]taxao.LineInput{
		{Quantity: mustQty(t, "1"), UnitPrice: mustMoney(t, "50.00"), TaxCode: "NOR"},
		{Quantity: mustQty(t, "1"), UnitPrice: mustMoney(t, "50.00"), TaxCode: "ISE"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.NetTotalCents != 10000 || doc.TaxTotalCents != 700 || doc.GrossTotalCents != 10700 {
		t.Fatalf("doc totals: net=%d tax=%d gross=%d", doc.NetTotalCents, doc.TaxTotalCents, doc.GrossTotalCents)
	}
}

func TestAO_TAX_001_unknown_tax_code_fail_closed(t *testing.T) {
	t.Parallel()
	_, err := taxao.CalculateLine(taxao.LineInput{
		Quantity: mustQty(t, "1"), UnitPrice: mustMoney(t, "1.00"), TaxCode: "OUT",
	})
	if !errors.Is(err, taxao.ErrUnknownTaxCode) {
		t.Fatalf("want ErrUnknownTaxCode, got %v", err)
	}
}

func TestKnownTaxCode_mvp_catalog(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"NOR", "RED", "INT", "ISE"} {
		if !taxao.KnownTaxCode(code) {
			t.Fatalf("expected known: %s", code)
		}
	}
	if taxao.KnownTaxCode("OUT") || taxao.KnownTaxCode("nor") {
		t.Fatal("OUT/lowercase must not be known in MVP catalog")
	}
}
