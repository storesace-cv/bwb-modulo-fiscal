// Package secretstore implements DEC-BO-001 plano B write-only secret refs (simulator).
//
// NOT real AGT credentials. NOT production KMS. Admin Metadata never returns plaintext.
package secretstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	EnvHomologation = "homologation"
	EnvProduction   = "production"

	StatusAbsent   = "absent"
	StatusPresent  = "present"
	StatusRotating = "rotating"
	StatusRevoked  = "revoked"
)

var (
	// ErrValidation is fail-closed input validation.
	ErrValidation = errors.New("secretstore: validação")
	// ErrWriteOnly is returned when plaintext read is requested via admin path.
	ErrWriteOnly = errors.New("secretstore: write-only (sem leitura de segredo)")
	// ErrNotFound is returned when a ref has no material.
	ErrNotFound = errors.New("secretstore: não encontrado")
	// ErrEnvIsolation rejects cross-environment copy.
	ErrEnvIsolation = errors.New("secretstore: isolamento HML/PRD")
	// ErrRevoked rejects use of revoked material.
	ErrRevoked = errors.New("secretstore: revogado")
)

// Ref identifies a secret slot (never the secret itself).
type Ref struct {
	Kind        string // e.g. producer_credential | producer_key | taxpayer_key
	Environment string // homologation | production
	SubjectID   string // taxpayer_id or "platform"
	Name        string // logical name
}

func (r Ref) Key() string {
	return strings.Join([]string{
		strings.TrimSpace(r.Kind),
		strings.TrimSpace(r.Environment),
		strings.TrimSpace(r.SubjectID),
		strings.TrimSpace(r.Name),
	}, "/")
}

// Metadata is the only view allowed to plano A / admin UI.
type Metadata struct {
	Ref            Ref
	Status         string
	Fingerprint    string // sha256 hex of plaintext (safe to show; not the secret)
	ExpiresAt      *time.Time
	LastVerifiedAt *time.Time
	Version        int
	Environment    string
}

// PutResult is returned after write-only provision (no plaintext).
type PutResult struct {
	Metadata Metadata
}

// AdminView is the plano B / plano A metadata surface: write-only + sanitized reads.
type AdminView interface {
	Put(ctx context.Context, ref Ref, plaintext []byte, expiresAt *time.Time) (PutResult, error)
	Rotate(ctx context.Context, ref Ref, plaintext []byte, expiresAt *time.Time) (PutResult, error)
	Revoke(ctx context.Context, ref Ref) (Metadata, error)
	Metadata(ctx context.Context, ref Ref) (Metadata, error)
	// Reveal is intentionally absent on AdminView.
}

// RuntimeReveal is reserved for the fiscal core (not backoffice). Simulator only.
type RuntimeReveal interface {
	Reveal(ctx context.Context, ref Ref) ([]byte, error)
}

// Memory is an in-process AES-GCM simulator (ephemeral process key).
type Memory struct {
	mu      sync.Mutex
	aead    cipher.AEAD
	entries map[string]entry
	now     func() time.Time
}

type entry struct {
	nonce      []byte
	ciphertext []byte
	meta       Metadata
}

// NewMemorySimulator builds a fail-closed in-memory vault. Process key is ephemeral.
func NewMemorySimulator(now func() time.Time) (*Memory, error) {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Memory{aead: aead, entries: map[string]entry{}, now: now}, nil
}

// Put provisions a secret write-only. Returns metadata only.
func (m *Memory) Put(ctx context.Context, ref Ref, plaintext []byte, expiresAt *time.Time) (PutResult, error) {
	if err := ctx.Err(); err != nil {
		return PutResult{}, err
	}
	if err := validateRef(ref); err != nil {
		return PutResult{}, err
	}
	if len(plaintext) == 0 {
		return PutResult{}, fmt.Errorf("%w: plaintext vazio", ErrValidation)
	}
	if len(plaintext) > MaxBytesForKind(ref.Kind) {
		return PutResult{}, fmt.Errorf("%w: plaintext demasiado grande", ErrValidation)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ref.Key()
	if e, ok := m.entries[key]; ok && e.meta.Status == StatusPresent {
		return PutResult{}, fmt.Errorf("%w: já presente (use Rotate)", ErrValidation)
	}
	meta, err := m.writeLocked(ref, plaintext, expiresAt, 1, StatusPresent)
	if err != nil {
		return PutResult{}, err
	}
	return PutResult{Metadata: meta}, nil
}

// Rotate replaces material; previous ciphertext is discarded (simulator).
func (m *Memory) Rotate(ctx context.Context, ref Ref, plaintext []byte, expiresAt *time.Time) (PutResult, error) {
	if err := ctx.Err(); err != nil {
		return PutResult{}, err
	}
	if err := validateRef(ref); err != nil {
		return PutResult{}, err
	}
	if len(plaintext) == 0 {
		return PutResult{}, fmt.Errorf("%w: plaintext vazio", ErrValidation)
	}
	if len(plaintext) > MaxBytesForKind(ref.Kind) {
		return PutResult{}, fmt.Errorf("%w: plaintext demasiado grande", ErrValidation)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ref.Key()
	prev := m.entries[key]
	ver := 1
	if prev.meta.Version > 0 {
		ver = prev.meta.Version + 1
	}
	meta, err := m.writeLocked(ref, plaintext, expiresAt, ver, StatusPresent)
	if err != nil {
		return PutResult{}, err
	}
	return PutResult{Metadata: meta}, nil
}

// Revoke marks the ref revoked and wipes ciphertext.
func (m *Memory) Revoke(ctx context.Context, ref Ref) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if err := validateRef(ref); err != nil {
		return Metadata{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := ref.Key()
	e, ok := m.entries[key]
	if !ok {
		return Metadata{}, ErrNotFound
	}
	e.nonce = nil
	e.ciphertext = nil
	e.meta.Status = StatusRevoked
	e.meta.Fingerprint = ""
	m.entries[key] = e
	return e.meta, nil
}

// Metadata returns sanitized fields only (never plaintext).
func (m *Memory) Metadata(ctx context.Context, ref Ref) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	if err := validateRef(ref); err != nil {
		return Metadata{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[ref.Key()]
	if !ok {
		return Metadata{Ref: ref, Status: StatusAbsent, Environment: ref.Environment}, nil
	}
	return e.meta, nil
}

// Reveal returns plaintext for runtime only. Admin UI must not call this.
func (m *Memory) Reveal(ctx context.Context, ref Ref) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRef(ref); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.entries[ref.Key()]
	if !ok {
		return nil, ErrNotFound
	}
	if e.meta.Status == StatusRevoked || len(e.ciphertext) == 0 {
		return nil, ErrRevoked
	}
	plain, err := m.aead.Open(nil, e.nonce, e.ciphertext, []byte(ref.Key()))
	if err != nil {
		return nil, err
	}
	now := m.now().UTC()
	e.meta.LastVerifiedAt = &now
	m.entries[ref.Key()] = e
	out := make([]byte, len(plain))
	copy(out, plain)
	return out, nil
}

// AdminRevealDenied always fails — documents that plano A/B admin cannot read secrets.
func AdminRevealDenied(_ context.Context, _ Ref) ([]byte, error) {
	return nil, ErrWriteOnly
}

// CopyAcrossEnvironments is rejected (HML≠PRD).
func (m *Memory) CopyAcrossEnvironments(_ context.Context, from, to Ref) error {
	if err := validateRef(from); err != nil {
		return err
	}
	if err := validateRef(to); err != nil {
		return err
	}
	if from.Environment == to.Environment {
		return fmt.Errorf("%w: ambientes iguais", ErrValidation)
	}
	return fmt.Errorf("%w: cópia %s→%s proibida", ErrEnvIsolation, from.Environment, to.Environment)
}

func (m *Memory) writeLocked(ref Ref, plaintext []byte, expiresAt *time.Time, version int, status string) (Metadata, error) {
	nonce := make([]byte, m.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return Metadata{}, err
	}
	ct := m.aead.Seal(nil, nonce, plaintext, []byte(ref.Key()))
	sum := sha256.Sum256(plaintext)
	fp := hex.EncodeToString(sum[:])
	meta := Metadata{
		Ref:         ref,
		Status:      status,
		Fingerprint: fp,
		ExpiresAt:   expiresAt,
		Version:     version,
		Environment: ref.Environment,
	}
	m.entries[ref.Key()] = entry{nonce: nonce, ciphertext: ct, meta: meta}
	return meta, nil
}

func validateRef(ref Ref) error {
	kind := strings.TrimSpace(ref.Kind)
	env := strings.TrimSpace(ref.Environment)
	subj := strings.TrimSpace(ref.SubjectID)
	name := strings.TrimSpace(ref.Name)
	if kind == "" || subj == "" || name == "" {
		return fmt.Errorf("%w: kind/subject/name obrigatórios", ErrValidation)
	}
	if !ValidKind(kind) {
		return fmt.Errorf("%w: kind não admitido", ErrValidation)
	}
	switch env {
	case EnvHomologation, EnvProduction:
	default:
		return fmt.Errorf("%w: environment deve ser homologation|production", ErrValidation)
	}
	return nil
}

// Ensure Memory implements both surfaces.
var (
	_ AdminView     = (*Memory)(nil)
	_ RuntimeReveal = (*Memory)(nil)
)
