package saftao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestMapPurchaseLedgerOmissionsAndExport(t *testing.T) {
	base := saftao.MinimalPurchaseInvoicesFixture()
	issued := time.Date(2026, 1, 20, 10, 0, 0, 0, time.UTC)
	incomplete := saftao.PurchaseLedgerRecord{ScopeID: "scope-pur", DocumentID: "p-inc", IssuedAt: issued}
	complete := saftao.PurchaseLedgerRecord{
		ScopeID:         "scope-pur",
		DocumentID:      "p-1",
		IssuedAt:        issued,
		InvoiceNo:       "FORN-FT-2026-0001",
		Hash:            "0",
		SourceID:        "AP1",
		PurchaseType:    saftao.PurchaseTypeFT,
		SupplierID:      "S1",
		TaxPayableCents: 1400,
		NetTotalCents:   10000,
		GrossTotalCents: 11400,
	}
	cfg := saftao.PurchaseLedgerMapConfig{
		ScopeID:              "scope-pur",
		PeriodStart:          saftao.MustDate("2026-01-01"),
		PeriodEnd:            saftao.MustDate("2026-12-31"),
		Header:               *base.Header,
		AllowedPurchaseTypes: []saftao.PurchaseType{saftao.PurchaseTypeFT},
		Suppliers:            base.MasterFiles.Supplier,
		ValidateAgainstXSD:   saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.MapPurchaseLedgerToExport(cfg, []saftao.PurchaseLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mapped != 1 || res.Omitted < 1 {
		t.Fatalf("mapped=%d omitted=%d", res.Mapped, res.Omitted)
	}
	for _, o := range res.Omissions {
		if strings.Contains(strings.ToLower(o.Reason), "nif") || strings.Contains(o.Reason, "5417") {
			t.Fatalf("sensitive omission: %+v", o)
		}
	}
	if strings.Contains(string(res.Export.XML), "<Line>") {
		t.Fatal("must not invent Line")
	}
	res2, err := saftao.MapPurchaseLedgerToExport(cfg, []saftao.PurchaseLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Export.SHA256 != res2.Export.SHA256 {
		t.Fatal("deterministic")
	}
}
