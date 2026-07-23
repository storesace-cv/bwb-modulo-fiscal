# Relatório S3C1 — deploy fechado + medição loopback

**Data (UTC):** 2026-07-22T12:01Z (medição); cleanup confirmado 12:02Z
**Squash / release:** `983a8013aba894a933429262cc5c281534b841b5`
**Schema no host:** `3` (`dirty=false`)
**Host:** `sandbox.fiscalmod.bwb.pt` / `194.9.62.239`
**Resultado S3C1:** **APROVADO**

Este relatório **não** contém passwords, tokens, DSN, hashes, NIF nem corpos de pedido/resposta.

## Âmbito executado

- Deploy fechado da release `983a8013…` via `update-staging.sh` (Nginx público **deny-all** mantido).
- Provisionamento efémero de scope/credencial sintéticos; medição `sustained` / `burst` / `replay` em `127.0.0.1:18080` com `fiscal-sandbox-measure`.
- Cleanup: revogação, remoção de tokens, desactivação de `:18080`, reconfirmação deny-all.
- **Não** executado: S3C2, instalação de conf Nginx pública aberta, abertura de `/v1/documents`.

## SHA / revision / schema / serviços

| Item | Valor |
|---|---|
| `current-sha` | `983a8013aba894a933429262cc5c281534b841b5` |
| `COMMIT` (release) | `983a8013aba894a933429262cc5c281534b841b5` |
| `health.revision` (loopback + HTTPS) | `983a8013aba894a933429262cc5c281534b841b5` |
| Schema | `3` / `dirty=false` |
| `bwb-fiscal-api` | active |
| Nginx | active |
| PostgreSQL | active |
| Auth runtime | `FISCAL_ENV=homologation` + `credential_store` |

## Portas e deny-all

| Superfície | Estado pós-cleanup |
|---|---|
| HTTPS `/v1/documents` | **403** deny-all |
| HTTPS `/v1/health` | 200 `status=ok` + revision correcta |
| `127.0.0.1:8080` | API loopback |
| `127.0.0.1:5432` | PG loopback |
| `127.0.0.1:18080` | **ABSENT** após cleanup |
| Externo 5432 / 8080 / 18080 | closed/timeout |

## Backups operacionais (sem conteúdo)

| Artefacto | Path |
|---|---|
| Env backup (helper) | id `20260722T115722Z-2f96fe45c0d8ad3cb2e21d8755f2988eb4a43dfd` |
| Env backup (updater) | id `20260722T115749Z-983a8013aba894a933429262cc5c281534b841b5` |
| Config Nginx pré-S3C1 | `/etc/bwb-modulo-fiscal/backups/s3c1-pre-20260722T115722Z/` |
| PG dump | `/var/backups/bwb-fiscal/fiscal-s3c1-pre-20260722T115722Z.dump` (`root:root` `0600`) |

## IDs sintéticos (permitidos)

| Tipo | ID |
|---|---|
| Scope carga/medição | `scope-s3c1-load-001` |
| Credencial medição (revogada) | `7bf2f42fc42a803d340e180f9be15586` |
| `created-by` | `s3c1-measure` |
| Motivo de revogação | `s3c1_done` |

Tokens, hashes, DSN e NIF: **omitidos**.

## Medição — resultados agregados

Base: `http://127.0.0.1:18080`. Zona provisória: `rate=10r/s`, `burst=20`, `limit_req_status 429`. Binário: `fiscal-sandbox-measure` da release activa. Helper host actualizado para a versão do commit (ver incidentes).

### sustained — passed=true (exit 0)

| Métrica | Valor | Limite |
|---|---:|---|
| attempted / http_responses | 300 / 300 | 300 |
| status_201 / 429 / 409 / 5xx / other | 300 / 0 / 0 / 0 / 0 | 429≤3; restantes 0 |
| transport_errors | 0 | 0 |
| duration_ms | 30104 | 28000–33000 |
| request_throughput_rps | ≈9.965 | 9–11 |
| p95_ms_201 / p99_ms_201 | 13 / 26 | ≤250 / ≤500 |

### burst — passed=true (exit 0)

| Métrica | Valor | Limite |
|---|---:|---|
| attempted / http_responses | 60 / 60 | ≤60 |
| status_201 | 21 | 20–25 |
| status_429 | 39 | restantes = 429 |
| 409 / 5xx / other / transport | 0 | 0 |

### replay — passed=true (exit 0)

| Métrica | Valor |
|---|---|
| status_201 | 2 |
| replay_identical | true |
| transport_errors | 0 |

## Cleanup (obrigatório)

| Passo | Estado |
|---|---|
| Revogar credencial | ok (`status=revoked`) |
| Auth pós-revogação | HTTP **401** com token antigo |
| Remover `measure.token` / `current.token` | ausentes |
| Desactivar `:18080` | ABSENT (após remoção sites-enabled/available + reload) |
| `/v1/documents` público | **403** |

## Incidentes

| Severidade | Fase | Descrição | Impacto | Resolução | Estado | Risco residual |
|---|---|---|---|---|---|---|
| Médio | S3C1 measure | Helper em `/usr/local/sbin` ainda com assinatura antiga `admin-sandbox-measure <sha40>` (sem perfil) | Medição bloqueada (`usage` error) | Instalação do `remote-deploy-helper.sh` do commit `983a8013…` no host | Corrigido | Helper não versionado automaticamente com a release; próximo deploy deve incluir passo explícito de sync do helper |
| Baixo | S3C1 measure | UFW SSH rate-limit após rajada de sessões sem mux | `Connection refused` temporário | Espera + `ControlMaster` mux | Corrigido | Reutilizar mux em ops futuras |
| Médio | S3C1 cleanup | Após `rm` do link em `sites-enabled` + reload, `:18080` ainda listado uma vez | Cleanup incompleto momentâneo | Remoção também de `sites-available` + reload; confirmação `ABSENT` | Corrigido | Baixo — procedimento de cleanup deve remover enabled **e** available e verificar `ss` |
| Informativo | Preflight | Release anterior `2f96fe45…` sem `health.revision` | Esperado pré-tooling | Deploy `983a8013…` | Encerrado | — |

## Conclusão

S3C1 **aprovado** com deny-all público preservado. Evidência de taxa sustentada ≈10 r/s e burst (201∈[20,25], resto 429) disponível para o PR S3C2 futuro. Evidência aceite para PR S3C2 (repo only; promoção no host só após merge).

Incidentes de desenho do helper/Nginx S3C2 encontrados em revisão pré-merge (timer/confirm/flock/boot/ACME/HSTS/exact location) estão documentados e corrigidos no runbook § «S3C2 — incidentes pré-merge» — **não** foram aplicados no host; deny-all permanece.
