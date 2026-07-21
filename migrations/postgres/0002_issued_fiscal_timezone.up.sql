-- DEC-TIME-001: fiscal issued_at context (timezone + offset). Forward-only.
-- Abort if any legacy documents or idempotency_records exist (canonical_v1 / missing temporal context).

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM fiscal.documents LIMIT 1)
     OR EXISTS (SELECT 1 FROM fiscal.idempotency_records LIMIT 1) THEN
    RAISE EXCEPTION
      'migration 0002 aborted: legacy documents or idempotency_records present; recreate the development database (canonical_v1 / no fiscal timezone). Do not recalculate hashes.';
  END IF;
END $$;

ALTER TABLE fiscal.documents
  ADD COLUMN issued_timezone TEXT NOT NULL,
  ADD COLUMN issued_offset_minutes INTEGER NOT NULL;

ALTER TABLE fiscal.documents
  ADD CONSTRAINT documents_issued_timezone_nonempty
    CHECK (length(trim(issued_timezone)) > 0);

ALTER TABLE fiscal.documents
  ADD CONSTRAINT documents_issued_offset_range
    CHECK (issued_offset_minutes BETWEEN -840 AND 840);
