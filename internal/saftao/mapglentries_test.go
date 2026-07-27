package saftao_test

import (
	"strings"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestMapGLEntriesLedgerOmissionsAndExport(t *testing.T) {
	base := saftao.MinimalGeneralLedgerEntriesFixture()
	tx := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	incomplete := saftao.GLEntriesLedgerRecord{
		ScopeID:       "scope-gle",
		DocumentID:    "gle-incomplete",
		TransactionAt: tx,
	}
	complete := saftao.GLEntriesLedgerRecord{
		ScopeID:           "scope-gle",
		DocumentID:        "gle-1",
		TransactionAt:     tx,
		SealedAt:          tx,
		JournalID:         "J1",
		JournalDesc:       "Diario sintetico",
		TransactionID:     "2026-01-15 J1 ARC001",
		Period:            1,
		SourceID:          "USER1",
		Description:       "Lancamento sintetico",
		DocArchivalNumber: "ARC001",
		TransactionType:   saftao.TransactionTypeN,
		DebitLines: []saftao.GLEntriesLedgerLine{{
			RecordID: "1", AccountID: "11.1", Description: "Debito sintetico", AmountCents: 5000,
		}},
		CreditLines: []saftao.GLEntriesLedgerLine{{
			RecordID: "2", AccountID: "11", Description: "Credito sintetico", AmountCents: 5000,
		}},
	}
	cfg := saftao.GLEntriesLedgerMapConfig{
		ScopeID:                 "scope-gle",
		PeriodStart:             saftao.MustDate("2026-01-01"),
		PeriodEnd:               saftao.MustDate("2026-12-31"),
		Header:                  *base.Header,
		AllowedTransactionTypes: []saftao.TransactionType{saftao.TransactionTypeN},
		GeneralLedgerAccounts:   base.MasterFiles.GeneralLedgerAccounts,
		ValidateAgainstXSD:      saftao.XSDValidatorAvailable(),
	}
	res, err := saftao.MapGLEntriesLedgerToExport(cfg, []saftao.GLEntriesLedgerRecord{incomplete, complete})
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
	if res.Export == nil || !strings.Contains(string(res.Export.XML), "GeneralLedgerEntries") {
		t.Fatal("expected GLE in export")
	}
	res2, err := saftao.MapGLEntriesLedgerToExport(cfg, []saftao.GLEntriesLedgerRecord{incomplete, complete})
	if err != nil {
		t.Fatal(err)
	}
	if res.Export.SHA256 != res2.Export.SHA256 {
		t.Fatal("GLE map export must be deterministic")
	}
}
