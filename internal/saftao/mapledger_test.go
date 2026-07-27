package saftao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestMapSalesLedgerPeriodScopeAndOmissions(t *testing.T) {
	start := saftao.MustDate("2026-01-01")
	end := saftao.MustDate("2026-01-31")
	base := saftao.MinimalSalesInvoiceFixture()

	complete := saftao.SalesLedgerRecord{
		ScopeID:       "scope-a",
		DocumentID:    "doc-1",
		DocumentType:  "invoice",
		SeriesCode:    "S001",
		FiscalSeq:     1,
		IssuedAt:      time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		SealedAt:      time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		CustomerTaxID: "999999999",
		CustomerName:  "Cliente",
		Lines: []saftao.SalesLedgerLine{{
			LineNo:         1,
			Description:    "Servico",
			QuantityScaled: 10000,
			UnitPriceCents: 10000,
			TaxCode:        "ISE",
		}},
		Hash:              "SYNTHETIC-HASH-NOT-A-SIGNATURE",
		HashControl:       "0",
		SourceID:          "POS1",
		CustomerID:        "C1",
		UnitOfMeasure:     "UN",
		TaxPercentage:     "0.00",
		ProductCodeByLine: map[int]string{1: "P1"},
	}
	outOfPeriod := complete
	outOfPeriod.DocumentID = "doc-old"
	outOfPeriod.IssuedAt = time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)

	missingHash := complete
	missingHash.DocumentID = "doc-no-hash"
	missingHash.Hash = ""

	otherScope := complete
	otherScope.DocumentID = "doc-other-scope"
	otherScope.ScopeID = "scope-b"

	cfg := saftao.LedgerMapConfig{
		ScopeID:             "scope-a",
		PeriodStart:         start,
		PeriodEnd:           end,
		Header:              *base.Header,
		AllowedInvoiceTypes: []saftao.InvoiceType{saftao.InvoiceTypeFT},
		EnabledGroups:       []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		IncludeEmptySales:   true,
		ValidateAgainstXSD:  saftao.XSDValidatorAvailable(),
	}

	res, err := saftao.MapSalesLedgerToExport(cfg, []saftao.SalesLedgerRecord{complete, outOfPeriod, missingHash, otherScope})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mapped != 1 {
		t.Fatalf("mapped=%d want 1", res.Mapped)
	}
	if res.Omitted < 3 {
		t.Fatalf("omitted=%d omissions=%v", res.Omitted, res.Omissions)
	}
	foundHashOmit := false
	for _, o := range res.Omissions {
		if o.DocumentID == "doc-no-hash" && o.Field == "hash" {
			foundHashOmit = true
		}
	}
	if !foundHashOmit {
		t.Fatalf("expected hash omission: %v", res.Omissions)
	}
	if !strings.Contains(string(res.Export.XML), "FT S001/1") {
		t.Fatalf("xml: %s", res.Export.XML)
	}
	if cfg.ValidateAgainstXSD && !res.Export.XSDChecked {
		t.Fatal("expected XSDChecked")
	}
}

func TestMapSalesLedgerDeterministic(t *testing.T) {
	start := saftao.MustDate("2026-01-01")
	end := saftao.MustDate("2026-12-31")
	base := saftao.MinimalSalesInvoiceFixture()
	rec := saftao.SalesLedgerRecord{
		ScopeID: "s", DocumentID: "d2", DocumentType: "invoice", SeriesCode: "A", FiscalSeq: 2,
		IssuedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		SealedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		Lines: []saftao.SalesLedgerLine{{
			LineNo: 1, Description: "X", QuantityScaled: 10000, UnitPriceCents: 500, TaxCode: "ISE",
		}},
		Hash: "H", HashControl: "0", SourceID: "SRC", CustomerID: "C9",
		UnitOfMeasure: "UN", TaxPercentage: "0.00", ProductCodeByLine: map[int]string{1: "PX"},
	}
	cfg := saftao.LedgerMapConfig{
		ScopeID: "s", PeriodStart: start, PeriodEnd: end, Header: *base.Header,
		AllowedInvoiceTypes: []saftao.InvoiceType{saftao.InvoiceTypeFT},
		EnabledGroups:       []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		IncludeEmptySales:   true,
	}
	a, err := saftao.MapSalesLedgerToExport(cfg, []saftao.SalesLedgerRecord{rec})
	if err != nil {
		t.Fatal(err)
	}
	b, err := saftao.MapSalesLedgerToExport(cfg, []saftao.SalesLedgerRecord{rec})
	if err != nil {
		t.Fatal(err)
	}
	if a.Export.SHA256 != b.Export.SHA256 {
		t.Fatal("map+export must be deterministic")
	}
}
