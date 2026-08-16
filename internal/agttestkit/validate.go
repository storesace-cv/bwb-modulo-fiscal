package agttestkit

import (
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
)

func validateRows(sheet string, rows []rawRow) (IdentityInventory, error) {
	inv := IdentityInventory{
		SheetName:  sheet,
		Algorithm:  "RSA",
		MinRSABits: MinRSABits,
		Notes: []string{
			"RSA PEM key pairs for AGT test/homologation fixtures — not X.509 certificates",
			"Do not substitute Basic Auth, softwareValidationNo, BWB registration, or productive AGT authorization",
			"Private material held in memory only during validation; never logged",
			"FE snapshot citations remain pending_validation and do not close normative conflicts",
		},
		SourceCitations: []string{
			"AO-FE-SNAP-HML-2026-07-25-API",
			"AO-FE-SNAP-HML-2026-07-25-ESTRUTURA",
			"AO-FE-SNAP-HML-2026-07-25-GESTAO",
			"AO-FE-SNAP-HML-2026-07-25-REGISTAR",
		},
	}

	seenNIF := map[string]string{}  // nif -> opaque (filled after fp)
	seenNome := map[string]string{} // nome -> opaque
	seenRef := map[string]struct{}{}

	allOK := true
	for _, row := range rows {
		priv, err := parseRSAPrivate(row.privPEM)
		if err != nil {
			return IdentityInventory{}, fmt.Errorf("%w: excel_row=%d", ErrInvalidPrivatePEM, row.excelRow)
		}
		pub, err := parseRSAPublic(row.pubPEM)
		if err != nil {
			return IdentityInventory{}, fmt.Errorf("%w: excel_row=%d", ErrInvalidPublicPEM, row.excelRow)
		}
		if !rsaPublicEqual(priv.PublicKey, pub) {
			allOK = false
			return IdentityInventory{}, fmt.Errorf("%w: excel_row=%d", ErrKeyPairMismatch, row.excelRow)
		}
		bits := priv.N.BitLen()
		meets := bits >= MinRSABits
		if !meets {
			allOK = false
			return IdentityInventory{}, fmt.Errorf("%w: excel_row=%d bits=%d min=%d", ErrRSATooSmall, row.excelRow, bits, MinRSABits)
		}
		fp := publicFingerprintSHA256(pub)
		ref := opaqueRefFromPublic(pub)
		if _, ok := seenRef[ref]; ok {
			return IdentityInventory{}, fmt.Errorf("%w", ErrDuplicateRef)
		}

		if prev, ok := seenNIF[row.nif]; ok {
			return IdentityInventory{}, fmt.Errorf("%w: refs=%s,%s", ErrDuplicateNIF, prev, ref)
		}
		if prev, ok := seenNome[row.nome]; ok {
			return IdentityInventory{}, fmt.Errorf("%w: refs=%s,%s", ErrDuplicateNome, prev, ref)
		}
		seenNIF[row.nif] = ref
		seenNome[row.nome] = ref
		seenRef[ref] = struct{}{}

		inv.Identities = append(inv.Identities, SanitizedIdentity{
			OpaqueRef:         ref,
			Algorithm:         "RSA",
			RSABits:           bits,
			PublicFingerprint: fp,
			PairMatch:         true,
			MeetsMinRSABits:   meets,
			NIFHashTruncated:  truncatedHash(row.nif),
			NomeHashTruncated: truncatedHash(row.nome),
		})
	}
	inv.IdentityCount = len(inv.Identities)
	inv.AllPairsValid = allOK && inv.IdentityCount > 0
	return inv, nil
}

func parseRSAPrivate(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, rest := pem.Decode(pemBytes)
	if block == nil {
		return nil, ErrInvalidPrivatePEM
	}
	if len(bytesTrimSpace(rest)) > 0 {
		// Reject concatenated / multiple PEM blocks in one cell.
		return nil, ErrInvalidPrivatePEM
	}
	switch block.Type {
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, ErrInvalidPrivatePEM
		}
		rk, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrUnsupportedKey
		}
		return rk, nil
	case "RSA PRIVATE KEY":
		rk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, ErrInvalidPrivatePEM
		}
		return rk, nil
	default:
		return nil, ErrUnsupportedKey
	}
}

func parseRSAPublic(pemBytes []byte) (*rsa.PublicKey, error) {
	block, rest := pem.Decode(pemBytes)
	if block == nil {
		return nil, ErrInvalidPublicPEM
	}
	if len(bytesTrimSpace(rest)) > 0 {
		return nil, ErrInvalidPublicPEM
	}
	switch block.Type {
	case "PUBLIC KEY":
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, ErrInvalidPublicPEM
		}
		rk, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, ErrUnsupportedKey
		}
		return rk, nil
	case "RSA PUBLIC KEY":
		rk, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, ErrInvalidPublicPEM
		}
		return rk, nil
	default:
		return nil, ErrUnsupportedKey
	}
}

func rsaPublicEqual(a rsa.PublicKey, b *rsa.PublicKey) bool {
	if b == nil || a.N == nil || b.N == nil {
		return false
	}
	return a.E == b.E && a.N.Cmp(b.N) == 0
}

func publicFingerprintSHA256(pub *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:])
}

func bytesTrimSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\r' || b[j-1] == '\t') {
		j--
	}
	return b[i:j]
}
