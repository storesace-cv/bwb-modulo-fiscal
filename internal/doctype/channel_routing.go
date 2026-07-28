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
	SAFTLayerWork    SAFTLayer = "WorkType"
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
	case "WorkType":
		return SAFTLayerWork, val
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
// AR dual L3 membership is enforced separately by CheckCDOC004Invariants.
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

// CDOC004Violation is a catalog row that breaks documented C-DOC-004 invariants (AR dual L3).
type CDOC004Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	SAFTStructure  string
	Reason         string
}

func (v CDOC004Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q l3=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.SAFTStructure, v.Reason)
}

const (
	canonicalVendasAR     = "bwb.ao.vendas.ar"
	canonicalPagamentosAR = "bwb.ao.pagamentos.ar"
)

// CheckCDOC004Invariants validates the AR dual-homonym seed:
// two distinct canonicals; InvoiceType=AR only under SalesInvoices; PaymentType=AR only under Payments.
// Does not choose «grupo único por emissão» and does not confirm AO-DOC-*.
func (r *Registry) CheckCDOC004Invariants() []CDOC004Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC004Violation, 0)

	vendas, okV := r.Lookup(canonicalVendasAR)
	pag, okP := r.Lookup(canonicalPagamentosAR)
	if !okV {
		out = append(out, CDOC004Violation{CodigoCanonico: canonicalVendasAR, Reason: "seed vendas.ar obrigatório (C-DOC-004)"})
	}
	if !okP {
		out = append(out, CDOC004Violation{CodigoCanonico: canonicalPagamentosAR, Reason: "seed pagamentos.ar obrigatório (C-DOC-004)"})
	}
	if okV && okP && vendas.CodigoCanonico == pag.CodigoCanonico {
		out = append(out, CDOC004Violation{
			CodigoCanonico: vendas.CodigoCanonico,
			FECode:         "AR",
			Reason:         "proibido colapsar vendas.ar e pagamentos.ar no mesmo canónico (C-DOC-004)",
		})
	}

	if okV {
		out = append(out, checkARHomonym(vendas, SAFTLayerInvoice, "SalesInvoices", "vendas")...)
	}
	if okP {
		out = append(out, checkARHomonym(pag, SAFTLayerPayment, "Payments", "pagamentos")...)
	}

	// Guard: any other FE=AR row must still bind L2↔L3 consistently.
	for _, e := range r.All() {
		if e.CodigoCanonico == canonicalVendasAR || e.CodigoCanonico == canonicalPagamentosAR {
			continue
		}
		if strings.TrimSpace(e.ChannelAdapters.FECode) != "AR" {
			continue
		}
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		switch {
		case layer == SAFTLayerInvoice && code == "AR" && structL3 != "SalesInvoices":
			out = append(out, CDOC004Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         "AR",
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "InvoiceType=AR exige L3 SalesInvoices (C-DOC-004)",
			})
		case layer == SAFTLayerPayment && code == "AR" && structL3 != "Payments":
			out = append(out, CDOC004Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         "AR",
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "PaymentType=AR exige L3 Payments (C-DOC-004)",
			})
		case layer != SAFTLayerInvoice && layer != SAFTLayerPayment:
			out = append(out, CDOC004Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         "AR",
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "FE=AR sem adaptador InvoiceType/PaymentType explícito (C-DOC-004)",
			})
		}
	}
	return out
}

func checkARHomonym(e Entry, wantLayer SAFTLayer, wantL3, wantGrupo string) []CDOC004Violation {
	out := make([]CDOC004Violation, 0)
	fe := strings.TrimSpace(e.ChannelAdapters.FECode)
	layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
	structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
	grupo := strings.TrimSpace(e.Grupo)

	if fe != "AR" {
		out = append(out, CDOC004Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         "homónimo AR exige FECode=AR",
		})
	}
	if layer != wantLayer || code != "AR" {
		out = append(out, CDOC004Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         fmt.Sprintf("exige adaptador %s=AR (C-DOC-004)", wantLayer),
		})
	}
	if structL3 != wantL3 {
		out = append(out, CDOC004Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         fmt.Sprintf("exige L3 %s (C-DOC-004)", wantL3),
		})
	}
	if grupo != wantGrupo {
		out = append(out, CDOC004Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         fmt.Sprintf("exige grupo=%s (C-DOC-004)", wantGrupo),
		})
	}
	if e.Activo != ActiveOff {
		out = append(out, CDOC004Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         "AR dual-L3 permanece off até fecho residual (C-DOC-004)",
		})
	}
	return out
}

// CDOC005Violation is a catalog row that breaks C-DOC-005 insurer dual L3 invariants.
type CDOC005Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	SAFTStructure  string
	Reason         string
}

func (v CDOC005Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q l3=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.SAFTStructure, v.Reason)
}

// Insurer FE codes that also exist as WorkType in the ASSOFT XSD (C-DOC-005).
var cdoc005InsurerFECodes = []string{"RP", "RE", "CS", "LD", "RA"}

func cdoc005CanonicalVendas(code string) string {
	return "bwb.ao.vendas." + strings.ToLower(code)
}

// CheckCDOC005Invariants validates insurer dual-membership fail-closed bindings:
// FE RP/RE/CS/LD/RA seeds under vendas must stay InvoiceType+SalesInvoices+off;
// XSD also lists the same literals under WorkType — never collapse L3 or invent FE→WorkType.
// Does not confirm AO-DOC-* and does not invent conferencia.* dual seeds.
func (r *Registry) CheckCDOC005Invariants() []CDOC005Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC005Violation, 0)
	for _, code := range cdoc005InsurerFECodes {
		canon := cdoc005CanonicalVendas(code)
		e, ok := r.Lookup(canon)
		if !ok {
			out = append(out, CDOC005Violation{
				CodigoCanonico: canon,
				FECode:         code,
				Reason:         "seed vendas." + strings.ToLower(code) + " obrigatório (C-DOC-005)",
			})
			continue
		}
		fe := strings.TrimSpace(e.ChannelAdapters.FECode)
		layer, c := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		grupo := strings.TrimSpace(e.Grupo)
		if fe != code {
			out = append(out, CDOC005Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "FECode deve coincidir com literal segurador " + code,
			})
		}
		if layer != SAFTLayerInvoice || c != code {
			out = append(out, CDOC005Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "exige InvoiceType=" + code + " até DEC-REG-003 (C-DOC-005); proibido WorkType no seed FE",
			})
		}
		if structL3 != "SalesInvoices" {
			out = append(out, CDOC005Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "exige L3 SalesInvoices (C-DOC-005)",
			})
		}
		if grupo != "vendas" {
			out = append(out, CDOC005Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "exige grupo=vendas (C-DOC-005)",
			})
		}
		if e.Activo != ActiveOff {
			out = append(out, CDOC005Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "segurador dual-L3 permanece off até fecho residual (C-DOC-005)",
			})
		}
	}

	// Guard: no FE=insurer row may bind WorkType (would invent FE→WorkType without DEC-REG-003).
	// SAF-T-only WorkType rows (FE empty) for these codes would need distinct canonicals — none today.
	for _, e := range r.All() {
		fe := strings.TrimSpace(e.ChannelAdapters.FECode)
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		isInsurerFE := false
		for _, c := range cdoc005InsurerFECodes {
			if fe == c {
				isInsurerFE = true
				break
			}
		}
		if isInsurerFE && layer == SAFTLayerWork {
			out = append(out, CDOC005Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "proibido FE→WorkType para códigos segurador sem DEC-REG-003 (C-DOC-005)",
			})
		}
		if layer == SAFTLayerWork && (code == "RP" || code == "RE" || code == "CS" || code == "LD" || code == "RA") {
			if structL3 != "WorkingDocuments" {
				out = append(out, CDOC005Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "WorkType=" + code + " exige L3 WorkingDocuments (C-DOC-005)",
				})
			}
			if fe != "" && fe != "∅" {
				// Already covered for insurer FE; keep generic clarity for non-empty FE.
				if !isInsurerFE {
					out = append(out, CDOC005Violation{
						CodigoCanonico: e.CodigoCanonico,
						FECode:         fe,
						SAFTType:       e.ChannelAdapters.SAFTType,
						SAFTStructure:  structL3,
						Reason:         "WorkType segurador com FE não-vazio exige decisão DEC-REG-003 (C-DOC-005)",
					})
				}
			}
		}
		if layer == SAFTLayerInvoice && (code == "RP" || code == "RE" || code == "CS" || code == "LD" || code == "RA") {
			if structL3 != "SalesInvoices" {
				out = append(out, CDOC005Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "InvoiceType=" + code + " exige L3 SalesInvoices (C-DOC-005)",
				})
			}
		}
	}
	return out
}
