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
- Fail2ban jail `sshd` ativo; `ignoreip` inclui IP do operador.
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
| Firewall | UFW activo |
| PostgreSQL | activo · loopback+TLS |
| systemd API | `bwb-fiscal-api` enabled+active |
| Nginx | activo · 80/443 |
| TLS | LE live · HSTS on · renew dry-run OK |

## Portas externas

| Porta | Resultado |
|---|---|
| 22 | OPEN |
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

## Incidentes

| Severidade | Fase | Causa | Impacto | Resolução | Estado | Risco residual |
|---|---|---|---|---|---|---|
| Alta | D2 remoto | TCP/22 `Connection refused` recorrente sob carga SSH (Fail2ban/sshd); VM chegou a ficar inacessível | Interrompeu bootstrap/deploy | Recuperação consola Euronodes; reboot VM 53143; `MaxStartups` ↑; `ignoreip` operador; AllowUsers `bwb-deploy` | Resolvido | Reincidência possível sob tempestade de ligações; preferir ControlMaster / ritmo no updater |
| Média | D2 TLS | Renew dry-run ACME falhou porque redirect HTTP→HTTPS interceptava challenge | Bloqueava HSTS | Redirect envolvido em `location /`; challenge em `^~ /.well-known` | Resolvido | Overlay Nginx no host diverge ligeiramente do template versionado (challenge) |
| Info | D2 acesso | Chave dedicada `bwb_fiscal_staging_ed25519` criada localmente mas acesso operacional usa `~/.ssh/digitalocean` | Confusão potencial de chaves | Documentado: chave operativa = digitalocean | Aceite | Manter inventário de chaves |

## Critério de fecho D2

Cumprido: health HTTPS `status=ok` **após** GRANTs; API loopback; documents deny-all; helper-only para deploy; migrate drop-priv; PG16+TLS require; NTP OK; backup restauro validado.

## Pendências operacionais (não bloqueiam fecho técnico)

- Política de retenção dos dumps em `/var/backups/bwb-fiscal/`.
- Alinhar template versionado Nginx (ACME vs redirect) num PR futuro.
- Considerar ControlMaster no updater live para reduzir storms SSH.
