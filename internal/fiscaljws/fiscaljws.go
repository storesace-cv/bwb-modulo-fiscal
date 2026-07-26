// Package fiscaljws provides RS256 JWS for the vertical slice with ephemeral RSA keys.
//
// NOT certified. NOT AGT HML/PRD. Does NOT implement official FE-RNG field sets (RM-FE-002).
// Private keys must never be persisted or committed.
package fiscaljws

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// Algorithm is the JWS alg for the slice envelope (technical).
	Algorithm = "RS256"
	// DefaultRSABits is the minimum RSA size for the ephemeral slice key.
	DefaultRSABits = 2048
)

var (
	// ErrInvalidJWS is returned when compact JWS verification fails.
	ErrInvalidJWS = errors.New("fiscaljws: JWS inválido")
)

// Signer holds an ephemeral RSA private key in memory only.
type Signer struct {
	priv *rsa.PrivateKey
}

// NewEphemeral generates a new in-memory RSA key pair. Private key is never written to disk.
func NewEphemeral(bits int) (*Signer, error) {
	if bits < DefaultRSABits {
		bits = DefaultRSABits
	}
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, fmt.Errorf("fiscaljws: generate key: %w", err)
	}
	return &Signer{priv: priv}, nil
}

// PublicKey returns the public half for verification (safe to expose in tests).
func (s *Signer) PublicKey() *rsa.PublicKey {
	if s == nil || s.priv == nil {
		return nil
	}
	return &s.priv.PublicKey
}

// PublicFingerprintSHA256 returns hex(sha256(SPKI DER)) for audit metadata (no private material).
func (s *Signer) PublicFingerprintSHA256() (string, error) {
	pub := s.PublicKey()
	if pub == nil {
		return "", fmt.Errorf("fiscaljws: nil signer")
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("fiscaljws: marshal public: %w", err)
	}
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:]), nil
}

// Envelope is the technical outbox→authority payload (≠ FE document signature fields).
type Envelope struct {
	Iss          string `json:"iss"`
	SubmissionID string `json:"submission_id"`
	DocumentID   string `json:"document_id"`
	Iat          int64  `json:"iat"`
	// Certified is always false for this adapter.
	Certified bool `json:"certified"`
}

// SignEnvelope builds a compact JWS RS256 over the technical envelope.
func (s *Signer) SignEnvelope(submissionID, documentID string, now time.Time) (compact string, err error) {
	if s == nil || s.priv == nil {
		return "", fmt.Errorf("fiscaljws: nil signer")
	}
	if submissionID == "" || documentID == "" {
		return "", fmt.Errorf("fiscaljws: submission_id/document_id obrigatórios")
	}
	env := Envelope{
		Iss:          "bwb-fiscal-slice",
		SubmissionID: submissionID,
		DocumentID:   documentID,
		Iat:          now.UTC().Unix(),
		Certified:    false,
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return s.SignCompact(payload)
}

// SignCompact signs an arbitrary payload as JWS compact serialization (header.payload.signature).
func (s *Signer) SignCompact(payload []byte) (string, error) {
	if s == nil || s.priv == nil {
		return "", fmt.Errorf("fiscaljws: nil signer")
	}
	header := map[string]string{"alg": Algorithm, "typ": "JOSE"}
	hb, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	encHeader := b64url(hb)
	encPayload := b64url(payload)
	signingInput := encHeader + "." + encPayload
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, s.priv, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("fiscaljws: sign: %w", err)
	}
	return signingInput + "." + b64url(sig), nil
}

// VerifyCompact verifies a compact JWS RS256 and returns the payload bytes.
func VerifyCompact(pub *rsa.PublicKey, compact string) ([]byte, error) {
	if pub == nil {
		return nil, fmt.Errorf("fiscaljws: nil public key")
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("%w: partes", ErrInvalidJWS)
	}
	hb, err := b64urlDecode(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: header", ErrInvalidJWS)
	}
	var header map[string]string
	if err := json.Unmarshal(hb, &header); err != nil {
		return nil, fmt.Errorf("%w: header json", ErrInvalidJWS)
	}
	if header["alg"] != Algorithm {
		return nil, fmt.Errorf("%w: alg", ErrInvalidJWS)
	}
	payload, err := b64urlDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: payload", ErrInvalidJWS)
	}
	sig, err := b64urlDecode(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: signature", ErrInvalidJWS)
	}
	signingInput := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		return nil, fmt.Errorf("%w: verify", ErrInvalidJWS)
	}
	return payload, nil
}

// ParseEnvelope verifies JWS and unmarshals the technical envelope.
func ParseEnvelope(pub *rsa.PublicKey, compact string) (Envelope, error) {
	raw, err := VerifyCompact(pub, compact)
	if err != nil {
		return Envelope{}, err
	}
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Envelope{}, fmt.Errorf("%w: envelope json", ErrInvalidJWS)
	}
	if env.Certified {
		return Envelope{}, fmt.Errorf("%w: certified=true proibido neste adaptador", ErrInvalidJWS)
	}
	if env.SubmissionID == "" || env.DocumentID == "" {
		return Envelope{}, fmt.Errorf("%w: envelope incompleto", ErrInvalidJWS)
	}
	return env, nil
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func b64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
