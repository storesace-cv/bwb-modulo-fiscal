// Package secadm enforces DEC-BO-001 plano B: owner-only access to secret administration.
//
// Does not implement MFA UI. Does not store secrets. Wraps secretstore.AdminView.
package secadm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/secretstore"
)

var (
	// ErrUnauthorized is returned when the actor is not the configured owner.
	ErrUnauthorized = errors.New("secadm: não autorizado (owner-only)")
	// ErrValidation is fail-closed config/input validation.
	ErrValidation = errors.New("secadm: validação")
)

// Actor is the authenticated principal calling SecAdm (never a secret).
type Actor struct {
	SubjectID string
}

// Gate authorizes plano B operations for a single owner subject.
type Gate struct {
	ownerID string
	store   secretstore.AdminView
}

// NewGate builds an owner-only gate. ownerSubjectID must be non-empty (e.g. configured operator id).
func NewGate(ownerSubjectID string, store secretstore.AdminView) (*Gate, error) {
	owner := strings.TrimSpace(ownerSubjectID)
	if owner == "" {
		return nil, fmt.Errorf("%w: ownerSubjectID obrigatório", ErrValidation)
	}
	if store == nil {
		return nil, fmt.Errorf("%w: store nil", ErrValidation)
	}
	return &Gate{ownerID: owner, store: store}, nil
}

func (g *Gate) authorize(actor Actor) error {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return fmt.Errorf("%w: actor vazio", ErrUnauthorized)
	}
	if actor.SubjectID != g.ownerID {
		return ErrUnauthorized
	}
	return nil
}

// Put provisions a secret if actor is owner.
func (g *Gate) Put(ctx context.Context, actor Actor, ref secretstore.Ref, plaintext []byte, expiresAt *time.Time) (secretstore.PutResult, error) {
	if err := g.authorize(actor); err != nil {
		return secretstore.PutResult{}, err
	}
	return g.store.Put(ctx, ref, plaintext, expiresAt)
}

// Rotate rotates material if actor is owner.
func (g *Gate) Rotate(ctx context.Context, actor Actor, ref secretstore.Ref, plaintext []byte, expiresAt *time.Time) (secretstore.PutResult, error) {
	if err := g.authorize(actor); err != nil {
		return secretstore.PutResult{}, err
	}
	return g.store.Rotate(ctx, ref, plaintext, expiresAt)
}

// Revoke revokes material if actor is owner.
func (g *Gate) Revoke(ctx context.Context, actor Actor, ref secretstore.Ref) (secretstore.Metadata, error) {
	if err := g.authorize(actor); err != nil {
		return secretstore.Metadata{}, err
	}
	return g.store.Revoke(ctx, ref)
}

// Metadata returns sanitized metadata. Owner-only for SecAdm surface
// (plano A uses a separate metadata adapter later, still without Reveal).
func (g *Gate) Metadata(ctx context.Context, actor Actor, ref secretstore.Ref) (secretstore.Metadata, error) {
	if err := g.authorize(actor); err != nil {
		return secretstore.Metadata{}, err
	}
	return g.store.Metadata(ctx, ref)
}

// ValidateOfflineRefs reveals key+cert ephemerally, validates offline, zeros plaintext.
// Never returns plaintext. ≠ external_verified / AGT.
func (g *Gate) ValidateOfflineRefs(ctx context.Context, actor Actor, keyRef, certRef secretstore.Ref, intermediatesPEM []byte, now time.Time) (secretstore.OfflineReport, error) {
	if err := g.authorize(actor); err != nil {
		return secretstore.OfflineReport{}, err
	}
	revealer, ok := g.store.(secretstore.RuntimeReveal)
	if !ok {
		return secretstore.OfflineReport{}, fmt.Errorf("%w: store sem Reveal runtime", ErrValidation)
	}
	keyPlain, err := revealer.Reveal(ctx, keyRef)
	if err != nil {
		return secretstore.OfflineReport{}, err
	}
	defer secretstore.ZeroBytes(keyPlain)
	certPlain, err := revealer.Reveal(ctx, certRef)
	if err != nil {
		return secretstore.OfflineReport{}, err
	}
	defer secretstore.ZeroBytes(certPlain)
	return secretstore.ValidateOfflineKeyCert(keyPlain, certPlain, intermediatesPEM, now)
}
