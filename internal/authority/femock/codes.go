package femock

import "fmt"

// FERNGDocStatus distinguishes emit-capable entries from reference-only catalog rows.
const (
	FERNGEmitActive       = "emit_active"       // may be scripted on this HTTP op
	FERNGReferenceBlocked = "reference_blocked" // catalogued; wire route blocked — never emitted
)

// FERNGEntry is a contextual FE-RNG catalog row (code × operation × source_id).
type FERNGEntry struct {
	Code       string
	Operation  string
	SourceID   string
	Status     string
	ThemeBrief string // inventory label only; ≠ invented AGT prose
}

// Contextual catalog. Active emit ops: softwareInfo, obterEstado, consultarFactura.
// REGISTAR/SOLICITAR codes remain reference-only while those routes are wire-blocked.
//
// Note: REGISTAR uses FE-RNG-031 (E40 jwsSignature); CONSULTAR* use FE-RNG-032 (E40).
// See AGT-Q-019 — do not cross-emit 031 on consultar ops.
var ferngCatalog = []FERNGEntry{
	// softwareInfo — software signature themes (REGISTAR table; compatible text in CONSULTAR*)
	{Code: "FE-RNG-010", Operation: "softwareInfo", SourceID: "AO-FE-SNAP-HML-2026-07-25-REGISTAR", Status: FERNGEmitActive, ThemeBrief: "jwsSoftwareSignature inválida (E08)"},
	{Code: "FE-RNG-011", Operation: "softwareInfo", SourceID: "AO-FE-SNAP-HML-2026-07-25-REGISTAR", Status: FERNGEmitActive, ThemeBrief: "jwsSoftwareSignature ≠ certificação (E39)"},

	// obterEstado — AO-FE-SNAP-HML-2026-07-25-CONSULTAR
	{Code: "FE-RNG-010", Operation: "obterEstado", SourceID: "AO-FE-SNAP-HML-2026-07-25-CONSULTAR", Status: FERNGEmitActive, ThemeBrief: "jwsSoftwareSignature inválida (E08)"},
	{Code: "FE-RNG-011", Operation: "obterEstado", SourceID: "AO-FE-SNAP-HML-2026-07-25-CONSULTAR", Status: FERNGEmitActive, ThemeBrief: "jwsSoftwareSignature ≠ certificação (E39)"},
	{Code: "FE-RNG-032", Operation: "obterEstado", SourceID: "AO-FE-SNAP-HML-2026-07-25-CONSULTAR", Status: FERNGEmitActive, ThemeBrief: "jwsSignature da chamada (E40)"},

	// consultarFactura — AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA
	{Code: "FE-RNG-010", Operation: "consultarFactura", SourceID: "AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA", Status: FERNGEmitActive, ThemeBrief: "jwsSoftwareSignature inválida (E08)"},
	{Code: "FE-RNG-011", Operation: "consultarFactura", SourceID: "AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA", Status: FERNGEmitActive, ThemeBrief: "jwsSoftwareSignature ≠ certificação (E39)"},
	{Code: "FE-RNG-032", Operation: "consultarFactura", SourceID: "AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA", Status: FERNGEmitActive, ThemeBrief: "jwsSignature da chamada (E40)"},

	// Reference-only (blocked HTTP routes — ScriptFERNG must reject)
	{Code: "FE-RNG-002", Operation: "registarFactura", SourceID: "AO-FE-SNAP-HML-2026-07-25-REGISTAR", Status: FERNGReferenceBlocked, ThemeBrief: "falta de parâmetro (E01)"},
	{Code: "FE-RNG-010", Operation: "registarFactura", SourceID: "AO-FE-SNAP-HML-2026-07-25-REGISTAR", Status: FERNGReferenceBlocked, ThemeBrief: "jwsSoftwareSignature inválida (E08)"},
	{Code: "FE-RNG-031", Operation: "registarFactura", SourceID: "AO-FE-SNAP-HML-2026-07-25-REGISTAR", Status: FERNGReferenceBlocked, ThemeBrief: "jwsSignature da chamada (E40)"},
	{Code: "FE-RNG-051", Operation: "solicitarSerie", SourceID: "AO-FE-SNAP-HML-2026-07-25-SOLICITAR", Status: FERNGReferenceBlocked, ThemeBrief: "código de série já em utilização (E31)"},
	{Code: "FE-RNG-080", Operation: "solicitarSerie", SourceID: "AO-FE-SNAP-HML-2026-07-25-SOLICITAR", Status: FERNGReferenceBlocked, ThemeBrief: "establishmentNumber desconhecido (E48)"},
}

// Mock-only technical codes (never attributed to AGT).
const (
	CodeUnauthorized        = "BWB-MOCK-UNAUTHORIZED"
	CodeBadRequest          = "BWB-MOCK-BAD-REQUEST"
	CodeMethodNotAllowed    = "BWB-MOCK-METHOD"
	CodeBodyTooLarge        = "BWB-MOCK-BODY-TOO-LARGE"
	CodeContentType         = "BWB-MOCK-CONTENT-TYPE"
	CodeJWSInvalid          = "BWB-MOCK-JWS-INVALID"
	CodeJWSTypRejected      = "BWB-MOCK-JWS-TYP"
	CodeRoleMismatch        = "BWB-MOCK-ROLE"
	CodeBindingMismatch     = "BWB-MOCK-BINDING"
	CodeProfileBlocked      = "BWB-MOCK-PROFILE-BLOCKED"
	CodeIdempotencyConflict = "BWB-MOCK-IDEMPOTENCY-CONFLICT"
	CodeFERNGUnknown        = "BWB-MOCK-FERNG-UNKNOWN"
	CodeFERNGOpMismatch     = "BWB-MOCK-FERNG-OP"
	CodeClosed              = "BWB-MOCK-CLOSED"
	CodeCancelled           = "BWB-MOCK-CANCELLED"
	CodeNotFound            = "BWB-MOCK-NOT-FOUND"
)

func lookupEmitFERNG(op, code string) (FERNGEntry, error) {
	for _, e := range ferngCatalog {
		if e.Operation == op && e.Code == code {
			if e.Status != FERNGEmitActive {
				return FERNGEntry{}, fmt.Errorf("%s: %s/%s", CodeFERNGOpMismatch, op, code)
			}
			return e, nil
		}
	}
	// Distinguish unknown op vs unknown pairing.
	opKnown := false
	for _, e := range ferngCatalog {
		if e.Operation == op {
			opKnown = true
			break
		}
	}
	if !opKnown {
		return FERNGEntry{}, fmt.Errorf("%s: unknown operation %s", CodeFERNGOpMismatch, op)
	}
	return FERNGEntry{}, fmt.Errorf("%s: %s not valid for %s", CodeFERNGUnknown, code, op)
}

// FERNGCatalog returns a copy of the contextual catalog.
func FERNGCatalog() []FERNGEntry {
	out := make([]FERNGEntry, len(ferngCatalog))
	copy(out, ferngCatalog)
	return out
}
