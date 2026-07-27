package saftao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestMapMovementLedgerOmissionsAndExport(t *testing.T) {
	base := saftao.MinimalMovementOfGoodsFixture()
	at := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	incomplete := saftao.MovementLedgerRecord{ScopeID: "scope-mov", DocumentID: "m-inc", MovementAt: at}
	complete := saftao.MovementLedgerRecord{
		ScopeID:        "scope-mov",
		DocumentID:     "m-1",
		MovementAt:     at,
		SealedAt:       at,
		DocumentNumber: "GR S001/1",
		Hash:           "SYNTHETIC-HASH-NOT-A-SIGNATURE",
		HashControl:    "0",
		MovementType:   saftao.MovementTypeGR,
		SourceID:       "POS1",
		CustomerID:     "C1",
		MovementStatus: saftao.MovementStatusN,
		SourceBilling:  saftao.SourceBillingP,
		StatusAt:       at,
		StartAt:        at,
		Lines: []saftao.MovementLedgerLine{{
			LineNo: 1, ProductCode: "P1", Description: "Mercadoria sintetica",
			QuantityScaled: 10000, UnitOfMeasure: "UN", UnitPriceCents: 5000, CreditCents: 5000,
			TaxType: "IVA", TaxCode: "NOR", TaxPercentage: "14.00",
		}},
		TaxPayableCents: 700, NetTotalCents: 5000, GrossTotalCents: 5700,
	}
	cfg := saftao.MovementLedgerMapConfig{
		ScopeID:              "scope-mov",
		PeriodStart:          saftao.MustDate("2026-01-01"),
		PeriodEnd:            saftao.MustDate("2026-12-31"),
		Header:               *base.Header,
		AllowedMovementTypes: []saftao.MovementType{saftao.MovementTypeGR},
		Customers:            base.MasterFiles.Customer,
		Products:             base.MasterFiles.Product,
		ValidateAgainstXSD:   saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.MapMovementLedgerToExport(cfg, []saftao.MovementLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mapped != 1 || res.Omitted < 1 {
		t.Fatalf("mapped=%d omitted=%d oms=%v", res.Mapped, res.Omitted, res.Omissions)
	}
	for _, o := range res.Omissions {
		if strings.Contains(strings.ToLower(o.Reason), "nif") {
			t.Fatalf("sensitive: %+v", o)
		}
	}
	res2, err := saftao.MapMovementLedgerToExport(cfg, []saftao.MovementLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Export.SHA256 != res2.Export.SHA256 {
		t.Fatal("deterministic")
	}
}
