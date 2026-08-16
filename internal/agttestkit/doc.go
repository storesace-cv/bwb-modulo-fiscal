// Package agttestkit inventories AGT homologation RSA test identities from an
// operator-supplied workbook path (RM-FEFIX-001).
//
// The workbook path is provided by the caller at runtime. Private key bytes stay
// in memory for the duration of validation and are zeroed afterwards. Sanitized
// inventory never includes PEM, NIF, or display names.
//
// These are RSA PEM key pairs for development/tests — not X.509 certificates,
// not Basic Auth, not softwareValidationNo, and not proof of BWB registration
// or productive AGT authorization. FE snapshot sources cited by provenance
// docs remain pending_validation.
//
// CI and the default automated suite must use WriteSyntheticWorkbook only.
// verify_no_local_deps remains the authority against unversioned tree deps.
package agttestkit
