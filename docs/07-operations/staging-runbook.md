# Staging runbook

**Environment label:** staging (not production).

**Hostname:** `sandbox.fiscalmod.bwb.pt` → API `https://sandbox.fiscalmod.bwb.pt/v1` · health `/v1/health`.

**Apex** `fiscalmod.bwb.pt` is reserved for production after operational approval.

**Progresso do projecto:** [ROADMAP.md](../../ROADMAP.md).

---

## Procedimento actual (pós-S3C2 CONFIRMED)

Estado operacional confirmado do sandbox BWB:

| Campo | Valor actual |
|---|---|
| Auth runtime | `FISCAL_ENV=homologation` + `FISCAL_AUTH_MODE=credential_store` |
| Significado de `homologation` | Designação **técnica** do ambiente sandbox BWB — **não** é homologação oficial AGT, nem certificação |
| Schema | `ExpectedVersion=12` / `dirty=false` |
| HTTPS `/v1/health` | 200 |
| HTTPS `/v1/documents` sem token | **401** |
| HTTPS `/admin/v1/ready` | 200 (`fail_closed`; IdP não configurado) — após **RM-OPS-007** |
| HTTPS `/admin/ui/login` | HTML login; login interactivo **indisponível** até IdP |
| Create / replay autenticados | **201** |
| Rate limit documentos | Nginx `10r/s`, `burst=20`, `limit_req_status 429` |
| S3C2 open state | **`confirmed`** |
| Timer open rollback | **inactive / disabled** |
| Medição `:18080` | **ausente** (desactivada após S3C2) |
| API | `127.0.0.1:8080` (não exposta externamente) |
| PostgreSQL | loopback apenas |
| Admin auth | `fail_closed` até IdP real — [admin-idp-onboarding.md](admin-idp-onboarding.md) |
| Grants pós-0012 | [grants-schema12-runtime-admin.sql](../../deploy/postgres/grants-schema12-runtime-admin.sql) (aplicar após migrate 3→12) |
| Kit POS | validação sandbox **9/9** — [s4-pos-kit-ops-validation-report.md](s4-pos-kit-ops-validation-report.md) |
| Gate pré-deploy `pg_dump` | validado (OPS-B2) — [b2-predeploy-pg-dump-gate-report.md](b2-predeploy-pg-dump-gate-report.md); INC-S4-003 **RESOLVIDO**; INC-B2-001 **aberto** |
| Promote schema12 | **RM-OPS-006 CONCLUÍDO** — [rm-ops-006-sandbox-schema12-promotion-report.md](rm-ops-006-sandbox-schema12-promotion-report.md) |
| Nginx admin proxy | **RM-OPS-007** — open conf com `/admin/v1/` + `/admin/ui`; deny-all sem admin |

Rotação de credenciais no sandbox: `fiscal-admin` issue/rotate/revoke via helper (`credential_store`). **Não** usar `dev_static` no sandbox.

`dev_static` existe apenas para desenvolvimento local com `FISCAL_ENV=development` — ver [local-dev.md](../06-delivery/local-dev.md).

Deny-all permanece disponível como **rollback/fail-safe** (`nginx-deny-all`), não como postura pública live actual.

Relatório de promoção: [s3c2-sandbox-promotion-report.md](s3c2-sandbox-promotion-report.md) (resultado final **CONFIRMED**).

---

## Layout

| Path | Purpose |
|---|---|
| `.env.example` | Template only (versioned) |
| `.env.local` | Operator SSH paths (ignored, `chmod 600`) |
| `.env.deploy.local` | Runtime allowlist → `fiscal.env` (ignored, `600`) |
| `.env.migrate.local` | Migration DSN → `migrate.env` (ignored, `600`) |
| `.env.admin.local` | Admin DSN → `admin.env` (ignored, `600`; S3A+) |
| `deploy/env.allowlist` | Allowed runtime keys |
| `deploy/migrate.env.allowlist` | Allowed migrate keys |
| `deploy/admin.env.allowlist` | Allowed admin keys (`DRIVER`+`URL` only) |
| `deploy/systemd/bwb-fiscal-api.service` | API unit; **only** `fiscal.env` |
| `deploy/nginx/bwb-fiscal-sandbox-http.conf` | HTTP bootstrap (no cert paths; IPv4 only in D1) |
| `deploy/nginx/bwb-fiscal-sandbox-tls.conf` | TLS site templates |
| `/opt/bwb-modulo-fiscal/releases/<sha>/` | Immutable release + `COMMIT` + `SHA256SUMS` |
| `/etc/bwb-modulo-fiscal/fiscal.env` | Runtime `root:root` `0600` |
| `/etc/bwb-modulo-fiscal/migrate.env` | Migration `root:root` `0600` |
| `/etc/bwb-modulo-fiscal/admin.env` | Admin DSN `root:root` `0600` (never systemd; env -i) |
| `/var/lib/bwb-fiscal-admin/tokens/` | Token files `bwb-fiscal-admin` `0700`/`0600` |
| `/etc/bwb-modulo-fiscal/backups/` | Config backups only (never under `/opt/releases`) |

## PostgreSQL roles

- **Runtime role:** CONNECT/USAGE + table privileges strictly needed by the API (SELECT/INSERT/UPDATE as required). Used in `fiscal.env`.
- **Migration role:** used only by `fiscal-migrate` via `migrate.env`. Never load into systemd. Never `source` on the server.
- **Drop-priv migrate user (D2):** `bwb-fiscal-migrate` (`nologin`, system user). The closed helper reads/validates `migrate.env` as root, then runs `fiscal-migrate` via `runuser`/`setpriv` with a cleaned environment (`FISCAL_DATABASE_DRIVER`, `FISCAL_DATABASE_URL` only). Release scripts/binaries are never executed as root.
- Listen on localhost only. API must not run as DB owner/superuser.

## SSH

- Private key stays in `~/.ssh`; `.env.local` stores only the path.
- Confirm host key **fingerprint from the cloud provider panel** before first connect; `ssh-keyscan` alone does not establish trust.
- Dedicated `UserKnownHostsFile` + `StrictHostKeyChecking=yes`.
- Forbidden: `StrictHostKeyChecking=no`, `UserKnownHostsFile=/dev/null`, ignoring pull/build/restart errors.
- Prefer `IdentitiesOnly=yes` and ControlMaster multiplexing; UFW on TCP/22 is ALLOW (not LIMIT) — see [ssh-ufw-limit-remediation-report.md](ssh-ufw-limit-remediation-report.md).

## Nginx (actual + rollback)

- **Live (CONFIRMED):** `location = /v1/documents` open autenticado + `limit_req` 10r/s / burst=20 / 429; health fora do limiter.
- **Rollback:** `nginx-deny-all` / `tls.deny.conf` — fail-safe; não é o estado público actual.
- Application generates `X-Request-Id`; Nginx clears inbound client `X-Request-Id`.
- Always `nginx -t` before reload; failed reload keeps previous config.

## Deploy / rollback

1. Build with `scripts/deploy/build-linux-release.sh` (`GOOS=linux` forced, `CGO_ENABLED=0`, `DEPLOY_GOARCH` amd64|arm64). Refuses dirty worktree; `SHA256SUMS` covers binaries, `lib/*`, `COMMIT`, `EXPECTED_SCHEMA_VERSION` (no release migrate runner).
2. `deploy-lock-acquire` → upload to remote temp → verify full manifest → immutable `releases/<sha>` (lateral) → **pre-deploy pg_dump gate** → env backup then install `0600` (restorable immediately after backup).
3. `migration_before` / `up` / `migration_after` use **`fiscal-migrate` from the new release** via the closed helper (drop-priv), never `current`, never as root.
4. Dirty migration **blocks** promotion.
5. **Before** activation: env restore on failure; binary not switched.
6. **After** activate/restart/health failure: re-read `current`; N-1 rollback (symlink + envs + restart + health) **only** if policy allows (`DEPLOY_N1_COMPAT_PROVEN=1` when schema changed). Otherwise roll-forward/manual.
7. Health accepts only JSON `"status":"ok"` (exact field); does **not** replace `fiscal-migrate version`.
8. Config install: temp file `0600` → atomic install by root under `/etc/bwb-modulo-fiscal/`. Never copy env into release dirs, logs, or reports.
9. D2 bootstrap: install helper + libs + sudoers + create `bwb-fiscal-migrate`. Libs em `/usr/local/lib/bwb-fiscal-deploy/`: `allowlist.sh`, `migrate.env.allowlist`, `admin.env.allowlist`, `parse_migrate_dsn.py`, `predeploy_pg.sh`. Install the versioned fragment **as-is** (no textual substitution): `install -m 0440 -o root -g root deploy/sudoers/bwb-fiscal-deploy /etc/sudoers.d/bwb-fiscal-deploy` then `visudo -cf /etc/sudoers.d/bwb-fiscal-deploy`. The rule is fixed to user `bwb-deploy` and only `/usr/local/sbin/bwb-fiscal-deploy-helper`.

## DNS / TLS (D2)

- A `sandbox.fiscalmod.bwb.pt` → `194.9.62.239`; AAAA only if IPv6 is configured and protected.
- Validate DNS before ACME. No DNS credentials in Git.
- Let’s Encrypt; TLS 1.2/1.3; HTTP→HTTPS redirect after cert issuance; HSTS without `includeSubDomains` only after renewal dry-run succeeds.

## Backups

Document and rehearse PostgreSQL backup + restore in D2. Config backups live under `/etc/bwb-modulo-fiscal/backups/`.

### Pre-deploy `pg_dump` gate (obrigatório no live path)

Antes de `backup-envs` / `install-env` / `migrate` / `activate` / `restart`, o updater:

1. `deploy-lock-acquire` (`/run/bwb-fiscal-deploy/deploy.lockdir`, meta `owner`/`acquired_at_utc`/`state`; stale >1800s sem auto-remove; corrupt/poisoned bloqueiam).
2. Upload + `install-release` **lateral** apenas (`releases/<sha>`; não altera `current`/envs/serviços).
3. `pre-deploy-pg-backup <sha> <backup_id>` — ordem canónica:
   - schema check origem (`SELECT version, dirty FROM public.bwb_schema_migrations;` — exactamente 1 linha, `dirty=false`);
   - `pg_dump -Fc` → workdir;
   - size > 0;
   - `pg_restore --list`;
   - instalação durável atómica em `/var/backups/bwb-fiscal/pre-deploy/<backup_id>.dump` (`O_EXCL` + fsync + `link` NOREPLACE; sem overwrite);
   - `createdb` como OS `postgres` (`-O fiscal_migrate`, `template0`, socket `/var/run/postgresql`, port `5432`);
   - `pg_restore` na BD temp;
   - validar schema temp == `schema_before`;
   - `dropdb` temp.
4. Só com `deploy_allowed=true` seguem mutações activas.
5. `deploy-lock-release` só se `DEPLOY_LOCK_ACQUIRED=1` e `DEPLOY_LOCK_POISONED=0`; falha de release ⇒ exit ≠ 0.

Identidades: dump/restore/psql = `bwb-fiscal-migrate` / `fiscal_migrate` via `PGSERVICEFILE`+`PGSERVICE`+`PGPASSFILE` (parser fechado; host só `127.0.0.1`, user `fiscal_migrate`, `sslmode=require`). **Sem** `CREATEDB`/`SUPERUSER` em `fiscal_migrate`. `createdb`/`dropdb` = OS `postgres` apenas.

Falhas: dump/`--list` ⇒ sem dump durável; restore após install ⇒ dump durável permanece; `dropdb` falhado ⇒ lock `poisoned` (sem release automático). Timeouts: dump/restore 900s; list/createdb/dropdb 60s; psql 30s; TERM→KILL 5s.

Locks `poisoned` / `stale` / `corrupt`: exigem **remediação humana**. O procedimento de teste/remediação **ainda não foi ensaiado** no sandbox (ver INC-B2-001). É **proibido** remover automaticamente estes locks (nenhuma rotina de auto-clean).

Libs do helper: `parse_migrate_dsn.py`, `predeploy_pg.sh` em `/usr/local/lib/bwb-fiscal-deploy/` junto com as allowlists. Relatório B2/fecho INC-S4-003: [b2-predeploy-pg-dump-gate-report.md](b2-predeploy-pg-dump-gate-report.md).

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

## D2 status (2026-07-21)

Bootstrap e primeiro deploy em `sandbox.fiscalmod.bwb.pt` concluídos. Relatório: [d2-staging-bootstrap-report.md](d2-staging-bootstrap-report.md).

---

## Histórico S3A / S3B / S3C (preservado)

As subsecções abaixo documentam a sequência histórica até à promoção. O **estado live actual** está na secção «Procedimento actual».

### S3A / S3B / S3C — sequência histórica

**S3A (repo):** artefacts de build/helper/`admin.env`/grants/Nginx fechado + candidato + medição + runbook.

**S3B (host):** deploy release S3A; migrate `2→3`; grants; provisionar scopes/credenciais; E2E em `http://127.0.0.1:8080`; medição de `limit_req` **apenas** em `http://127.0.0.1:18080`; HTTPS público manteve `/v1/documents` **deny-all** durante S3B.

**S3C-tooling (repo):** binário Go `fiscal-sandbox-measure` + `Health.revision` (SHA40).

**S3C1 (ops):** medição em `:18080` com perfis fechados; 443 documents ainda deny-all; remover `:18080` no fim da medição.

**S3C2 (repo + ops):** valores finais `rate=10r/s` / `burst=20`; site open + deny-all versionados; helper `nginx-open-arm` / `nginx-open-confirm` / `nginx-deny-all` + timer 5 min + boot-recovery.

### S3C2 — abertura controlada (fail-safe)

Artefactos no release (`nginx/tls.open.conf`, `nginx/tls.deny.conf`, `nginx/limit-req-documents.conf`, units systemd rollback + boot-recovery):

| Op fechada | Comportamento |
|---|---|
| `nginx-open-arm <sha40>` | Instala open; exige timer `is-active`; falha → `deny_restored` ou `emergency_nginx_stop` (só se Nginx **comprovadamente** inactivo) ou `emergency_stop_failed` (CRITICAL); reload falhado usa o mesmo fail-closed; **nunca** `arm_ok` sem timer |
| `nginx-open-confirm <sha40>` | Grava `state=confirmed` **antes** de `stop`/`disable` do timer; falha de stop reporta erro mas mantém `confirmed` (fire = noop) |
| `nginx-deny-all <sha40>` | Instala deny-all, `nginx -t`, reload, **probe 403 com retry** (401 transitório até deadline curto); falha → mesmo fail-closed do arm; cancela timer |
| `nginx-open-rollback-fire` | Alvo do timer: se `armed`, deny-all + `state=rolled_back` se 403; senão estado terminal — **nunca** `armed` com timer inactive; se `confirmed`, noop |
| `nginx-open-boot-recovery` | Unit `Before=nginx.service`: se `armed`, deny on disk + `nginx -t`; `confirmed` noop. Drop-in `Requires=`/`After=` recovery |

Todas as ops `nginx-open-*` / `nginx-deny-all` tomam **flock exclusivo**. Updater **não** activa open. Legacy `install-nginx-open` continua rejeitado.

### S3C2 — incidentes pré-merge (corrigidos no Draft)

| # | Achado | Risco | Correcção |
|---|---|---|---|
| 1 | Open aplicado antes do timer active | Nginx aberto sem fail-safe | Fail-closed `deny_restored` ou `emergency_nginx_stop`; exigir `is-active` |
| 2 | Confirm cancelava timer antes de `confirmed` | Aberto + `armed` sem timer | State `confirmed` antes de stop |
| 3 | Sem serialização arm/confirm/rollback | Corrida confirm vs fire | `flock` exclusivo |
| 4 | Reboot / `Before=` sem Requires | Nginx arranca apesar de recovery falhada | Drop-in `Requires=`/`After=` + deny on disk + `-t` |
| 5 | `return 301` server-level | ACME quebrado (D2) | Redirect sob `location /` |
| 6 | HSTS comentado nos templates | Live perderia HSTS | HSTS nos dois artefactos |
| 7 | `location /v1/documents` prefix | Paths semelhantes à API | `location = /v1/documents` |
| 8 | `fail_closed` com `set +e` ignorava restore | Podia reportar fail-closed com Nginx ainda aberto | Validar cada passo; fallback stop |
| 9 | `emergency_stopped` sem prova | Anunciava stop sem `is-active` | Só após inactivo; senão `emergency_stop_failed` |
| 10 | Reload arm com restore fraco | Exposição após reload ambíguo | Mesmo `nginx_fail_closed_deny` |
| 11 | Probe deny-all único; 401 pós-reload → unit failed + `armed` (I1 sandbox) | Aberto/inconsistente sem fail-closed | Retry 403 (401 transitório); deny-all/rollback-fire fail-closed; nunca `armed`+timer inactive |

### S3C2 — promoção sandbox (histórico → resultado final)

Relatório completo: [s3c2-sandbox-promotion-report.md](s3c2-sandbox-promotion-report.md).

Sequência preservada:

1. **Primeira tentativa:** promoção **ROLLED_BACK** (gate do timer / I1) — sem `confirm`.
2. **Correcção I1:** probe deny-all com retry; fail-closed rigoroso (PR #17).
3. **Reteste:** `TIMER_ROLLBACK_APPROVED` no Ubuntu.
4. **Promoção final:** **CONFIRMED** — open autenticado; timer inactive/disabled; cleanup de credenciais efémeras.

O relatório histórico preserva a tentativa ROLLED_BACK e a confirmação posterior. O estado operacional a usar pelos operadores é o da secção «Procedimento actual».

### S3C1 — matriz e thresholds (aprovados; histórico de medição)

Base histórica de medição: `http://127.0.0.1:18080` (hoje **ausente** no host confirmado). Helper: `admin-sandbox-measure <sha40> sustained|burst|replay`.

| Perfil | Pedidos | Pacing | Concorrência | Duração wall | Persistência máx. |
|---|---:|---|---:|---|---:|
| `sustained` | 300 | 10 r/s média (±10% request throughput); agenda monotónica sem catch-up | 1 | 28–33 s | ≤300 docs |
| `burst` | ≤60 | rajada | ≤5 | curta | ≤60 docs |
| `replay` | 2 | sequencial; mesma key/body | 1 | N/A | 1 doc |

Thresholds: `5xx=0`; `409=0`; `other=0`; sustained p95≤250 ms / p99≤500 ms **só em respostas 201**; burst 201 ∈ [20,25] e restantes 429; replay ambos 201 idênticos.

### Topologia (evolução)

| Superfície | Path/porto | S3A | S3B | Actual (pós-S3C2 CONFIRMED) |
|---|---|---|---|---|
| Público TLS | `443` `/v1/documents` | deny-all | deny-all | **aberto autenticado** + `limit_req` |
| Medição | `127.0.0.1:18080` | versionado | activo loopback | **ausente** |
| API directa | `127.0.0.1:8080` | N/A | E2E | debug/loopback |

### admin.env e custódia

| Path | Owner | Mode | Quem lê |
|---|---|---|---|
| `/etc/bwb-modulo-fiscal/admin.env` | `root:root` | `0600` | helper (root) apenas |
| `/var/lib/bwb-fiscal-admin/tokens/` | `bwb-fiscal-admin` | `0700` | só `bwb-fiscal-admin` |
| ficheiro token | `bwb-fiscal-admin` | `0600` | E2E/admin CLI |

Fluxo: helper root → parser allowlist → `env -i` → drop para `bwb-fiscal-admin`. DSN/token **nunca** em argv/stdout/logs.

Ops allowlisted: `admin-scope-create`, `admin-credential-issue|rotate|revoke`, `admin-sandbox-e2e`, `admin-sandbox-measure <sha> sustained|burst|replay`, `admin-sandbox-ab-revoke-gate`, `nginx-open-arm|nginx-open-confirm|nginx-deny-all|nginx-open-boot-recovery`.

### Grants PostgreSQL

- Script: `deploy/postgres/grants-schema3-runtime-admin.sql` — grants explícitos por objeto; **não** cria roles.
- Roles `fiscal_migrate` / `fiscal_runtime` / `fiscal_admin` criadas no bootstrap S3B.

### Sequência S3B (operador — histórico)

Documentada para reexecução de ambientes deny-all / medição. No sandbox actual confirmado, o endpoint público já não está em deny-all; ver «Procedimento actual».

### Rollback deny-all (fail-safe)

1. `nginx-deny-all <sha40>` (ou restaurar site deny versionado).
2. `nginx -t` && reload.
3. Confirmar `/v1/health` OK e `/v1/documents` 403 (quando deny activo).
4. Isto é **recuperação**, não o estado normal pós-CONFIRMED.

### SSH multiplexado

Reutilizar o mux do updater (`ControlMaster` + `ControlPath` + `ControlPersist`). Não abrir tempestade TCP. `deploy_ssh_mux_stop` no EXIT. Proibido `StrictHostKeyChecking=no`.

---

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
| Alto | D1 review | `sudo -n bash` / comandos privilegiados genéricos | Root equivalente para a chave de deploy | Helper fechado + sudoers só para o helper | Corrigido | Bootstrap D2 instala helper/sudoers |
| Médio | D1 review | Envs novos ficavam após falha pré-ativação | Config parcial/inconsistente | Restore/remoção transacional + testes | Corrigido | — |
| Médio | D1 review | `HEALTH_URL` arbitrário no live path | Probe no destino errado / interpolação | URL fixa `127.0.0.1:8080` | Corrigido | — |
| Crítico | D1 review | Helper executava `remote-migrate-run.sh`/`fiscal-migrate` da release como root | RCE root via chave deploy + SHA256SUMS arbitrário | Drop-priv `bwb-fiscal-migrate`; runner removido da release | Corrigido | Bootstrap D2 cria o user |
| Alto | D1 review | `ENVS_INSTALLED` só após ambos os envs; falha no 2.º sem restore | Config parcial (`fiscal.env` novo) | `ENVS_RESTORABLE` pós-backup; cada SCP/install com `pre_activate_fail` | Corrigido | — |
| Alto | D1 review | `restart` falhava após `activate` sem rollback/relatório | Release ativa inconsistente | Rotina `post_activate_fail` + re-leitura de `current` | Corrigido | — |
| Médio | D1 review | Health aceitava `"ok"` em qualquer campo JSON | Falso positivo de health | Matcher estrito `"status":"ok"` | Corrigido | — |
| Alto | D1 review | `run_remote_health` sob `if` ignorava falha real do healthcheck (`set -e` desativado) | `promote=ok` com API unhealthy | Captura explícita do exit status | Corrigido | — |

Report deploy incidents with severity, phase, description, impact, resolution, state, residual risk — **without** secret values.
