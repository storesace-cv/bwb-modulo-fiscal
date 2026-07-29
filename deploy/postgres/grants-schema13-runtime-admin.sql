-- Operational grants after migration 0013 (NOT a golang-migrate file).
-- Apply as DB owner / a role with GRANT privilege (typically postgres OS or fiscal_migrate).
-- Explicit per-object grants only. Never CREATE ROLE. Never invent LOGIN credentials.
--
-- Complements grants-schema12 for secret_store_entries (ciphertext only; ≠ plaintext).
-- Origin: RM-AGTPREP-014 durable encrypted SecretStore scaffolding.

BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_migrate') THEN
    RAISE EXCEPTION 'grants-schema13: required role fiscal_migrate does not exist';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_runtime') THEN
    RAISE EXCEPTION 'grants-schema13: required role fiscal_runtime does not exist';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_admin') THEN
    RAISE EXCEPTION 'grants-schema13: required role fiscal_admin does not exist';
  END IF;
END
$$;

GRANT USAGE ON SCHEMA fiscal TO fiscal_runtime;
GRANT USAGE ON SCHEMA fiscal TO fiscal_admin;

REVOKE ALL ON TABLE fiscal.secret_store_entries FROM fiscal_runtime, fiscal_admin;

-- fiscal-api monolith (SecAdm write-only + metadata) uses fiscal_runtime DSN
GRANT SELECT, INSERT, UPDATE ON fiscal.secret_store_entries TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.secret_store_entries TO fiscal_admin;

REVOKE ALL ON ALL TABLES IN SCHEMA fiscal FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE fiscal_migrate IN SCHEMA fiscal
  REVOKE ALL ON TABLES FROM fiscal_runtime, fiscal_admin;
ALTER DEFAULT PRIVILEGES FOR ROLE fiscal_migrate IN SCHEMA fiscal
  REVOKE ALL ON SEQUENCES FROM fiscal_runtime, fiscal_admin;

COMMIT;
