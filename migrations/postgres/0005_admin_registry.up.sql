-- Admin registry foundation (DEC-BO-001 plano A): taxpayers, establishments, scope_bindings.
-- No secrets, credentials, private keys, tokens, or private URLs.

CREATE TABLE fiscal.taxpayers (
  taxpayer_id TEXT NOT NULL PRIMARY KEY,
  nif TEXT NOT NULL,
  legal_name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT taxpayers_id_nonempty CHECK (length(trim(taxpayer_id)) > 0),
  CONSTRAINT taxpayers_nif_nonempty CHECK (length(trim(nif)) > 0),
  CONSTRAINT taxpayers_nif_unique UNIQUE (nif),
  CONSTRAINT taxpayers_legal_name_nonempty CHECK (length(trim(legal_name)) > 0),
  CONSTRAINT taxpayers_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE TABLE fiscal.establishments (
  establishment_id TEXT NOT NULL PRIMARY KEY,
  taxpayer_id TEXT NOT NULL REFERENCES fiscal.taxpayers (taxpayer_id) ON DELETE RESTRICT,
  code TEXT NOT NULL,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT establishments_id_nonempty CHECK (length(trim(establishment_id)) > 0),
  CONSTRAINT establishments_code_nonempty CHECK (length(trim(code)) > 0),
  CONSTRAINT establishments_name_nonempty CHECK (length(trim(name)) > 0),
  CONSTRAINT establishments_status_check CHECK (status IN ('active', 'inactive')),
  CONSTRAINT establishments_taxpayer_code_unique UNIQUE (taxpayer_id, code)
);

CREATE TABLE fiscal.scope_bindings (
  scope_id TEXT NOT NULL PRIMARY KEY,
  taxpayer_id TEXT NOT NULL REFERENCES fiscal.taxpayers (taxpayer_id) ON DELETE RESTRICT,
  establishment_id TEXT NOT NULL REFERENCES fiscal.establishments (establishment_id) ON DELETE RESTRICT,
  environment TEXT NOT NULL,
  iana_timezone TEXT NOT NULL,
  series_effective_code TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT scope_bindings_id_nonempty CHECK (length(trim(scope_id)) > 0),
  CONSTRAINT scope_bindings_environment_check CHECK (
    environment IN ('homologation', 'production', 'development')
  ),
  CONSTRAINT scope_bindings_timezone_nonempty CHECK (length(trim(iana_timezone)) > 0),
  CONSTRAINT scope_bindings_series_nonempty CHECK (length(trim(series_effective_code)) > 0),
  CONSTRAINT scope_bindings_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX establishments_taxpayer_id_idx ON fiscal.establishments (taxpayer_id);
CREATE INDEX scope_bindings_taxpayer_id_idx ON fiscal.scope_bindings (taxpayer_id);
CREATE INDEX scope_bindings_establishment_id_idx ON fiscal.scope_bindings (establishment_id);
