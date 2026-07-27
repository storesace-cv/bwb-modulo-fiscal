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

func TestStoreListMovementsForSAFT_Unsupported(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "saft-mov-gap.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	store := persistence.NewStore(sqlDB, persistence.DialectSQLite)
	_, err = store.ListMovementsForSAFT(ctx, persistence.SAFTMovementQuery{
		ScopeID: "s", IssuedFrom: saftao.MustDate("2026-01-01"), IssuedTo: saftao.MustDate("2026-12-31"),
	})
	if !errors.Is(err, persistence.ErrUnsupported) {
		t.Fatalf("got %v", err)
	}
}

func TestSyntheticMovementLedger(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	src := persistence.SyntheticMovementLedger{Records: []saftao.MovementLedgerRecord{
		{ScopeID: "s1", DocumentID: "b", MovementAt: at},
		{ScopeID: "s1", DocumentID: "a", MovementAt: at},
	}}
	out, err := src.ListMovementsForSAFT(ctx, persistence.SAFTMovementQuery{
		ScopeID: "s1", IssuedFrom: saftao.MustDate("2026-01-01"), IssuedTo: saftao.MustDate("2026-12-31"),
	})
	if err != nil || len(out) != 2 || out[0].DocumentID != "a" {
		t.Fatalf("%v %+v", err, out)
	}
}
