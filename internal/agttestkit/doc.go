// Package agttestkit inventories AGT homologation RSA test identities and holds
// them in memory for generic RSA-SHA256 signing (RM-FEFIX-001 / RM-FEFIX-002).
//
// Workbook path is caller-supplied (no default to a real workbook). Private key
// bytes stay in memory for validation/custody and are wiped on Close. Sanitized
// listings never include PEM, NIF, display names, or full public fingerprints.
//
// Inside private custody, each taxpayer workbook row keeps taxpayerNIF and
// sourceLabel bound to the same key pair. sourceLabel is the origin designation
// from the NOME column (entity/contribuinte test profile label) after structural
// trimming only — it is not a confirmed tax-regime classification and is never
// listed, logged, JSON-encoded, persisted, or exposed over HTTP.
//
// IdentityProvider.Signer returns an opaque crypto.Signer proxy that resolves
// the private key under lock. It never returns the stored *rsa.PrivateKey, so
// consumers cannot type-assert to extract D/Primes. Sign fails after Close.
//
// Providers: workbook custody, ephemeral producer keys, or a SecretStore PEM
// adapter so consumers can switch sources without API changes.
//
// These are RSA PEM key pairs for development/tests — not X.509 certificates,
// not Basic Auth, not softwareValidationNo, and not proof of BWB registration
// or productive AGT authorization. FE snapshot sources remain pending_validation.
//
// This package does not implement AGT JWS claim sets (RM-FEFIX-003).
// CI must use WriteSyntheticWorkbook only. verify_no_local_deps remains authoritative.
package agttestkit
