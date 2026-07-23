# Relatório S4 — validação operacional do kit POS (sandbox)

**Data (UTC):** 2026-07-23T15:09Z (kit); cleanup 2026-07-23T15:10Z  
**Repo `main` (kit/docs):** `2f0339a31a17da08a31226e0e7f84cd61e76e31d`  
**Release activa no host:** `10141af3cdd9cda16cfef46bbe5a4f0c9e522815`  
**Host:** `sandbox.fiscalmod.bwb.pt` / `194.9.62.239`  
**URL canónica:** `https://sandbox.fiscalmod.bwb.pt/v1`  
**Resultado S4 ops:** **REPROVADO** (`rate_429` FAIL; restantes PASS)

Este relatório **não** contém passwords, tokens, DSN, NIF, hashes de conteúdo, IDs de documentos fiscais, IDs externos de integrador, corpos de pedido/resposta nem correladores internos do kit.

## Âmbito executado

- Preflight: `main` == `origin/main` == `2f0339a…`, working tree limpo; deps locais (bash/curl/jq/openssl) nas versões mínimas; health HTTPS 200; POST `/v1/documents` sem token → 401; portas externas 5432/8080/18080 fechadas/timeout.
- Scope sintético dedicado `scope-s4-val-001` (NIF/timezone/série alinhados às fixtures do kit; `environment=homologation`).
- Credencial A emitida → ficheiro kit-ready 52 bytes (sem LF) transferido via SCP 0600 → revogada → 401 confirmado.
- Credencial B emitida → transferida igual → kit local `scripts/integration/pos-sandbox-kit.sh` (sem `--allow-loopback-test`, sem `BWB_POS_KIT_*`).
- Cleanup: B revogada → 401; tokens/relatório temporários removidos no Mac e no servidor; scope e auditoria preservados; health 200; portas externas reconfirmadas fechadas.
- **Não** executado: deploy, alteração de Nginx/runtime/BD schema, push/PR.

## SHA / revision / serviços

| Item | Valor |
|---|---|
| `current` (host) | `10141af3cdd9cda16cfef46bbe5a4f0c9e522815` |
| `health.version` | `0.2.5-staging` |
| `health.revision` | `10141af3cdd9cda16cfef46bbe5a4f0c9e522815` |
| Auth runtime | `FISCAL_ENV=homologation` + `credential_store` |
| Nginx `limit_req` documentos | `rate=10r/s`, `burst=20`, `limit_req_status 429` (S3C1/S3C2) |

Nota: o host permanece na release `10141af3…` (intencional — sem deploy nesta validação). O kit em execução é o de `main` `2f0339a…`.

## Casos do kit (sanitizados)

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

## Contagem `rate_429`

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

Padrão de carga do kit: 30 POSTs em ondas de 5 concorrentes (aguarda conclusão de cada onda). Com `rate=10r/s` e `burst=20`, este padrão não gerou 429 no sandbox.

## Relatório do kit

- Relatório temporário 0600 sem token/NIF/IDs de documento/IDs externos (scan local PASS).
- tmpdir do kit (`/tmp/bwb-pos-kit.*`) removido pelo trap EXIT.
- Sem redirecionamentos HTTP observados nos casos PASS (validação implícita do kit contra URL canónica HTTPS).

## Credenciais / cleanup

| Item | Estado |
|---|---|
| Scope `scope-s4-val-001` | **activo** (não eliminado) |
| Credencial A | revogada; probe POST → 401 |
| Credencial B | revogada; probe POST → 401 |
| Ficheiros token Mac | eliminados |
| Ficheiros token servidor (`issue-scope-s4-val-001-*.token`) | eliminados |
| `/home/ubuntu/s4-xfer` | removido |
| Documentos / auditoria | preservados |
| Health pós-cleanup | 200 |
| Externo 5432 / 8080 / 18080 | closed/timeout |

Transferência: `sudo` + `install`/`python3` para ficheiro 0600 do utilizador `ubuntu`, SCP, shred da cópia de xfer. Remoção do LF final do ficheiro admin (53→52 bytes) para cumprir o contrato do kit; sem alteração de sudoers/helper.

## Incidentes

### INC-S4-001 — `rate_429` sem nenhum 429

| Campo | Valor |
|---|---|
| Severidade | Alta (bloqueia aprovação operacional S4) |
| Causa (hipótese fundamentada) | O kit dispara 30 pedidos em 6 ondas de 5; o Nginx público está em `10r/s` + `burst=20`. A pressão efectiva fica abaixo do limiar que produz 429. |
| Impacto | Validação S4 incompleta; 8/9 casos PASS; critério de rate limit não demonstrado contra o sandbox real. |
| Resolução | Cleanup concluído; **sem** alteração de código/Nginx nesta sessão. Parar para análise (ajustar padrão de carga do kit vs. limites aprovados S3C1/S3C2, ou redefinir critério operacional). |
| Estado | Aberto |
| Risco residual | Integradores podem passar o kit localmente em mock e falhar/omitir prova de 429 no sandbox real; falsa confiança no caso `rate_429`. |

### INC-S4-002 — formato do ficheiro de token admin vs kit (observação)

| Campo | Valor |
|---|---|
| Severidade | Baixa (contornada na transferência; não bloqueou) |
| Causa | `fiscal-admin` grava token + LF (53 bytes); o kit exige exactamente 52 bytes sem CR/LF. |
| Impacto | Ficheiro bruto do helper não é utilizável pelo kit sem normalização. |
| Resolução | Na transferência operacional, removeu-se apenas o LF final; conteúdo do token inalterado. |
| Estado | Mitigado operacionalmente; melhoria de produto pendente (admin sem LF **ou** kit a aceitar um LF final único, com decisão explícita). |
| Risco residual | Operadores que copiem o ficheiro bruto falham o kit com erro de validação. |

## Decisões / bloqueios

1. **Não** alterar o kit nem o Nginx para forçar PASS nesta corrida.
2. Validação S4 contra sandbox real: **não aprovada** até `rate_429` PASS com evidência sanitizada.
3. Scope `scope-s4-val-001` mantido activo para auditoria/reexecução futura; credenciais A/B revogadas.
