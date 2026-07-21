-- Sandbox POS S1: scopes, api_credentials, audit_events (forward-only; no schema namespace).
-- DEC-AUTH-001 / DEC-ROTATE-001 / DEC-AUDIT-001 / DEC-CREDS-001 / DEC-MIG-001
PRAGMA foreign_keys = ON;

CREATE TABLE scopes (
  scope_id TEXT NOT NULL PRIMARY KEY,
  taxpayer_nif TEXT NOT NULL,
  iana_timezone TEXT NOT NULL,
  series_effective_code TEXT NOT NULL,
  environment TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  CONSTRAINT scopes_scope_id_nonempty CHECK (length(trim(scope_id)) > 0),
  CONSTRAINT scopes_taxpayer_nif_nonempty CHECK (length(trim(taxpayer_nif)) > 0),
  CONSTRAINT scopes_iana_timezone_nonempty CHECK (length(trim(iana_timezone)) > 0),
  CONSTRAINT scopes_series_code_nonempty CHECK (length(trim(series_effective_code)) > 0),
  CONSTRAINT scopes_environment_check CHECK (environment IN ('homologation', 'development')),
  CONSTRAINT scopes_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE TABLE api_credentials (
  credential_id TEXT NOT NULL PRIMARY KEY,
  scope_id TEXT NOT NULL REFERENCES scopes (scope_id) ON DELETE RESTRICT,
  token_hash BLOB NOT NULL,
  status TEXT NOT NULL,
  expires_at TEXT NULL,
  grace_until TEXT NULL,
  rotated_from TEXT NULL,
  revoked_at TEXT NULL,
  created_at TEXT NOT NULL,
  created_by TEXT NOT NULL,
  CONSTRAINT api_credentials_token_hash_len CHECK (length(token_hash) = 32),
  CONSTRAINT api_credentials_token_hash_unique UNIQUE (token_hash),
  CONSTRAINT api_credentials_id_scope_unique UNIQUE (credential_id, scope_id),
  CONSTRAINT api_credentials_status_check CHECK (status IN ('active', 'grace', 'revoked')),
  CONSTRAINT api_credentials_created_by_nonempty CHECK (length(trim(created_by)) > 0),
  CONSTRAINT api_credentials_revoked_coherent CHECK (
    (status = 'revoked' AND revoked_at IS NOT NULL)
    OR (status <> 'revoked' AND revoked_at IS NULL)
  ),
  CONSTRAINT api_credentials_grace_coherent CHECK (
    (status = 'grace' AND grace_until IS NOT NULL)
    OR (status <> 'grace')
  ),
  CONSTRAINT api_credentials_no_self_rotate CHECK (
    rotated_from IS NULL OR rotated_from <> credential_id
  ),
  CONSTRAINT api_credentials_rotated_from_same_scope
    FOREIGN KEY (rotated_from, scope_id)
    REFERENCES api_credentials (credential_id, scope_id)
);

CREATE UNIQUE INDEX api_credentials_one_active_per_scope
  ON api_credentials (scope_id)
  WHERE status = 'active';

CREATE UNIQUE INDEX api_credentials_one_grace_per_scope
  ON api_credentials (scope_id)
  WHERE status = 'grace';

CREATE TABLE audit_events (
  event_id TEXT NOT NULL PRIMARY KEY,
  occurred_at TEXT NOT NULL,
  credential_id TEXT NULL,
  scope_id TEXT NULL,
  action TEXT NOT NULL,
  result TEXT NOT NULL,
  reason_code TEXT NULL,
  request_id TEXT NULL,
  CONSTRAINT audit_events_action_nonempty CHECK (length(trim(action)) > 0),
  CONSTRAINT audit_events_result_nonempty CHECK (length(trim(result)) > 0)
);

CREATE TRIGGER audit_events_no_update
BEFORE UPDATE ON audit_events
BEGIN
  SELECT RAISE(ABORT, 'audit_events is append-only');
END;

CREATE TRIGGER audit_events_no_delete
BEFORE DELETE ON audit_events
BEGIN
  SELECT RAISE(ABORT, 'audit_events is append-only');
END;
