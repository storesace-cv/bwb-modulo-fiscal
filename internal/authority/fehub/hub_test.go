package fehub_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/authority/fehub"
)

func TestFixtureAllowsTransport(t *testing.T) {
	h := fehub.NewFixture()
	if err := h.AssertTransportAllowed(); err != nil {
		t.Fatal(err)
	}
	v := h.View()
	if v.ExternalVerified || !v.TransportAllowed || v.Kind != string(fehub.KindFixture) {
		t.Fatalf("%+v", v)
	}
}

func TestReservedDeniesTransport(t *testing.T) {
	for _, k := range []fehub.Kind{fehub.KindHML, fehub.KindPRD} {
		h, err := fehub.NewReserved(k)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.AssertTransportAllowed(); !errors.Is(err, fehub.ErrTransportDenied) {
			t.Fatalf("%s: %v", k, err)
		}
	}
}

func TestRejectPlaintextSecretsInSlots(t *testing.T) {
	h := fehub.NewFixture()
	_, err := h.WithSlots(fehub.MetadataSlots{CredentialRef: "-----BEGIN PRIVATE KEY-----"})
	if !errors.Is(err, fehub.ErrSecretRejected) {
		t.Fatalf("%v", err)
	}
	ok, err := h.WithSlots(fehub.MetadataSlots{CredentialRef: "secretstore:producer_cred_ref"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ok.View().Note, "BEGIN") {
		t.Fatal("leak")
	}
}
