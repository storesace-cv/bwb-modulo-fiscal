// Package fehub holds sanitized FE authority environment metadata (RM-FEFIX-005).
//
// Kinds: fixture_agt (BWB-MOCK active) | homologation_agt | production_agt (reserved).
// Never stores secrets. external_verified is always false. mock≠HML≠PRD≠homologação AGT.
package fehub

import (
	"errors"
	"fmt"
	"strings"
)

// Kind selects which authority metadata lane is configured.
type Kind string

const (
	KindFixture = Kind("fixture_agt")
	KindHML     = Kind("homologation_agt")
	KindPRD     = Kind("production_agt")
)

var (
	ErrUnknownKind     = errors.New("fehub: unknown kind")
	ErrTransportDenied = errors.New("fehub: transport denied for kind")
	ErrSecretRejected  = errors.New("fehub: plaintext secret rejected")
)

// MetadataSlots are future-facing refs only (names/pointers — never secret values).
type MetadataSlots struct {
	EndpointBaseRef         string `json:"endpoint_base_ref,omitempty"`
	CredentialRef           string `json:"credential_ref,omitempty"`
	CertificateRef          string `json:"certificate_ref,omitempty"`
	SoftwareValidationNoRef string `json:"software_validation_no_ref,omitempty"`
	ValidityNote            string `json:"validity_note,omitempty"`
}

// Hub is the in-memory metadata hub for one FE authority lane.
type Hub struct {
	Kind  Kind
	Slots MetadataSlots
	Note  string
}

// NewFixture returns the only transport-enabled hub (BWB-MOCK / local fixture).
func NewFixture() Hub {
	return Hub{
		Kind: KindFixture,
		Note: "fixture_agt=BWB-MOCK local; ≠ HML; ≠ PRD; ≠ homologação AGT; external_verified=false",
	}
}

// NewReserved builds a HML/PRD hub with empty slots (transport fail-closed).
func NewReserved(kind Kind) (Hub, error) {
	switch kind {
	case KindHML, KindPRD:
		return Hub{
			Kind: kind,
			Note: string(kind) + " reserved until official credentials/endpoints; transport denied; external_verified=false",
		}, nil
	default:
		return Hub{}, fmt.Errorf("%w: %s", ErrUnknownKind, kind)
	}
}

// WithSlots copies hub with validated slots (reject values that look like secrets).
func (h Hub) WithSlots(slots MetadataSlots) (Hub, error) {
	if err := rejectSecrets(slots); err != nil {
		return Hub{}, err
	}
	h.Slots = slots
	return h, nil
}

func rejectSecrets(s MetadataSlots) error {
	for _, v := range []string{s.EndpointBaseRef, s.CredentialRef, s.CertificateRef, s.SoftwareValidationNoRef, s.ValidityNote} {
		low := strings.ToLower(v)
		if strings.Contains(low, "begin ") || strings.Contains(low, "password") ||
			strings.Contains(low, "basic ") || strings.Contains(v, "-----") {
			return ErrSecretRejected
		}
	}
	return nil
}

// AllowTransport is true only for fixture_agt (local BWB-MOCK).
func (h Hub) AllowTransport() bool {
	return h.Kind == KindFixture
}

// AssertTransportAllowed fails closed for HML/PRD/unknown.
func (h Hub) AssertTransportAllowed() error {
	if !h.AllowTransport() {
		return fmt.Errorf("%w: %s (use fixture_agt for BWB-MOCK only)", ErrTransportDenied, h.Kind)
	}
	return nil
}

// PublicView is owner-safe metadata (no secrets; external_verified always false).
type PublicView struct {
	Kind             string        `json:"kind"`
	TransportAllowed bool          `json:"transport_allowed"`
	ExternalVerified bool          `json:"external_verified"`
	Slots            MetadataSlots `json:"slots"`
	Note             string        `json:"note"`
	Labels           []string      `json:"labels"`
}

// View returns a sanitized snapshot.
func (h Hub) View() PublicView {
	return PublicView{
		Kind:             string(h.Kind),
		TransportAllowed: h.AllowTransport(),
		ExternalVerified: false,
		Slots:            h.Slots,
		Note:             h.Note,
		Labels: []string{
			"mock≠HML≠PRD",
			"fixture_ok≠AGT_accepted",
			"pending_validation_sources",
		},
	}
}
