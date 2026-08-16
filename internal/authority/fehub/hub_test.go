package fehub_test

import (
	"errors"
	"strings"
	"testing"
	"unsafe"

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
	// Markers are synthetic (no high-entropy tokens) but must match reject heuristics.
	cases := []fehub.MetadataSlots{
		{CredentialRef: "-----BEGIN PRIVATE KEY-----"},
		{CredentialRef: "Bearer SYNTHETIC_NOT_A_JWT"},
		{CredentialRef: "bearer abc.def.ghi"},
		{CredentialRef: "Basic SYNTHETIC_USERPASS"},
		{CredentialRef: "password=SYNTHETIC"},
		{EndpointBaseRef: "postgres://user:pass@localhost/db"},
		{CertificateRef: "api_key=SYNTHETIC_PLACEHOLDER"},
		{ValidityNote: "authorization: Bearer x"},
		{CredentialRef: "user:secret@host"},
		{CredentialRef: "eyJ_SYNTHETIC_NOT_A_REAL_TOKEN"},
		{CredentialRef: "bwb_sbox_" + strings.Repeat("Z", 43)},
		{CredentialRef: " secretstore:producer_cred_ref"},
		{CredentialRef: "secretstore:producer_cred_ref "},
	}
	for i, slots := range cases {
		if _, err := h.WithSlots(slots); !errors.Is(err, fehub.ErrSecretRejected) {
			t.Fatalf("case %d: want ErrSecretRejected, got %v", i, err)
		}
	}
	ok, err := h.WithSlots(fehub.MetadataSlots{CredentialRef: "secretstore:producer_cred_ref"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(ok.View().Note, "BEGIN") {
		t.Fatal("leak")
	}
}

func TestViewScrubsBypassMutation(t *testing.T) {
	// Simulate reflective/unsafe bypass of WithSlots (RM-FEFIX-006 #2).
	h := fehub.NewFixture()
	// rawHub MUST mirror unexported fehub.Hub field order/types exactly (kind, slots, note).
	type rawHub struct {
		kind  fehub.Kind
		slots fehub.MetadataSlots
		note  string
	}
	rp := (*rawHub)(unsafe.Pointer(&h))
	rp.slots.CredentialRef = "Bearer SYNTHETIC_NOT_A_JWT"
	v := h.View()
	if v.Slots.CredentialRef != "" {
		t.Fatalf("View must scrub bypassed secret, got %q", v.Slots.CredentialRef)
	}
	if v.ExternalVerified {
		t.Fatal("external_verified")
	}
}
