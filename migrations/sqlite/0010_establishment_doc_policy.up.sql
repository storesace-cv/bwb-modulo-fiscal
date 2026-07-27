-- Establishment document group/type activation (DEC-PROD-003 / RM-BO-013).
-- No secrets; canonical codes only from seed catalog (never invent FE-RNG).

CREATE TABLE establishment_doc_group_config (
  establishment_id TEXT NOT NULL REFERENCES establishments (establishment_id) ON DELETE RESTRICT,
  environment TEXT NOT NULL,
  grupo TEXT NOT NULL,
  active INTEGER NOT NULL,
  updated_at TEXT NOT NULL,
  CONSTRAINT establishment_doc_group_config_pk PRIMARY KEY (establishment_id, environment, grupo),
  CONSTRAINT establishment_doc_group_config_environment_check CHECK (
    environment IN ('homologation', 'production', 'development')
  ),
  CONSTRAINT establishment_doc_group_config_grupo_check CHECK (
    grupo IN ('vendas', 'movimentacao', 'conferencia', 'pagamentos', 'compras')
  ),
  CONSTRAINT establishment_doc_group_config_active_check CHECK (active IN (0, 1))
);

CREATE TABLE establishment_doc_type_config (
  establishment_id TEXT NOT NULL REFERENCES establishments (establishment_id) ON DELETE RESTRICT,
  environment TEXT NOT NULL,
  codigo_canonico TEXT NOT NULL,
  active INTEGER NOT NULL,
  updated_at TEXT NOT NULL,
  CONSTRAINT establishment_doc_type_config_pk PRIMARY KEY (establishment_id, environment, codigo_canonico),
  CONSTRAINT establishment_doc_type_config_environment_check CHECK (
    environment IN ('homologation', 'production', 'development')
  ),
  CONSTRAINT establishment_doc_type_config_canon_nonempty CHECK (length(trim(codigo_canonico)) > 0),
  CONSTRAINT establishment_doc_type_config_active_check CHECK (active IN (0, 1))
);

CREATE INDEX establishment_doc_group_config_est_idx
  ON establishment_doc_group_config (establishment_id);
CREATE INDEX establishment_doc_type_config_est_idx
  ON establishment_doc_type_config (establishment_id);
