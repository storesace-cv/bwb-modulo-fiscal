package doctype

import "strings"

// Availability reasons (sanitized; no NIF / secrets / invented FE-RNG codes).
const (
	ReasonGroupInactive       = "group_inactive"
	ReasonTypeInactive        = "type_inactive"
	ReasonAGTPending          = "agt_pending"
	ReasonSAFTIncompatible    = "saft_incompatible"
	ReasonFEMatrixUnsupported = "fe_matrix_unsupported"
	ReasonUnknownCanonical    = "unknown_canonical"
)

// Normative states that block emission until compliance closes the gap (RM-BO-013).
// hipotese remains allowlisted when activo=on (slice FT/NC); conflito/pending_validation do not.
func IsAGTPending(estadoNormativo string) bool {
	switch strings.TrimSpace(estadoNormativo) {
	case "pending_validation", "conflito":
		return true
	default:
		return false
	}
}

// AvailabilityConfig is establishment/environment activation (DEC-PROD-003).
// Missing group ⇒ active. Missing type override ⇒ seed activo.
type AvailabilityConfig struct {
	GroupActive        map[string]bool // key = grupo
	TypeActiveOverride map[string]bool // key = codigo_canonico; present ⇒ override seed
}

// AvailabilityInput feeds ComputeAvailability (DEC-PROD-005/006 + DEC-PROD-004).
type AvailabilityInput struct {
	FEEnrollmentStatus string // not_enrolled|pending|active|suspended
	Config             AvailabilityConfig
}

// TypeAvailability is one computed row for admin/ops (no invented codes).
type TypeAvailability struct {
	Grupo           string   `json:"grupo"`
	CodigoCanonico  string   `json:"codigo_canonico"`
	Designacao      string   `json:"designacao"`
	Available       bool     `json:"available"`
	Reasons         []string `json:"reasons"`
	Eligibility     string   `json:"eligibility"`
	FECode          string   `json:"fe_code,omitempty"`
	SAFTType        string   `json:"saft_type,omitempty"`
	EstadoNormativo string   `json:"estado_normativo"`
	SeedActivo      string   `json:"seed_activo"`
	EffectiveActivo bool     `json:"effective_activo"`
	GroupActive     bool     `json:"group_active"`
}

// AvailabilityReport groups computed availability (5 groups + types).
type AvailabilityReport struct {
	FEEnrollmentStatus string              `json:"fe_enrollment_status"`
	FEAderiu           bool                `json:"fe_aderiu"` // ≡ status==active (DEC-PROD-004); not a stored boolean model
	Groups             []GroupAvailability `json:"groups"`
	Types              []TypeAvailability  `json:"types"`
}

// GroupAvailability is one of the five L3 product groups.
type GroupAvailability struct {
	Grupo  string `json:"grupo"`
	Active bool   `json:"active"`
}

// ComputeAvailability evaluates catalog types under config + FE enrolment (fail-closed).
// Does not invent FE-RNG / documentType codes; only uses seed adapters.
func (r *Registry) ComputeAvailability(in AvailabilityInput) AvailabilityReport {
	feStatus := strings.TrimSpace(in.FEEnrollmentStatus)
	if feStatus == "" {
		feStatus = "not_enrolled"
	}
	feAderiu := feStatus == "active"

	groups := make([]GroupAvailability, 0, 5)
	for _, g := range Groups() {
		groups = append(groups, GroupAvailability{Grupo: g, Active: groupIsActive(in.Config, g)})
	}

	types := make([]TypeAvailability, 0)
	if r == nil {
		return AvailabilityReport{FEEnrollmentStatus: feStatus, FEAderiu: feAderiu, Groups: groups, Types: types}
	}
	for _, e := range r.All() {
		types = append(types, evaluateType(e, in.Config, feAderiu))
	}
	return AvailabilityReport{
		FEEnrollmentStatus: feStatus,
		FEAderiu:           feAderiu,
		Groups:             groups,
		Types:              types,
	}
}

func groupIsActive(cfg AvailabilityConfig, grupo string) bool {
	if cfg.GroupActive == nil {
		return true
	}
	v, ok := cfg.GroupActive[grupo]
	if !ok {
		return true
	}
	return v
}

func typeIsActive(e Entry, cfg AvailabilityConfig) bool {
	if cfg.TypeActiveOverride != nil {
		if v, ok := cfg.TypeActiveOverride[e.CodigoCanonico]; ok {
			return v
		}
	}
	return e.Activo == ActiveOn
}

func evaluateType(e Entry, cfg AvailabilityConfig, feAderiu bool) TypeAvailability {
	groupActive := groupIsActive(cfg, e.Grupo)
	effective := typeIsActive(e, cfg)
	reasons := make([]string, 0, 4)

	if !groupActive {
		reasons = append(reasons, ReasonGroupInactive)
	}
	if !effective {
		reasons = append(reasons, ReasonTypeInactive)
	}
	if IsAGTPending(e.EstadoNormativo) {
		reasons = append(reasons, ReasonAGTPending)
	}

	elig := strings.TrimSpace(e.ChannelAdapters.Eligibility)
	feCode := e.ChannelAdapters.FECode
	saft := e.ChannelAdapters.SAFTType
	saftStruct := e.ChannelAdapters.SAFTStructure
	hasSAFT := saft != "" || saftStruct != ""
	hasFE := feCode != ""

	if feAderiu {
		// DEC-PROD-006: with FE active, only FE-matrix-supported types (L4 present in seed).
		if !hasFE {
			reasons = append(reasons, ReasonFEMatrixUnsupported)
		}
	} else {
		// Without FE: only SAF-T-compatible (SAF-T or ambos with SAF-T adapters). FE-only blocked.
		if !hasSAFT || elig == "FE" {
			reasons = append(reasons, ReasonSAFTIncompatible)
		}
	}

	return TypeAvailability{
		Grupo:           e.Grupo,
		CodigoCanonico:  e.CodigoCanonico,
		Designacao:      e.Designacao,
		Available:       len(reasons) == 0,
		Reasons:         reasons,
		Eligibility:     elig,
		FECode:          feCode,
		SAFTType:        saft,
		EstadoNormativo: e.EstadoNormativo,
		SeedActivo:      e.Activo,
		EffectiveActivo: effective,
		GroupActive:     groupActive,
	}
}
