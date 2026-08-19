-- Persistent FE fixture queue (RM-FEFIX-007): workbook identities → BWB-MOCK boundary.

CREATE TABLE fe_fixture_submissions (
  id TEXT NOT NULL PRIMARY KEY,
  operation TEXT NOT NULL,
  state TEXT NOT NULL,
  identity_ref TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  mock_request_id TEXT NOT NULL DEFAULT '',
  mock_code TEXT NOT NULL DEFAULT '',
  source_id TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  next_attempt_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CONSTRAINT fe_fixture_submissions_id_nonempty CHECK (length(trim(id)) > 0),
  CONSTRAINT fe_fixture_submissions_operation_check CHECK (
    operation IN ('softwareInfo', 'obterEstado', 'consultarFactura')
  ),
  CONSTRAINT fe_fixture_submissions_state_check CHECK (
    state IN (
      'queued_for_fixture_boundary',
      'fixture_boundary_in_flight',
      'fixture_boundary_ok',
      'fixture_boundary_reject',
      'fixture_boundary_profile_blocked',
      'fixture_boundary_transport_failed',
      'fixture_boundary_dead'
    )
  ),
  CONSTRAINT fe_fixture_submissions_identity_ref_nonempty CHECK (length(trim(identity_ref)) > 0),
  CONSTRAINT fe_fixture_submissions_idempotency_nonempty CHECK (length(trim(idempotency_key)) > 0),
  CONSTRAINT fe_fixture_submissions_payload_json_nonempty CHECK (length(trim(payload_json)) > 0),
  CONSTRAINT fe_fixture_submissions_attempts_nonnegative CHECK (attempts >= 0),
  CONSTRAINT fe_fixture_submissions_idem_unique UNIQUE (idempotency_key, operation)
);

CREATE INDEX fe_fixture_submissions_claim_idx ON fe_fixture_submissions (state, next_attempt_at);
