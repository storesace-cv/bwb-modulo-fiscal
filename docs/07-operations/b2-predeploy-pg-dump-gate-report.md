# Relatório B2/B3 — gate obrigatório `pg_dump` pré-deploy (INC-S4-003)

**Data (UTC):** 2026-07-25T00:42Z–00:45Z (B2 ops); documentação B3 nesta branch  
**Host:** `sandbox.fiscalmod.bwb.pt` / `194.9.62.239`  
**Fingerprint Ed25519 (D2):** `SHA256:I5NU5TgFEAggzCb6K0iHF3F+mXGNuLrzdbTTJgiipag`  
**Repo base:** `main` = `origin/main` = `e39638fa1595e228c65918d7ee9c4e46ac748daa`  
**Resultado B2:** **APROVADO** (gate real + falha induzida fail-closed)  
**INC-S4-003:** **RESOLVIDO** (fundamentação abaixo)

Este relatório **não** contém passwords, tokens, DSN, PGPASSFILE, conteúdo de dumps, NIF, IDs fiscais, corpos HTTP nem URLs de ligação.

**Âmbito B3:** só documentação a partir de evidências sanitizadas já recolhidas em B2. **Sem** novo SSH, deploy, backup ou reexecução de testes.

---

## Estado inicial (pré-B2)

| Campo | Valor |
|---|---|
| Release activa (`current`) | `5d7c14b01af1f4855a41ee2b4f251af96dd5b726` |
| `health.revision` (HTTPS + loopback) | `5d7c14b01af1f4855a41ee2b4f251af96dd5b726` |
| Schema | `version=3`, `dirty=false`, exactamente **1** linha em `public.bwb_schema_migrations` |
| Serviços | `bwb-fiscal-api`, `nginx`, `postgresql` = active |
| Nginx open | `state=confirmed` (sha de open state inalterado nesta sessão) |
| Timer rollback open | `inactive` / `disabled` |
| HTTPS `/v1/health` | 200 |
| POST `/v1/documents` sem token | 401 |
| Portas externas 5432 / 8080 / 18080 | closed/filtered |
| Helper B1 (`parse_migrate_dsn.py`, `predeploy_pg.sh`, ops lock/gate) | **ausente** no host |
| Dumps em `pre-deploy/` | 0 |

---

## Instalação controlada do helper B1

- Fora do updater protegido (instalação root directa, conforme [staging-runbook.md](staging-runbook.md)).
- Backup pré-install: directório `b2-helper-install-20260725T004246Z` sob `/var/backups/bwb-fiscal/` — modo directório **0700** (verificado); artefactos anteriores do helper/libs/sudoers preservados a **0600**.
- Checksums locais (`e39638fa…`) = artefactos instalados; `visudo -cf` OK; ownership helper `root:root` `0755`; libs `root:root` `0644`; sudoers `0440`.
- Ops confirmadas: `deploy-lock-acquire` / `pre-deploy-pg-backup`.

---

## Privilégios PostgreSQL

| Role | SUPERUSER | CREATEDB |
|---|---|---|
| `fiscal_migrate` | false | false |
| `fiscal_runtime` | false | false |
| `fiscal_admin` | false | false |

Prova adicional: `createdb`/`dropdb` smoke como OS `postgres` (socket `/var/run/postgresql`, port `5432`, `-O fiscal_migrate`, `template0`) OK; BD smoke removida. Roles runtime/admin **sem** privilégios novos.

---

## Deploy live + gate (release `e39638fa…`)

### Ordem observada (updater)

1. `deploy_lock=acquired` (`backup_id` abaixo)  
2. Upload + `install-release` **lateral** (`releases/e39638fa…`; `current` ainda N-1)  
3. `pre_deploy_pg_backup=ok` (`deploy_allowed=true` só aqui)  
4. `backup-envs` → `install-env` → migrate `3→3` dirty=false → activate → restart → health  
5. `promote=ok` / `done`  
6. `lock_release=ok`

### Identificadores e dump (sanitizado)

| Campo | Valor |
|---|---|
| `backup_id` (promote) | `20260725T004315Z-e39638fa1595e228c65918d7ee9c4e46ac748daa` |
| Directório `pre-deploy/` | `root:root` **0700** (verificado) |
| Ficheiro dump | `root:root` **0600**, **185123** bytes |
| `pg_restore --list` | OK (68 linhas TOC; sem conteúdo listado neste relatório) |
| Temp DB | criada, schema validado (= origem `3` / `dirty=false`), **eliminada** |
| Workdir efémero | removido |
| Lock após promote | ausente |

### Release / health pós-promote

| Campo | Valor |
|---|---|
| Release anterior | `5d7c14b01af1f4855a41ee2b4f251af96dd5b726` |
| Release final | `e39638fa1595e228c65918d7ee9c4e46ac748daa` |
| `current-sha` = `health.revision` | `e39638fa1595e228c65918d7ee9c4e46ac748daa` |
| Schema | `3` / `dirty=false` / 1 linha |
| Serviços | active / active / active |
| Nginx | `state=confirmed`; timer inactive/disabled |
| HTTPS health / POST sem token | 200 / 401 |
| Portas 5432 / 8080 / 18080 | closed/filtered |

---

## Falha induzida (pós-instalação durável)

| Campo | Valor |
|---|---|
| Mecanismo | Pré-criação da BD temp canónica (nome derivado do `backup_id`) para forçar falha de `createdb` **após** dump durável; reversível; **sem** parar PostgreSQL; **sem** corromper schema/dados |
| `backup_id` | `20260725T004408Z-e39638fa1595e228c65918d7ee9c4e46ac748daa` |
| Resultado gate | `backup_created=true`, `restore_verified=false`, `deploy_allowed=false`, `error=temp_db_create_failed`, `lock_state=held` |
| Dump falha | `root:root` **0600**, **185178** bytes — **preservado** |
| Dump promote (`…004315Z…`) | **não sobrescrito** |
| Mutações activas | **nenhuma** (`current`, mtimes de envs, schema, API, Nginx idênticos antes/depois) |
| Lock | libertado por `deploy-lock-release` (não poisoned); BD temp pré-criada removida |

---

## Lock poisoned

**Não ensaiado em B2.** O runbook não define procedimento seguro/reversível de teste nem passos de remediação humana. Não foi improvisado. Ver **INC-B2-001** (não reabre INC-S4-003).

---

## Backups preservados (evidência)

| Artefacto | Modo | Notas |
|---|---|---|
| Directório `/var/backups/bwb-fiscal/pre-deploy/` | **0700** | verificado |
| Dump promote `…004315Z-e39638fa….dump` | **0600** | gate live |
| Dump falha induzida `…004408Z-e39638fa….dump` | **0600** | fail-closed |
| Directório `b2-helper-install-20260725T004246Z` | **0700** | pré-install helper |
| Env backup updater (`backup_id` = `…004315Z…`) | ficheiros env tipicamente **0600** | sob `/etc/bwb-modulo-fiscal/backups/` (conteúdo não listado) |

---

## Segredos

- Outputs do updater e do gate filtrados; scan negativo a padrões `postgres://user:pass@`, `PGPASSWORD=`, chaves privadas.
- Relatório sem DSN, passwords, PGPASSFILE, TOC/conteúdo de dump ou dados fiscais.

---

## INC-S4-003 — fecho

| Campo | Valor |
|---|---|
| Severidade original | Média (rastreabilidade / recuperação pontual) |
| Estado | **RESOLVIDO** |
| Fundamentação | (1) B1 mergeado em `main` @ `e39638fa…` (PR #23); (2) helper/libs do gate instalados no sandbox; (3) dump pré-deploy real criado no promote; (4) `pg_restore --list` + restore temp + schema 1 linha / dirty=false validados; (5) falha induzida com `deploy_allowed=false` e sem mutação activa; (6) deploy saudável (health/revision/serviços/Nginx/portas) |
| Risco residual | Retenção/rotação manual de dumps; remediação de lock poisoned/stale/corrupt ainda não ensaiada (INC-B2-001) |

---

## INC-B2-001 — prova poisoned não executada

| Campo | Valor |
|---|---|
| Severidade | Baixa / processual |
| Causa | Ausência de procedimento seguro, reversível e explícito no runbook para induzir/remediar lock `poisoned` |
| Impacto | Cobertura B2 incompleta no eixo poisoned; **não** afecta a validade do gate no caminho feliz nem a falha induzida pós-dump |
| Resolução | Skip documentado; runbook actualizado para exigir remediação humana e proibir auto-remoção |
| Estado | **Aberto** |
| Nota | **Não** reabre INC-S4-003 |

---

## Confirmações de processo

- B3: documentação apenas; **sem** novo acesso SSH/deploy/backup/testes.
- Pasta `local/` não utilizada como fonte nem incluída em artefactos versionados.
