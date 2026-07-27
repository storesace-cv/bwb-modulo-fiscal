-- Establishment document series metadata (RM-BO-014 / DEC-PROD-004).
-- Local admin registry only — NOT fiscal numbering authority; no AGT solicitarSerie.
-- Codes unique per establishment+environment; lifecycle draft→active→closed (no reuse/backwards).

CREATE TABLE fiscal.establishment_series (
  series_id TEXT NOT NULL PRIMARY KEY,
  establishment_id TEXT NOT NULL REFERENCES fiscal.establishments (establishment_id) ON DELETE RESTRICT,
  environment TEXT NOT NULL,
  codigo_canonico TEXT NOT NULL,
  code TEXT NOT NULL,
  status TEXT NOT NULL,
  valid_from TIMESTAMPTZ NOT NULL,
  valid_to TIMESTAMPTZ NULL,
  version INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT establishment_series_id_nonempty CHECK (length(trim(series_id)) > 0),
  CONSTRAINT establishment_series_code_nonempty CHECK (length(trim(code)) > 0),
  CONSTRAINT establishment_series_canon_nonempty CHECK (length(trim(codigo_canonico)) > 0),
  CONSTRAINT establishment_series_environment_check CHECK (
    environment IN ('homologation', 'production', 'development')
  ),
  CONSTRAINT establishment_series_status_check CHECK (
    status IN ('draft', 'active', 'closed')
  ),
  CONSTRAINT establishment_series_version_positive CHECK (version >= 1),
  CONSTRAINT establishment_series_code_unique UNIQUE (establishment_id, environment, code)
);

CREATE UNIQUE INDEX establishment_series_one_active_per_type_idx
  ON fiscal.establishment_series (establishment_id, environment, codigo_canonico)
  WHERE status = 'active';

CREATE INDEX establishment_series_est_env_idx
  ON fiscal.establishment_series (establishment_id, environment);
