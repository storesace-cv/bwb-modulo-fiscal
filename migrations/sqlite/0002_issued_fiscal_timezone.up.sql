-- DEC-TIME-001: fiscal issued_at context. Forward-only.
-- Abort if legacy documents or idempotency_records exist (canonical_v1 / missing temporal context).
-- Empty-table rebuild: create documents_new, drop documents, rename — do NOT RENAME first
-- (SQLite rewrites child FKs to the renamed table name).

CREATE TABLE _mig_0002_guard (
  ok INTEGER NOT NULL CHECK (ok = 1)
);
INSERT INTO _mig_0002_guard (ok)
SELECT CASE
  WHEN (SELECT COUNT(*) FROM documents) + (SELECT COUNT(*) FROM idempotency_records) > 0 THEN 0
  ELSE 1
END;
DROP TABLE _mig_0002_guard;

PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS documents_no_update;
DROP TRIGGER IF EXISTS documents_no_delete;

CREATE TABLE documents_new (
  id TEXT NOT NULL PRIMARY KEY,
  scope_id TEXT NOT NULL,
  external_id TEXT NOT NULL,
  document_type TEXT NOT NULL,
  currency TEXT NOT NULL,
  issued_at TEXT NOT NULL,
  issued_timezone TEXT NOT NULL,
  issued_offset_minutes INTEGER NOT NULL,
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
  CONSTRAINT documents_issued_timezone_nonempty CHECK (length(trim(issued_timezone)) > 0),
  CONSTRAINT documents_issued_offset_range CHECK (issued_offset_minutes BETWEEN -840 AND 840),
  CONSTRAINT documents_scope_external_unique UNIQUE (scope_id, external_id),
  CONSTRAINT documents_scope_series_seq_unique UNIQUE (scope_id, series_code, fiscal_seq)
);

DROP TABLE documents;
ALTER TABLE documents_new RENAME TO documents;

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

PRAGMA foreign_keys = ON;
