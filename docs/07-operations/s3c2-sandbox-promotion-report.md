# Relatório S3C2 — promoção controlada sandbox (Ubuntu)

**Data (UTC):** 2026-07-23 (preflight ~11:51Z → cleanup ~12:03Z)
**Squash / release alvo:** `11c58841dea76de4c252457151ff4fd2d0ae741d`
**Host:** `sandbox.fiscalmod.bwb.pt` / `194.9.62.239`
**Resultado da promoção:** **ROLLED_BACK** (não CONFIRMADA)

Este relatório **não** contém passwords, tokens, DSN, hashes, NIF nem corpos de pedido/resposta.

## Veredicto

A promoção pública **não** foi confirmada. O gate do timer real falhou: o serviço de rollback disparou, mas o probe HTTP pós-deny devolveu `401` em vez de `403`, o unit falhou e o `state` ficou `armed` até remediação manual com `nginx-deny-all`. Protocolo de falha aplicado: deny-all comprovado (`403`), **sem** `nginx-open-confirm`.

O teste de **boot recovery** passou. Cleanup (revogação + remoção de tokens S3C2) concluído. Postura final do host: fechada.

## Âmbito executado

| Passo | Resultado |
|---|---|
| 1. Preflight fechado + backups | OK |
| 2. Deploy fechado `11c5884…` (deny-all mantido) | OK |
| 3. Teste real do timer Ubuntu | **FALHOU** (incidente I1) |
| 4. Teste real de boot recovery | OK |
| 5. Promoção final + `nginx-open-confirm` | **NÃO EXECUTADO** (bloqueado por I1) |
| 6. Cleanup (revoke + scrub tokens) | OK |

## SHA / revision / schema / serviços (final)

| Item | Valor |
|---|---|
| `origin/main` / `HEAD` local no início | `11c58841dea76de4c252457151ff4fd2d0ae741d` |
| Release pré-deploy | `983a8013…` |
| `current-sha` final | `11c58841dea76de4c252457151ff4fd2d0ae741d` |
| `health.revision` (`/v1/health`) | `11c58841dea76de4c252457151ff4fd2d0ae741d` |
| Schema (preflight) | `3` / `dirty=false` |
| `bwb-fiscal-api` | active |
| Nginx | active |
| PostgreSQL | active |
| `nginx-open.state` | `state=boot_recovered` |
| Timer rollback | inactive |
| Auth runtime | `FISCAL_ENV=homologation` + credential store |

## Portas e superfície pública (final)

| Superfície | Estado |
|---|---|
| HTTPS `/v1/documents` (local + externo) | **403** deny-all |
| HTTPS `/v1/health` | 200 + revision correcta |
| `127.0.0.1:8080` | API loopback |
| `127.0.0.1:5432` | PG loopback |
| `127.0.0.1:18080` | **ABSENT** |
| Bind público 5432 / 8080 / 18080 | ausente (`closed_externally_ok`) |

## Backups preflight (sem conteúdo)

| Artefacto | Notas |
|---|---|
| Bundle root | `/root/bwb-s3c2-preflight-20260723T115143Z` |
| Nginx site + helper + sudoers + units | incluídos no bundle |
| Env files | mode `0600` |
| Dump PostgreSQL | `fiscal.pg.dump` validado com `pg_restore -l` |

## IDs sintéticos (permitidos)

| Tipo | ID |
|---|---|
| Scope timer | `s3c2-timer-20260723T115359Z` |
| Credencial (revogada) | `0300dc885e7e941c09534735cc659cc3` |
| Motivo de revogação | `s3c2_promotion_cleanup` |
| NIF / token / DSN | **omitidos** |

Documentos e auditoria sintéticos: **preservados** (sem delete).

## 1–2. Preflight e deploy fechado

- Main alinhada com `11c5884…`.
- Pré: release `983a8013…`, schema 3/dirty=false, serviços active, `/v1/documents=403`, `:18080` ausente, disco/NTP OK.
- Deploy via `update-staging.sh` + sync helper/sudoers/units/drop-in Nginx.
- `visudo -cf`, `systemd-analyze verify`, `daemon-reload`, `nginx -t` OK.
- Pós-deploy fechado: `current-sha=COMMIT=health.revision=11c5884…`, deny-all, público 403.

## 3. Teste real do timer (incidente)

### O que correu

1. `nginx-open-arm` → `state=armed`, timer **active**, site open canónico (exact `location = /v1/documents`, HSTS, ACME, `rate=10r/s`, `burst=20`), `:18080` ausente.
2. Gates pré-expiração (sem confirm): health 200+revision; sem token=401; token inválido=401; create=201; replay=201 idêntico (após correcção: Idempotency-Key UUID).
3. Timer real disparou ~`2026-07-23T11:59:02Z`.

### Falha

```
bwb-fiscal-nginx-open-rollback.service → failed
error: documents probe expected 403 got 401
```

- Após o fire: timer `inactive`, `state` ainda `armed`, unit `failed`.
- Causa provável: race pós-`nginx reload` — workers antigos ainda serviam o site open (API → 401 sem token) enquanto o ficheiro deny já estava instalado; `op_nginx_deny_all` usa `nginx_verify_documents_403` que faz `die` **sem** caminho fail-closed/`emergency_stop` (ao contrário do arm).
- Remediação imediata: `nginx-deny-all <sha>` → `state=denied`, `documents=403`, site deny-all, serviços active.

### Implicação

Gate de rollback automático **não** comprovou sucesso end-to-end. Promoção final e `nginx-open-confirm` **bloqueados** pelo protocolo de falha.

## 4. Teste real de boot recovery — OK

1. Re-armar (`state=armed`, timer active).
2. Reboot controlado (`boot_id` novo; uptime 0 min @ 12:02Z).
3. SSH restabelecido com backoff + multiplexing.
4. Evidência de ordem:
   - recovery exit `12:02:04Z`
   - nginx ActiveEnter `12:02:05Z`
   - unit `Before=nginx.service`; nginx `Requires=`/`After=` boot-recovery
5. Resultado: `nginx_open_boot_recovery=ok … action=deny_config_pre_nginx`, `state=boot_recovered`, site deny-all, `/v1/documents=403`, timer inactive, `:18080` ausente.
6. Após arranque: API/PG ficaram active; `/v1/health=200` + revision correcta.

## 5. Promoção final — não executada

Não houve segundo arm para gates completos (NIF mismatch, rate-limit 429 externo, confirm). Motivo: falha do gate do timer (I1). Host permanece fechado.

## 6. Cleanup

| Acção | Resultado |
|---|---|
| `admin-credential-revoke` | `status=revoked` |
| Prova API loopback pós-revoke | **401** |
| Remoção `current.token` + token S3C2 + meta `/root/bwb-s3c2-cred-timer.txt` | OK (`*s3c2*` = 0) |
| Tokens S3B/S3C1 anteriores | preservados (fora de âmbito) |
| Docs/auditoria sintéticos | preservados |
| Outputs deste relatório | sem token/DSN/NIF/hash/payload |

## Incidentes reais

### I1 — CRITICAL/ops: rollback timer falhou o probe (401 vs 403)

| Campo | Valor |
|---|---|
| Quando | 2026-07-23T11:59:03Z |
| Unit | `bwb-fiscal-nginx-open-rollback.service` |
| Sintoma | `documents probe expected 403 got 401`; state ficou `armed` |
| Exposição | Janela com state inconsistente; deny no disco pode já ter sido aplicado, mas o fail-safe não fechou o ciclo nem fez emergency stop |
| Mitigação imediata | `nginx-deny-all` manual → 403 + `state=denied` |
| Follow-up recomendado | Retry/backoff no probe pós-reload; em falha de `op_nginx_deny_all` durante `rollback-fire`, invocar o mesmo fail-closed que o arm (`deny_restored` / `emergency_nginx_stop`) |

### I2 — menor: path de health

Deny/open canónicos expõem `location = /v1/health` (não `/health`). Probes com `/health` devolvem 404 sob deny-all.

### I3 — operacional: reboot SSH

O primeiro `systemctl reboot` após o arm foi cortado (SIGPIPE/SSH); reboot efectivamente emitido na reconexão. Boot recovery validado no ciclo seguinte.

## Métricas sanitizadas (timer pré-falha)

| Gate | Resultado |
|---|---|
| health | 200 + revision `11c5884…` |
| sem token | 401 |
| token inválido | 401 |
| create (token efémero) | 201 |
| replay | 201, payload estável / idêntico |
| confirm | **não** executado |
| rate-limit 429 / NIF 403 / TLS externo completo pós-confirm | **não** executados (promoção abortada) |

## Estado final do host (após promoção inicial 2026-07-23 ~12:03Z)

- **Promoção:** `ROLLED_BACK` (não `CONFIRMED`)
- **Nginx open state:** `boot_recovered`
- **Público `/v1/documents`:** 403
- **Release activa:** `11c58841dea76de4c252457151ff4fd2d0ae741d`
- **Serviços:** nginx / postgresql / bwb-fiscal-api active
- **Timer:** inactive; sem exposição residual open

## Reteste timer pós-fix (2026-07-23)

**Resultado:** **TIMER_ROLLBACK_APPROVED**
**Release:** `10141af3cdd9cda16cfef46bbe5a4f0c9e522815` (squash merge PR #17)
**Confirm / promoção final:** **não** executados (aguarda nova autorização)

### Preflight e deploy fechado

| Item | Valor |
|---|---|
| Preflight release | `11c5884…`; documents=403; health ok; schema `3`/`dirty=false`; serviços active; timer inactive; `:18080` ABSENT |
| Backup bundle | `/root/bwb-s3c2-timer-retest-preflight-20260723T124837Z` (0600; PG dump `pg_restore -l` ok) |
| Env backup-id | `20260723T124837Z-10141af3…` |
| Deploy | `update-staging.sh` → promote ok; helper/sudoers/units/drop-in sincronizados; `visudo -cf`, `systemd-analyze verify`, `daemon-reload`, `nginx -t` OK |
| Pós-deploy | `current-sha=COMMIT=health.revision=10141af3…`; schema 3/dirty=false; documents local+ext=403; deny-all; timer inactive |

### Arm → espera real 5 min → rollback

| Evento | UTC |
|---|---|
| `nginx-open-arm` | 2026-07-23T12:49:41Z |
| Timer NEXT | 2026-07-23T12:54:42Z |
| Rollback fire / state | 2026-07-23T12:54:43Z |

| Gate | Resultado |
|---|---|
| `state` após arm | `armed`; timer **active**; Nginx active; documents sem token=**401**; `:18080` ABSENT |
| `nginx-open-confirm` | **não** executado |
| Unit rollback | `Result=success` (Finished / Deactivated successfully) |
| `state` final | **`rolled_back`** (nunca `armed`) |
| Timer final | inactive |
| `/v1/documents` local | **403** |
| `/v1/documents` externo | **403** |
| Nginx | active (sem emergency stop) |
| Probe (journal, só códigos) | `attempt=1 code=401` → `attempt=2 code=403`; `attempts=2 codes=401,403 result=ok` |
| `nginx_deny_all_ok` / `rollback_fire=ok` | sim |
| Emergência | nenhuma (`emergency_*=0`) |

### Estado do host após reteste

- Release: `10141af3cdd9cda16cfef46bbe5a4f0c9e522815`
- `state=rolled_back`; site deny-all; documents 403; timer inactive; serviços active; `:18080` ABSENT
- Promoção pública continua **não confirmada**

### Incidentes neste reteste

Nenhum incidente crítico. O retry 401→403 comportou-se como desenhado (I1 corrigido).

## Próximos passos

1. ~~Corrigir probe/fail-closed~~ — feito em PR #17 (`10141af3…`); reteste timer **APPROVED**.
2. Promoção final + `nginx-open-confirm` **só** com nova autorização explícita.
3. Não abrir `/v1/documents` publicamente até essa autorização.
