# Relatório S4 — validação operacional do kit POS (sandbox)

**Última actualização (UTC):** 2026-07-24T16:21Z (revalidação + cleanup)  
**Resultado S4 ops (cumulativo):** **APROVADO** na corrida 2 (após deploy `5d7c14b…`); corrida 1 permanece documentada como **REPROVADO**.  
**Host:** `sandbox.fiscalmod.bwb.pt` / `194.9.62.239`  
**URL canónica:** `https://sandbox.fiscalmod.bwb.pt/v1`

Este relatório **não** contém passwords, tokens, DSN, NIF, hashes de conteúdo, IDs de documentos fiscais, IDs externos de integrador, corpos de pedido/resposta nem correladores internos do kit.

Documento **cumulativo**: a corrida 1 (falha) não é apagada; a corrida 2 regista deploy + revalidação com o kit e o `fiscal-admin` da release corrigida.

---

## Corrida 1 — 2026-07-23 (REPROVADO)

**Data (UTC):** 2026-07-23T15:09Z (kit); cleanup 2026-07-23T15:10Z  
**Repo `main` (kit/docs):** `2f0339a31a17da08a31226e0e7f84cd61e76e31d`  
**Release activa no host:** `10141af3cdd9cda16cfef46bbe5a4f0c9e522815`  
**Resultado:** **REPROVADO** (`rate_429` FAIL; restantes PASS)

### Âmbito executado

- Preflight: `main` == `origin/main` == `2f0339a…`, working tree limpo; deps locais (bash/curl/jq/openssl) nas versões mínimas; health HTTPS 200; POST `/v1/documents` sem token → 401; portas externas 5432/8080/18080 fechadas/timeout.
- Scope sintético dedicado `scope-s4-val-001` (NIF/timezone/série alinhados às fixtures do kit; `environment=homologation`).
- Credencial A emitida → ficheiro kit-ready 52 bytes (sem LF) transferido via SCP 0600 → revogada → 401 confirmado.
- Credencial B emitida → transferida igual → kit local `scripts/integration/pos-sandbox-kit.sh` (sem `--allow-loopback-test`, sem `BWB_POS_KIT_*`).
- Cleanup: B revogada → 401; tokens/relatório temporários removidos no Mac e no servidor; scope e auditoria preservados; health 200; portas externas reconfirmadas fechadas.
- **Não** executado: deploy, alteração de Nginx/runtime/BD schema, push/PR.

### SHA / revision / serviços

| Item | Valor |
|---|---|
| `current` (host) | `10141af3cdd9cda16cfef46bbe5a4f0c9e522815` |
| `health.version` | `0.2.5-staging` |
| `health.revision` | `10141af3cdd9cda16cfef46bbe5a4f0c9e522815` |
| Auth runtime | `FISCAL_ENV=homologation` + `credential_store` |
| Nginx `limit_req` documentos | `rate=10r/s`, `burst=20`, `limit_req_status 429` (S3C1/S3C2) |

Nota: o host permanecia na release `10141af3…` (intencional — sem deploy nesta validação). O kit em execução era o de `main` `2f0339a…`.

### Casos do kit (sanitizados)

| Caso | Estado | HTTP / resultado |
|---|---|---|
| `create_201` | PASS | 201 |
| `replay` | PASS | 201 (campos estáveis idênticos) |
| `idempotency_conflict` | PASS | 409 |
| `external_id_conflict` | PASS | 409 |
| `scope_mismatch` | PASS | 403 |
| `validation_422` | PASS | 422 |
| `unauthorized_bad_token` | PASS | 401 |
| `token_revoked_401` | PASS | 401 |
| `rate_429` | **FAIL** | ver contagem abaixo |

Sumário kit: pass=8, fail=1, not_run=0. Exit code=1.

### Contagem `rate_429` (corrida 1)

| Métrica | Valor |
|---:|---:|
| HTTP 201 | 30 |
| HTTP 429 | 0 |
| HTTP 5xx | 0 |
| Erros de transporte | 0 |
| other | 0 |
| collected | 30 |
| alive (PIDs) | 0 |

Critério exigido: ≥1×429, 0×5xx, 0 transporte, exactamente 30 resultados. **Não cumprido** (0×429).

Padrão de carga do kit (então): 30 POSTs em ondas de 5 concorrentes (aguarda conclusão de cada onda). Com `rate=10r/s` e `burst=20`, este padrão não gerou 429 no sandbox.

### Transferência / token (corrida 1)

Transferência: `sudo` + `install`/`python3` para ficheiro 0600 do utilizador `ubuntu`, SCP, shred da cópia de xfer. **Remoção do LF final** do ficheiro admin (53→52 bytes) para cumprir o contrato do kit; sem alteração de sudoers/helper.

### Cleanup (corrida 1)

| Item | Estado |
|---|---|
| Scope `scope-s4-val-001` | **activo** (não eliminado) |
| Credenciais A/B | revogadas; probe POST → 401 |
| Ficheiros token Mac / servidor / `s4-xfer` | eliminados |
| Documentos / auditoria | preservados |
| Health pós-cleanup | 200 |
| Externo 5432 / 8080 / 18080 | closed/timeout |

### Incidentes abertos após corrida 1

#### INC-S4-001 — `rate_429` sem nenhum 429

| Campo | Valor |
|---|---|
| Severidade | Alta (bloqueia aprovação operacional S4) |
| Causa (hipótese fundamentada) | O kit disparava 30 pedidos em 6 ondas de 5; o Nginx público está em `10r/s` + `burst=20`. A pressão efectiva ficava abaixo do limiar que produz 429. |
| Impacto | Validação S4 incompleta; 8/9 casos PASS; critério de rate limit não demonstrado contra o sandbox real. |
| Resolução (corrida 1) | Cleanup concluído; **sem** alteração de código/Nginx nessa sessão. |
| Estado após corrida 1 | Aberto |

#### INC-S4-002 — formato do ficheiro de token admin vs kit

| Campo | Valor |
|---|---|
| Severidade | Baixa (contornada na transferência; não bloqueou) |
| Causa | `fiscal-admin` gravava token + LF (53 bytes); o kit exige exactamente 52 bytes sem CR/LF. |
| Impacto | Ficheiro bruto do helper não era utilizável pelo kit sem normalização. |
| Resolução (corrida 1) | Na transferência operacional, removeu-se apenas o LF final. |
| Estado após corrida 1 | Mitigado operacionalmente; melhoria de produto pendente |

### Decisões / bloqueios (corrida 1)

1. **Não** alterar o kit nem o Nginx para forçar PASS nessa corrida.
2. Validação S4 contra sandbox real: **não aprovada** até `rate_429` PASS com evidência sanitizada.
3. Scope `scope-s4-val-001` mantido activo para auditoria/reexecução futura; credenciais A/B revogadas.

Branch local de preservação da corrida 1: `ops/s4-pos-kit-ops-validation-report` @ `275c43b0dea1eb5a916ca8930df9b662e0ad890e`.

---

## Corrida 2 — 2026-07-24 (APROVADO)

**Data (UTC):** deploy ~16:18Z; kit 16:20:50Z–16:20:59Z; cleanup ~16:21Z  
**Repo `main` (kit + admin):** `5d7c14b01af1f4855a41ee2b4f251af96dd5b726`  
**Release activa no host (após deploy):** `5d7c14b01af1f4855a41ee2b4f251af96dd5b726`  
**Resultado:** **APROVADO** (9/9 PASS, incluindo `rate_429`)

### Pré-requisito de deploy (sudoers)

O `update-staging.sh` via `bwb-deploy` falhou inicialmente com `sudo: a password is required`. Em `/etc/sudoers.d/bwb-fiscal-deploy` estava o placeholder literal `DEPLOY_USER` (User_Alias indefinido) em vez de `bwb-deploy`.

Correcção no host (via `ubuntu`, backup prévio, `visudo -cf` OK, modo `0440`):

```text
bwb-deploy ALL=(root) NOPASSWD: /usr/local/sbin/bwb-fiscal-deploy-helper
```

Sem alargar o sudoers a outros binários. Após a correcção, `update-staging.sh` concluiu com `report done`.

### Deploy controlado

| Item | Valor |
|---|---|
| Previous release | `10141af3cdd9cda16cfef46bbe5a4f0c9e522815` |
| Active release | `5d7c14b01af1f4855a41ee2b4f251af96dd5b726` |
| Schema before/after | `3` / `3` (dirty=false; sem migração) |
| `health.revision` | `5d7c14b01af1f4855a41ee2b4f251af96dd5b726` |
| Nginx documentos | inalterado: `rate=10r/s`, `burst=20` |

Motivo do deploy antes da revalidação: provar também o `fiscal-admin --output-file` sem newline; kit novo contra host antigo voltaria a exigir o contorno do LF e deixaria metade da correcção sem prova.

### Âmbito executado (corrida 2)

- Preflight: `main`/`origin/main` = `5d7c14b…`, tree limpa no momento do deploy; health 200 + revision correcta; POST sem token → 401; portas 5432/8080/18080 fechadas/timeout.
- Scope `scope-s4-val-001` reutilizado (activo desde corrida 1).
- Credencial A: ficheiro **bruto** do admin = **52 bytes**, `raw_has_lf=0`, `raw_has_cr=0` (sem strip); SCP 0600; revogada → 401.
- Credencial B: igual prova de 52 bytes raw; kit local sem `--allow-loopback-test` / sem `BWB_POS_KIT_*`.
- Cleanup: B revogada → 401; tokens/xfer eliminados; scope + auditoria preservados; health 200; portas externas fechadas.

### Casos do kit (sanitizados) — corrida 2

| Caso | Estado | HTTP / resultado |
|---|---|---|
| `create_201` | PASS | 201 |
| `replay` | PASS | 201 |
| `idempotency_conflict` | PASS | 409 |
| `external_id_conflict` | PASS | 409 |
| `scope_mismatch` | PASS | 403 |
| `validation_422` | PASS | 422 |
| `unauthorized_bad_token` | PASS | 401 |
| `token_revoked_401` | PASS | 401 |
| `rate_429` | **PASS** | ver contagem abaixo |

Sumário kit: pass=9, fail=0, not_run=0. Exit code=0.

### Contagem `rate_429` (corrida 2)

| Métrica | Valor |
|---:|---:|
| HTTP 201 | 22 |
| HTTP 429 | 8 |
| HTTP 5xx | 0 |
| Erros de transporte | 0 |
| other | 0 |
| collected | 30 |
| alive (PIDs) | 0 |

Critério: ≥1×429, 0×5xx, 0 transporte, exactamente 30 resultados. **Cumprido**.

Padrão de carga do kit (release `5d7c14b…`): rajada **sincronizada** de 30 pedidos (workers ready → gate único), alinhada a `10r/s` + `burst=20`.

### Prova INC-S4-002 (corrida 2)

| Verificação | Resultado |
|---|---|
| Bytes do ficheiro admin (raw, sem strip) | 52 |
| CR no ficheiro | 0 |
| LF no ficheiro | 0 |
| Transferência | `install` byte-a-byte + SCP; **sem** remoção de newline |
| Aceitação pelo kit | PASS (`--token-file` / `--revoked-token-file`) |

Nota de domínio observada na emissão: `Issue` permite no máximo **uma** credencial `active` por scope (`ErrCredentialConflict`); segunda emissão só após revoke — comportamento esperado, não incidente S4.

### Cleanup (corrida 2)

| Item | Estado |
|---|---|
| Scope `scope-s4-val-001` | **activo** (preservado) |
| Credenciais A/B (reval) | revogadas; probe POST → 401 |
| Ficheiros token Mac / `issue-scope-s4-val-001-*` / `current.token` / `s4-xfer` | eliminados |
| Documentos / auditoria | preservados |
| Health pós-cleanup | 200; revision `5d7c14b…` |
| Externo 5432 / 8080 / 18080 | closed/timeout |

### Estado dos incidentes após corrida 2

| ID | Estado | Evidência |
|---|---|---|
| INC-S4-001 | **Fechado** | `rate_429` PASS: 8×429, 22×201, collected=30, 0×5xx/transporte; Nginx inalterado |
| INC-S4-002 | **Fechado** | `fiscal-admin --output-file` na release `5d7c14b…` grava 52 bytes sem CR/LF; kit aceita ficheiro bruto |

### Decisões (corrida 2)

1. Deploy controlado da release de correcção **antes** da revalidação — necessário para prova conjunta kit + admin.
2. Nginx `10r/s`/`burst=20` **não** alterado.
3. Relatório actualizado cumulativamente nesta branch; **sem** push/PR até revisão humana.
4. Scope `scope-s4-val-001` mantido activo para rastreio.
