// Package signsep encodes C-SIGN-001 fail-closed separation:
// SAF-T document Hash chain (DE 74/19 n.º34) ≠ FE JWS/RS256 (DE 683/25).
//
// Does not implement fiscal signing algorithms and does not confirm AO-CRYPTO-*.
package signsep

import (
	"fmt"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/fiscaljws"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/saftao"
)

// Mechanism identifiers (documentation / guards only — not wire formats).
const (
	MechanismSAFTHash = "saft_hash_chain" // DE 74/19 Anexo I n.º34 (RSA+SHA-1 cited; AO-* pending)
	MechanismFEJWS    = "fe_jws_rs256"    // DE 683/25 + FE HML jwsDocumentSignature
)

// FETechnicalAlgorithm is the only algorithm allowed for the ephemeral vertical-slice JWS helper.
// Distinct from any future confirmed SAF-T Hash algorithm.
const FETechnicalAlgorithm = fiscaljws.Algorithm // RS256

// SAFTCitedHashDigest is the digest named in DE 74/19 n.º34 OCR (C-SIGN-001 fact).
// It is NOT implemented here and MUST NOT be treated as FE JWS alg.
const SAFTCitedHashDigest = "SHA-1"

// Violation is a C-SIGN-001 separation breach.
type Violation struct {
	Code   string
	Reason string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s", v.Code, v.Reason)
}

// CheckInvariants verifies compile-time / package-level separation used by the module.
func CheckInvariants() []Violation {
	out := make([]Violation, 0, 4)

	if MechanismSAFTHash == MechanismFEJWS {
		out = append(out, Violation{Code: "mechanism_ids", Reason: "identificadores SAF-T e FE colidem"})
	}
	if FETechnicalAlgorithm == "" {
		out = append(out, Violation{Code: "fe_alg", Reason: "algoritmo FE técnico ausente"})
	}
	if FETechnicalAlgorithm == SAFTCitedHashDigest {
		out = append(out, Violation{Code: "alg_collision", Reason: "RS256 FE não pode igualar digest SAF-T citado"})
	}
	if saftao.PendingHashAlgorithm == "" {
		out = append(out, Violation{Code: "pending_hash", Reason: "PendingHashAlgorithm obrigatório (C-SIGN-001)"})
	}
	if strings.EqualFold(string(saftao.PendingHashAlgorithm), FETechnicalAlgorithm) {
		out = append(out, Violation{
			Code:   "pending_vs_fe",
			Reason: "marcador PendingHashAlgorithm não pode ser o alg FE RS256",
		})
	}
	meta := saftao.Meta()
	if meta.Certified {
		out = append(out, Violation{Code: "saft_certified", Reason: "fundação SAF-T não pode alegar Certified"})
	}
	if meta.Status != "pending_validation" {
		out = append(out, Violation{Code: "saft_status", Reason: "XSD deve permanecer pending_validation"})
	}
	return out
}

// RejectConflatedAlgorithm reports whether a declared algorithm string improperly
// mixes FE JWS and SAF-T Hash semantics (fail-closed guard for callers).
func RejectConflatedAlgorithm(declared string, mechanism string) error {
	d := strings.TrimSpace(declared)
	m := strings.TrimSpace(mechanism)
	if d == "" || m == "" {
		return fmt.Errorf("signsep: algoritmo/mecanismo vazios")
	}
	switch m {
	case MechanismFEJWS:
		if !strings.EqualFold(d, FETechnicalAlgorithm) && d != "pending_external" {
			return fmt.Errorf("signsep: mecanismo FE exige %s ou pending_external (got %q)", FETechnicalAlgorithm, d)
		}
		if strings.EqualFold(d, SAFTCitedHashDigest) || strings.EqualFold(d, "SHA1") {
			return fmt.Errorf("signsep: SHA-1 SAF-T não é algoritmo FE JWS (C-SIGN-001)")
		}
		return nil
	case MechanismSAFTHash:
		if strings.EqualFold(d, FETechnicalAlgorithm) {
			return fmt.Errorf("signsep: RS256 FE não assina Hash SAF-T (C-SIGN-001)")
		}
		// Algorithm remains pending AO-*; only reject known FE conflation here.
		return nil
	default:
		return fmt.Errorf("signsep: mecanismo desconhecido %q", m)
	}
}
