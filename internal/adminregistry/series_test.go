package adminregistry_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/doctype"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestSeriesLifecycleNoReuse(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "series.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)

	tp, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{NIF: "5000000399", LegalName: "Series Co"})
	if err != nil {
		t.Fatal(err)
	}
	est, err := reg.CreateEstablishment(ctx, adminregistry.CreateEstablishmentInput{
		TaxpayerID: tp.ID, Code: "S1", Name: "Loja S",
	})
	if err != nil {
		t.Fatal(err)
	}
	s1, err := reg.CreateSeries(ctx, adminregistry.CreateSeriesInput{
		EstablishmentID: est.ID, Environment: adminregistry.EnvHomologation,
		CodigoCanonico: doctype.CanonicalFT, Code: "A",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s1.Status != adminregistry.SeriesDraft || s1.Version != 1 {
		t.Fatalf("%+v", s1)
	}
	_, err = reg.CreateSeries(ctx, adminregistry.CreateSeriesInput{
		EstablishmentID: est.ID, Environment: adminregistry.EnvHomologation,
		CodigoCanonico: doctype.CanonicalNC, Code: "A",
	})
	if !errors.Is(err, adminregistry.ErrConflict) {
		t.Fatalf("reuse code want conflict, got %v", err)
	}

	active, err := reg.TransitionSeries(ctx, adminregistry.TransitionSeriesInput{
		SeriesID: s1.ID, ExpectedVersion: 1, ToStatus: adminregistry.SeriesActive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.Status != adminregistry.SeriesActive || active.Version != 2 {
		t.Fatalf("%+v", active)
	}
	_, err = reg.TransitionSeries(ctx, adminregistry.TransitionSeriesInput{
		SeriesID: s1.ID, ExpectedVersion: 1, ToStatus: adminregistry.SeriesClosed,
	})
	if !errors.Is(err, adminregistry.ErrConflict) {
		t.Fatalf("stale version want conflict, got %v", err)
	}
	_, err = reg.TransitionSeries(ctx, adminregistry.TransitionSeriesInput{
		SeriesID: s1.ID, ExpectedVersion: 2, ToStatus: adminregistry.SeriesDraft,
	})
	if !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("backwards want validation, got %v", err)
	}
	closed, err := reg.TransitionSeries(ctx, adminregistry.TransitionSeriesInput{
		SeriesID: s1.ID, ExpectedVersion: 2, ToStatus: adminregistry.SeriesClosed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != adminregistry.SeriesClosed || closed.ValidTo == nil {
		t.Fatalf("%+v", closed)
	}
	_, err = reg.TransitionSeries(ctx, adminregistry.TransitionSeriesInput{
		SeriesID: s1.ID, ExpectedVersion: 3, ToStatus: adminregistry.SeriesActive,
	})
	if !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("reopen closed want validation, got %v", err)
	}
}
