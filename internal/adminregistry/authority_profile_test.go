package adminregistry_test

import (
	"path/filepath"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func TestAuthorityProfileCRUD_NoSecrets(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "auth-profile.db")
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
		Environment:       "development",
		DisplayName:       "bad",
		AllowedOperations: []string{"registarFactura"},
	})
	if err == nil {
		t.Fatal("development environment must fail-closed")
	}

	_, err = reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment:       adminregistry.EnvHomologation,
		DisplayName:       "HML prep",
		AllowedOperations: []string{"inventedOp"},
	})
	if err == nil {
		t.Fatal("unknown operation must fail-closed")
	}

	_, err = reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment:     adminregistry.EnvHomologation,
		DisplayName:     "leak",
		PendingExternal: map[string]string{"note": "-----BEGIN PRIVATE KEY-----"},
	})
	if err == nil {
		t.Fatal("secret-like pending_external must fail-closed")
	}

	p, err := reg.CreateAuthorityProfile(ctx, adminregistry.CreateAuthorityProfileInput{
		Environment:           adminregistry.EnvHomologation,
		DisplayName:           "HML prep",
		AllowedOperations:     []string{"registarFactura", "obterEstado"},
		PendingExternal:       map[string]string{"base_path_conflict": "C-FE-001"},
		ProducerCredentialRef: "platform/hml/cred",
		AlgorithmDeclared:     "RS256",
		FingerprintSanitized:  "sha256:deadbeef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.ExternalVerified || p.Status != adminregistry.AuthorityStatusDraft {
		t.Fatalf("got %+v", p)
	}

	got, err := reg.GetAuthorityProfile(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "HML prep" || len(got.AllowedOperations) != 2 {
		t.Fatalf("%+v", got)
	}

	trueVal := true
	upd, err := reg.UpdateAuthorityProfile(ctx, adminregistry.UpdateAuthorityProfileInput{
		ProfileID:    p.ID,
		Status:       adminregistry.AuthorityStatusValidated,
		ConfigReady:  &trueVal,
		SecretsReady: &trueVal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !upd.ConfigReady || !upd.SecretsReady || upd.ExternalVerified {
		t.Fatalf("%+v", upd)
	}

	list, err := reg.ListAuthorityProfiles(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}
