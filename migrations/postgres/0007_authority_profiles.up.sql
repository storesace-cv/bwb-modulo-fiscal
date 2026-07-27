-- AuthorityProfile: public AGT prep config (DEC-BO-004 / RM-AGTPREP-002).
-- No secrets, PEM, PKCS#12, passwords, tokens, or private URLs.

CREATE TABLE fiscal.authority_profiles (
  profile_id TEXT NOT NULL PRIMARY KEY,
  environment TEXT NOT NULL,
  taxpayer_id TEXT NULL,
  scope_id TEXT NULL,
  display_name TEXT NOT NULL,
  status TEXT NOT NULL,
  allowed_operations TEXT NOT NULL,
  pending_external TEXT NOT NULL,
  producer_credential_ref TEXT NOT NULL DEFAULT '',
  producer_key_ref TEXT NOT NULL DEFAULT '',
  certificate_ref TEXT NOT NULL DEFAULT '',
  algorithm_declared TEXT NOT NULL DEFAULT '',
  key_id_sanitized TEXT NOT NULL DEFAULT '',
  fingerprint_sanitized TEXT NOT NULL DEFAULT '',
  expires_at TIMESTAMPTZ NULL,
  config_ready BOOLEAN NOT NULL DEFAULT FALSE,
  secrets_ready BOOLEAN NOT NULL DEFAULT FALSE,
  offline_validated BOOLEAN NOT NULL DEFAULT FALSE,
  external_verified BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT authority_profiles_id_nonempty CHECK (length(trim(profile_id)) > 0),
  CONSTRAINT authority_profiles_environment_check CHECK (
    environment IN ('homologation', 'production')
  ),
  CONSTRAINT authority_profiles_display_nonempty CHECK (length(trim(display_name)) > 0),
  CONSTRAINT authority_profiles_status_check CHECK (
    status IN ('draft', 'validated', 'active', 'revoked')
  ),
  CONSTRAINT authority_profiles_external_verified_false CHECK (external_verified = FALSE)
);

CREATE INDEX authority_profiles_environment_idx ON fiscal.authority_profiles (environment);
CREATE INDEX authority_profiles_status_idx ON fiscal.authority_profiles (status);
