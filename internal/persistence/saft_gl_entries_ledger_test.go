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

func TestStoreListGLEntriesForSAFT_Unsupported(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "saft-gle-gap.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	store := persistence.NewStore(sqlDB, persistence.DialectSQLite)

	_, err = store.ListGLEntriesForSAFT(ctx, persistence.SAFTGLEntriesQuery{
		ScopeID:    "scope",
		IssuedFrom: saftao.MustDate("2026-01-01"),
		IssuedTo:   saftao.MustDate("2026-12-31"),
	})
	if !errors.Is(err, persistence.ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func TestSyntheticGLEntriesLedger_ScopePeriodCaps(t *testing.T) {
	ctx := context.Background()
	tx := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	src := persistence.SyntheticGLEntriesLedger{Records: []saftao.GLEntriesLedgerRecord{
		{ScopeID: "s1", DocumentID: "b", JournalID: "J2", TransactionAt: tx},
		{ScopeID: "s1", DocumentID: "a", JournalID: "J1", TransactionAt: tx.Add(time.Hour)},
		{ScopeID: "other", DocumentID: "x", JournalID: "J1", TransactionAt: tx},
		{ScopeID: "s1", DocumentID: "old", JournalID: "J1", TransactionAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}}
	out, err := src.ListGLEntriesForSAFT(ctx, persistence.SAFTGLEntriesQuery{
		ScopeID:    "s1",
		IssuedFrom: saftao.MustDate("2026-01-01"),
		IssuedTo:   saftao.MustDate("2026-12-31"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].DocumentID != "a" || out[1].DocumentID != "b" {
		t.Fatalf("got %+v", out)
	}
	_, err = src.ListGLEntriesForSAFT(ctx, persistence.SAFTGLEntriesQuery{
		ScopeID: "s1", IssuedFrom: saftao.MustDate("2026-01-01"), IssuedTo: saftao.MustDate("2026-12-31"), MaxDocuments: 1,
	})
	if !errors.Is(err, persistence.ErrValidation) {
		t.Fatalf("want ErrValidation on MaxDocuments, got %v", err)
	}
}
