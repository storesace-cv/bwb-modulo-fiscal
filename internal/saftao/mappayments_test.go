package saftao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestMapPaymentsLedgerOmissionsAndExport(t *testing.T) {
	base := saftao.MinimalPaymentsFixture()
	tx := time.Date(2026, 1, 16, 11, 0, 0, 0, time.UTC)
	incomplete := saftao.PaymentLedgerRecord{
		ScopeID:       "scope-pay",
		DocumentID:    "pay-incomplete",
		TransactionAt: tx,
	}
	complete := saftao.PaymentLedgerRecord{
		ScopeID:            "scope-pay",
		DocumentID:         "pay-1",
		TransactionAt:      tx,
		SealedAt:           tx,
		PaymentRefNo:       "RC S001/1",
		PaymentType:        saftao.PaymentTypeRC,
		SourceID:           "POS1",
		CustomerID:         "C1",
		PaymentStatus:      saftao.PaymentStatusN,
		SourcePayment:      saftao.SourcePaymentP,
		PaymentStatusAt:    tx,
		PaymentMechanism:   "NU",
		PaymentAmountCents: 11400,
		Lines: []saftao.PaymentLedgerLine{{
			LineNo:        1,
			OriginatingON: "FT S001/1",
			InvoiceDate:   saftao.MustDate("2026-01-15"),
			CreditCents:   11400,
		}},
		TaxPayableCents: 1400,
		NetTotalCents:   10000,
		GrossTotalCents: 11400,
	}
	cfg := saftao.PaymentLedgerMapConfig{
		ScopeID:             "scope-pay",
		PeriodStart:         saftao.MustDate("2026-01-01"),
		PeriodEnd:           saftao.MustDate("2026-12-31"),
		Header:              *base.Header,
		AllowedPaymentTypes: []saftao.PaymentType{saftao.PaymentTypeRC},
		Customers:           base.MasterFiles.Customer,
		ValidateAgainstXSD:  saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.MapPaymentsLedgerToExport(cfg, []saftao.PaymentLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mapped != 1 || res.Omitted < 1 {
		t.Fatalf("mapped=%d omitted=%d oms=%v", res.Mapped, res.Omitted, res.Omissions)
	}
	for _, o := range res.Omissions {
		low := strings.ToLower(o.Reason + o.Field)
		if strings.Contains(low, "nif") || strings.Contains(low, "token") || strings.Contains(o.Reason, "5417") {
			t.Fatalf("omission must not leak sensitive data: %+v", o)
		}
	}
	if res.Export == nil || res.Export.NumberOfPayments != 1 {
		t.Fatal("expected one payment in export")
	}
	if !strings.Contains(string(res.Export.XML), "RC S001/1") {
		t.Fatal("payment missing from XML")
	}
	res2, err := saftao.MapPaymentsLedgerToExport(cfg, []saftao.PaymentLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Export.SHA256 != res2.Export.SHA256 {
		t.Fatal("payment map export must be deterministic")
	}
}
