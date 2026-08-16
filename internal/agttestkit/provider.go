package agttestkit

import (
	"crypto"
	"crypto/rsa"
	"errors"
)

// Identity roles for sanitized listings (RM-FEFIX-002).
const (
	RoleTaxpayerTest       = "taxpayer_test"
	RoleProducerEphemeral  = "producer_ephemeral"
	RoleSecretStoreAdapter = "secretstore_adapter"
)

var (
	ErrRefRequired       = errors.New("agttestkit: identity ref required")
	ErrRefNotFound       = errors.New("agttestkit: identity ref not found")
	ErrRefAmbiguous      = errors.New("agttestkit: identity ref ambiguous")
	ErrProviderClosed    = errors.New("agttestkit: identity provider closed")
	ErrDuplicateRef      = errors.New("agttestkit: duplicate opaque identity ref")
	ErrSignerUnavailable = errors.New("agttestkit: signer unavailable")
	ErrVerifyFailed      = errors.New("agttestkit: signature verify failed")
)

// SanitizedRef is the only consumer-facing identity metadata (no PEM/NIF/name/fingerprint).
type SanitizedRef struct {
	Ref       string `json:"ref"`
	Algorithm string `json:"algorithm"`
	RSABits   int    `json:"rsa_bits"`
	Role      string `json:"role"`
}

// IdentityProvider resolves opaque refs to in-memory RSA crypto.Signer values.
// Workbook, ephemeral producer, and future SecretStore adapters implement this
// so consumers do not depend on custody source.
type IdentityProvider interface {
	List() []SanitizedRef
	Signer(ref string) (crypto.Signer, error)
	// Verify checks RSA-SHA256 PKCS#1 v1.5 over SHA-256(message).
	Verify(ref string, message, signature []byte) error
	Close() error
}

// rsaPrivateKey is satisfied by *rsa.PrivateKey (crypto.Signer).
var _ crypto.Signer = (*rsa.PrivateKey)(nil)
