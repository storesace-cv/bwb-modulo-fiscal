-- Operational grants after migration 0003 (NOT a golang-migrate file).
-- Apply as DB owner / fiscal_migrate. Explicit per-object grants only.
-- No generic ALTER DEFAULT PRIVILEGES granting SELECT/INSERT/UPDATE to runtime/admin.
--
-- Origin: S3A plan. Re-run after each future migration with an updated explicit script.

BEGIN;

-- Ensure roles exist (cluster-level; idempotent).
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_migrate') THEN
    CREATE ROLE fiscal_migrate NOLOGIN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_runtime') THEN
    CREATE ROLE fiscal_runtime LOGIN;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_admin') THEN
    CREATE ROLE fiscal_admin LOGIN;
  END IF;
END
$$;

-- Schema usage
GRANT USAGE ON SCHEMA fiscal TO fiscal_runtime;
GRANT USAGE ON SCHEMA fiscal TO fiscal_admin;

-- Strip prior broad table privileges on auth surfaces, then grant explicitly.
REVOKE ALL ON TABLE fiscal.scopes FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.api_credentials FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.audit_events FROM fiscal_runtime, fiscal_admin;

GRANT SELECT ON fiscal.scopes TO fiscal_runtime;
GRANT SELECT ON fiscal.api_credentials TO fiscal_runtime;
GRANT INSERT ON fiscal.audit_events TO fiscal_runtime;

GRANT SELECT, INSERT ON fiscal.scopes TO fiscal_admin;
GRANT SELECT, INSERT ON fiscal.api_credentials TO fiscal_admin;
-- Column-level UPDATE only (revoke/rotate/grace). Table-level UPDATE must stay revoked.
GRANT UPDATE (status, grace_until, revoked_at) ON fiscal.api_credentials TO fiscal_admin;
GRANT INSERT ON fiscal.audit_events TO fiscal_admin;

-- SealInTx / document persistence (runtime only; explicit, no admin)
REVOKE ALL ON TABLE fiscal.documents FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.document_lines FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.ledger_events FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.outbox_messages FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.idempotency_records FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.series_counters FROM fiscal_runtime, fiscal_admin;

GRANT SELECT, INSERT ON fiscal.documents TO fiscal_runtime;
GRANT SELECT, INSERT ON fiscal.document_lines TO fiscal_runtime;
GRANT SELECT, INSERT ON fiscal.ledger_events TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.outbox_messages TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.idempotency_records TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.series_counters TO fiscal_runtime;

-- Sequences used by inserts (existing objects only; not a future DEFAULT PRIVILEGE grant)
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA fiscal TO fiscal_runtime;

REVOKE ALL ON ALL TABLES IN SCHEMA fiscal FROM PUBLIC;

-- Prefer removal of broad defaults for migrate-owned future objects.
ALTER DEFAULT PRIVILEGES FOR ROLE fiscal_migrate IN SCHEMA fiscal
  REVOKE ALL ON TABLES FROM fiscal_runtime, fiscal_admin;
ALTER DEFAULT PRIVILEGES FOR ROLE fiscal_migrate IN SCHEMA fiscal
  REVOKE ALL ON SEQUENCES FROM fiscal_runtime, fiscal_admin;

COMMIT;
