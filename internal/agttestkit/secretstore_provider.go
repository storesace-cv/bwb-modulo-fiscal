package agttestkit

import (
	"context"
	"fmt"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

// SecretStorePEMBinding maps an opaque identity ref to a SecretStore slot that
// holds PEM private-key bytes (taxpayer_key / producer_key). Used to prove the
// IdentityProvider interface can be backed by SecretStore without changing consumers.
type SecretStorePEMBinding struct {
	OpaqueRef string
	StoreRef  secretstore.Ref
	Role      string
}

// OpenSecretStorePEMProvider loads PEM private keys via RuntimeReveal into memory.
// Fail-closed: any missing/invalid binding aborts with no usable provider.
// This is an adapter for tests and future swap — not real AGT credentials.
func OpenSecretStorePEMProvider(ctx context.Context, reveal secretstore.RuntimeReveal, bindings []SecretStorePEMBinding) (IdentityProvider, error) {
	if reveal == nil {
		return nil, fmt.Errorf("%w: reveal required", ErrSignerUnavailable)
	}
	if len(bindings) == 0 {
		return nil, ErrNoIdentities
	}
	held := make([]heldIdentity, 0, len(bindings))
	seen := map[string]struct{}{}
	for _, b := range bindings {
		ref := normalizeRef(b.OpaqueRef)
		if ref == "" {
			wipeHeld(held)
			return nil, ErrRefRequired
		}
		if _, ok := seen[ref]; ok {
			wipeHeld(held)
			return nil, fmt.Errorf("%w", ErrDuplicateRef)
		}
		role := b.Role
		if role == "" {
			role = RoleSecretStoreAdapter
		}
		pemBytes, err := reveal.Reveal(ctx, b.StoreRef)
		if err != nil {
			wipeHeld(held)
			return nil, fmt.Errorf("%w", ErrSignerUnavailable)
		}
		priv, err := parseRSAPrivate(pemBytes)
		zeroBytes(pemBytes)
		if err != nil {
			wipeHeld(held)
			return nil, fmt.Errorf("%w", ErrInvalidPrivatePEM)
		}
		bits := priv.N.BitLen()
		if bits < MinRSABits {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w", ErrRSATooSmall)
		}
		// Stable opaque ref must match material; reject caller mismatch.
		want := opaqueRefFromPublic(&priv.PublicKey)
		if ref != want {
			wipeRSAPrivate(priv)
			wipeHeld(held)
			return nil, fmt.Errorf("%w", ErrRefAmbiguous)
		}
		seen[ref] = struct{}{}
		held = append(held, heldIdentity{ref: ref, role: role, bits: bits, priv: priv})
	}
	p, err := newMemoryProvider(held)
	if err != nil {
		wipeHeld(held)
		return nil, err
	}
	return p, nil
}
