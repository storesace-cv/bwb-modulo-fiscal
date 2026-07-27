# RM-OPS-006 — Promoção sandbox `main` → schema 12

**Host:** `sandbox.fiscalmod.bwb.pt` (`194.9.62.239`) — **nunca** o apex de produção.  
**Data (UTC):** 2026-07-27  
**Itens:** `RM-OPS-006`, `RM-BO-018`, `RM-OPS-007`

## Relatório separado (pós-deploy)

| # | Declaração | Estado |
|---|---|---|
| 1 | **Código online** — `health.revision` = `current-sha` = tip promovido | **SIM** — `c27b75ef7c354e54f009243784dc3a0b0a12c4f5` (revalidar = `origin/main` após squash merge deste PR) |
| 2 | **Admin UI alcançável** — HTTPS `/admin/ui/login` HTML; `/admin/ui/` protegido 401 | **SIM** (após `nginx-open-arm` + `confirm` RM-OPS-007) |
| 3 | **Login interactivo indisponível** — até IdP real + `oidc_jwt` + redirect authorize | **SIM** (`admin_interactive_login=unavailable`, `admin_oidc=not_configured`, `admin_auth_mode=fail_closed`) |

## Pré-voo (auditoria)

| Check | Resultado |
|---|---|
| SSH | `~/.ssh/digitalocean` + `IdentitiesOnly=yes` + known_hosts dedicado |
| `main` limpo | tip pré-promote `dca98a4…`; pós-#120 `c82255c…` |
| ExpectedVersion código | `12` |
| Allowlist deploy default | `DEPLOY_EXPECTED_SCHEMA_VERSION_DEFAULT=12` (PR #120) |
| Schema live pré-deploy | `version=3` `dirty=false` |
| Disco | ~142 GiB livres em `/` |
| Portas | PG `127.0.0.1:5432`; API `127.0.0.1:8080`; 80/443 públicos |
| Timer open rollback | inactive / unit file disabled (estado `confirmed` S3C2) |
| Deploy lock | ausente pré-deploy |
| `FISCAL_ENV` | `homologation` (técnico BWB ≠ AGT) |
| API pública | `credential_store`; sem token → 401 preservado |
| AGT | sem credenciais reais; simulator não activado em production |
| IdP admin | sem IdP local inseguro; fail-closed |

## Gate e promote (`c82255c…`)

Updater: `scripts/deploy/update-staging.sh` (lock → upload → `install-release` lateral → **pre-deploy pg_dump gate** → env backup/install → migrate new binary → activate).

| Campo | Valor |
|---|---|
| Deploy lock id | `20260727T185834Z-c82255cba1bb135c0ed9df52ba6e65514ecc9988` |
| `migration_before` | `3` `dirty=false` |
| `migration_after` | `12` `dirty=false` |
| `DEPLOY_N1_COMPAT_PROVEN` | `0` (schema mudou → roll-forward only) |
| `promote` | `ok` |
| `current-sha` | `c82255cba1bb135c0ed9df52ba6e65514ecc9988` |
| HTTPS `/v1/health` | 200; `revision=c82255c…`; `version=0.2.73-staging` |
| HTTPS `POST /v1/documents` sem token | **401** |
| PG/API externos | fechados (`5432`/`8080`) |
| Nginx `-t` | ok |
| Lock pós-deploy | libertado |
| Segredos em journal | sem matches de password/PEM/NIF |

### Grants pós-0012

Aplicado como root (fora do helper): `deploy/postgres/grants-schema12-runtime-admin.sql` → roles `fiscal_runtime` / `fiscal_admin` no schema `fiscal`.  
**Nota:** grants **não** vão nas migrations (CI sem roles runtime/admin).

### Admin readiness (loopback)

`GET http://127.0.0.1:8080/admin/v1/ready`:

- `admin_auth_mode=fail_closed`
- `admin_oidc=not_configured`
- `admin_interactive_login=unavailable`
- `database=ok`

Antes de RM-OPS-007, Nginx open **não** fazia proxy de `/admin/*` (404 público).

## RM-OPS-007 — proxy Nginx admin (sandbox) — CONCLUÍDO

Alteração canónica em `deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf`:

- `location ^~ /admin/v1/` → upstream API
- `location ^~ /admin/ui` → upstream API
- `tls.deny.conf` permanece **sem** proxy admin (fail-safe)

### Execução 2026-07-27T19:02Z

| Passo | Resultado |
|---|---|
| Deploy tip | `c27b75ef7c354e54f009243784dc3a0b0a12c4f5` (schema 12→12 `dirty=false`) |
| `nginx-open-arm` | `nginx_open_arm_ok` + timer activo |
| HTTPS `/v1/health` | 200; `revision=c27b75e…` |
| HTTPS `POST /v1/documents` | **401** |
| HTTPS `/admin/v1/ready` | 200; fail_closed / oidc not_configured / interactive unavailable |
| HTTPS `/admin/ui/login` | **200** HTML «Entrar · BWB Admin» |
| HTTPS `/admin/ui/` | **401** |
| `nginx-open-confirm` | `nginx_open_confirm_ok` |
| Estado final | `state=confirmed` sha=`c27b75e…`; timer **inactive/disabled** |

## Incidentes

| ID | Severidade | Descrição | Estado |
|---|---|---|---|
| INC-OPS-006-1 | Alto (pré-deploy) | Default allowlist schema `3` vs código `12` | **RESOLVIDO** em PR #120 (`EXPECTED_SCHEMA_VERSION` default 12) |
| INC-OPS-006-2 | Alto (pós-migrate) | Tabelas 0004…0012 sem grants automáticos | **RESOLVIDO** com `grants-schema12-runtime-admin.sql` aplicado no host |
| INC-OPS-006-3 | Médio | Nginx open sem `/admin/*` → UI/ready públicos 404 | **RESOLVIDO** via RM-OPS-007 |

## Evidências de código

- PR #120 — schema12 readiness + IdP diagnostics: https://github.com/storesace-cv/bwb-modulo-fiscal/pull/120  
- PR de promoção/Nginx — (preencher após merge)

## Não feito / bloqueios conscientes

- IdP real / `oidc_jwt` no sandbox: **bloqueado** até decisão de segurança (fornecedor + tenants); allowlist `fiscal.env` ainda sem chaves OIDC admin.
- Credenciais AGT reais: **não** criadas.
- Apex `fiscalmod.bwb.pt`: **fora de âmbito**.
