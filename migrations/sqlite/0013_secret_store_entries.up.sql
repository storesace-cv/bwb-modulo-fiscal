-- Durable encrypted-at-rest SecretStore entries (RM-AGTPREP-014 / DEC-BO-001 plano B).
-- Ciphertext + nonce only. NEVER plaintext, PEM, PKCS#12, passwords, or tokens.
-- Master key lives outside the DB (env/KMS reference); ≠ production KMS/HSM decision.

CREATE TABLE secret_store_entries (
  ref_key TEXT NOT NULL PRIMARY KEY,
  kind TEXT NOT NULL,
  environment TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  fingerprint TEXT NOT NULL DEFAULT '',
  cipher_alg TEXT NOT NULL DEFAULT 'AES-256-GCM',
  nonce BLOB NOT NULL,
  ciphertext BLOB NOT NULL,
  version INTEGER NOT NULL,
  expires_at TEXT NULL,
  last_verified_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CONSTRAINT secret_store_entries_ref_key_nonempty CHECK (length(trim(ref_key)) > 0),
  CONSTRAINT secret_store_entries_kind_check CHECK (
    kind IN ('producer_credential', 'producer_key', 'taxpayer_key', 'certificate')
  ),
  CONSTRAINT secret_store_entries_environment_check CHECK (
    environment IN ('homologation', 'production')
  ),
  CONSTRAINT secret_store_entries_subject_nonempty CHECK (length(trim(subject_id)) > 0),
  CONSTRAINT secret_store_entries_name_nonempty CHECK (length(trim(name)) > 0),
  CONSTRAINT secret_store_entries_status_check CHECK (
    status IN ('present', 'rotating', 'revoked')
  ),
  CONSTRAINT secret_store_entries_cipher_alg_check CHECK (cipher_alg = 'AES-256-GCM'),
  CONSTRAINT secret_store_entries_version_positive CHECK (version >= 1),
  CONSTRAINT secret_store_entries_material_by_status CHECK (
    (
      status = 'revoked'
      AND length(nonce) = 0
      AND length(ciphertext) = 0
      AND fingerprint = ''
    )
    OR (
      status IN ('present', 'rotating')
      AND length(nonce) > 0
      AND length(ciphertext) > 0
      AND length(trim(fingerprint)) > 0
    )
  )
);

CREATE INDEX secret_store_entries_env_kind_idx
  ON secret_store_entries (environment, kind);

CREATE INDEX secret_store_entries_subject_idx
  ON secret_store_entries (subject_id);
