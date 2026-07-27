package persistence_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestStoreListPaymentsForSAFT_Unsupported(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "saft-pay-gap.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	store := persistence.NewStore(sqlDB, persistence.DialectSQLite)

	_, err = store.ListPaymentsForSAFT(ctx, persistence.SAFTPaymentQuery{
		ScopeID:    "scope",
		IssuedFrom: saftao.MustDate("2026-01-01"),
		IssuedTo:   saftao.MustDate("2026-12-31"),
	})
	if !errors.Is(err, persistence.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestSyntheticPaymentLedger_ScopePeriodCaps(t *testing.T) {
	ctx := context.Background()
	tx := time.Date(2026, 1, 16, 11, 0, 0, 0, time.UTC)
	src := persistence.SyntheticPaymentLedger{Records: []saftao.PaymentLedgerRecord{
		{ScopeID: "s1", DocumentID: "b", TransactionAt: tx},
		{ScopeID: "s1", DocumentID: "a", TransactionAt: tx.Add(time.Hour)},
		{ScopeID: "other", DocumentID: "x", TransactionAt: tx},
		{ScopeID: "s1", DocumentID: "old", TransactionAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}}
	out, err := src.ListPaymentsForSAFT(ctx, persistence.SAFTPaymentQuery{
		ScopeID:    "s1",
		IssuedFrom: saftao.MustDate("2026-01-01"),
		IssuedTo:   saftao.MustDate("2026-12-31"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].DocumentID != "a" && out[0].DocumentID != "b" {
		// deterministic: DocumentID ascending → a then b
	}
	if len(out) != 2 || out[0].DocumentID != "a" || out[1].DocumentID != "b" {
		t.Fatalf("got %+v", out)
	}

	_, err = src.ListPaymentsForSAFT(ctx, persistence.SAFTPaymentQuery{
		ScopeID:      "s1",
		IssuedFrom:   saftao.MustDate("2026-01-01"),
		IssuedTo:     saftao.MustDate("2026-12-31"),
		MaxDocuments: 1,
	})
	if !errors.Is(err, persistence.ErrValidation) {
		t.Fatalf("want MaxDocuments fail-closed, got %v", err)
	}
}
