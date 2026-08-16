// Package fejws implements compact JWS RS256 for FE preparation (RM-FEFIX-003).
//
// Generic engine only: signs/verifies exact payload bytes. Does not invent FE
// claim sets, JWT/JOSE typ defaults, or SAF-T hash chains (see signsep / C-SIGN-001).
// Sources cited by profiles remain pending_validation.
package fejws

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const Algorithm = "RS256"

var (
	ErrInvalidCompact     = errors.New("fejws: invalid compact jws")
	ErrInvalidHeader      = errors.New("fejws: invalid protected header")
	ErrAlgorithm          = errors.New("fejws: algorithm rejected")
	ErrCritical           = errors.New("fejws: unknown critical header")
	ErrDuplicateHeaderKey = errors.New("fejws: duplicate protected header key")
	ErrEmptySignature     = errors.New("fejws: empty signature")
	ErrVerifyFailed       = errors.New("fejws: signature verify failed")
	ErrSigner             = errors.New("fejws: signer unavailable")
	ErrPublicKey          = errors.New("fejws: public key unavailable")
)

// ProtectedHeader is the JWS protected header. Alg is always RS256 when signing.
// Typ is optional and must be set explicitly by callers/profiles — never defaulted
// (JWT vs JOSE remains open: C-FE-JWS-TYP-001).
type ProtectedHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ,omitempty"`
}

// SignCompact builds header.payload.signature over exact payload bytes (no JSON rewrite).
// extraHeader may set typ or other non-critical keys; alg is forced to RS256.
// Unknown crit values are rejected. Payload is encoded as-is (base64url).
func SignCompact(signer crypto.Signer, payload []byte, header ProtectedHeader) (string, error) {
	if signer == nil {
		return "", ErrSigner
	}
	if header.Alg != "" && header.Alg != Algorithm {
		return "", ErrAlgorithm
	}
	header.Alg = Algorithm
	hb, err := marshalProtectedHeader(header)
	if err != nil {
		return "", err
	}
	encHeader := b64url(hb)
	encPayload := b64url(payload)
	signingInput := encHeader + "." + encPayload
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := signer.Sign(rand.Reader, sum[:], crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("%w", ErrSigner)
	}
	if len(sig) == 0 {
		return "", ErrEmptySignature
	}
	return signingInput + "." + b64url(sig), nil
}

// VerifyCompact verifies a compact JWS RS256 and returns the exact payload bytes.
func VerifyCompact(pub crypto.PublicKey, compact string) ([]byte, error) {
	parts := strings.Split(compact, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidCompact
	}
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return nil, ErrInvalidCompact
	}
	hb, err := b64urlDecodeCanonical(parts[0])
	if err != nil {
		return nil, ErrInvalidCompact
	}
	payload, err := b64urlDecodeCanonical(parts[1])
	if err != nil {
		return nil, ErrInvalidCompact
	}
	sig, err := b64urlDecodeCanonical(parts[2])
	if err != nil {
		return nil, ErrInvalidCompact
	}
	if len(sig) == 0 {
		return nil, ErrEmptySignature
	}
	hdr, err := parseProtectedHeader(hb)
	if err != nil {
		return nil, err
	}
	if hdr.Alg != Algorithm {
		return nil, ErrAlgorithm
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok || rsaPub == nil {
		return nil, ErrPublicKey
	}
	signingInput := parts[0] + "." + parts[1]
	sum := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, sum[:], sig); err != nil {
		return nil, ErrVerifyFailed
	}
	return payload, nil
}

func marshalProtectedHeader(h ProtectedHeader) ([]byte, error) {
	// Deterministic key order: alg first, then typ if present.
	type ordered struct {
		Alg string `json:"alg"`
		Typ string `json:"typ,omitempty"`
	}
	return json.Marshal(ordered{Alg: h.Alg, Typ: h.Typ})
}

func parseProtectedHeader(raw []byte) (ProtectedHeader, error) {
	keys, err := objectKeysRejectDuplicates(raw)
	if err != nil {
		return ProtectedHeader{}, err
	}
	var h ProtectedHeader
	if err := json.Unmarshal(raw, &h); err != nil {
		return ProtectedHeader{}, ErrInvalidHeader
	}
	if h.Alg == "" {
		return ProtectedHeader{}, ErrAlgorithm
	}
	if h.Alg == "none" || h.Alg == "None" || h.Alg == "NONE" {
		return ProtectedHeader{}, ErrAlgorithm
	}
	if h.Alg != Algorithm {
		return ProtectedHeader{}, ErrAlgorithm
	}
	// Reject unknown critical headers if present via raw map.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return ProtectedHeader{}, ErrInvalidHeader
	}
	if critRaw, ok := probe["crit"]; ok {
		var crit []string
		if err := json.Unmarshal(critRaw, &crit); err != nil {
			return ProtectedHeader{}, ErrCritical
		}
		for _, c := range crit {
			if c != "alg" && c != "typ" {
				return ProtectedHeader{}, ErrCritical
			}
			if _, exists := keys[c]; !exists {
				return ProtectedHeader{}, ErrCritical
			}
		}
	}
	return h, nil
}

func objectKeysRejectDuplicates(raw []byte) (map[string]struct{}, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, ErrInvalidHeader
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil, ErrInvalidHeader
	}
	keys := map[string]struct{}{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, ErrInvalidHeader
		}
		key, ok := tok.(string)
		if !ok {
			return nil, ErrInvalidHeader
		}
		if _, exists := keys[key]; exists {
			return nil, ErrDuplicateHeaderKey
		}
		keys[key] = struct{}{}
		if err := skipValue(dec); err != nil {
			return nil, ErrInvalidHeader
		}
	}
	tok, err = dec.Token()
	if err != nil {
		return nil, ErrInvalidHeader
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '}' {
		return nil, ErrInvalidHeader
	}
	return keys, nil
}

func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); ok {
		switch delim {
		case '{', '[':
			for dec.More() {
				if delim == '{' {
					if _, err := dec.Token(); err != nil {
						return err
					}
				}
				if err := skipValue(dec); err != nil {
					return err
				}
			}
			_, err = dec.Token()
			return err
		}
	}
	return nil
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func b64urlDecodeCanonical(s string) ([]byte, error) {
	if strings.ContainsAny(s, "=+/") {
		return nil, fmt.Errorf("non-canonical base64url")
	}
	out, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if b64url(out) != s {
		return nil, fmt.Errorf("non-canonical base64url encoding")
	}
	return out, nil
}

// PublicRSA extracts *rsa.PublicKey from a crypto.Signer Public() result.
func PublicRSA(signer crypto.Signer) (*rsa.PublicKey, error) {
	if signer == nil {
		return nil, ErrSigner
	}
	pub := signer.Public()
	rk, ok := pub.(*rsa.PublicKey)
	if !ok || rk == nil {
		return nil, ErrPublicKey
	}
	return rk, nil
}
