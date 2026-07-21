# Staging runbook (PR D1 — repository artefacts only)

**Environment label:** staging (not production).  
**Hostname:** `sandbox.fiscalmod.bwb.pt` → API `https://sandbox.fiscalmod.bwb.pt/v1` · health `/v1/health`.  
**Apex** `fiscalmod.bwb.pt` is reserved for production after real auth + operational approval.  
**Runtime constraint:** while auth is `dev_static`, `FISCAL_ENV` must remain `development` (code).

D1 delivers scripts, systemd, Nginx templates, allowlists, and docs. **D2** (after merge) performs DNS, TLS, hardening, and host install. Do not run live SSH update from this repo until D2.

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
| `deploy/nginx/bwb-fiscal-sandbox-http.conf` | HTTP bootstrap (no cert paths) |
| `deploy/nginx/bwb-fiscal-sandbox-tls.conf` | TLS site (enable after ACME) |
| `/opt/bwb-modulo-fiscal/releases/<sha>/` | Immutable release + `COMMIT` + `SHA256SUMS` |
| `/etc/bwb-modulo-fiscal/fiscal.env` | Runtime `root:root` `0600` |
| `/etc/bwb-modulo-fiscal/migrate.env` | Migration `root:root` `0600` |
| `/etc/bwb-modulo-fiscal/backups/` | Config backups only (never under `/opt/releases`) |

## PostgreSQL roles

- **Runtime role:** CONNECT/USAGE + table privileges strictly needed by the API (SELECT/INSERT/UPDATE as required). Used in `fiscal.env`.
- **Migration role:** used only by `fiscal-migrate` via `migrate.env`. Never load into systemd.
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

## Deploy / rollback

1. Build with `scripts/deploy/build-linux-release.sh` (`GOOS=linux`, `CGO_ENABLED=0`, `DEPLOY_GOARCH` from D2 `uname -m`).
2. Verify `SHA256SUMS` on the server before install; reject commit mismatch.
3. Record `migration_before` / `migration_after` / `dirty` (no DSNs in logs).
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

bash tests/deploy/run-tests.sh
bash scripts/deploy/check-antipatterns.sh
```

## Incidentes

Report deploy incidents with severity, phase, description, impact, resolution, state, residual risk — **without** secret values.
