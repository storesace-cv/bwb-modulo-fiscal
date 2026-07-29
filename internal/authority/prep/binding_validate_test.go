package prep_test

import (
	"context"
	"errors"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminregistry"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/prep"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestValidateProfileBindingsSecretsReadyRequiresPresent(t *testing.T) {
	p := adminregistry.AuthorityProfile{
		ID: "p1", Environment: adminregistry.EnvHomologation,
		AllowedOperations:     []string{"registarFactura"},
		ProducerCredentialRef: "cred", ProducerKeyRef: "key", CertificateRef: "cert",
		SecretsReady: true,
	}
	lookup := func(ref secretstore.Ref) (secretstore.Metadata, error) {
		return secretstore.Metadata{Ref: ref, Status: secretstore.StatusAbsent, Environment: ref.Environment}, nil
	}
	v, err := prep.ValidateProfileBindings(context.Background(), p, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if v.Valid || v.ExternalVerified {
		t.Fatalf("%+v", v)
	}
	if err := prep.AssertBindingsForWrite(context.Background(), p, lookup); !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateProfileBindingsConflictOps(t *testing.T) {
	p := adminregistry.AuthorityProfile{
		ID: "p2", Environment: adminregistry.EnvHomologation,
		AllowedOperations: []string{"solicitarSerie"},
		SecretsReady:      false,
		Status:            adminregistry.AuthorityStatusDraft,
	}
	lookup := func(ref secretstore.Ref) (secretstore.Metadata, error) {
		return secretstore.Metadata{Ref: ref, Status: secretstore.StatusAbsent}, nil
	}
	v, err := prep.ValidateProfileBindings(context.Background(), p, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if v.Valid || v.OpsPathStatuses["solicitarSerie"] != prep.PathStatusConflictOpen {
		t.Fatalf("%+v", v)
	}
	if err := prep.AssertBindingsForWrite(context.Background(), p, lookup); !errors.Is(err, adminregistry.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}

func TestValidateProfileBindingsOK(t *testing.T) {
	p := adminregistry.AuthorityProfile{
		ID: "p3", Environment: adminregistry.EnvHomologation,
		AllowedOperations:     []string{"registarFactura", "obterEstado"},
		ProducerCredentialRef: "cred", ProducerKeyRef: "key", CertificateRef: "cert",
		SecretsReady: true, Status: adminregistry.AuthorityStatusValidated,
	}
	lookup := func(ref secretstore.Ref) (secretstore.Metadata, error) {
		return secretstore.Metadata{
			Ref: ref, Status: secretstore.StatusPresent, Environment: ref.Environment, Version: 1, Fingerprint: "aa",
		}, nil
	}
	v, err := prep.ValidateProfileBindings(context.Background(), p, lookup)
	if err != nil || !v.Valid {
		t.Fatalf("%+v %v", v, err)
	}
	if err := prep.AssertBindingsForWrite(context.Background(), p, lookup); err != nil {
		t.Fatal(err)
	}
}
