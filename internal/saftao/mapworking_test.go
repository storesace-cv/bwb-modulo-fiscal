package saftao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestMapWorkingLedgerOmissionsAndExport(t *testing.T) {
	base := saftao.MinimalWorkingDocumentsFixture()
	at := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	incomplete := saftao.WorkingLedgerRecord{ScopeID: "scope-wrk", DocumentID: "w-inc", WorkAt: at}
	complete := saftao.WorkingLedgerRecord{
		ScopeID: "scope-wrk", DocumentID: "w-1", WorkAt: at, SealedAt: at,
		DocumentNumber: "PF S001/1", Hash: "SYNTHETIC-HASH-NOT-A-SIGNATURE", HashControl: "0",
		WorkType: saftao.WorkTypePF, SourceID: "POS1", CustomerID: "C1",
		WorkStatus: saftao.WorkStatusN, SourceBilling: saftao.SourceBillingP, StatusAt: at,
		Lines: []saftao.WorkingLedgerLine{{
			LineNo: 1, ProductCode: "P1", Description: "Servico conferencia",
			QuantityScaled: 10000, UnitOfMeasure: "UN", UnitPriceCents: 2500, CreditCents: 2500,
			TaxType: "IVA", TaxCode: "NOR", TaxPercentage: "14.00",
		}},
		TaxPayableCents: 350, NetTotalCents: 2500, GrossTotalCents: 2850,
	}
	cfg := saftao.WorkingLedgerMapConfig{
		ScopeID: "scope-wrk", PeriodStart: saftao.MustDate("2026-01-01"), PeriodEnd: saftao.MustDate("2026-12-31"),
		Header: *base.Header, AllowedWorkTypes: []saftao.WorkType{saftao.WorkTypePF},
		Customers: base.MasterFiles.Customer, Products: base.MasterFiles.Product,
		ValidateAgainstXSD: saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.MapWorkingLedgerToExport(cfg, []saftao.WorkingLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mapped != 1 || res.Omitted < 1 {
		t.Fatalf("mapped=%d omitted=%d", res.Mapped, res.Omitted)
	}
	for _, o := range res.Omissions {
		if strings.Contains(strings.ToLower(o.Reason), "nif") {
			t.Fatalf("sensitive: %+v", o)
		}
	}
	res2, err := saftao.MapWorkingLedgerToExport(cfg, []saftao.WorkingLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Export.SHA256 != res2.Export.SHA256 {
		t.Fatal("deterministic")
	}
}
