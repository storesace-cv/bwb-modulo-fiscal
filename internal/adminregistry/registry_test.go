package adminregistry_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbtest"
)

func TestAdminRegistrySQLite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "admin.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	v, dirty, err := dbmigrate.Version(dbmigrate.DialectSQLite, path)
	if err != nil || dirty || v != dbmigrate.ExpectedVersion {
		t.Fatalf("version=%d dirty=%v err=%v want %d", v, dirty, err, dbmigrate.ExpectedVersion)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runRegistrySuite(t, ctx, adminregistry.New(sqlDB, adminregistry.DialectSQLite, fixedNow))
}

func TestAdminRegistryPostgres(t *testing.T) {
	dsn, cleanup := dbtest.OpenIsolatedPostgres(t)
	defer cleanup()
	ctx := context.Background()
	if err := dbmigrate.Up(dbmigrate.DialectPostgres, dsn); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenPostgres(ctx, db.PostgresConfig{URL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	runRegistrySuite(t, ctx, adminregistry.New(sqlDB, adminregistry.DialectPostgres, fixedNow))
}

func fixedNow() time.Time {
	return time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
}

func runRegistrySuite(t *testing.T, ctx context.Context, reg *adminregistry.Registry) {
	t.Helper()

	t.Run("create_taxpayer_establishment_binding", func(t *testing.T) {
		tp, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{
			NIF: "5000000001", LegalName: "Contribuinte Demo Lda",
		})
		if err != nil {
			t.Fatal(err)
		}
		est, err := reg.CreateEstablishment(ctx, adminregistry.CreateEstablishmentInput{
			TaxpayerID: tp.ID, Code: "LOJA1", Name: "Loja 1",
		})
		if err != nil {
			t.Fatal(err)
		}
		bind, err := reg.CreateScopeBinding(ctx, adminregistry.CreateScopeBindingInput{
			ScopeID: "scope-demo-1", TaxpayerID: tp.ID, EstablishmentID: est.ID,
			Environment: adminregistry.EnvHomologation, IANATimezone: "Africa/Luanda",
			SeriesEffectiveCode: "A",
		})
		if err != nil {
			t.Fatal(err)
		}
		if bind.ScopeID != "scope-demo-1" || bind.Environment != adminregistry.EnvHomologation {
			t.Fatalf("%+v", bind)
		}
		updated, err := reg.UpdateScopeConfig(ctx, adminregistry.UpdateScopeConfigInput{
			ScopeID: bind.ScopeID, Environment: adminregistry.EnvProduction,
			IANATimezone: "Europe/Lisbon", SeriesEffectiveCode: "B", Status: adminregistry.StatusInactive,
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Environment != adminregistry.EnvProduction || updated.SeriesEffectiveCode != "B" ||
			updated.IANATimezone != "Europe/Lisbon" || updated.Status != adminregistry.StatusInactive {
			t.Fatalf("update: %+v", updated)
		}
		if updated.TaxpayerID != tp.ID || updated.EstablishmentID != est.ID {
			t.Fatalf("immutable ids changed: %+v", updated)
		}
		got, err := reg.GetTaxpayer(ctx, tp.ID)
		if err != nil || got.NIF != "5000000001" {
			t.Fatalf("%+v %v", got, err)
		}
	})

	t.Run("duplicate_nif_conflict", func(t *testing.T) {
		_, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{
			NIF: "5000000002", LegalName: "A",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{
			NIF: "5000000002", LegalName: "B",
		})
		if !errors.Is(err, adminregistry.ErrConflict) {
			t.Fatalf("want ErrConflict, got %v", err)
		}
	})

	t.Run("establishment_wrong_taxpayer_rejected", func(t *testing.T) {
		a, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{NIF: "5000000003", LegalName: "A"})
		if err != nil {
			t.Fatal(err)
		}
		b, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{NIF: "5000000004", LegalName: "B"})
		if err != nil {
			t.Fatal(err)
		}
		est, err := reg.CreateEstablishment(ctx, adminregistry.CreateEstablishmentInput{
			TaxpayerID: a.ID, Code: "X", Name: "X",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = reg.CreateScopeBinding(ctx, adminregistry.CreateScopeBindingInput{
			ScopeID: "scope-mismatch", TaxpayerID: b.ID, EstablishmentID: est.ID,
			Environment: adminregistry.EnvDevelopment, IANATimezone: "Africa/Luanda",
			SeriesEffectiveCode: "B",
		})
		if !errors.Is(err, adminregistry.ErrValidation) {
			t.Fatalf("want ErrValidation, got %v", err)
		}
	})

	t.Run("empty_nif_rejected", func(t *testing.T) {
		_, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{NIF: "  ", LegalName: "X"})
		if !errors.Is(err, adminregistry.ErrValidation) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("invalid_timezone_rejected", func(t *testing.T) {
		tp, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{NIF: "5000000005", LegalName: "TZ"})
		if err != nil {
			t.Fatal(err)
		}
		est, err := reg.CreateEstablishment(ctx, adminregistry.CreateEstablishmentInput{
			TaxpayerID: tp.ID, Code: "TZ1", Name: "TZ",
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = reg.CreateScopeBinding(ctx, adminregistry.CreateScopeBindingInput{
			ScopeID: "scope-bad-tz", TaxpayerID: tp.ID, EstablishmentID: est.ID,
			Environment: adminregistry.EnvHomologation, IANATimezone: "Not/AZone",
			SeriesEffectiveCode: "A",
		})
		if !errors.Is(err, adminregistry.ErrValidation) {
			t.Fatalf("want ErrValidation, got %v", err)
		}
	})

	t.Run("fe_enrollment_per_environment", func(t *testing.T) {
		tp, err := reg.CreateTaxpayer(ctx, adminregistry.CreateTaxpayerInput{NIF: "5000000006", LegalName: "FE Co"})
		if err != nil {
			t.Fatal(err)
		}
		got, err := reg.UpsertFEEnrollment(ctx, adminregistry.UpsertFEEnrollmentInput{
			TaxpayerID: tp.ID, Environment: adminregistry.EnvHomologation, Status: adminregistry.FEEnrollmentActive,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !adminregistry.FEAderiu(got.Status) || got.Environment != adminregistry.EnvHomologation {
			t.Fatalf("%+v", got)
		}
		_, err = reg.UpsertFEEnrollment(ctx, adminregistry.UpsertFEEnrollmentInput{
			TaxpayerID: tp.ID, Environment: adminregistry.EnvProduction, Status: adminregistry.FEEnrollmentPending,
		})
		if err != nil {
			t.Fatal(err)
		}
		list, err := reg.ListFEEnrollments(ctx, tp.ID)
		if err != nil || len(list) != 2 {
			t.Fatalf("list=%v err=%v", list, err)
		}
		if adminregistry.EffectiveFEStatus(list, adminregistry.EnvProduction) != adminregistry.FEEnrollmentPending {
			t.Fatalf("%+v", list)
		}
		_, err = reg.UpsertFEEnrollment(ctx, adminregistry.UpsertFEEnrollmentInput{
			TaxpayerID: tp.ID, Environment: adminregistry.EnvHomologation, Status: "boolean",
		})
		if !errors.Is(err, adminregistry.ErrValidation) {
			t.Fatalf("want validation, got %v", err)
		}
	})
}
