package doctype

import (
	"fmt"
	"strings"
)

// C-DOC-003 / DEC-PROD-006 fail-closed channel routing helpers.
// Does not invent SAF-T structures for FE-only types and does not confirm AO-DOC-*.

// SAFTLayer classifies a catalog SAFTType adapter string.
type SAFTLayer string

const (
	SAFTLayerNone    SAFTLayer = ""
	SAFTLayerInvoice SAFTLayer = "InvoiceType"
	SAFTLayerPayment SAFTLayer = "PaymentType"
	SAFTLayerOther   SAFTLayer = "other"
)

// ParseSAFTTypeAdapter parses catalog values like "InvoiceType=FT" or "PaymentType=RC".
// Empty adapter ⇒ none. Unknown prefix ⇒ other (fail-closed for C-DOC-003 checks).
func ParseSAFTTypeAdapter(saftType string) (layer SAFTLayer, code string) {
	s := strings.TrimSpace(saftType)
	if s == "" || s == "∅" {
		return SAFTLayerNone, ""
	}
	prefix, val, ok := strings.Cut(s, "=")
	if !ok {
		return SAFTLayerOther, s
	}
	prefix = strings.TrimSpace(prefix)
	val = strings.TrimSpace(val)
	switch prefix {
	case "InvoiceType":
		return SAFTLayerInvoice, val
	case "PaymentType":
		return SAFTLayerPayment, val
	default:
		return SAFTLayerOther, val
	}
}

// CDOC003Violation is a catalog row that breaks documented C-DOC-003 invariants.
type CDOC003Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	Reason         string
}

func (v CDOC003Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.Reason)
}

// CheckCDOC003Invariants validates seed adapters against the documented conflict:
// FA is FE-only (no InvoiceType/PaymentType); RC/RG must not be InvoiceType.
// AR may appear in InvoiceType or PaymentType (distinct L3) — not flagged here.
// Returns violations; empty slice means seed respects the fail-closed binding.
func (r *Registry) CheckCDOC003Invariants() []CDOC003Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC003Violation, 0)
	for _, e := range r.All() {
		fe := strings.TrimSpace(e.ChannelAdapters.FECode)
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		elig := strings.TrimSpace(e.ChannelAdapters.Eligibility)

		switch fe {
		case "FA":
			if layer != SAFTLayerNone {
				out = append(out, CDOC003Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					Reason:         "FA é FE-only (C-DOC-003); proibido adaptador SAF-T InvoiceType/PaymentType",
				})
			}
			if elig != "" && elig != "FE" {
				out = append(out, CDOC003Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					Reason:         "FA exige eligibility=FE (DEC-PROD-006)",
				})
			}
		case "RC", "RG":
			if layer == SAFTLayerInvoice {
				out = append(out, CDOC003Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					Reason:         "RC/RG L4 não mapeiam para InvoiceType (C-DOC-003); usar Payments/PaymentType",
				})
			}
			if layer == SAFTLayerPayment && code != fe {
				out = append(out, CDOC003Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					Reason:         "PaymentType do seed diverge do FE code RC/RG",
				})
			}
		}

		// Guard: never claim InvoiceType=FA/RC/RG in seed.
		if layer == SAFTLayerInvoice && (code == "FA" || code == "RC" || code == "RG") {
			out = append(out, CDOC003Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				Reason:         "InvoiceType não enumera FA/RC/RG no XSD ASSOFT (C-DOC-003)",
			})
		}
	}
	return out
}
