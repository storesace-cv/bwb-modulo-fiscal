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
	SAFTLayerNone     SAFTLayer = ""
	SAFTLayerInvoice  SAFTLayer = "InvoiceType"
	SAFTLayerPayment  SAFTLayer = "PaymentType"
	SAFTLayerWork     SAFTLayer = "WorkType"
	SAFTLayerPurchase SAFTLayer = "PurchaseType"
	SAFTLayerMovement SAFTLayer = "MovementType"
	SAFTLayerOther    SAFTLayer = "other"
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
	case "PurchaseType":
		return SAFTLayerPurchase, val
	case "MovementType":
		return SAFTLayerMovement, val
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

// CDOC006Violation is a catalog row that breaks C-DOC-006 RC PaymentType vs PurchaseType invariants.
type CDOC006Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	SAFTStructure  string
	Reason         string
}

func (v CDOC006Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q l3=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.SAFTStructure, v.Reason)
}

const (
	canonicalPagamentosRC = "bwb.ao.pagamentos.rc"
	canonicalComprasRC    = "bwb.ao.compras.rc"
)

// CheckCDOC006Invariants validates RC dual-homonym fail-closed bindings:
// PaymentType=RC under Payments (FE RC) ≠ PurchaseType=RC under PurchaseInvoices (FE empty).
// Does not invent bijection L4↔L2 and does not confirm AO-DOC-*.
func (r *Registry) CheckCDOC006Invariants() []CDOC006Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC006Violation, 0)

	pag, okP := r.Lookup(canonicalPagamentosRC)
	com, okC := r.Lookup(canonicalComprasRC)
	if !okP {
		out = append(out, CDOC006Violation{CodigoCanonico: canonicalPagamentosRC, Reason: "seed pagamentos.rc obrigatório (C-DOC-006)"})
	}
	if !okC {
		out = append(out, CDOC006Violation{CodigoCanonico: canonicalComprasRC, Reason: "seed compras.rc obrigatório (C-DOC-006)"})
	}
	if okP && okC && pag.CodigoCanonico == com.CodigoCanonico {
		out = append(out, CDOC006Violation{
			CodigoCanonico: pag.CodigoCanonico,
			FECode:         "RC",
			Reason:         "proibido colapsar pagamentos.rc e compras.rc no mesmo canónico (C-DOC-006)",
		})
	}

	if okP {
		fe := strings.TrimSpace(pag.ChannelAdapters.FECode)
		layer, code := ParseSAFTTypeAdapter(pag.ChannelAdapters.SAFTType)
		structL3 := strings.TrimSpace(pag.ChannelAdapters.SAFTStructure)
		grupo := strings.TrimSpace(pag.Grupo)
		if fe != "RC" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: pag.CodigoCanonico,
				FECode:         fe,
				SAFTType:       pag.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "pagamentos.rc exige FECode=RC (C-DOC-006)",
			})
		}
		if layer != SAFTLayerPayment || code != "RC" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: pag.CodigoCanonico,
				FECode:         fe,
				SAFTType:       pag.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "pagamentos.rc exige PaymentType=RC (C-DOC-006); ≠ PurchaseType",
			})
		}
		if structL3 != "Payments" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: pag.CodigoCanonico,
				FECode:         fe,
				SAFTType:       pag.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "pagamentos.rc exige L3 Payments (C-DOC-006)",
			})
		}
		if grupo != "pagamentos" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: pag.CodigoCanonico,
				FECode:         fe,
				SAFTType:       pag.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "pagamentos.rc exige grupo=pagamentos (C-DOC-006)",
			})
		}
		if pag.Activo != ActiveOff {
			out = append(out, CDOC006Violation{
				CodigoCanonico: pag.CodigoCanonico,
				FECode:         fe,
				SAFTType:       pag.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "pagamentos.rc permanece off até fecho residual (C-DOC-006)",
			})
		}
	}

	if okC {
		fe := strings.TrimSpace(com.ChannelAdapters.FECode)
		layer, code := ParseSAFTTypeAdapter(com.ChannelAdapters.SAFTType)
		structL3 := strings.TrimSpace(com.ChannelAdapters.SAFTStructure)
		grupo := strings.TrimSpace(com.Grupo)
		if fe != "" && fe != "∅" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.rc é SAF-T-only (FE vazio); proibido FE=RC no lado compras (C-DOC-006)",
			})
		}
		if layer != SAFTLayerPurchase || code != "RC" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.rc exige PurchaseType=RC (C-DOC-006); ≠ PaymentType",
			})
		}
		if structL3 != "PurchaseInvoices" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.rc exige L3 PurchaseInvoices (C-DOC-006)",
			})
		}
		if grupo != "compras" {
			out = append(out, CDOC006Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.rc exige grupo=compras (C-DOC-006)",
			})
		}
		if com.Activo != ActiveOff {
			out = append(out, CDOC006Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.rc permanece off até fecho residual (C-DOC-006)",
			})
		}
	}

	// Guard: never claim InvoiceType=RC (also C-DOC-003) or mix L3 for RC adapters.
	for _, e := range r.All() {
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		if code != "RC" {
			continue
		}
		switch layer {
		case SAFTLayerPayment:
			if structL3 != "Payments" {
				out = append(out, CDOC006Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         e.ChannelAdapters.FECode,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "PaymentType=RC exige L3 Payments (C-DOC-006)",
				})
			}
		case SAFTLayerPurchase:
			if structL3 != "PurchaseInvoices" {
				out = append(out, CDOC006Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         e.ChannelAdapters.FECode,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "PurchaseType=RC exige L3 PurchaseInvoices (C-DOC-006)",
				})
			}
		case SAFTLayerInvoice:
			out = append(out, CDOC006Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         e.ChannelAdapters.FECode,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "InvoiceType=RC proibido (C-DOC-003/006)",
			})
		}
	}
	return out
}

// CDOC007Violation is a catalog row that breaks C-DOC-007 GR MovementType vs WorkType invariants.
type CDOC007Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	SAFTStructure  string
	Reason         string
}

func (v CDOC007Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q l3=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.SAFTStructure, v.Reason)
}

const (
	canonicalMovimentacaoGR = "bwb.ao.movimentacao.gr"
	canonicalConferenciaGR  = "bwb.ao.conferencia.gr"
)

// CheckCDOC007Invariants validates GR dual-homonym fail-closed bindings:
// MovementType=GR under MovementOfGoods ≠ WorkType=GR under WorkingDocuments.
// Both are SAF-T-only (FE empty). Does not invent FE L4 for GR and does not confirm AO-DOC-*.
func (r *Registry) CheckCDOC007Invariants() []CDOC007Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC007Violation, 0)

	mov, okM := r.Lookup(canonicalMovimentacaoGR)
	conf, okC := r.Lookup(canonicalConferenciaGR)
	if !okM {
		out = append(out, CDOC007Violation{CodigoCanonico: canonicalMovimentacaoGR, Reason: "seed movimentacao.gr obrigatório (C-DOC-007)"})
	}
	if !okC {
		out = append(out, CDOC007Violation{CodigoCanonico: canonicalConferenciaGR, Reason: "seed conferencia.gr obrigatório (C-DOC-007; XSD WorkType=GR)"})
	}
	if okM && okC && mov.CodigoCanonico == conf.CodigoCanonico {
		out = append(out, CDOC007Violation{
			CodigoCanonico: mov.CodigoCanonico,
			Reason:         "proibido colapsar movimentacao.gr e conferencia.gr (C-DOC-007)",
		})
	}

	if okM {
		out = append(out, checkGRHomonym(mov, SAFTLayerMovement, "MovementOfGoods", "movimentacao")...)
	}
	if okC {
		out = append(out, checkGRHomonym(conf, SAFTLayerWork, "WorkingDocuments", "conferencia")...)
	}

	for _, e := range r.All() {
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		if code != "GR" {
			continue
		}
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		fe := strings.TrimSpace(e.ChannelAdapters.FECode)
		switch layer {
		case SAFTLayerMovement:
			if structL3 != "MovementOfGoods" {
				out = append(out, CDOC007Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "MovementType=GR exige L3 MovementOfGoods (C-DOC-007)",
				})
			}
		case SAFTLayerWork:
			if structL3 != "WorkingDocuments" {
				out = append(out, CDOC007Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         fe,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "WorkType=GR exige L3 WorkingDocuments (C-DOC-007)",
				})
			}
		}
		if fe != "" {
			out = append(out, CDOC007Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         fe,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "GR seed é SAF-T-only; proibido inventar FE L4 para GR (C-DOC-007)",
			})
		}
	}
	return out
}

func checkGRHomonym(e Entry, wantLayer SAFTLayer, wantL3, wantGrupo string) []CDOC007Violation {
	out := make([]CDOC007Violation, 0)
	fe := strings.TrimSpace(e.ChannelAdapters.FECode)
	layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
	structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
	grupo := strings.TrimSpace(e.Grupo)

	if fe != "" {
		out = append(out, CDOC007Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         "GR exige FE vazio (SAF-T-only; C-DOC-007)",
		})
	}
	if layer != wantLayer || code != "GR" {
		out = append(out, CDOC007Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         fmt.Sprintf("exige adaptador %s=GR (C-DOC-007)", wantLayer),
		})
	}
	if structL3 != wantL3 {
		out = append(out, CDOC007Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         fmt.Sprintf("exige L3 %s (C-DOC-007)", wantL3),
		})
	}
	if grupo != wantGrupo {
		out = append(out, CDOC007Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         fmt.Sprintf("exige grupo=%s (C-DOC-007)", wantGrupo),
		})
	}
	if e.Activo != ActiveOff {
		out = append(out, CDOC007Violation{
			CodigoCanonico: e.CodigoCanonico,
			FECode:         fe,
			SAFTType:       e.ChannelAdapters.SAFTType,
			SAFTStructure:  structL3,
			Reason:         "GR dual-L3 permanece off até fecho residual (C-DOC-007)",
		})
	}
	return out
}

// CDOC008Violation is a catalog row that breaks C-DOC-008 InvoiceType vs PurchaseType invariants.
type CDOC008Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	SAFTStructure  string
	Reason         string
}

func (v CDOC008Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q l3=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.SAFTStructure, v.Reason)
}

// Shared L2 literals present in both InvoiceType and PurchaseType (XSD) with dual seeds.
var cdoc008DualCodes = []string{"FT", "NC"}

func cdoc008CanonicalVendas(code string) string {
	return "bwb.ao.vendas." + strings.ToLower(code)
}

func cdoc008CanonicalCompras(code string) string {
	return "bwb.ao.compras." + strings.ToLower(code)
}

// CheckCDOC008Invariants validates InvoiceType vs PurchaseType dual-homonym fail-closed bindings
// for FT/NC: vendas (FE+InvoiceType+SalesInvoices; may be ActiveOn per DEC-REG-003) ≠
// compras (FE empty+PurchaseType+PurchaseInvoices; must stay off).
// Does not invent FE for compras and does not confirm AO-DOC-*.
func (r *Registry) CheckCDOC008Invariants() []CDOC008Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC008Violation, 0)
	for _, code := range cdoc008DualCodes {
		vCanon := cdoc008CanonicalVendas(code)
		cCanon := cdoc008CanonicalCompras(code)
		vendas, okV := r.Lookup(vCanon)
		compras, okC := r.Lookup(cCanon)
		if !okV {
			out = append(out, CDOC008Violation{CodigoCanonico: vCanon, FECode: code, Reason: "seed vendas." + strings.ToLower(code) + " obrigatório (C-DOC-008)"})
		}
		if !okC {
			out = append(out, CDOC008Violation{CodigoCanonico: cCanon, Reason: "seed compras." + strings.ToLower(code) + " obrigatório (C-DOC-008; XSD PurchaseType)"})
		}
		if okV && okC && vendas.CodigoCanonico == compras.CodigoCanonico {
			out = append(out, CDOC008Violation{
				CodigoCanonico: vendas.CodigoCanonico,
				FECode:         code,
				Reason:         "proibido colapsar vendas e compras para " + code + " (C-DOC-008)",
			})
		}
		if okV {
			fe := strings.TrimSpace(vendas.ChannelAdapters.FECode)
			layer, c := ParseSAFTTypeAdapter(vendas.ChannelAdapters.SAFTType)
			structL3 := strings.TrimSpace(vendas.ChannelAdapters.SAFTStructure)
			grupo := strings.TrimSpace(vendas.Grupo)
			if fe != code {
				out = append(out, CDOC008Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "vendas." + strings.ToLower(code) + " exige FECode=" + code,
				})
			}
			if layer != SAFTLayerInvoice || c != code {
				out = append(out, CDOC008Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "vendas exige InvoiceType=" + code + " (C-DOC-008); ≠ PurchaseType",
				})
			}
			if structL3 != "SalesInvoices" {
				out = append(out, CDOC008Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "vendas exige L3 SalesInvoices (C-DOC-008)",
				})
			}
			if grupo != "vendas" {
				out = append(out, CDOC008Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "exige grupo=vendas (C-DOC-008)",
				})
			}
			// DEC-REG-003: FT/NC may be ActiveOn on vendas; no force-off here.
		}
		if okC {
			fe := strings.TrimSpace(compras.ChannelAdapters.FECode)
			layer, c := ParseSAFTTypeAdapter(compras.ChannelAdapters.SAFTType)
			structL3 := strings.TrimSpace(compras.ChannelAdapters.SAFTStructure)
			grupo := strings.TrimSpace(compras.Grupo)
			if fe != "" {
				out = append(out, CDOC008Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras." + strings.ToLower(code) + " é SAF-T-only; proibido FE L4 (C-DOC-008)",
				})
			}
			if layer != SAFTLayerPurchase || c != code {
				out = append(out, CDOC008Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras exige PurchaseType=" + code + " (C-DOC-008); ≠ InvoiceType",
				})
			}
			if structL3 != "PurchaseInvoices" {
				out = append(out, CDOC008Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras exige L3 PurchaseInvoices (C-DOC-008)",
				})
			}
			if grupo != "compras" {
				out = append(out, CDOC008Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "exige grupo=compras (C-DOC-008)",
				})
			}
			if compras.Activo != ActiveOff {
				out = append(out, CDOC008Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras." + strings.ToLower(code) + " permanece off até DEC-REG-003 (C-DOC-008)",
				})
			}
		}
	}

	// Guard: L2 literal FT/NC must bind L3 correctly when present.
	for _, e := range r.All() {
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		if code != "FT" && code != "NC" {
			continue
		}
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		switch layer {
		case SAFTLayerInvoice:
			if structL3 != "SalesInvoices" {
				out = append(out, CDOC008Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         e.ChannelAdapters.FECode,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "InvoiceType=" + code + " exige L3 SalesInvoices (C-DOC-008)",
				})
			}
		case SAFTLayerPurchase:
			if structL3 != "PurchaseInvoices" {
				out = append(out, CDOC008Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         e.ChannelAdapters.FECode,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "PurchaseType=" + code + " exige L3 PurchaseInvoices (C-DOC-008)",
				})
			}
		}
	}
	return out
}

// CDOC009Violation is a catalog row that breaks C-DOC-009 AR PurchaseType third-L3 invariants.
type CDOC009Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	SAFTStructure  string
	Reason         string
}

func (v CDOC009Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q l3=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.SAFTStructure, v.Reason)
}

const canonicalComprasAR = "bwb.ao.compras.ar"

// CheckCDOC009Invariants validates the third AR L3 leg (PurchaseType under PurchaseInvoices):
// distinct from C-DOC-004 vendas.ar (InvoiceType) and pagamentos.ar (PaymentType).
// compras.ar is SAF-T-only (FE empty) and off. Does not confirm AO-DOC-*.
func (r *Registry) CheckCDOC009Invariants() []CDOC009Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC009Violation, 0)

	vendas, okV := r.Lookup(canonicalVendasAR)
	pag, okP := r.Lookup(canonicalPagamentosAR)
	com, okC := r.Lookup(canonicalComprasAR)
	if !okV {
		out = append(out, CDOC009Violation{CodigoCanonico: canonicalVendasAR, Reason: "seed vendas.ar obrigatório (C-DOC-004/009)"})
	}
	if !okP {
		out = append(out, CDOC009Violation{CodigoCanonico: canonicalPagamentosAR, Reason: "seed pagamentos.ar obrigatório (C-DOC-004/009)"})
	}
	if !okC {
		out = append(out, CDOC009Violation{CodigoCanonico: canonicalComprasAR, Reason: "seed compras.ar obrigatório (C-DOC-009; XSD PurchaseType=AR)"})
	}
	if okV && okC && vendas.CodigoCanonico == com.CodigoCanonico {
		out = append(out, CDOC009Violation{CodigoCanonico: vendas.CodigoCanonico, Reason: "proibido colapsar vendas.ar e compras.ar (C-DOC-009)"})
	}
	if okP && okC && pag.CodigoCanonico == com.CodigoCanonico {
		out = append(out, CDOC009Violation{CodigoCanonico: pag.CodigoCanonico, Reason: "proibido colapsar pagamentos.ar e compras.ar (C-DOC-009)"})
	}

	if okC {
		fe := strings.TrimSpace(com.ChannelAdapters.FECode)
		layer, code := ParseSAFTTypeAdapter(com.ChannelAdapters.SAFTType)
		structL3 := strings.TrimSpace(com.ChannelAdapters.SAFTStructure)
		grupo := strings.TrimSpace(com.Grupo)
		if fe != "" {
			out = append(out, CDOC009Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.ar é SAF-T-only; proibido FE L4 (C-DOC-009)",
			})
		}
		if layer != SAFTLayerPurchase || code != "AR" {
			out = append(out, CDOC009Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.ar exige PurchaseType=AR (C-DOC-009); ≠ InvoiceType/PaymentType",
			})
		}
		if structL3 != "PurchaseInvoices" {
			out = append(out, CDOC009Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.ar exige L3 PurchaseInvoices (C-DOC-009)",
			})
		}
		if grupo != "compras" {
			out = append(out, CDOC009Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.ar exige grupo=compras (C-DOC-009)",
			})
		}
		if com.Activo != ActiveOff {
			out = append(out, CDOC009Violation{
				CodigoCanonico: com.CodigoCanonico,
				FECode:         fe,
				SAFTType:       com.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "compras.ar permanece off até fecho residual (C-DOC-009)",
			})
		}
	}

	for _, e := range r.All() {
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		if code != "AR" || layer != SAFTLayerPurchase {
			continue
		}
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		if structL3 != "PurchaseInvoices" {
			out = append(out, CDOC009Violation{
				CodigoCanonico: e.CodigoCanonico,
				FECode:         e.ChannelAdapters.FECode,
				SAFTType:       e.ChannelAdapters.SAFTType,
				SAFTStructure:  structL3,
				Reason:         "PurchaseType=AR exige L3 PurchaseInvoices (C-DOC-009)",
			})
		}
	}
	return out
}

// CDOC010Violation is a catalog row that breaks C-DOC-010 InvoiceType vs PurchaseType
// invariants for remaining shared L2 literals (beyond FT/NC in C-DOC-008 and AR in C-DOC-009).
type CDOC010Violation struct {
	CodigoCanonico string
	FECode         string
	SAFTType       string
	SAFTStructure  string
	Reason         string
}

func (v CDOC010Violation) String() string {
	return fmt.Sprintf("%s fe=%q saft=%q l3=%q: %s", v.CodigoCanonico, v.FECode, v.SAFTType, v.SAFTStructure, v.Reason)
}

// Remaining InvoiceType ∩ PurchaseType L2 literals with dual seeds (XSD), excluding
// FT/NC (C-DOC-008) and AR (C-DOC-009 / triple with PaymentType).
var cdoc010DualCodes = []string{"FR", "GF", "FG", "AC", "AF", "TV"}

func cdoc010CanonicalVendas(code string) string {
	return "bwb.ao.vendas." + strings.ToLower(code)
}

func cdoc010CanonicalCompras(code string) string {
	return "bwb.ao.compras." + strings.ToLower(code)
}

func cdoc010CodeSet() map[string]struct{} {
	out := make(map[string]struct{}, len(cdoc010DualCodes))
	for _, c := range cdoc010DualCodes {
		out[c] = struct{}{}
	}
	return out
}

// CheckCDOC010Invariants validates InvoiceType vs PurchaseType dual-homonym fail-closed
// bindings for FR/GF/FG/AC/AF/TV: vendas (FE+InvoiceType+SalesInvoices; off) ≠
// compras (FE empty+PurchaseType+PurchaseInvoices; off). Does not invent FE for compras
// and does not confirm AO-DOC-*.
func (r *Registry) CheckCDOC010Invariants() []CDOC010Violation {
	if r == nil {
		return nil
	}
	out := make([]CDOC010Violation, 0)
	for _, code := range cdoc010DualCodes {
		vCanon := cdoc010CanonicalVendas(code)
		cCanon := cdoc010CanonicalCompras(code)
		vendas, okV := r.Lookup(vCanon)
		compras, okC := r.Lookup(cCanon)
		if !okV {
			out = append(out, CDOC010Violation{CodigoCanonico: vCanon, FECode: code, Reason: "seed vendas." + strings.ToLower(code) + " obrigatório (C-DOC-010)"})
		}
		if !okC {
			out = append(out, CDOC010Violation{CodigoCanonico: cCanon, Reason: "seed compras." + strings.ToLower(code) + " obrigatório (C-DOC-010; XSD PurchaseType)"})
		}
		if okV && okC && vendas.CodigoCanonico == compras.CodigoCanonico {
			out = append(out, CDOC010Violation{
				CodigoCanonico: vendas.CodigoCanonico,
				FECode:         code,
				Reason:         "proibido colapsar vendas e compras para " + code + " (C-DOC-010)",
			})
		}
		if okV {
			fe := strings.TrimSpace(vendas.ChannelAdapters.FECode)
			layer, c := ParseSAFTTypeAdapter(vendas.ChannelAdapters.SAFTType)
			structL3 := strings.TrimSpace(vendas.ChannelAdapters.SAFTStructure)
			grupo := strings.TrimSpace(vendas.Grupo)
			if fe != code {
				out = append(out, CDOC010Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "vendas." + strings.ToLower(code) + " exige FECode=" + code,
				})
			}
			if layer != SAFTLayerInvoice || c != code {
				out = append(out, CDOC010Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "vendas exige InvoiceType=" + code + " (C-DOC-010); ≠ PurchaseType",
				})
			}
			if structL3 != "SalesInvoices" {
				out = append(out, CDOC010Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "vendas exige L3 SalesInvoices (C-DOC-010)",
				})
			}
			if grupo != "vendas" {
				out = append(out, CDOC010Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "exige grupo=vendas (C-DOC-010)",
				})
			}
			if vendas.Activo != ActiveOff {
				out = append(out, CDOC010Violation{
					CodigoCanonico: vendas.CodigoCanonico,
					FECode:         fe,
					SAFTType:       vendas.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "vendas." + strings.ToLower(code) + " permanece off até DEC-REG-003 (C-DOC-010)",
				})
			}
		}
		if okC {
			fe := strings.TrimSpace(compras.ChannelAdapters.FECode)
			layer, c := ParseSAFTTypeAdapter(compras.ChannelAdapters.SAFTType)
			structL3 := strings.TrimSpace(compras.ChannelAdapters.SAFTStructure)
			grupo := strings.TrimSpace(compras.Grupo)
			if fe != "" {
				out = append(out, CDOC010Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras." + strings.ToLower(code) + " é SAF-T-only; proibido FE L4 (C-DOC-010)",
				})
			}
			if layer != SAFTLayerPurchase || c != code {
				out = append(out, CDOC010Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras exige PurchaseType=" + code + " (C-DOC-010); ≠ InvoiceType",
				})
			}
			if structL3 != "PurchaseInvoices" {
				out = append(out, CDOC010Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras exige L3 PurchaseInvoices (C-DOC-010)",
				})
			}
			if grupo != "compras" {
				out = append(out, CDOC010Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "exige grupo=compras (C-DOC-010)",
				})
			}
			if compras.Activo != ActiveOff {
				out = append(out, CDOC010Violation{
					CodigoCanonico: compras.CodigoCanonico,
					FECode:         fe,
					SAFTType:       compras.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "compras." + strings.ToLower(code) + " permanece off até DEC-REG-003 (C-DOC-010)",
				})
			}
		}
	}

	codes := cdoc010CodeSet()
	for _, e := range r.All() {
		layer, code := ParseSAFTTypeAdapter(e.ChannelAdapters.SAFTType)
		if _, ok := codes[code]; !ok {
			continue
		}
		structL3 := strings.TrimSpace(e.ChannelAdapters.SAFTStructure)
		switch layer {
		case SAFTLayerInvoice:
			if structL3 != "SalesInvoices" {
				out = append(out, CDOC010Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         e.ChannelAdapters.FECode,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "InvoiceType=" + code + " exige L3 SalesInvoices (C-DOC-010)",
				})
			}
		case SAFTLayerPurchase:
			if structL3 != "PurchaseInvoices" {
				out = append(out, CDOC010Violation{
					CodigoCanonico: e.CodigoCanonico,
					FECode:         e.ChannelAdapters.FECode,
					SAFTType:       e.ChannelAdapters.SAFTType,
					SAFTStructure:  structL3,
					Reason:         "PurchaseType=" + code + " exige L3 PurchaseInvoices (C-DOC-010)",
				})
			}
		}
	}
	return out
}
