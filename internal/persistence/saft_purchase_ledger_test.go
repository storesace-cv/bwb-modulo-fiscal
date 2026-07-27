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

func TestStoreListPurchasesForSAFT_Unsupported(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "saft-pur-gap.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	store := persistence.NewStore(sqlDB, persistence.DialectSQLite)
	_, err = store.ListPurchasesForSAFT(ctx, persistence.SAFTPurchaseQuery{
		ScopeID: "s", IssuedFrom: saftao.MustDate("2026-01-01"), IssuedTo: saftao.MustDate("2026-12-31"),
	})
	if !errors.Is(err, persistence.ErrUnsupported) {
		t.Fatalf("got %v", err)
	}
}

func TestSyntheticPurchaseLedger(t *testing.T) {
	ctx := context.Background()
	issued := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	src := persistence.SyntheticPurchaseLedger{Records: []saftao.PurchaseLedgerRecord{
		{ScopeID: "s1", DocumentID: "b", IssuedAt: issued},
		{ScopeID: "s1", DocumentID: "a", IssuedAt: issued},
		{ScopeID: "s1", DocumentID: "old", IssuedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}}
	out, err := src.ListPurchasesForSAFT(ctx, persistence.SAFTPurchaseQuery{
		ScopeID: "s1", IssuedFrom: saftao.MustDate("2026-01-01"), IssuedTo: saftao.MustDate("2026-12-31"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].DocumentID != "a" {
		t.Fatalf("%+v", out)
	}
}
