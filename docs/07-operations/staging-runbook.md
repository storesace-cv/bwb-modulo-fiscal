# Staging runbook (PR D1 — repository artefacts only)

**Environment label:** staging (not production).

**Hostname:** `sandbox.fiscalmod.bwb.pt` → API `https://sandbox.fiscalmod.bwb.pt/v1` · health `/v1/health`.

**Apex** `fiscalmod.bwb.pt` is reserved for production after real auth + operational approval.

**Runtime constraint:** while auth is `dev_static`, `FISCAL_ENV` must remain `development` (code).

D1 delivers scripts, systemd, Nginx templates, allowlists, and docs. Live updater path is implemented in-repo and covered by PATH mocks; **D2** performs DNS, TLS, hardening, and the first real host install. Do not point live SSH at the server until D2.

## Layout

| Path | Purpose |
|---|---|
| `.env.example` | Template only (versioned) |
| `.env.local` | Operator SSH paths (ignored, `chmod 600`) |
| `.env.deploy.local` | Runtime allowlist → `fiscal.env` (ignored, `600`) |
| `.env.migrate.local` | Migration DSN → `migrate.env` (ignored, `600`) |
| `deploy/env.allowlist` | Allowed runtime keys |
| `deploy/migrate.env.allowlist` | Allowed migrate keys |
| `deploy/systemd/bwb-fiscal-api.service` | API unit; **only** `fiscal.env` |
| `deploy/nginx/bwb-fiscal-sandbox-http.conf` | HTTP bootstrap (no cert paths; IPv4 only in D1) |
| `deploy/nginx/bwb-fiscal-sandbox-tls.conf` | TLS site (enable after ACME; IPv4 only in D1) |
| `/opt/bwb-modulo-fiscal/releases/<sha>/` | Immutable release + `COMMIT` + `SHA256SUMS` |
| `/etc/bwb-modulo-fiscal/fiscal.env` | Runtime `root:root` `0600` |
| `/etc/bwb-modulo-fiscal/migrate.env` | Migration `root:root` `0600` |
| `/etc/bwb-modulo-fiscal/backups/` | Config backups only (never under `/opt/releases`) |

## PostgreSQL roles

- **Runtime role:** CONNECT/USAGE + table privileges strictly needed by the API (SELECT/INSERT/UPDATE as required). Used in `fiscal.env`.
- **Migration role:** used only by `fiscal-migrate` via `migrate.env`. Never load into systemd. Never `source` on the server — use `remote-migrate-run.sh`.
- Listen on localhost only. API must not run as DB owner/superuser.

## SSH

- Private key stays in `~/.ssh`; `.env.local` stores only the path.
- Confirm host key **fingerprint from the cloud provider panel** before first connect; `ssh-keyscan` alone does not establish trust.
- Dedicated `UserKnownHostsFile` + `StrictHostKeyChecking=yes`.
- Forbidden: `StrictHostKeyChecking=no`, `UserKnownHostsFile=/dev/null`, ignoring pull/build/restart errors.

## Nginx

- Versioned `/v1/documents` is **`deny all`**. Real IP allowlists are a **non-versioned** server overlay in D2 — never commit open placeholders.
- `/v1/health` may be public (no secrets in payload).
- Application generates `X-Request-Id`; Nginx clears inbound client `X-Request-Id`.
- Always `nginx -t` before reload; failed reload keeps previous config.
- HTTP bootstrap and TLS configs are separate so bootstrap never references missing certificates.
- IPv6 `listen [::]` is **not** enabled in D1; add only in D2 after address, firewall, and AAAA decision.

## Deploy / rollback

1. Build with `scripts/deploy/build-linux-release.sh` (`GOOS=linux` forced, `CGO_ENABLED=0`, `DEPLOY_GOARCH` amd64|arm64). Refuses dirty worktree; `SHA256SUMS` covers `fiscal-api`, `fiscal-migrate`, and `COMMIT`.
2. Upload to remote temp → verify `COMMIT`/`SHA256SUMS` → immutable `releases/<sha>` → atomic env install `0600`.
3. `migration_before` / `up` / `migration_after` use **`fiscal-migrate` from the new release**, never `current`.
4. Dirty migration **blocks** promotion.
5. **Before** migration: binary rollback allowed.
6. **After** schema-changing migration: automatic binary rollback **only** if `DEPLOY_N1_COMPAT_PROVEN=1` (explicit N-1 compatibility proof). Otherwise roll-forward/manual — never restore a binary that cannot write new NOT NULL columns.
7. Health check does **not** replace `fiscal-migrate version`.
8. Config install: temp file `0600` → atomic install by root under `/etc/bwb-modulo-fiscal/`. Never copy env into release dirs, logs, or reports.

## Token rotation (`dev_static`)

Rotate `FISCAL_AUTH_DEV_TOKEN` when compromised, when operators leave, or on a scheduled cadence. Update `fiscal.env` only; restart API; never log the token.

## DNS / TLS (D2)

- A `sandbox.fiscalmod.bwb.pt` → `194.9.62.239`; AAAA only if IPv6 is configured and protected.
- Validate DNS before ACME. No DNS credentials in Git.
- Let’s Encrypt; TLS 1.2/1.3; HTTP→HTTPS redirect after cert issuance; HSTS without `includeSubDomains` only after renewal dry-run succeeds.

## Backups

Document and rehearse PostgreSQL backup + restore in D2. Config backups live under `/etc/bwb-modulo-fiscal/backups/`.

## Scripts

```bash
# Local dry-run (no SSH)
export DEPLOY_DRY_RUN=1 EXPECTED_COMMIT="$(git rev-parse HEAD)" DEPLOY_GOARCH=amd64
bash scripts/deploy/update-staging.sh

# Live path with mocked ssh/scp/sudo/systemctl (no network)
# See tests/deploy/run-tests.sh

bash tests/deploy/run-tests.sh
bash scripts/deploy/check-antipatterns.sh
```

## Incidentes (D1 review)

| Severidade | Fase | Descrição | Impacto | Resolução | Estado | Risco residual |
|---|---|---|---|---|---|---|
| Médio | D1 review | Updater live ausente apesar de D1 o prometer (`update-staging.sh` stub D2-only) | Deploy real impossível / dry-run apenas | Implementado caminho live + mocks PATH | Corrigido | Execução real só em D2 |
| Médio | D1 review | `migrate-remote` usava `fiscal-migrate` de `current` em vez da nova release | Schema/binário desalinhados no `up` | Runner + `RELEASE_DIR` da nova release via sudo | Corrigido | — |
| Alto | D1 review | `source migrate.env` podia interpretar conteúdo como shell | RCE/config injection via DSN/token | Leitura segura + validação exacta; sem `source`/`eval` | Corrigido | — |
| Médio | D1 review | Artefacto podia ser produzido de working tree diferente do `COMMIT` | Release mentia sobre o commit | Build recusa dirty tree; manifesto completo | Corrigido | — |
| Baixo | D1 review | `git diff --check` com trailing whitespace apesar do relatório OK | Qualidade/CI falsa | Removido; CI usa `base...HEAD` | Corrigido | — |
| Alto | D1 review | `sudo`/`systemctl` exigidos no Mac do operador | Deploy impossível fora do servidor | Operações privilegiadas só via SSH remoto | Corrigido | — |
| Alto | D1 review | Healthcheck no host do operador; `promote=ok` prematuro | Falso positivo de deploy | Health em `127.0.0.1` remoto; promote só após health | Corrigido | — |
| Médio | D1 review | Upload `/tmp` previsível; sem backup/restore de envs | Race/leak; rollback incompleto | `mktemp -d` 0700; backups root 0600 + restore | Corrigido | — |
| Médio | D1 review | Manifesto incompleto; schema não validado após `up` | Helpers/schema desalinhados | `EXPECTED_SCHEMA_VERSION` + checksum de helpers | Corrigido | — |

Report deploy incidents with severity, phase, description, impact, resolution, state, residual risk — **without** secret values.
