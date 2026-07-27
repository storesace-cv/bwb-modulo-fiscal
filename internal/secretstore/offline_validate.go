package secretstore

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"
)

// OfflineReport is a sanitized offline validation result (≠ AGT / external_verified).
type OfflineReport struct {
	OK                bool
	PairMatch         bool
	ChainOK           bool
	WithinValidity    bool
	PurposeOK         bool
	PurposeNote       string // e.g. pending_external when KU unknown
	FingerprintSHA256 string
	NotBefore         string
	NotAfter          string
	KeyBits           int
	AlgorithmNote     string // RSA|ECDSA + pending_external markers — not JWS claims
	Issues            []string
}

// ValidateOfflineKeyCert checks pair, chain (self or provided intermediates),
// validity window, basic key usage, and cert fingerprint. Password never used here
// (material already decrypted/PEM). Does NOT contact AGT.
func ValidateOfflineKeyCert(keyPEM, certPEM []byte, intermediatesPEM []byte, now time.Time) (OfflineReport, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	rep := OfflineReport{PurposeNote: "pending_external"}
	if len(keyPEM) == 0 || len(certPEM) == 0 {
		return OfflineReport{}, fmt.Errorf("%w: key/cert PEM obrigatórios", ErrValidation)
	}
	priv, err := parsePrivateKeyPEM(keyPEM)
	if err != nil {
		return OfflineReport{}, err
	}
	cert, err := parseCertificatePEM(certPEM)
	if err != nil {
		return OfflineReport{}, err
	}
	sum := sha256.Sum256(cert.Raw)
	rep.FingerprintSHA256 = hex.EncodeToString(sum[:])
	rep.NotBefore = cert.NotBefore.UTC().Format(time.RFC3339)
	rep.NotAfter = cert.NotAfter.UTC().Format(time.RFC3339)

	rep.PairMatch = publicKeysEqual(priv.Public(), cert.PublicKey)
	if !rep.PairMatch {
		rep.Issues = append(rep.Issues, "par chave-certificado não coincide")
	}

	rep.WithinValidity = !now.Before(cert.NotBefore) && !now.After(cert.NotAfter)
	if !rep.WithinValidity {
		rep.Issues = append(rep.Issues, "fora da validade")
	}

	intermediates := x509.NewCertPool()
	roots := x509.NewCertPool()
	if len(intermediatesPEM) > 0 {
		if !intermediates.AppendCertsFromPEM(intermediatesPEM) {
			rep.Issues = append(rep.Issues, "intermediates PEM inválidos")
		}
	}
	// Self-signed: treat leaf as root for prep validation only.
	if bytes.Equal(cert.RawSubject, cert.RawIssuer) {
		roots.AddCert(cert)
	}
	chains, err := cert.Verify(x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   now,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})
	rep.ChainOK = err == nil && len(chains) > 0
	if !rep.ChainOK {
		if bytes.Equal(cert.RawSubject, cert.RawIssuer) {
			// Verify may still fail on self-signed with empty roots in some cases — already added.
			rep.ChainOK = err == nil
		}
		if !rep.ChainOK {
			rep.Issues = append(rep.Issues, "cadeia não verificável offline (pending_external se CA AGT desconhecida)")
		}
	}

	rep.PurposeOK, rep.PurposeNote = purposeCheck(cert)
	if !rep.PurposeOK {
		rep.Issues = append(rep.Issues, "finalidade/KU insuficiente para assinatura")
	}

	switch k := priv.(type) {
	case *rsa.PrivateKey:
		rep.KeyBits = k.N.BitLen()
		rep.AlgorithmNote = "RSA"
		if rep.KeyBits < 2048 {
			rep.Issues = append(rep.Issues, "RSA < 2048 (observado em docs FE archived; pending_validation)")
			rep.PurposeOK = false
		}
	case *ecdsa.PrivateKey:
		rep.KeyBits = k.Curve.Params().BitSize
		rep.AlgorithmNote = "ECDSA/pending_external"
		rep.Issues = append(rep.Issues, "ECDSA não confirmado para FE JWS (pending_external)")
		rep.PurposeOK = false
	default:
		rep.AlgorithmNote = "pending_external"
		rep.Issues = append(rep.Issues, "tipo de chave não admitido")
		rep.PurposeOK = false
	}

	rep.OK = rep.PairMatch && rep.ChainOK && rep.WithinValidity && rep.PurposeOK
	return rep, nil
}

func purposeCheck(cert *x509.Certificate) (bool, string) {
	// FE JWS/RS256 docs archived imply signing; do not invent ExtKeyUsage OIDs.
	if cert.KeyUsage == 0 {
		return true, "pending_external" // KU absent — accept for prep with note
	}
	if cert.KeyUsage&x509.KeyUsageDigitalSignature != 0 || cert.KeyUsage&x509.KeyUsageContentCommitment != 0 {
		return true, "digital_signature"
	}
	return false, "pending_external"
}

func parsePrivateKeyPEM(raw []byte) (crypto.Signer, error) {
	rest := raw
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case "PRIVATE KEY":
			k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%w: PKCS#8", ErrValidation)
			}
			signer, ok := k.(crypto.Signer)
			if !ok {
				return nil, fmt.Errorf("%w: chave não assinante", ErrValidation)
			}
			return signer, nil
		case "RSA PRIVATE KEY":
			k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%w: PKCS#1", ErrValidation)
			}
			return k, nil
		case "EC PRIVATE KEY":
			k, err := x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%w: EC", ErrValidation)
			}
			return k, nil
		}
	}
	return nil, fmt.Errorf("%w: PEM chave ausente", ErrValidation)
}

func parseCertificatePEM(raw []byte) (*x509.Certificate, error) {
	rest := raw
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%w: certificado", ErrValidation)
		}
		return c, nil
	}
	return nil, fmt.Errorf("%w: PEM certificado ausente", ErrValidation)
}

func publicKeysEqual(a, b any) bool {
	switch ak := a.(type) {
	case *rsa.PublicKey:
		bk, ok := b.(*rsa.PublicKey)
		return ok && ak.N.Cmp(bk.N) == 0 && ak.E == bk.E
	case *ecdsa.PublicKey:
		bk, ok := b.(*ecdsa.PublicKey)
		return ok && ak.Curve == bk.Curve && ak.X.Cmp(bk.X) == 0 && ak.Y.Cmp(bk.Y) == 0
	default:
		return false
	}
}
