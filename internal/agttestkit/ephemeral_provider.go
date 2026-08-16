package agttestkit

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

// OpenEphemeralProducerProvider creates a single in-memory RSA producer key for
// future software/producer signing tests. The key is generated per call, never
// persisted, and must not be confused with taxpayer keys from the AGT workbook.
func OpenEphemeralProducerProvider(bits int) (IdentityProvider, error) {
	if bits < MinRSABits {
		bits = MinRSABits
	}
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("agttestkit: ephemeral producer key: %w", err)
	}
	ref := opaqueRefFromPublic(&priv.PublicKey)
	// Domain-separated listing prefix via role; ref stays opaque (same scheme).
	held := []heldIdentity{{
		ref:  ref,
		role: RoleProducerEphemeral,
		bits: priv.N.BitLen(),
		priv: priv,
	}}
	p, err := newMemoryProvider(held)
	if err != nil {
		wipeHeld(held)
		return nil, err
	}
	return p, nil
}
