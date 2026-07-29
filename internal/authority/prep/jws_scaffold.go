package prep

import (
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/signsep"
)

// JWS claim / profile validation states (scaffolding; ≠ RM-FE-002 / AGT).
const (
	ClaimsStatusPendingExternal = "pending_external"
)

// JWSProfileScaffold is owner-facing metadata for future FE JWS — no invented claims.
type JWSProfileScaffold struct {
	AlgorithmDeclared       string
	ClaimsStatus            string
	MechanismID             string
	SAFTMechanismSeparated  string
	ExternalVerified        bool
	InventedClaimsForbidden bool
	SourceNote              string
	ConflictSeparationNote  string
}

// JWSProfileScaffoldDefault returns the fail-closed scaffold (always external_verified=false).
func JWSProfileScaffoldDefault() JWSProfileScaffold {
	return JWSProfileScaffold{
		AlgorithmDeclared:       signsep.FETechnicalAlgorithm, // RS256 technical observation
		ClaimsStatus:            ClaimsStatusPendingExternal,
		MechanismID:             signsep.MechanismFEJWS,
		SAFTMechanismSeparated:  signsep.MechanismSAFTHash,
		ExternalVerified:        false,
		InventedClaimsForbidden: true,
		SourceNote:              "DEC-BO-004 + signsep/C-SIGN-001; claims exactos pendentes RM-FE-002 / AGT",
		ConflictSeparationNote:  "FE JWS ≠ SAF-T Hash (C-SIGN-001); não inventar campos JWS",
	}
}
