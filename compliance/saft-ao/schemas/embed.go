// Package saftschemas embeds the ASSOFT SAF-T (AO) XSD snapshot (SRC-B2).
// source_id: AO-SAFT-XSD-1.01_01 — status pending_validation (≠ certificado AGT).
package saftschemas

import "embed"

//go:embed SAFTAO1.01_01.xsd SHA256SUMS.txt LICENSE NOTICE.md
var Files embed.FS

const (
	XSDFileName     = "SAFTAO1.01_01.xsd"
	ExpectedSHA256  = "e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631"
	SourceID        = "AO-SAFT-XSD-1.01_01"
	TargetNamespace = "urn:OECD:StandardAuditFile-Tax:AO_1.01_01"
	SchemaVersion   = "1.01_01"
)
