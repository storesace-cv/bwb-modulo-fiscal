# Relatório D2 — bootstrap staging `sandbox.fiscalmod.bwb.pt`

**Data (UTC):** 2026-07-21
**SHA ativo:** `be26a3fdcff03636de8ce2f5ace2a994f4a9913e`
**Schema:** `2` (`dirty=false`)
**Host:** `modulo-fiscal` / `194.9.62.239`
**SO:** Ubuntu 22.04.5 LTS · kernel 5.15 · arch amd64

Este relatório **não** contém passwords, tokens, chaves privadas nem DSN.

## Checks executados (sem segredos)

- DNS A `sandbox.fiscalmod.bwb.pt` → `194.9.62.239` (1.1.1.1 / 8.8.8.8); sem AAAA.
- Host key Ed25519 validada: `SHA256:I5NU5TgFEAggzCb6K0iHF3F+mXGNuLrzdbTTJgiipag`.
- SSH: `PermitRootLogin no`, `PasswordAuthentication no`, `AuthenticationMethods publickey`, `AllowUsers ubuntu bwb-deploy`.
- UFW ativo: 22 LIMIT; 80/443 ALLOW; 5432/8080 não expostos.
- Fail2ban jail `sshd` ativo; `ignoreip` apenas loopback (sem IP público dinâmico).
- Helper `/usr/local/sbin/bwb-fiscal-deploy-helper` + libs; sudoers só helper para `bwb-deploy`.
- PostgreSQL **16.14**; `listen_addresses=localhost`; `ssl=on`; `sslmode=require` validado.
- Roles `fiscal_migrate` (owner DB) e `fiscal_runtime` (sem DDL); DEFAULT PRIVILEGES pós-migrate.
- Nginx HTTP→HTTPS; Let’s Encrypt emitido; renew `--dry-run` OK; HSTS sem `includeSubDomains`.
- API `bwb-fiscal-api` active; bind `127.0.0.1:8080`.
- Health interno e HTTPS externo: `{"status":"ok",...}`.
- `/v1/documents` → HTTP 403 (deny all).
- `pg_dump` + restore em DB temporária → schema 2; dumps de evidência mantidos em `/var/backups/bwb-fiscal/` (`0600`).

## Fingerprints (apenas)

| Item | Fingerprint |
|---|---|
| Host Ed25519 | `SHA256:I5NU5TgFEAggzCb6K0iHF3F+mXGNuLrzdbTTJgiipag` |
| Chave operador (`~/.ssh/digitalocean`) | `SHA256:FpiaaXl5K3m1Vr0UYbp+aoLKJkLcTyE3LAwvTHMrDqY` |

## Versões

| Componente | Versão |
|---|---|
| Ubuntu | 22.04.5 LTS |
| PostgreSQL | 16.14 (pgdg) |
| Nginx | 1.18.0 (Ubuntu) |
| Certbot | 1.21.0 |
| Release app | commit `be26a3fd…` · `FISCAL_APP_VERSION=0.2.5-staging` |

## Estado NTP / SSH / FW / PG / systemd / Nginx / TLS

| Área | Estado |
|---|---|
| NTP | sincronizado · TZ `Etc/UTC` |
| SSH | activo · root/password bloqueados · 2 sessões OK |
| Firewall | UFW activo · 22 LIMIT (6 NEW/30s → REJECT) |
| PostgreSQL | activo · loopback+TLS |
| systemd API | `bwb-fiscal-api` enabled+active |
| Nginx | activo · 80/443 |
| TLS | LE live · HSTS on · renew dry-run OK |

## Portas externas

| Porta | Resultado |
|---|---|
| 22 | OPEN (exceto sob UFW LIMIT) |
| 80 | OPEN |
| 443 | OPEN |
| 5432 | filtrada/timeout |
| 8080 | filtrada/timeout |

## Provas de privilégio (sem DSN)

- Migration: `version=0→2`, `dirty=false`.
- Runtime: acesso a objectos `fiscal.*` após GRANTs.
- Runtime: `CREATE TABLE` → permission denied.
- Systemd: apenas `fiscal.env` (sem migrate.env).

## Backup / restore

- Dumps: `/var/backups/bwb-fiscal/fiscal-*.dump` mode `0600`.
- Restore test: DB `fiscal_restore_test` criada, restaurada (schema 2), removida.
- Evidência **não** apagada (retenção a definir).

## Auditoria SSH (incidente “TCP/22 refused sob carga”)

### Erro exacto observado pelo Cursor

`ssh: connect to host 194.9.62.239 port 22: Connection refused` (TCP/22 REJECT), sob rajadas de novas ligações SSH/SCP do agente/updater. O terminal do operador (poucas ligações) continuava a funcionar fora da janela de rate-limit.

### Sintoma vs causa

| | Descrição |
|---|---|
| **Sintoma** | `Connection refused` em TCP/22; por vezes `kex_exchange_identification: Connection closed by remote host` em probes curtos |
| **Causa comprovada** | UFW `LIMIT` na porta 22: `recent --seconds 30 --hitcount 6` → chain `ufw-user-limit` → `REJECT`. Cada `ssh`/`scp` do updater abria uma **nova** TCP (sem ControlMaster). ≥6 NEW/30s do mesmo IP ⇒ refused **antes** do sshd |
| **Não causa (sem evidência)** | Fail2ban ban (`Total banned: 0` no boot actual; journal fail2ban só arranques de serviço); MaxStartups drop (sem linhas MaxStartups no journal sshd); Permission denied; Too many authentication failures |

### Evidência (UTC 2026-07-21)

| Fonte | Achado |
|---|---|
| Reprodução | burst TCP 8× sem intervalo: connects 1–3 OK, 4–8 `Connection refused` (`errno 61`) @ `17:40:14Z` |
| UFW rules | `-m recent --update --seconds 30 --hitcount 6 -j ufw-user-limit` → `REJECT` |
| Fail2ban | `Currently banned: 0`, `Total banned: 0`, `Total failed: 0` após boot `17:32` |
| sshd | Pré-reboot `16:50–17:32`: 36 `Connection from`, 25 Accepted, 0 MaxStartups; último Accepted `17:26:31`; reboot `17:32` |
| Reboot | `last`: boot `17:32` (e boots anteriores `16:20`/`16:21`/`16:38`) — **recuperação, não correção da causa** |
| Scripts (pré-fix) | ~16 novas TCP no happy-path live; sem ControlMaster/ControlPersist |
| Mux pós-fix | 16 invokes `bwb-deploy` @ `17:46:36–17:46:38` → **1** `Connection from` + **1** Accepted; fail2ban 0 bans; `NRestarts=0` |
| Mux 20 | 20 invokes @ `17:45:10–17:45:13` → `ok=20 fail=0` |
| Sequencial sem mux | 20× com gap 1s → `ok=5 fail=15` (confirma LIMIT, não “SSH em baixo”) |

### Correções aplicadas (após evidência)

**Cliente (causa real):**

- `scripts/deploy/lib/allowlist.sh`: `ControlMaster=auto`, `ControlPersist=120`, `ControlPath` curto (`/tmp/bwb-ssh-$UID/%C`), `IdentitiesOnly`, retries limitados com backoff em `deploy_ssh_run`/`deploy_scp_run`, contagem de invocações via `DEPLOY_SSH_INVOKE_LOG`.
- `update-staging.sh` / `migrate-remote.sh` / `healthcheck.sh`: usam mux + cleanup `-O exit`.
- `tests/deploy/run-tests.sh`: asserts de mux e de que o path live excederia UFW LIMIT sem mux.

**Servidor (limpeza de mitigações injustificadas — não escondem o defeito do cliente):**

- Removido `ignoreip` com IP público dinâmico `109.50.171.37` (apenas `127.0.0.1/8 ::1`).
- Removido drop-in `MaxStartups 30:50:100` sem métricas → default efectivo `10:30:100`.
- `sshd -t` + `systemctl reload ssh` (sem reboot); root/password continuam off; UFW LIMIT **mantido**.

### Estado do incidente

**Mitigado (causa comprovada; correção no cliente).** Não marcado “Resolvido” apenas por reboot/limites. Fail2ban **não** foi a causa demonstrável neste incidente.

### Riscos residuais

- Qualquer cliente que abra ≥6 **novas** TCP/22 em 30s (agente IDE, loops, scans) volta a ver `Connection refused` — comportamento intencional do UFW LIMIT.
- `MaxSessions 2` limita multiplex concorrente; o updater é sequencial (OK).
- Agent SSH com várias chaves sem `IdentitiesOnly` gera `Failed publickey` extra (visível nos logs); o deploy path força `IdentitiesOnly=yes`.
- Dry-run do updater na branch de docs exige `EXPECTED_COMMIT=HEAD` e worktree limpa ou `DEPLOY_TEST_OUT_ROOT` (não é falha SSH).

## Incidentes

| Severidade | Fase | Causa | Impacto | Resolução | Estado | Risco residual |
|---|---|---|---|---|---|---|
| Alta | D2 remoto | UFW LIMIT (6 NEW TCP/22 / 30s → REJECT) + tempestade de ligações novas do updater/agente sem ControlMaster | `Connection refused`; bootstrap/deploy interrompidos | ControlMaster/ControlPersist no deploy; retries backoff; remoção de `ignoreip` dinâmico e MaxStartups sem métrica; UFW LIMIT mantido | Mitigado | Clientes sem mux ainda disparam LIMIT |
| Média | D2 TLS | Renew dry-run ACME falhou porque redirect HTTP→HTTPS interceptava challenge | Bloqueava HSTS | Redirect envolvido em `location /`; challenge em `^~ /.well-known` | Resolvido | Overlay Nginx no host diverge ligeiramente do template versionado (challenge) |
| Info | D2 acesso | Chave dedicada `bwb_fiscal_staging_ed25519` criada localmente mas acesso operacional usa `~/.ssh/digitalocean` | Confusão potencial de chaves | Documentado: chave operativa = digitalocean | Aceite | Manter inventário de chaves |

## Critério de fecho D2

Cumprido: health HTTPS `status=ok` **após** GRANTs; API loopback; documents deny-all; helper-only para deploy; migrate drop-priv; PG16+TLS require; NTP OK; backup restauro validado.

## Pendências operacionais (não bloqueiam fecho técnico)

- Política de retenção dos dumps em `/var/backups/bwb-fiscal/`.
- Alinhar template versionado Nginx (ACME vs redirect) num PR futuro.
