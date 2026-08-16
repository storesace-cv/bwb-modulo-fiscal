// Package fehub holds sanitized FE authority environment metadata (RM-FEFIX-005/006).
//
// Kinds: fixture_agt (BWB-MOCK active) | homologation_agt | production_agt (reserved).
// Never stores secrets. external_verified is always false. mock≠HML≠PRD≠homologação AGT.
package fehub

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
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
// Fields are unexported so callers must use constructors / WithSlots (RM-FEFIX-006).
type Hub struct {
	kind  Kind
	slots MetadataSlots
	note  string
}

// Kind returns the authority lane.
func (h Hub) Kind() Kind { return h.kind }

// NewFixture returns the only transport-enabled hub (BWB-MOCK / local fixture).
func NewFixture() Hub {
	return Hub{
		kind: KindFixture,
		note: "fixture_agt=BWB-MOCK local; ≠ HML; ≠ PRD; ≠ homologação AGT; external_verified=false",
	}
}

// NewReserved builds a HML/PRD hub with empty slots (transport fail-closed).
func NewReserved(kind Kind) (Hub, error) {
	switch kind {
	case KindHML, KindPRD:
		return Hub{
			kind: kind,
			note: string(kind) + " reserved until official credentials/endpoints; transport denied; external_verified=false",
		}, nil
	default:
		return Hub{}, fmt.Errorf("%w: %s", ErrUnknownKind, kind)
	}
}

// WithSlots copies hub with validated slots (reject values that look like secrets).
func (h Hub) WithSlots(slots MetadataSlots) (Hub, error) {
	if err := validateSlots(slots); err != nil {
		return Hub{}, err
	}
	h.slots = slots
	return h, nil
}

func validateSlots(s MetadataSlots) error {
	for _, v := range []string{s.EndpointBaseRef, s.CredentialRef, s.CertificateRef, s.SoftwareValidationNoRef, s.ValidityNote} {
		if err := validateSlotValue(v); err != nil {
			return err
		}
	}
	return nil
}

func validateSlotValue(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if looksLikeSecret(v) || !isSafeRefCharset(v) {
		return ErrSecretRejected
	}
	return nil
}

func looksLikeSecret(v string) bool {
	low := strings.ToLower(v)
	if strings.Contains(v, "-----") || strings.Contains(low, "begin ") {
		return true
	}
	needles := []string{
		"password", "passwd", "private key", "privatekey",
		"basic ", "bearer ", "authorization", "api_key", "apikey", "access_token",
		"refresh_token", "client_secret", "dsn", "postgres://", "postgresql://",
		"mysql://", "mongodb://", "redis://", "amqp://", "jdbc:",
	}
	for _, n := range needles {
		if strings.Contains(low, n) {
			return true
		}
	}
	// "secret" alone, but allow opaque refs like secretstore:…
	if strings.Contains(low, "secret") && !strings.Contains(low, "secretstore") {
		return true
	}
	if strings.HasPrefix(low, "bearer") || strings.HasPrefix(low, "basic") {
		return true
	}
	// JWT-ish / opaque token blobs
	if strings.HasPrefix(v, "eyJ") || strings.HasPrefix(low, "eyj") {
		return true
	}
	if strings.Contains(v, "@") && strings.Contains(v, ":") {
		return true // user:pass@host style
	}
	if strings.Contains(low, "://") {
		return true
	}
	return false
}

func isSafeRefCharset(v string) bool {
	if len(v) > 128 {
		return false
	}
	for _, r := range v {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case r == '_' || r == '-' || r == '.' || r == ':' || r == '/':
		default:
			return false
		}
	}
	return true
}

// AllowTransport is true only for fixture_agt (local BWB-MOCK).
func (h Hub) AllowTransport() bool {
	return h.kind == KindFixture
}

// AssertTransportAllowed fails closed for HML/PRD/unknown.
func (h Hub) AssertTransportAllowed() error {
	if !h.AllowTransport() {
		return fmt.Errorf("%w: %s (use fixture_agt for BWB-MOCK only)", ErrTransportDenied, h.kind)
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

// View returns a sanitized snapshot. Invalid/secret-like slots are scrubbed fail-closed.
func (h Hub) View() PublicView {
	slots := h.slots
	if err := validateSlots(slots); err != nil {
		slots = MetadataSlots{}
	}
	return PublicView{
		Kind:             string(h.kind),
		TransportAllowed: h.AllowTransport(),
		ExternalVerified: false,
		Slots:            slots,
		Note:             h.note,
		Labels: []string{
			"mock≠HML≠PRD",
			"fixture_ok≠AGT_accepted",
			"pending_validation_sources",
		},
	}
}
