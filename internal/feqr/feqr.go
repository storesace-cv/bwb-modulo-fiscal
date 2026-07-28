// Package feqr encodes C-FE-QR-001 fail-closed handling for printed FE QR Code.
//
// Does not generate QR images and does not confirm AO-* / RM-ENG-007.
package feqr

import (
	"fmt"
	"strings"
)

// ConflictOpen remains true until AGT closes URL divergence DE 683 vs FE HML.
const ConflictOpen = true

// Parameters aligned across DE 683 Anexo III OCR and FE HML snapshot (pending_validation).
const (
	ModulesPerSide = 33
	ECCPercent     = 15
	PNGSizePx      = 350 // from FE HML snapshot only; DE 683 OCR does not state px in extract
)

// CandidateURLHost bases cited by sources (do not pick one while ConflictOpen).
const (
	HostPortalContribuinte = "portaldocontribuinte.minfin.gov.ao" // DE 683 OCR @19194–19195
	HostQuiosqueAGT        = "quiosqueagt.minfin.gov.ao"          // FE HML QRCODE snapshot
)

// Violation is a C-FE-QR-001 guard breach.
type Violation struct {
	Code   string
	Reason string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s", v.Code, v.Reason)
}

// CheckInvariants verifies package-level QR separation / pending state.
func CheckInvariants() []Violation {
	out := make([]Violation, 0, 4)
	if ModulesPerSide != 33 {
		out = append(out, Violation{Code: "modules", Reason: "inventário exige 33×33"})
	}
	if ECCPercent != 15 {
		out = append(out, Violation{Code: "ecc", Reason: "inventário exige ECC 15%"})
	}
	if HostPortalContribuinte == HostQuiosqueAGT {
		out = append(out, Violation{Code: "hosts", Reason: "hosts candidatos colidem"})
	}
	if !ConflictOpen {
		out = append(out, Violation{
			Code:   "conflict_flag",
			Reason: "ConflictOpen=false exige fecho documentado C-FE-QR-001",
		})
	}
	return out
}

// RejectAmbiguousQRURL rejects any attempt to treat a concrete QR URL as confirmed
// while C-FE-QR-001 is open, or to mix both host families.
func RejectAmbiguousQRURL(rawURL string) error {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return fmt.Errorf("feqr: url vazia")
	}
	hasPortal := strings.Contains(u, HostPortalContribuinte)
	hasQuiosque := strings.Contains(u, HostQuiosqueAGT)
	if hasPortal && hasQuiosque {
		return fmt.Errorf("feqr: URL mistura hosts DE 683 e FE HML (C-FE-QR-001)")
	}
	if ConflictOpen {
		return fmt.Errorf("feqr: C-FE-QR-001 aberto — recusar URL de QR até confirmação AGT")
	}
	return nil
}

// BuildPrintedQRURL always fails while ConflictOpen (no «correct» URL inventada).
func BuildPrintedQRURL(host, documentNo, nifEmissor string) (string, error) {
	_ = host
	_ = documentNo
	_ = nifEmissor
	if ConflictOpen {
		return "", fmt.Errorf("feqr: C-FE-QR-001 aberto — não construir URL de QR impresso")
	}
	return "", fmt.Errorf("feqr: construção só após fecho C-FE-QR-001")
}

// EncodeDocumentNoSpaces applies the only shared rule: spaces → %%20.
func EncodeDocumentNoSpaces(documentNo string) string {
	return strings.ReplaceAll(documentNo, " ", "%20")
}
