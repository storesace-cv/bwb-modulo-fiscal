// Package agttestkit inventories AGT homologation RSA test identities and holds
// them in memory for generic RSA-SHA256 signing (RM-FEFIX-001 / RM-FEFIX-002).
//
// Workbook path is caller-supplied (no default to a real workbook). Private key
// bytes stay in memory for validation/custody and are wiped on Close. Sanitized
// listings never include PEM, NIF, display names, or full public fingerprints.
//
// IdentityProvider is backed by workbook custody, ephemeral producer keys, or a
// SecretStore PEM adapter so consumers can switch sources without API changes.
//
// These are RSA PEM key pairs for development/tests — not X.509 certificates,
// not Basic Auth, not softwareValidationNo, and not proof of BWB registration
// or productive AGT authorization. FE snapshot sources remain pending_validation.
//
// This package does not implement AGT JWS claim sets (RM-FEFIX-003).
// CI must use WriteSyntheticWorkbook only. verify_no_local_deps remains authoritative.
package agttestkit
