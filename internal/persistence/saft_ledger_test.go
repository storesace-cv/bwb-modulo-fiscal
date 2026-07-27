package persistence_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/persistence"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

func TestListSealedSalesForSAFT_SQLite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "saft-ledger.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	store := persistence.NewStore(sqlDB, persistence.DialectSQLite)

	scope := "scope-saft-ledger"
	req := sampleSealReq(scope, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaa10", "ext-saft-1", "S001", "10.00")
	if _, err := store.SealInTx(ctx, req); err != nil {
		t.Fatalf("seal: %v", err)
	}

	from := saftao.MustDate("2024-01-01")
	to := saftao.MustDate("2026-12-31")
	recs, err := store.ListSealedSalesForSAFT(ctx, persistence.SAFTSalesQuery{
		ScopeID:    scope,
		IssuedFrom: from,
		IssuedTo:   to,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("recs=%d", len(recs))
	}
	r := recs[0]
	if r.ScopeID != scope || r.DocumentType != "invoice" || r.SeriesCode != "S001" || r.FiscalSeq != 1 {
		t.Fatalf("record: %+v", r)
	}
	if len(r.Lines) != 1 || r.Lines[0].UnitPriceCents != 1000 {
		t.Fatalf("lines: %+v", r.Lines)
	}
	// Enrichment fields empty — MapSalesLedger must omit without inventing Hash.
	if r.Hash != "" || r.CustomerID != "" || r.SourceID != "" {
		t.Fatal("must not invent SAF-T enrichment from ledger alone")
	}

	// Period filter excludes future-only window.
	empty, err := store.ListSealedSalesForSAFT(ctx, persistence.SAFTSalesQuery{
		ScopeID:    scope,
		IssuedFrom: saftao.MustDate("2099-01-01"),
		IssuedTo:   saftao.MustDate("2099-12-31"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("want empty, got %d", len(empty))
	}

	// End-to-end: loader + map with enrichment → export.
	base := saftao.MinimalSalesInvoiceFixture()
	r.Hash = "SYNTHETIC-HASH-NOT-A-SIGNATURE"
	r.HashControl = "0"
	r.SourceID = "POS1"
	r.CustomerID = "C1"
	r.UnitOfMeasure = "UN"
	r.TaxPercentage = "0.00"
	r.ProductCodeByLine = map[int]string{1: "P1"}
	r.Lines[0].TaxCode = "ISE"
	cfg := saftao.LedgerMapConfig{
		ScopeID:             scope,
		PeriodStart:         from,
		PeriodEnd:           to,
		Header:              *base.Header,
		AllowedInvoiceTypes: []saftao.InvoiceType{saftao.InvoiceTypeFT},
		EnabledGroups:       []saftao.DocumentGroup{saftao.GroupSalesInvoices},
		IncludeEmptySales:   true,
		ValidateAgainstXSD:  saftao.XSDValidatorAvailable(),
	}
	mapped, err := saftao.MapSalesLedgerToExport(cfg, []saftao.SalesLedgerRecord{r})
	if err != nil {
		t.Fatal(err)
	}
	if mapped.Mapped != 1 {
		t.Fatalf("mapped=%d omissions=%v", mapped.Mapped, mapped.Omissions)
	}
	if mapped.Export == nil || len(mapped.Export.XML) == 0 {
		t.Fatal("empty export")
	}
}
