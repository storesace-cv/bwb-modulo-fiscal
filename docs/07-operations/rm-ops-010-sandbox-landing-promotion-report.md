# RM-OPS-010 — Promoção sandbox landing `GET /`

**Host:** `sandbox.fiscalmod.bwb.pt` (`194.9.62.239`) — **nunca** o apex de produção.
**Data (UTC):** 2026-07-28
**Item:** `RM-OPS-010` (código já em `main` via PR #133)

## Distinção obrigatória

| Sinal | Significado |
|---|---|
| `GET /v1/health` | Disponibilidade / liveness |
| `GET /` (landing HTML) | **Só UX** — orientação; **não** é health check |
| 404 em path desconhecido | Esperado (allowlist API) |

`FISCAL_ENV=homologation` continua designação técnica BWB — **não** homologação AGT.

## Relatório (pós-deploy)

| # | Declaração | Estado |
|---|---|---|
| 1 | **Código online** — `health.revision` = `current-sha` = tip promovido | **SIM** — `8063763e4bc345815f09b061e13c0782c52781fb` |
| 2 | **Schema limpo** — `version=12` `dirty=false` | **SIM** |
| 3 | **Serviços healthy** — `bwb-fiscal-api` / `nginx` / `postgresql` | **SIM** (`active`) |
| 4 | **Landing** — HTTPS `GET /` 200 HTML amigável | **SIM** (após `nginx-open-arm` + `confirm`) |
| 5 | **Health** — HTTPS `/v1/health` 200 | **SIM** (`revision=8063763…`; `version=0.2.74-staging`) |
| 6 | **Admin login UI** — HTTPS `/admin/ui/login` 200 | **SIM** |
| 7 | **Auth documentos** — `POST /v1/documents` sem token → **401** | **SIM** (inalterado) |
| 8 | **Rate limit documentos** — burst → **429** presente | **SIM** (inalterado; 10r/s burst=20) |
| 9 | **Portas protegidas** — PG/`8080` só loopback; `:18080` ausente | **SIM** |
| 10 | **Nginx open** — `state=confirmed` no tip promovido | **SIM** |

## Pré-voo

| Check | Resultado |
|---|---|
| Tip `origin/main` | `8063763e4bc345815f09b061e13c0782c52781fb` (squash #133) |
| Release anterior | `38f86fb52367937e6595192ab39da68097ff4a53` |
| Deploy lock pré-deploy | ausente |
| Schema live pré-deploy | `12` / `dirty=false` (roll-forward only; `DEPLOY_N1_COMPAT_PROVEN=0`) |
| ExpectedVersion código | `12` |

## Gate e promote

Updater: `scripts/deploy/update-staging.sh` (lock → upload → `install-release` lateral → **pre-deploy pg_dump gate** → env backup/install → migrate new binary → activate → lock release).

| Campo | Valor |
|---|---|
| Deploy lock id | `20260728T220138Z-8063763e4bc345815f09b061e13c0782c52781fb` |
| `pre_deploy_pg_backup` | `ok` (gate obrigatório) |
| `migration_before` | `12` `dirty=false` |
| `migration_after` | `12` `dirty=false` |
| `promote` | `ok` → `current->8063763…` |
| `lock_release` | `ok` |
| Rollback controls | envs restorable pós-backup; previous release `38f86fb…` intacta |

Sem passwords, DSN, NIF, tokens ou conteúdo fiscal em logs deste relatório.

## Nginx open (landing)

O updater **não** activa o site open. Após promote, o site live ainda apontava para open SHA antigo (`1c425ec…`) **sem** `location = /` → HTTPS `GET /` = nginx 404 enquanto loopback API já servia landing 200.

| Passo | Resultado |
|---|---|
| `nginx-open-arm 8063763…` | `nginx_open_arm_ok` + timer activo |
| HTTPS `GET /` | **200** HTML «BWB Módulo Fiscal (sandbox)» + disclaimer «404 na raiz ≠ serviço em baixo» |
| Regressão health / admin / documents | 200 / 200 / **401** |
| Path desconhecido | **404** |
| `nginx-open-confirm 8063763…` | `nginx_open_confirm_ok` |
| Estado final | `state=confirmed` sha=`8063763…`; timer **inactive/disabled** |

Deny-all permanece disponível como fail-safe (`nginx-deny-all`) **sem** proxy da raiz.

## Incidentes

| ID | Severidade | Descrição | Estado |
|---|---|---|---|
| INC-OPS-010-1 | Baixo (UX) | Após promote, open SHA antigo → `GET /` público 404 apesar da API nova | **RESOLVIDO** com `nginx-open-arm` + `confirm` no tip |

## Evidências

- Código: PR #133 — https://github.com/storesace-cv/bwb-modulo-fiscal/pull/133
- Docs UX: [sandbox-root-landing.md](sandbox-root-landing.md)
- Runbook: [staging-runbook.md](staging-runbook.md)
