package adminregistry_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAuthorityProfileActivationGateFailClosed(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "auth-gate.db")
	if err := dbmigrate.Up(dbmigrate.DialectSQLite, path); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.OpenSQLite(ctx, db.SQLiteConfig{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	reg := adminregistry.New(sqlDB, adminregistry.DialectSQLite, nil)

	_, err = reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: adminregistry.EnvHomologation,
		DisplayName: "gate create active",
		Status:      adminregistry.AuthorityStatusActive,
	})
	if !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("create active without readiness want ErrValidation, got %v", err)
	}

	p, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment: adminregistry.EnvHomologation,
		DisplayName: "gate draft",
		Status:      adminregistry.AuthorityStatusDraft,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID: p.ID,
		Status:    adminregistry.AuthorityStatusActive,
	})
	if !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("activate without flags want ErrValidation, got %v", err)
	}

	trueVal := true
	falseVal := false
	_, err = reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID:        p.ID,
		Status:           adminregistry.AuthorityStatusActive,
		ConfigReady:      &trueVal,
		SecretsReady:     &trueVal,
		OfflineValidated: &falseVal,
	})
	if !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("partial readiness want ErrValidation, got %v", err)
	}

	active, err := reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID:        p.ID,
		Status:           adminregistry.AuthorityStatusActive,
		ConfigReady:      &trueVal,
		SecretsReady:     &trueVal,
		OfflineValidated: &trueVal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.Status != adminregistry.AuthorityStatusActive || active.ExternalVerified {
		t.Fatalf("active must keep external_verified=false: %+v", active)
	}

	// Clearing a readiness flag while active must fail-closed (no silent demotion).
	_, err = reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID:    p.ID,
		SecretsReady: &falseVal,
	})
	if !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("clear secrets_ready while active want ErrValidation, got %v", err)
	}
	still, err := reg.GetAuthorityProfile(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if still.Status != adminregistry.AuthorityStatusActive || !still.SecretsReady || still.ExternalVerified {
		t.Fatalf("failed update must not mutate: %+v", still)
	}
}
