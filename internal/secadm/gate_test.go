package secadm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secadm"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestOwnerCanPutOperatorDenied(t *testing.T) {
	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := secadm.NewGate("owner-jorge", mem)
	if err != nil {
		t.Fatal(err)
	}
	ref := secretstore.Ref{
		Kind: "producer_credential", Environment: secretstore.EnvHomologation,
		SubjectID: "platform", Name: "agt",
	}
	_, err = gate.Put(context.Background(), secadm.Actor{SubjectID: "ops-user"}, ref, []byte("x"), nil)
	if !errors.Is(err, secadm.ErrUnauthorized) {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
	res, err := gate.Put(context.Background(), secadm.Actor{SubjectID: "owner-jorge"}, ref, []byte("x"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Metadata.Status != secretstore.StatusPresent {
		t.Fatalf("%+v", res.Metadata)
	}
	meta, err := gate.Metadata(context.Background(), secadm.Actor{SubjectID: "owner-jorge"}, ref)
	if err != nil || meta.Fingerprint == "" {
		t.Fatalf("%+v %v", meta, err)
	}
}

func TestEmptyOwnerRejected(t *testing.T) {
	mem, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = secadm.NewGate("  ", mem)
	if !errors.Is(err, secadm.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}
