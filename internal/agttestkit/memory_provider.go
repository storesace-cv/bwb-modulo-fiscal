package agttestkit

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
)

// heldIdentity is private custody state. taxpayerNIF and sourceLabel bind the
// workbook row's entity designation to the key pair. They are never exported,
// listed, logged, JSON-tagged, or persisted.
type heldIdentity struct {
	ref         string
	role        string
	bits        int
	taxpayerNIF string // unexported; no json tags
	sourceLabel string // origin designation from NOME column (not a tax regime)
	priv        *rsa.PrivateKey
}

// memoryProvider is the shared in-memory custody backend for workbook, ephemeral,
// and SecretStore-adapter constructors.
type memoryProvider struct {
	mu     sync.RWMutex
	closed bool
	byRef  map[string]*heldIdentity
	order  []string
}

func newMemoryProvider(held []heldIdentity) (*memoryProvider, error) {
	p := &memoryProvider{byRef: make(map[string]*heldIdentity, len(held))}
	for i := range held {
		h := held[i]
		if h.ref == "" || h.priv == nil {
			wipeHeld(held)
			return nil, ErrSignerUnavailable
		}
		if _, ok := p.byRef[h.ref]; ok {
			wipeHeld(held)
			return nil, fmt.Errorf("%w", ErrDuplicateRef)
		}
		cp := h
		p.byRef[h.ref] = &cp
		p.order = append(p.order, h.ref)
	}
	return p, nil
}

func (p *memoryProvider) List() []SanitizedRef {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return nil
	}
	out := make([]SanitizedRef, 0, len(p.order))
	for _, ref := range p.order {
		h := p.byRef[ref]
		if h == nil || h.priv == nil {
			continue
		}
		out = append(out, SanitizedRef{
			Ref:       h.ref,
			Algorithm: "RSA",
			RSABits:   h.bits,
			Role:      h.role,
		})
	}
	return out
}

func (p *memoryProvider) Signer(ref string) (crypto.Signer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return nil, ErrProviderClosed
	}
	ref = normalizeRef(ref)
	if ref == "" {
		return nil, ErrRefRequired
	}
	h, ok := p.byRef[ref]
	if !ok || h == nil || h.priv == nil {
		return nil, ErrRefNotFound
	}
	// Opaque proxy — never return the stored *rsa.PrivateKey.
	return &opaqueSigner{p: p, ref: ref}, nil
}

func (p *memoryProvider) Verify(ref string, message, signature []byte) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return ErrProviderClosed
	}
	ref = normalizeRef(ref)
	if ref == "" {
		return ErrRefRequired
	}
	h, ok := p.byRef[ref]
	if !ok || h == nil || h.priv == nil {
		return ErrRefNotFound
	}
	return verifyRSASHA256(&h.priv.PublicKey, message, signature)
}

func (p *memoryProvider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	for _, h := range p.byRef {
		wipeIdentity(h)
	}
	p.byRef = nil
	p.order = nil
	return nil
}

func (p *memoryProvider) ValidateTaxpayerBinding(ref, suppliedNIF string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return ErrProviderClosed
	}
	ref = normalizeRef(ref)
	if ref == "" {
		return ErrRefRequired
	}
	h, ok := p.byRef[ref]
	if !ok || h == nil || h.priv == nil {
		return ErrRefNotFound
	}
	if h.role != RoleTaxpayerTest {
		return ErrRoleMismatch
	}
	supplied := strings.TrimSpace(suppliedNIF)
	if supplied == "" || h.taxpayerNIF == "" {
		return ErrBindingMismatch
	}
	if supplied != h.taxpayerNIF {
		return ErrBindingMismatch
	}
	return nil
}

func (p *memoryProvider) RequireRole(ref, expectedRole string) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return ErrProviderClosed
	}
	ref = normalizeRef(ref)
	if ref == "" {
		return ErrRefRequired
	}
	expectedRole = strings.TrimSpace(expectedRole)
	if expectedRole == "" {
		return ErrRoleMismatch
	}
	h, ok := p.byRef[ref]
	if !ok || h == nil || h.priv == nil {
		return ErrRefNotFound
	}
	if h.role != expectedRole {
		return ErrRoleMismatch
	}
	return nil
}

// opaqueSigner implements crypto.Signer without exposing the private key object.
// Sign/Public resolve under the provider lock and fail after Close.
type opaqueSigner struct {
	p   *memoryProvider
	ref string
}

var _ crypto.Signer = (*opaqueSigner)(nil)

func (s *opaqueSigner) Public() crypto.PublicKey {
	if s == nil || s.p == nil {
		return nil
	}
	s.p.mu.RLock()
	defer s.p.mu.RUnlock()
	if s.p.closed {
		return nil
	}
	h, ok := s.p.byRef[s.ref]
	if !ok || h == nil || h.priv == nil || h.priv.N == nil {
		return nil
	}
	// Independent public copy — never share the private key's PublicKey pointer graph
	// in a way that could reach D/Primes (rsa.PublicKey has no private fields; still copy N).
	return &rsa.PublicKey{
		N: new(big.Int).Set(h.priv.N),
		E: h.priv.E,
	}
}

func (s *opaqueSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if s == nil || s.p == nil {
		return nil, ErrSignerUnavailable
	}
	s.p.mu.RLock()
	defer s.p.mu.RUnlock()
	if s.p.closed {
		return nil, ErrProviderClosed
	}
	h, ok := s.p.byRef[s.ref]
	if !ok || h == nil || h.priv == nil {
		return nil, ErrSignerUnavailable
	}
	return h.priv.Sign(rand, digest, opts)
}

func normalizeRef(ref string) string {
	return strings.TrimSpace(ref)
}

// SignMessageRSA SHA-256 hashes message then signs with PKCS#1 v1.5 (generic; ≠ AGT JWS).
func SignMessageRSA(signer crypto.Signer, message []byte) ([]byte, error) {
	if signer == nil {
		return nil, ErrSignerUnavailable
	}
	sum := sha256.Sum256(message)
	return signer.Sign(rand.Reader, sum[:], crypto.SHA256)
}

func verifyRSASHA256(pub *rsa.PublicKey, message, signature []byte) error {
	if pub == nil {
		return ErrSignerUnavailable
	}
	sum := sha256.Sum256(message)
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], signature); err != nil {
		return ErrVerifyFailed
	}
	return nil
}

func wipeHeld(held []heldIdentity) {
	for i := range held {
		wipeIdentity(&held[i])
	}
}

func wipeIdentity(h *heldIdentity) {
	if h == nil {
		return
	}
	wipeRSAPrivate(h.priv)
	h.priv = nil
	h.taxpayerNIF = ""
	h.sourceLabel = ""
	h.ref = ""
	h.role = ""
	h.bits = 0
}

func wipeRSAPrivate(k *rsa.PrivateKey) {
	if k == nil {
		return
	}
	if k.D != nil {
		k.D.SetInt64(0)
	}
	for _, p := range k.Primes {
		if p != nil {
			p.SetInt64(0)
		}
	}
	if k.Precomputed.Dp != nil {
		k.Precomputed.Dp.SetInt64(0)
	}
	if k.Precomputed.Dq != nil {
		k.Precomputed.Dq.SetInt64(0)
	}
	if k.Precomputed.Qinv != nil {
		k.Precomputed.Qinv.SetInt64(0)
	}
	k.Precomputed = rsa.PrecomputedValues{}
	if k.N != nil {
		k.N.SetInt64(0)
	}
	k.E = 0
}
