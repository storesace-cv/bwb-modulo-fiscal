package femock

import "fmt"

// Catalogued FE-RNG codes that the harness may script (source_id from FE matrix).
// Descriptions are abbreviated inventory labels — not invented AGT prose.
var allowlistedFERNG = map[string]string{
	"FE-RNG-002": "AO-FE-SNAP-HML-2026-07-25-REGISTAR",
	"FE-RNG-010": "AO-FE-SNAP-HML-2026-07-25-REGISTAR",
	"FE-RNG-031": "AO-FE-SNAP-HML-2026-07-25-REGISTAR",
	"FE-RNG-051": "AO-FE-SNAP-HML-2026-07-25-SOLICITAR",
	"FE-RNG-080": "AO-FE-SNAP-HML-2026-07-25-SOLICITAR",
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
	CodeClosed              = "BWB-MOCK-CLOSED"
	CodeCancelled           = "BWB-MOCK-CANCELLED"
	CodeNotFound            = "BWB-MOCK-NOT-FOUND"
)

func ferngSourceID(code string) (string, error) {
	src, ok := allowlistedFERNG[code]
	if !ok {
		return "", fmt.Errorf("%s: %s", CodeFERNGUnknown, code)
	}
	return src, nil
}

// AllowlistedFERNG returns a copy of scriptable FE-RNG codes → source_id.
func AllowlistedFERNG() map[string]string {
	out := make(map[string]string, len(allowlistedFERNG))
	for k, v := range allowlistedFERNG {
		out[k] = v
	}
	return out
}
