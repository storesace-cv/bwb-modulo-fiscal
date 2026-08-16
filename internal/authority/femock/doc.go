// Package femock is a strictly local HTTP test double for AGT FE transport prep (RM-FEFIX-004).
//
// Namespace: /mock/agt-fe/v1/... — never /sigt/fe/...
// JWS typ: BWB-MOCK only (≠ JWT/JOSE AGT; C-FE-JWS-TYP-001 remains open).
// Success here ≠ homologação / aceitação AGT. No network to AGT HML/PRD.
// Distinct from internal/authority/simulator (in-process submission stub via fiscaljws).
package femock

const (
	// TypMock is the only protected-header typ accepted by this mock.
	TypMock = "BWB-MOCK"

	// PathPrefix is the exclusive mock route namespace.
	PathPrefix = "/mock/agt-fe/v1"

	DefaultMaxBody = 16 << 10 // 16 KiB
)
