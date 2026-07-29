# RM-OPS-011 — Promoção sandbox `main` → schema 13 + SecretStore

**Host:** `sandbox.fiscalmod.bwb.pt` (`194.9.62.239`) — **nunca** o apex de produção.
**Data (UTC):** 2026-07-29
**Itens:** `RM-OPS-011` (fecho) · readiness [#154](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/154)

## Relatório pós-deploy

| # | Declaração | Estado |
|---|---|---|
| 1 | **Código online** — `health.revision` = tip promovido | **SIM** — `0fbac6ad9c92d1b87258f6a47571021a721807b7` |
| 2 | **Schema** — `version=13` `dirty=false` | **SIM** (`public.bwb_schema_migrations`) |
| 3 | **Tabela** `fiscal.secret_store_entries` | **SIM** |
| 4 | **Grants schema13** (`fiscal_runtime` SELECT/INSERT) | **SIM** |
| 5 | **Master key** presente em `fiscal.env` (0600) | **SIM** — fingerprint SHA-256 `368b49fa…` (valor **nunca** no Git/logs) |
| 6 | **API sem token** | **401** em `POST /v1/documents` |
| 7 | **Admin** fail_closed / OIDC not_configured / interactive unavailable | **SIM** |
| 8 | **UI login** HTTPS | **200** |
| 9 | **Journal** sem PEM/password/master_key | **SIM** (`journal_clean`) |

## Pré-voo

| Check | Resultado |
|---|---|
| Tip pré-promote | `8063763e4bc345815f09b061e13c0782c52781fb` (schema 12) |
| Tip pós-promote | `0fbac6ad9c92d1b87258f6a47571021a721807b7` |
| Allowlist default | `DEPLOY_EXPECTED_SCHEMA_VERSION_DEFAULT=13` (PR #154) |
| `fiscal.env` allowlist | `FISCAL_SECRETSTORE_BACKEND` + `FISCAL_SECRETSTORE_MASTER_KEY` + `FISCAL_AUTHORITY` |
| `FISCAL_ENV` | `homologation` (técnico BWB ≠ AGT) |
| `FISCAL_AUTHORITY` | `simulator` |
| AGT | sem credenciais reais |

## Gate e promote

Updater: `scripts/deploy/update-staging.sh`

| Campo | Valor |
|---|---|
| Deploy lock id | `20260729T110949Z-0fbac6ad9c92d1b87258f6a47571021a721807b7` |
| `migration_before` | `12` `dirty=false` |
| `migration_after` | `13` `dirty=false` |
| `promote` | `ok` |
| `current-sha` | `0fbac6ad9c92d1b87258f6a47571021a721807b7` |
| HTTPS `/v1/health` | 200; `revision=0fbac6a…` |
| Lock pós-deploy | libertado |

### Grants pós-0013

Aplicado como `postgres` (fora do helper): [`deploy/postgres/grants-schema13-runtime-admin.sql`](../../deploy/postgres/grants-schema13-runtime-admin.sql).
Grants **não** vão nas migrations (CI sem roles runtime/admin).

## Incidentes

Nenhum. Master key gerada localmente no operador e instalada via `install-env` allowlisted — valor ausente de Git, journal e relatório (só fingerprint).

## Evidências de código

- PR #154 — readiness allowlist/schema default: https://github.com/storesace-cv/bwb-modulo-fiscal/pull/154
- PR deste relatório — fecho RM-OPS-011
- ≠ homologação oficial AGT; ≠ KMS; ciphertext-only em BD
