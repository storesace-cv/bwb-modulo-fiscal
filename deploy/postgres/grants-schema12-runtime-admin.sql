-- Operational grants after migrations through 0012 (NOT a golang-migrate file).
-- Apply as DB owner / a role with GRANT privilege (typically postgres OS or fiscal_migrate).
-- Explicit per-object grants only. Never CREATE ROLE. Never invent LOGIN credentials.
--
-- Complements deploy/postgres/grants-schema3-runtime-admin.sql for tables added in
-- 0004…0012 (authority outbox, admin registry, FE enrollment, doc policy, series, ops).
-- Re-run after future migrations with an updated explicit script.
-- Origin: RM-OPS-006 sandbox promote to ExpectedVersion=12.

BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_migrate') THEN
    RAISE EXCEPTION 'grants-schema12: required role fiscal_migrate does not exist';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_runtime') THEN
    RAISE EXCEPTION 'grants-schema12: required role fiscal_runtime does not exist';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_admin') THEN
    RAISE EXCEPTION 'grants-schema12: required role fiscal_admin does not exist';
  END IF;
END
$$;

GRANT USAGE ON SCHEMA fiscal TO fiscal_runtime;
GRANT USAGE ON SCHEMA fiscal TO fiscal_admin;

-- Foundation / outbox (runtime) — reaffirm after schema evolves
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

-- 0003 credential surfaces
REVOKE ALL ON TABLE fiscal.scopes FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.api_credentials FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.audit_events FROM fiscal_runtime, fiscal_admin;

GRANT SELECT ON fiscal.scopes TO fiscal_runtime;
GRANT SELECT ON fiscal.api_credentials TO fiscal_runtime;
GRANT INSERT ON fiscal.audit_events TO fiscal_runtime;

GRANT SELECT, INSERT ON fiscal.scopes TO fiscal_admin;
GRANT SELECT, INSERT ON fiscal.api_credentials TO fiscal_admin;
GRANT UPDATE (status, grace_until, revoked_at) ON fiscal.api_credentials TO fiscal_admin;
GRANT INSERT ON fiscal.audit_events TO fiscal_admin;

-- 0004 authority attempts/responses (outbox worker)
REVOKE ALL ON TABLE fiscal.authority_attempts FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.authority_responses FROM fiscal_runtime, fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.authority_attempts TO fiscal_runtime;
GRANT SELECT, INSERT ON fiscal.authority_responses TO fiscal_runtime;

-- 0005…0012 admin registry / ops — fiscal-api monolith uses fiscal_runtime DSN
REVOKE ALL ON TABLE fiscal.taxpayers FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.establishments FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.scope_bindings FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.admin_audit_events FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.authority_profiles FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.taxpayer_fe_enrollments FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.establishment_doc_group_config FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.establishment_doc_type_config FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.establishment_series FROM fiscal_runtime, fiscal_admin;
REVOKE ALL ON TABLE fiscal.admin_ops_action_idempotency FROM fiscal_runtime, fiscal_admin;

GRANT SELECT, INSERT, UPDATE ON fiscal.taxpayers TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishments TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.scope_bindings TO fiscal_runtime;
GRANT SELECT, INSERT ON fiscal.admin_audit_events TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.authority_profiles TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.taxpayer_fe_enrollments TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishment_doc_group_config TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishment_doc_type_config TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishment_series TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.admin_ops_action_idempotency TO fiscal_runtime;

-- fiscal-admin CLI (admin.env) — same objects
GRANT SELECT, INSERT, UPDATE ON fiscal.taxpayers TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishments TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.scope_bindings TO fiscal_admin;
GRANT SELECT, INSERT ON fiscal.admin_audit_events TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.authority_profiles TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.taxpayer_fe_enrollments TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishment_doc_group_config TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishment_doc_type_config TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.establishment_series TO fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.admin_ops_action_idempotency TO fiscal_admin;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA fiscal TO fiscal_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA fiscal TO fiscal_admin;

REVOKE ALL ON ALL TABLES IN SCHEMA fiscal FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE fiscal_migrate IN SCHEMA fiscal
  REVOKE ALL ON TABLES FROM fiscal_runtime, fiscal_admin;
ALTER DEFAULT PRIVILEGES FOR ROLE fiscal_migrate IN SCHEMA fiscal
  REVOKE ALL ON SEQUENCES FROM fiscal_runtime, fiscal_admin;

COMMIT;
