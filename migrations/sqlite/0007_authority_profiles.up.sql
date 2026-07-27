-- AuthorityProfile: public AGT prep config (DEC-BO-004 / RM-AGTPREP-002).
-- No secrets, PEM, PKCS#12, passwords, tokens, or private URLs.

CREATE TABLE authority_profiles (
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
  expires_at TEXT NULL,
  config_ready INTEGER NOT NULL DEFAULT 0,
  secrets_ready INTEGER NOT NULL DEFAULT 0,
  offline_validated INTEGER NOT NULL DEFAULT 0,
  external_verified INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CONSTRAINT authority_profiles_id_nonempty CHECK (length(trim(profile_id)) > 0),
  CONSTRAINT authority_profiles_environment_check CHECK (
    environment IN ('homologation', 'production')
  ),
  CONSTRAINT authority_profiles_display_nonempty CHECK (length(trim(display_name)) > 0),
  CONSTRAINT authority_profiles_status_check CHECK (
    status IN ('draft', 'validated', 'active', 'revoked')
  ),
  CONSTRAINT authority_profiles_external_verified_false CHECK (external_verified = 0)
);

CREATE INDEX authority_profiles_environment_idx ON authority_profiles (environment);
CREATE INDEX authority_profiles_status_idx ON authority_profiles (status);
