// Package agttestkit inventories AGT homologation RSA test identities from a
// local workbook (RM-FEFIX-001).
//
// Material is read only from a path supplied by the caller (typically under
// local/, which is gitignored). Private key bytes stay in memory for the
// duration of validation and are zeroed afterwards. Sanitized inventory never
// includes PEM, NIF, or display names.
//
// These are RSA PEM key pairs for development/tests — not X.509 certificates,
// not Basic Auth, not softwareValidationNo, and not proof of BWB registration
// or productive AGT authorization. FE snapshot sources cited by provenance
// docs remain pending_validation.
package agttestkit
