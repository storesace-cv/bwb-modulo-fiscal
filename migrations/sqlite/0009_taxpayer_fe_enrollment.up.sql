-- FE enrollment status per taxpayer + environment (DEC-PROD-004 / RM-BO-012).
-- Canonical enum — never a boolean. No secrets / NIF duplication.

CREATE TABLE taxpayer_fe_enrollments (
  taxpayer_id TEXT NOT NULL REFERENCES taxpayers (taxpayer_id) ON DELETE RESTRICT,
  environment TEXT NOT NULL,
  status TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CONSTRAINT taxpayer_fe_enrollments_pk PRIMARY KEY (taxpayer_id, environment),
  CONSTRAINT taxpayer_fe_enrollments_environment_check CHECK (
    environment IN ('homologation', 'production', 'development')
  ),
  CONSTRAINT taxpayer_fe_enrollments_status_check CHECK (
    status IN ('not_enrolled', 'pending', 'active', 'suspended')
  )
);

CREATE INDEX taxpayer_fe_enrollments_environment_idx
  ON taxpayer_fe_enrollments (environment);
