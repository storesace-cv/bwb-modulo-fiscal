package secretstore_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

func TestWriteOnlyPutMetadataNoPlaintext(t *testing.T) {
	store, err := secretstore.NewMemorySimulator(func() time.Time {
		return time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	ref := secretstore.Ref{
		Kind: "producer_credential", Environment: secretstore.EnvHomologation,
		SubjectID: "platform", Name: "agt-basic",
	}
	secret := []byte("not-a-real-agt-secret")
	res, err := store.Put(context.Background(), ref, secret, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Metadata.Status != secretstore.StatusPresent || res.Metadata.Fingerprint == "" {
		t.Fatalf("%+v", res.Metadata)
	}
	meta, err := store.Metadata(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Fingerprint != res.Metadata.Fingerprint {
		t.Fatalf("fp mismatch")
	}
	// Admin path cannot reveal.
	if _, err := secretstore.AdminRevealDenied(context.Background(), ref); !errors.Is(err, secretstore.ErrWriteOnly) {
		t.Fatalf("want ErrWriteOnly, got %v", err)
	}
	// Runtime reveal works in simulator only.
	got, err := store.Reveal(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(secret) {
		t.Fatalf("reveal mismatch")
	}
	meta2, err := store.Metadata(context.Background(), ref)
	if err != nil || meta2.LastVerifiedAt == nil {
		t.Fatalf("last verified: %+v %v", meta2, err)
	}
}

func TestHMLPRDCopyRejected(t *testing.T) {
	store, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	from := secretstore.Ref{Kind: secretstore.KindProducerCredential, Environment: secretstore.EnvHomologation, SubjectID: "platform", Name: "x"}
	to := secretstore.Ref{Kind: secretstore.KindProducerCredential, Environment: secretstore.EnvProduction, SubjectID: "platform", Name: "x"}
	err = store.CopyAcrossEnvironments(context.Background(), from, to)
	if !errors.Is(err, secretstore.ErrEnvIsolation) {
		t.Fatalf("got %v", err)
	}
}

func TestRevokeWipesReveal(t *testing.T) {
	store, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	ref := secretstore.Ref{Kind: "taxpayer_key", Environment: secretstore.EnvHomologation, SubjectID: "tp1", Name: "jws"}
	if _, err := store.Put(context.Background(), ref, []byte("ephemeral-test-key"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Revoke(context.Background(), ref); err != nil {
		t.Fatal(err)
	}
	meta, err := store.Metadata(context.Background(), ref)
	if err != nil || meta.Status != secretstore.StatusRevoked || meta.Fingerprint != "" {
		t.Fatalf("%+v %v", meta, err)
	}
	if _, err := store.Reveal(context.Background(), ref); !errors.Is(err, secretstore.ErrRevoked) {
		t.Fatalf("got %v", err)
	}
}

func TestRotateIncrementsVersion(t *testing.T) {
	store, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	ref := secretstore.Ref{Kind: "producer_key", Environment: secretstore.EnvProduction, SubjectID: "platform", Name: "rsa"}
	r1, err := store.Put(context.Background(), ref, []byte("v1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := store.Rotate(context.Background(), ref, []byte("v2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if r1.Metadata.Version != 1 || r2.Metadata.Version != 2 {
		t.Fatalf("%d %d", r1.Metadata.Version, r2.Metadata.Version)
	}
	if r1.Metadata.Fingerprint == r2.Metadata.Fingerprint {
		t.Fatal("fingerprint should change on rotate")
	}
}

func TestListMetadataEnvironmentIsolation(t *testing.T) {
	store, err := secretstore.NewMemorySimulator(nil)
	if err != nil {
		t.Fatal(err)
	}
	hml := secretstore.Ref{Kind: "producer_credential", Environment: secretstore.EnvHomologation, SubjectID: "platform", Name: "a"}
	prd := secretstore.Ref{Kind: "producer_credential", Environment: secretstore.EnvProduction, SubjectID: "platform", Name: "a"}
	if _, err := store.Put(context.Background(), hml, []byte("hml-only"), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Put(context.Background(), prd, []byte("prd-only"), nil); err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListMetadata(context.Background(), secretstore.EnvHomologation)
	if err != nil || len(rows) != 1 || rows[0].Environment != secretstore.EnvHomologation {
		t.Fatalf("%+v %v", rows, err)
	}
	if _, err := store.ListMetadata(context.Background(), "development"); !errors.Is(err, secretstore.ErrValidation) {
		t.Fatalf("got %v", err)
	}
}
