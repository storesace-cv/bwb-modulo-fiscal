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
| `.env.admin.local` | Admin DSN → `admin.env` (ignored, `600`; S3A+) |
| `deploy/env.allowlist` | Allowed runtime keys |
| `deploy/migrate.env.allowlist` | Allowed migrate keys |
| `deploy/admin.env.allowlist` | Allowed admin keys (`DRIVER`+`URL` only) |
| `deploy/systemd/bwb-fiscal-api.service` | API unit; **only** `fiscal.env` |
| `deploy/nginx/bwb-fiscal-sandbox-http.conf` | HTTP bootstrap (no cert paths; IPv4 only in D1) |
| `deploy/nginx/bwb-fiscal-sandbox-tls.conf` | TLS site (enable after ACME; IPv4 only in D1) |
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

## Nginx

- Versioned `/v1/documents` is **`deny all`**. Real IP allowlists are a **non-versioned** server overlay in D2 — never commit open placeholders.
- `/v1/health` may be public (no secrets in payload).
- Application generates `X-Request-Id`; Nginx clears inbound client `X-Request-Id`.
- Always `nginx -t` before reload; failed reload keeps previous config.
- HTTP bootstrap and TLS configs are separate so bootstrap never references missing certificates.
- IPv6 `listen [::]` is **not** enabled in D1; add only in D2 after address, firewall, and AAAA decision.

## Deploy / rollback

1. Build with `scripts/deploy/build-linux-release.sh` (`GOOS=linux` forced, `CGO_ENABLED=0`, `DEPLOY_GOARCH` amd64|arm64). Refuses dirty worktree; `SHA256SUMS` covers binaries, `lib/*`, `COMMIT`, `EXPECTED_SCHEMA_VERSION` (no release migrate runner).
2. Upload to remote temp → verify full manifest → immutable `releases/<sha>` (full manifest again on `install-release`/`activate`) → env backup then install `0600` (restorable immediately after backup).
3. `migration_before` / `up` / `migration_after` use **`fiscal-migrate` from the new release** via the closed helper (drop-priv), never `current`, never as root.
4. Dirty migration **blocks** promotion.
5. **Before** activation: env restore on failure; binary not switched.
6. **After** activate/restart/health failure: re-read `current`; N-1 rollback (symlink + envs + restart + health) **only** if policy allows (`DEPLOY_N1_COMPAT_PROVEN=1` when schema changed). Otherwise roll-forward/manual.
7. Health accepts only JSON `"status":"ok"` (exact field); does **not** replace `fiscal-migrate version`.
8. Config install: temp file `0600` → atomic install by root under `/etc/bwb-modulo-fiscal/`. Never copy env into release dirs, logs, or reports.
9. D2 bootstrap: install helper + libs + sudoers + create `bwb-fiscal-migrate` (see `deploy/sudoers/bwb-fiscal-deploy`).

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

## D2 status (2026-07-21)

Bootstrap e primeiro deploy em `sandbox.fiscalmod.bwb.pt` concluídos. Relatório: [d2-staging-bootstrap-report.md](d2-staging-bootstrap-report.md).

## S3A / S3B / S3C — credential_store staging (Nginx)

**S3A (este PR, só repositório):** artefacts de build/helper/`admin.env`/grants/Nginx fechado + candidato + medição + runbook. **Sem SSH, sem deploy, sem alteração DNS/Nginx remoto.**

**S3B (pós-merge S3A, no host):** deploy release S3A; migrate `2→3` se necessário; aplicar `deploy/postgres/grants-schema3-runtime-admin.sql`; auditoria de privilégios; provisionar scopes/credenciais via helper; E2E em `http://127.0.0.1:8080`; medição de `limit_req` **apenas** em `http://127.0.0.1:18080`; HTTPS público mantém `/v1/documents` **deny-all**.

**S3C-tooling (repo):** binário Go `fiscal-sandbox-measure` + `Health.revision` (SHA40). **Sem** abertura pública; deny-all intacto.

**S3C1 (ops, pós-deploy tooling):** medição em `:18080` com perfis fechados; 443 documents permanece deny-all; remover `:18080` no fim.

**S3C2 (PR repo, pós-S3C1):** valores finais `rate=10r/s` / `burst=20`; site open + deny-all versionados no release; helper `nginx-open-arm` / `nginx-open-confirm` / `nginx-deny-all` + timer systemd 5 min + `nginx-open-boot-recovery`; flock em `/var/lock/bwb-fiscal-nginx-open.lock`; `:18080` desactivado no arm. **Promoção no host só após merge.**

### S3C2 — abertura controlada (fail-safe)

Artefactos no release (`nginx/tls.open.conf`, `nginx/tls.deny.conf`, `nginx/limit-req-documents.conf`, units systemd rollback + boot-recovery):

| Op fechada | Comportamento |
|---|---|
| `nginx-open-arm <sha40>` | Instala open; exige timer `is-active`; falha → `deny_restored` ou `emergency_nginx_stop` (só se Nginx **comprovadamente** inactivo) ou `emergency_stop_failed` (CRITICAL); reload falhado usa o mesmo fail-closed; **nunca** `arm_ok` sem timer |
| `nginx-open-confirm <sha40>` | Grava `state=confirmed` **antes** de `stop`/`disable` do timer; falha de stop reporta erro mas mantém `confirmed` (fire = noop) |
| `nginx-deny-all <sha40>` | Instala deny-all, `nginx -t`, reload, **probe 403 com retry** (401 transitório até deadline curto); falha → mesmo fail-closed do arm (`deny_restored` / `emergency_nginx_stop` / `emergency_stop_failed`); cancela timer |
| `nginx-open-rollback-fire` | Alvo do timer: se `armed`, deny-all + `state=rolled_back` se 403; senão estado terminal `denied`/`emergency_stopped`/`emergency_stop_failed` — **nunca** `armed` com timer inactive; se `confirmed`, noop |
| `nginx-open-boot-recovery` | Unit `Before=nginx.service`: se `armed`, deny on disk + `nginx -t`; `confirmed` noop. Drop-in `nginx.service.d/*`: `Requires=`/`After=` recovery — falha armed **impede** `nginx.service` |

Todas as ops `nginx-open-*` / `nginx-deny-all` tomam **flock exclusivo** em path fixo root-owned. Sem paths/URLs/comandos arbitrários do operador. Updater **não** activa open. Legacy `install-nginx-open` continua rejeitado.

Open/deny: `location = /v1/documents`; HSTS `max-age=31536000` sem `includeSubDomains`; ACME em `^~ /.well-known/acme-challenge/`; redirect HTTPS só em `location /`. Open: `limit_req` burst=20 + `limit_req_status 429`; health fora do limiter; `X-Request-Id` inbound limpo.

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

### S3C2 — promoção sandbox (resultado)

Relatório: `docs/07-operations/s3c2-sandbox-promotion-report.md`. Resultado: **ROLLED_BACK** (não confirmada). Re-tentar timer real no Ubuntu só após merge deste fail-closed; **sem** `confirm` até o timer comprovar 403.

### S3C1 — matriz e thresholds (aprovados)

Base fixa: `http://127.0.0.1:18080`. Helper: `admin-sandbox-measure <sha40> sustained|burst|replay` (operador **não** controla URL/taxa/concorrência/token path).

| Perfil | Pedidos | Pacing | Concorrência | Duração wall | Persistência máx. |
|---|---:|---|---:|---|---:|
| `sustained` | 300 | 10 r/s média (±10% request throughput); agenda monotónica sem catch-up | 1 | 28–33 s | ≤300 docs |
| `burst` | ≤60 | rajada | ≤5 | curta | ≤60 docs |
| `replay` | 2 | sequencial; mesma key/body | 1 | N/A | 1 doc |

Thresholds: `5xx=0`; `409=0`; `other=0`; sustained p95≤250 ms / p99≤500 ms **só em respostas 201** (nearest-rank); 429 latência reportada em separado; request throughput ∈ [9,11] r/s **apenas sobre `http_responses`** (não conta `transport_errors`); 429≤3/300 em sustained; burst 201 ∈ [20,25] e restantes 429; replay ambos 201 com payload tipado estável idêntico. Sem `Retry-After` (Nginx não envia por omissão).

Contadores do relatório JSON: `attempted` (todas as tentativas), `http_responses` (com status HTTP), `transport_errors` (sem resposta HTTP); `passed` + `failure_codes` sanitizados. Falha de threshold ou transporte emite JSON completo e exit ≠ 0 (não esconde resultados atrás de apenas `measure_failed`). Em falha de transporte o perfil cancela o restante de forma controlada e preserva métricas já recolhidas. Percentis (`p50`/`p95`/`p99`) usam nearest-rank sobre latências µs→ms das classes 201 e 429.

Token: só `/var/lib/bwb-fiscal-admin/tokens/measure.token` — dir real (não symlink), mode sem grupo/outros; ficheiro regular (não symlink), owner euid, `0600`. Output: JSON agregado sem token/NIF/body/DSN/URL/Authorization.

Health público: `version` (rótulo `FISCAL_APP_VERSION`) + `revision` (SHA40 lowercase do artefacto em release; `dev` **apenas** com `FISCAL_ENV=development`). Gate release: `health.revision == COMMIT` (SHA40 exacto, sem normalização de case).

### Topologia

| Superfície | Path/porto | Estado em S3A | Estado em S3B | Estado pós-S3C |
|---|---|---|---|---|
| Público TLS | `443` `/v1/documents` | deny-all (instalável) | deny-all | aberto + `limit_req` final |
| Candidato aberto | `deploy/nginx/candidates/*.open.candidate.conf` | versionado; **não activável** pelo helper/updater | não activar | fundido na canónica |
| Medição | `127.0.0.1:18080` | ficheiro versionado | activo só loopback | desactivado + verificado |
| API directa | `127.0.0.1:8080` | N/A | gates E2E (sem medir 429 aqui) | opcional debug |

Zone provisória (S3A/S3B): `deploy/nginx/http.d/bwb-limit-req-documents-provisional.conf` — `10r/s`, `burst=20` (idêntica no candidato e na medição). Health fora do `limit_req`; `X-Request-Id` inbound limpo (`proxy_set_header X-Request-Id ""`).

### admin.env e custódia

| Path | Owner | Mode | Quem lê |
|---|---|---|---|
| `/etc/bwb-modulo-fiscal/admin.env` | `root:root` | `0600` | helper (root) apenas |
| `/var/lib/bwb-fiscal-admin/tokens/` | `bwb-fiscal-admin` | `0700` | só `bwb-fiscal-admin` |
| ficheiro token | `bwb-fiscal-admin` | `0600` | E2E/admin CLI |

Fluxo: helper root → parser allowlist (`FISCAL_DATABASE_DRIVER`, `FISCAL_DATABASE_URL`) → `env -i` → drop para `bwb-fiscal-admin`. DSN/token **nunca** em argv/stdout/logs. `bwb-deploy` não lê `admin.env` nem tokens. `--output-file` é sempre escolhido pelo helper sob o dir de tokens.

Ops allowlisted: `admin-scope-create`, `admin-credential-issue|rotate|revoke`, `admin-sandbox-e2e`, `admin-sandbox-measure <sha> sustained|burst|replay`, `admin-sandbox-ab-revoke-gate`, `nginx-open-arm|nginx-open-confirm|nginx-deny-all|nginx-open-boot-recovery` (só pós-merge S3C2). Rejeitadas: `install-nginx-open` / `activate-open-candidate`.

### Grants PostgreSQL (S3A artefact / S3B apply)

- Script: `deploy/postgres/grants-schema3-runtime-admin.sql` — grants explícitos por objeto; **não** cria roles.
- Roles `fiscal_migrate` / `fiscal_runtime` / `fiscal_admin` e respetivas credenciais LOGIN são criadas no **bootstrap S3B** com autoridade apropriada; o script de grants falha se alguma role obrigatória estiver ausente.
- Sem `CREATE ROLE` e sem DEFAULT PRIVILEGES genéricos para runtime/admin.

### Teto de medição (S3B histórico / S3C1 actual)

- **S3B (legado):** ≤60 req / ≤5 conc / ≤60 s (burst curto; não decide rate final).
- **S3C1:** ver matriz acima (`fiscal-sandbox-measure` Go); base `http://127.0.0.1:18080` apenas.

### Sequência S3B (operador)

1. SSH multiplexado (`ControlMaster`) com key/known_hosts do `.env.local` — fingerprint do painel do provider.
2. Backup PG + restore de validação (fora deste runbook detalhado).
3. `update-staging.sh` com `.env.deploy.local` / `.env.migrate.local` / `.env.admin.local` (`chmod 600`).
4. Confirmar `version=3 dirty=false`; aplicar grants SQL como owner; testes negativos de privilégio.
5. Bootstrap OS se em falta: user `bwb-fiscal-admin`, dirs tokens, `admin.env.allowlist` em `/usr/local/lib/bwb-fiscal-deploy/`, sudoers só para o helper.
6. Activar conf de medição loopback + zone `http.d` (não o candidato aberto); `nginx -t` && reload.
7. Helper: scope ops + scope carga; issue credenciais; E2E casos allowlisted em `:8080`.
8. Copiar token de carga para `measure.token` (helper/path allowlisted); `admin-sandbox-measure`.
9. Registar evidência rate/burst/`Retry-After` (sem segredos). Revogar credencial de carga.
10. Prova A→B: revogar A; E2E com B; A deve falhar auth.
11. Prova externa: `:18080` inacessível de fora (UFW); `:443` documents ainda 403 deny-all.

### Rollback deny-all (se abertura falhar em S3C)

1. Restaurar `deploy/nginx/bwb-fiscal-sandbox-tls.conf` (documents `deny all`) como site activo.
2. `nginx -t` && reload.
3. Desactivar/remover conf `:18080`.
4. Verificar `/v1/health` OK e `/v1/documents` 403.

### SSH multiplexado

Reutilizar o mux do updater (`ControlMaster` + `ControlPath` + `ControlPersist`). Não abrir tempestade TCP. `deploy_ssh_mux_stop` no EXIT. Proibido `StrictHostKeyChecking=no`.

### Critérios para abrir PR S3C

- Evidência S3B de 429 sob tetos; valores finais de `rate`/`burst` documentados.
- HTTPS público ainda deny-all até merge+apply S3C.
- Nenhum token/DSN/NIF completo em logs ou PRs.

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
