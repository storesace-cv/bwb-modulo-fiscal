-- Core fiscal tables for SQLite WAL Edge (forward-only; no schema namespace).
PRAGMA foreign_keys = ON;

CREATE TABLE idempotency_records (
  scope_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash BLOB NOT NULL,
  document_id TEXT NULL,
  state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (scope_id, idempotency_key),
  CONSTRAINT idempotency_records_scope_id_nonempty CHECK (length(trim(scope_id)) > 0),
  CONSTRAINT idempotency_records_key_nonempty CHECK (length(trim(idempotency_key)) > 0),
  CONSTRAINT idempotency_records_hash_len CHECK (length(request_hash) = 32),
  CONSTRAINT idempotency_records_state_check CHECK (state IN ('in_progress', 'completed'))
);

CREATE TABLE documents (
  id TEXT NOT NULL PRIMARY KEY,
  scope_id TEXT NOT NULL,
  external_id TEXT NOT NULL,
  document_type TEXT NOT NULL,
  currency TEXT NOT NULL,
  issued_at TEXT NOT NULL,
  requested_series TEXT NULL,
  series_code TEXT NOT NULL,
  fiscal_seq INTEGER NOT NULL,
  seller_tax_id TEXT NOT NULL,
  seller_name TEXT NOT NULL,
  customer_tax_id TEXT NULL,
  customer_name TEXT NULL,
  created_at TEXT NOT NULL,
  sealed_at TEXT NOT NULL,
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
BEFORE UPDATE ON documents
BEGIN
  SELECT RAISE(ABORT, 'documents is append-only');
END;

CREATE TRIGGER documents_no_delete
BEFORE DELETE ON documents
BEGIN
  SELECT RAISE(ABORT, 'documents is append-only');
END;

CREATE TABLE document_lines (
  document_id TEXT NOT NULL REFERENCES documents (id) ON DELETE RESTRICT,
  line_no INTEGER NOT NULL,
  line_id TEXT NOT NULL,
  description TEXT NOT NULL,
  quantity_scaled INTEGER NOT NULL,
  unit_price_cents INTEGER NOT NULL,
  tax_code TEXT NOT NULL,
  PRIMARY KEY (document_id, line_no),
  CONSTRAINT document_lines_line_id_nonempty CHECK (length(trim(line_id)) > 0),
  CONSTRAINT document_lines_description_nonempty CHECK (length(trim(description)) > 0),
  CONSTRAINT document_lines_quantity_positive CHECK (quantity_scaled > 0),
  CONSTRAINT document_lines_unit_price_nonneg CHECK (unit_price_cents >= 0),
  CONSTRAINT document_lines_tax_code_nonempty CHECK (length(trim(tax_code)) > 0)
);

CREATE TRIGGER document_lines_no_update
BEFORE UPDATE ON document_lines
BEGIN
  SELECT RAISE(ABORT, 'document_lines is append-only');
END;

CREATE TRIGGER document_lines_no_delete
BEFORE DELETE ON document_lines
BEGIN
  SELECT RAISE(ABORT, 'document_lines is append-only');
END;

CREATE TABLE series_counters (
  scope_id TEXT NOT NULL,
  series_code TEXT NOT NULL,
  last_seq INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (scope_id, series_code),
  CONSTRAINT series_counters_scope_id_nonempty CHECK (length(trim(scope_id)) > 0),
  CONSTRAINT series_counters_series_code_nonempty CHECK (length(trim(series_code)) > 0),
  CONSTRAINT series_counters_last_seq_nonneg CHECK (last_seq >= 0)
);

CREATE TABLE ledger_events (
  id TEXT NOT NULL PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES documents (id) ON DELETE RESTRICT,
  seq INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  from_status TEXT NULL,
  to_status TEXT NOT NULL,
  created_at TEXT NOT NULL,
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
BEFORE UPDATE ON ledger_events
BEGIN
  SELECT RAISE(ABORT, 'ledger_events is append-only');
END;

CREATE TRIGGER ledger_events_no_delete
BEFORE DELETE ON ledger_events
BEGIN
  SELECT RAISE(ABORT, 'ledger_events is append-only');
END;

CREATE TABLE outbox_messages (
  id TEXT NOT NULL PRIMARY KEY,
  document_id TEXT NOT NULL REFERENCES documents (id) ON DELETE RESTRICT,
  message_type TEXT NOT NULL,
  submission_id TEXT NOT NULL,
  state TEXT NOT NULL,
  available_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CONSTRAINT outbox_messages_submission_id_unique UNIQUE (submission_id),
  CONSTRAINT outbox_messages_submission_id_nonempty CHECK (length(trim(submission_id)) > 0),
  CONSTRAINT outbox_messages_message_type_check CHECK (message_type IN ('authority_submission')),
  CONSTRAINT outbox_messages_state_check CHECK (state IN ('pending', 'in_flight', 'succeeded', 'dead'))
);
