package secretstore

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// MasterKeyBytes is AES-256 key length.
	MasterKeyBytes = 32
	// CipherAlgAES256GCM is the only admitted durable cipher (scaffolding ≠ KMS).
	CipherAlgAES256GCM = "AES-256-GCM"
)

// ParseMasterKey decodes FISCAL_SECRETSTORE_MASTER_KEY (base64 or hex, 32 bytes).
// Never log or return the key material in error messages.
func ParseMasterKey(raw string) ([]byte, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, fmt.Errorf("%w: master key ausente", ErrValidation)
	}
	var key []byte
	var err error
	switch {
	case len(s) == MasterKeyBytes*2 && isHex(s):
		key, err = hex.DecodeString(s)
	default:
		key, err = base64.StdEncoding.DecodeString(s)
		if err != nil {
			key, err = base64.RawStdEncoding.DecodeString(s)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("%w: master key encoding inválido", ErrValidation)
	}
	if len(key) != MasterKeyBytes {
		return nil, fmt.Errorf("%w: master key deve ter %d bytes", ErrValidation, MasterKeyBytes)
	}
	out := make([]byte, MasterKeyBytes)
	copy(out, key)
	return out, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	if len(key) != MasterKeyBytes {
		return nil, fmt.Errorf("%w: master key length", ErrValidation)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: cipher", ErrValidation)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: gcm", ErrValidation)
	}
	return aead, nil
}

func isHex(s string) bool {
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
