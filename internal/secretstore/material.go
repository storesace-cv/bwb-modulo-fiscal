package secretstore

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"unicode/utf8"

	"software.sslmate.com/src/go-pkcs12"
)

// Material kinds admitted for AGT prep (DEC-BO-004). Do not invent AGT endpoints.
const (
	KindProducerCredential = "producer_credential"
	KindProducerKey        = "producer_key"
	KindTaxpayerKey        = "taxpayer_key"
	KindCertificate        = "certificate"
)

// Material encodings supported for import (docs archived / pending_validation).
const (
	EncodingCredential = "credential"
	EncodingPEM        = "pem"
	EncodingPKCS12     = "pkcs12"
)

// Upload size limits (fail-closed).
const (
	MaxCredentialBytes = 8 << 10  // 8 KiB
	MaxPEMBytes        = 64 << 10 // 64 KiB
	MaxPKCS12Bytes     = 256 << 10
)

// MaterialInput is ephemeral ingest (password never persisted).
type MaterialInput struct {
	Kind     string
	Encoding string
	Bytes    []byte
	Password []byte // PKCS#12 unlock only; caller must zero after Prepare
}

// PreparedMaterial is storage-ready bytes + sanitized format note (no password).
type PreparedMaterial struct {
	StorageBytes []byte
	Encoding     string
	FormatNote   string // e.g. pem_private_key | pem_certificate | pkcs12 | credential
}

// Prepare validates encoding/kind/size and returns bytes for SecretStore Put/Rotate.
// Password is used only to verify PKCS#12 and is never copied into PreparedMaterial.
func Prepare(in MaterialInput) (PreparedMaterial, error) {
	kind := strings.TrimSpace(in.Kind)
	enc := strings.TrimSpace(in.Encoding)
	if !ValidKind(kind) {
		return PreparedMaterial{}, fmt.Errorf("%w: kind", ErrValidation)
	}
	if len(in.Bytes) == 0 {
		return PreparedMaterial{}, fmt.Errorf("%w: material vazio", ErrValidation)
	}
	switch enc {
	case EncodingCredential:
		if kind != KindProducerCredential {
			return PreparedMaterial{}, fmt.Errorf("%w: encoding credential só para producer_credential", ErrValidation)
		}
		if len(in.Bytes) > MaxCredentialBytes {
			return PreparedMaterial{}, fmt.Errorf("%w: credential demasiado grande", ErrValidation)
		}
		if !utf8.Valid(in.Bytes) {
			return PreparedMaterial{}, fmt.Errorf("%w: credential não UTF-8", ErrValidation)
		}
		out := append([]byte(nil), in.Bytes...)
		return PreparedMaterial{StorageBytes: out, Encoding: enc, FormatNote: "credential"}, nil
	case EncodingPEM:
		if kind != KindProducerKey && kind != KindTaxpayerKey && kind != KindCertificate {
			return PreparedMaterial{}, fmt.Errorf("%w: pem incompatível com kind", ErrValidation)
		}
		if len(in.Bytes) > MaxPEMBytes {
			return PreparedMaterial{}, fmt.Errorf("%w: PEM demasiado grande", ErrValidation)
		}
		note, err := validatePEM(in.Bytes, kind)
		if err != nil {
			return PreparedMaterial{}, err
		}
		out := append([]byte(nil), in.Bytes...)
		return PreparedMaterial{StorageBytes: out, Encoding: enc, FormatNote: note}, nil
	case EncodingPKCS12:
		if kind != KindProducerKey && kind != KindTaxpayerKey && kind != KindCertificate {
			return PreparedMaterial{}, fmt.Errorf("%w: pkcs12 incompatível com kind", ErrValidation)
		}
		if len(in.Bytes) > MaxPKCS12Bytes {
			return PreparedMaterial{}, fmt.Errorf("%w: PKCS#12 demasiado grande", ErrValidation)
		}
		if err := validatePKCS12(in.Bytes, in.Password); err != nil {
			return PreparedMaterial{}, err
		}
		out := append([]byte(nil), in.Bytes...)
		return PreparedMaterial{StorageBytes: out, Encoding: enc, FormatNote: "pkcs12"}, nil
	default:
		return PreparedMaterial{}, fmt.Errorf("%w: encoding (credential|pem|pkcs12)", ErrValidation)
	}
}

// ValidKind reports whether kind is in the SecAdm allowlist.
func ValidKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case KindProducerCredential, KindProducerKey, KindTaxpayerKey, KindCertificate:
		return true
	default:
		return false
	}
}

// MaxBytesForKind returns the fail-closed size cap for JSON plaintext puts by kind.
func MaxBytesForKind(kind string) int {
	switch strings.TrimSpace(kind) {
	case KindProducerCredential:
		return MaxCredentialBytes
	case KindProducerKey, KindTaxpayerKey, KindCertificate:
		return MaxPEMBytes
	default:
		return MaxCredentialBytes
	}
}

func validatePEM(raw []byte, kind string) (string, error) {
	rest := raw
	var blocks int
	var note string
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		blocks++
		switch block.Type {
		case "CERTIFICATE":
			if _, err := x509.ParseCertificate(block.Bytes); err != nil {
				return "", fmt.Errorf("%w: certificado PEM inválido", ErrValidation)
			}
			if note == "" {
				note = "pem_certificate"
			}
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			switch block.Type {
			case "PRIVATE KEY":
				if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
					return "", fmt.Errorf("%w: chave PEM PKCS#8 inválida", ErrValidation)
				}
			case "RSA PRIVATE KEY":
				if _, err := x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
					return "", fmt.Errorf("%w: chave PEM PKCS#1 inválida", ErrValidation)
				}
			case "EC PRIVATE KEY":
				if _, err := x509.ParseECPrivateKey(block.Bytes); err != nil {
					return "", fmt.Errorf("%w: chave PEM EC inválida", ErrValidation)
				}
			}
			note = "pem_private_key"
		default:
			return "", fmt.Errorf("%w: tipo PEM %q não admitido (pending_external)", ErrValidation, block.Type)
		}
	}
	if blocks == 0 {
		return "", fmt.Errorf("%w: PEM sem blocos", ErrValidation)
	}
	switch kind {
	case KindCertificate:
		if note != "pem_certificate" {
			return "", fmt.Errorf("%w: certificate exige PEM CERTIFICATE", ErrValidation)
		}
	case KindProducerKey, KindTaxpayerKey:
		if note != "pem_private_key" {
			return "", fmt.Errorf("%w: key exige PEM PRIVATE KEY", ErrValidation)
		}
	}
	return note, nil
}

func validatePKCS12(raw, password []byte) error {
	if len(raw) < 4 || raw[0] != 0x30 {
		return fmt.Errorf("%w: PKCS#12 não parece DER", ErrValidation)
	}
	// Password required to prove unlockability; never stored.
	if len(password) == 0 {
		return fmt.Errorf("%w: password PKCS#12 obrigatória (efémera)", ErrValidation)
	}
	key, cert, err := pkcs12.Decode(raw, string(password))
	if err != nil {
		_, _, _, err2 := pkcs12.DecodeChain(raw, string(password))
		if err2 != nil {
			return fmt.Errorf("%w: PKCS#12 inválido ou password incorrecta", ErrValidation)
		}
		return nil
	}
	if key == nil && cert == nil {
		return fmt.Errorf("%w: PKCS#12 vazio", ErrValidation)
	}
	return nil
}

// ZeroBytes overwrites a buffer (password / plaintext).
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// LooksLikePEM is a cheap check for UI/API routing hints (not a validator).
func LooksLikePEM(b []byte) bool {
	return bytes.Contains(b, []byte("-----BEGIN "))
}
