// Package saftao is the Angola SAF-T structural foundation (RM-SAFT-001…017).
//
// Builds on XSD SAFTAO1.01_01 (source_id AO-SAFT-XSD-1.01_01, pending_validation).
// Does NOT claim AGT validation, certification, or AO-* compliance.
// Distinct from FE JWS/RS256.
package saftao

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"

	saftschemas "github.com/storesace-cv/bwb-modulo-fiscal/compliance/saft-ao/schemas"
)

// SchemaMeta describes the embedded XSD snapshot used by this package.
type SchemaMeta struct {
	SourceID        string
	TargetNamespace string
	SchemaVersion   string
	SHA256          string
	Status          string // always pending_validation until AGT confirms
	Certified       bool   // always false in this foundation
}

// Meta returns immutable metadata for the embedded XSD.
func Meta() SchemaMeta {
	return SchemaMeta{
		SourceID:        saftschemas.SourceID,
		TargetNamespace: saftschemas.TargetNamespace,
		SchemaVersion:   saftschemas.SchemaVersion,
		SHA256:          saftschemas.ExpectedSHA256,
		Status:          "pending_validation",
		Certified:       false,
	}
}

// VerifyEmbeddedXSD checks byte integrity of the embedded XSD against the catalog hash.
func VerifyEmbeddedXSD() error {
	raw, err := fs.ReadFile(saftschemas.Files, saftschemas.XSDFileName)
	if err != nil {
		return fmt.Errorf("saftao: ler XSD: %w", err)
	}
	sum := sha256.Sum256(raw)
	got := hex.EncodeToString(sum[:])
	if got != saftschemas.ExpectedSHA256 {
		return fmt.Errorf("saftao: SHA-256 XSD divergente (integridade)")
	}
	return nil
}

// XSDBytes returns the embedded XSD (for inventory/tools; not a runtime dependency on local/).
func XSDBytes() ([]byte, error) {
	return fs.ReadFile(saftschemas.Files, saftschemas.XSDFileName)
}
