-- Create application schema and core fiscal tables (forward-only).
CREATE SCHEMA fiscal;

CREATE OR REPLACE FUNCTION fiscal.reject_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  RAISE EXCEPTION '% is append-only: % not allowed', TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME, TG_OP;
END;
$$;

CREATE TABLE fiscal.idempotency_records (
  scope_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash BYTEA NOT NULL,
  document_id TEXT NULL,
  state TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (scope_id, idempotency_key),
  CONSTRAINT idempotency_records_scope_id_nonempty CHECK (length(trim(scope_id)) > 0),
  CONSTRAINT idempotency_records_key_nonempty CHECK (length(trim(idempotency_key)) > 0),
  CONSTRAINT idempotency_records_hash_len CHECK (octet_length(request_hash) = 32),
  CONSTRAINT idempotency_records_state_check CHECK (state IN ('in_progress', 'completed'))
);

CREATE TABLE fiscal.documents (
  id TEXT NOT NULL PRIMARY KEY,
  scope_id TEXT NOT NULL,
  external_id TEXT NOT NULL,
  document_type TEXT NOT NULL,
  currency TEXT NOT NULL,
  issued_at TIMESTAMPTZ NOT NULL,
  requested_series TEXT NULL,
  series_code TEXT NOT NULL,
  fiscal_seq BIGINT NOT NULL,
  seller_tax_id TEXT NOT NULL,
  seller_name TEXT NOT NULL,
  customer_tax_id TEXT NULL,
  customer_name TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  sealed_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT documents_scope_id_nonempty CHECK (length(trim(scope_id)) > 0),
  CONSTRAINT documents_external_id_nonempty CHECK (length(trim(external_id)) > 0),
  CONSTRAINT documents_document_type_check CHECK (document_type IN ('invoice', 'credit_note')),
  CONSTRAINT documents_currency_aoa CHECK (currency = 'AOA'),
  CONSTRAINT documents_series_code_nonempty CHECK (length(trim(series_code)) > 0),
  CONSTRAINT documents_seller_tax_id_nonempty CHECK (length(trim(seller_tax_id)) > 0),
  CONSTRAINT documents_seller_name_nonempty CHECK (length(trim(seller_name)) > 0),
  CONSTRAINT documents_scope_external_unique UNIQUE (scope_id, external_id),
  CONSTRAINT documents_scope_series_seq_unique UNIQUE (scope_id, series_code, fiscal_seq)
);

CREATE TRIGGER documents_no_update
  BEFORE UPDATE ON fiscal.documents
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TRIGGER documents_no_delete
  BEFORE DELETE ON fiscal.documents
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TABLE fiscal.document_lines (
  document_id TEXT NOT NULL REFERENCES fiscal.documents (id) ON DELETE RESTRICT,
  line_no INTEGER NOT NULL,
  line_id TEXT NOT NULL,
  description TEXT NOT NULL,
  quantity_scaled BIGINT NOT NULL,
  unit_price_cents BIGINT NOT NULL,
  tax_code TEXT NOT NULL,
  PRIMARY KEY (document_id, line_no),
  CONSTRAINT document_lines_line_id_nonempty CHECK (length(trim(line_id)) > 0),
  CONSTRAINT document_lines_description_nonempty CHECK (length(trim(description)) > 0),
  CONSTRAINT document_lines_quantity_positive CHECK (quantity_scaled > 0),
  CONSTRAINT document_lines_unit_price_nonneg CHECK (unit_price_cents >= 0),
  CONSTRAINT document_lines_tax_code_nonempty CHECK (length(trim(tax_code)) > 0)
);

CREATE TRIGGER document_lines_no_update
  BEFORE UPDATE ON fiscal.document_lines
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TRIGGER document_lines_no_delete
  BEFORE DELETE ON fiscal.document_lines
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TABLE fiscal.series_counters (
  scope_id TEXT NOT NULL,
  series_code TEXT NOT NULL,
  last_seq BIGINT NOT NULL DEFAULT 0,
  PRIMARY KEY (scope_id, series_code),
  CONSTRAINT series_counters_scope_id_nonempty CHECK (length(trim(scope_id)) > 0),
  CONSTRAINT series_counters_series_code_nonempty CHECK (length(trim(series_code)) > 0),
  CONSTRAINT series_counters_last_seq_nonneg CHECK (last_seq >= 0)
);

CREATE TABLE fiscal.ledger_events (
  id TEXT NOT NULL PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES fiscal.documents (id) ON DELETE RESTRICT,
  seq BIGINT NOT NULL,
  event_type TEXT NOT NULL,
  from_status TEXT NULL,
  to_status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ledger_events_seq_positive CHECK (seq > 0),
  CONSTRAINT ledger_events_document_seq_unique UNIQUE (document_id, seq),
  CONSTRAINT ledger_events_event_type_nonempty CHECK (length(trim(event_type)) > 0),
  CONSTRAINT ledger_events_from_status_check CHECK (
    from_status IS NULL OR from_status IN (
      'received',
      'validated',
      'sealed_locally',
      'queued_for_authority',
      'authority_processing',
      'authority_accepted',
      'authority_rejected',
      'authority_outcome_unknown',
      'rejected',
      'contingency_pending'
    )
  ),
  CONSTRAINT ledger_events_to_status_check CHECK (
    to_status IN (
      'received',
      'validated',
      'sealed_locally',
      'queued_for_authority',
      'authority_processing',
      'authority_accepted',
      'authority_rejected',
      'authority_outcome_unknown',
      'rejected',
      'contingency_pending'
    )
  )
);

CREATE TRIGGER ledger_events_no_update
  BEFORE UPDATE ON fiscal.ledger_events
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TRIGGER ledger_events_no_delete
  BEFORE DELETE ON fiscal.ledger_events
  FOR EACH ROW EXECUTE FUNCTION fiscal.reject_mutation();

CREATE TABLE fiscal.outbox_messages (
  id TEXT NOT NULL PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES fiscal.documents (id) ON DELETE RESTRICT,
  message_type TEXT NOT NULL,
  submission_id TEXT NOT NULL,
  state TEXT NOT NULL,
  available_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT outbox_messages_submission_id_unique UNIQUE (submission_id),
  CONSTRAINT outbox_messages_submission_id_nonempty CHECK (length(trim(submission_id)) > 0),
  CONSTRAINT outbox_messages_message_type_check CHECK (message_type IN ('authority_submission')),
  CONSTRAINT outbox_messages_state_check CHECK (state IN ('pending', 'in_flight', 'succeeded', 'dead'))
);
