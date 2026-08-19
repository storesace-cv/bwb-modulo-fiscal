-- Operational grants after migration 0014 (NOT a golang-migrate file).
-- fe_fixture_submissions: persistent BWB-MOCK queue (ciphertext-free metadata only).

BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_migrate') THEN
    RAISE EXCEPTION 'grants-schema14: required role fiscal_migrate does not exist';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_runtime') THEN
    RAISE EXCEPTION 'grants-schema14: required role fiscal_runtime does not exist';
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'fiscal_admin') THEN
    RAISE EXCEPTION 'grants-schema14: required role fiscal_admin does not exist';
  END IF;
END
$$;

GRANT USAGE ON SCHEMA fiscal TO fiscal_runtime;
GRANT USAGE ON SCHEMA fiscal TO fiscal_admin;

REVOKE ALL ON TABLE fiscal.fe_fixture_submissions FROM fiscal_runtime, fiscal_admin;
GRANT SELECT, INSERT, UPDATE ON fiscal.fe_fixture_submissions TO fiscal_runtime;
GRANT SELECT, INSERT, UPDATE ON fiscal.fe_fixture_submissions TO fiscal_admin;

REVOKE ALL ON ALL TABLES IN SCHEMA fiscal FROM PUBLIC;

ALTER DEFAULT PRIVILEGES FOR ROLE fiscal_migrate IN SCHEMA fiscal
  REVOKE ALL ON TABLES FROM fiscal_runtime, fiscal_admin;

COMMIT;
