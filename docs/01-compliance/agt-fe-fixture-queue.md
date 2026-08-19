# Fila persistente FE fixture (RM-FEFIX-007 / RM-FEFIX-008)

**Estado:** fila SQL + runtime `fiscal-api` mock-only (**RM-FEFIX-007/008 CONCLUÍDO** em `main`). Cadeia FEFIX-001…008 fecha o âmbito **executável** do plano Excel/chaves. **≠** homologação AGT; **≠** `authority_accepted`; wire HML/PRD bloqueado (**AGT-Q-003**, **RM-FE-001**).

## Objectivo

Fechar o plano Excel/chaves (PR5–PR6) ligando identidades RSA de teste (`agttestkit`) à fila SQL `fe_fixture_submissions`, worker com retries, e boundary BWB-MOCK (`feboundary` → `femock`).

## Pacotes

| Pacote | Papel |
|---|---|
| [`internal/agttestkit`](../../internal/agttestkit/doc.go) | Inventário/validação workbook (read-only; CI sintético) |
| [`internal/authority/fefixqueue`](../../internal/authority/fefixqueue/doc.go) | Fila SQL + worker |
| [`internal/authority/feboundary`](../../internal/authority/feboundary/boundary.go) | Assinatura BWB-MOCK + HTTP loopback |
| [`internal/authority/femock`](../../internal/authority/femock/doc.go) | Mock FE local |
| [`internal/authority/fixtruntime`](../../internal/authority/fixtruntime/doc.go) | Arranque loopback femock + worker no `fiscal-api` |
| [`internal/authority/prep`](../../internal/authority/prep/fixture_identities.go) | Catálogo sanitizado admin |

## Configuração operador

| Variável | Uso |
|---|---|
| `FISCAL_AGT_TEST_WORKBOOK` | Path absoluto ao workbook AGT em `local/` (fora Git). **Obrigatório** para activar runtime fixture. |
| `FISCAL_FE_FIXTURE_MOCK_USER` | Basic Auth sintético loopback (default `bwb-fixture-mock`) |
| `FISCAL_FE_FIXTURE_MOCK_PASS` | Password sintética loopback (gerada se omitida) |
| `FISCAL_FE_FIXTURE_WORKER_INTERVAL` | Poll da fila (default `2s`) |
| Admin `GET /admin/v1/authority/fixture-identities` | Lista refs sanitizadas (owner-only) |
| Admin `GET /admin/v1/authority/fixture-hub` | Metadados hub `fixture_agt` |
| Admin `GET /admin/v1/authority/fixture-runtime` | Estado runtime (loopback, contagem identidades) |
| Admin `GET/POST /admin/v1/authority/fixture-submissions` | Listar / enfileirar submissões mock |
| Admin `GET /admin/v1/authority/fixture-submissions/{id}` | Detalhe submissão |
| Admin `POST /admin/v1/authority/fixture-submissions/process-next` | Processar manualmente uma fila (worker também corre em background) |

## Estados persistidos

Mesmos de [`agt-fe-boundary-hub.md`](agt-fe-boundary-hub.md), mais `fixture_boundary_dead` após esgotar retries.

- Retries/backoff em falha de transporte mock (5 tentativas, 5s).
- Reclaim `in_flight` stale (30s).
- Idempotência `(idempotency_key, operation)`.
- `IsAGTAccepted()` **sempre** false.

## Operações activas (mock)

`softwareInfo`, `obterEstado`, `consultarFactura` — restantes permanecem `blocked_conflict` (wire JWS AGT).

## Explicitamente fora de âmbito

- Wire JWS AGT (`C-FE-JWS-TYP-001`, `AGT-Q-003`…`019`).
- Endpoints/credenciais HML/PRD reais (`RM-FE-001` ADIADO).
- Mapear `fixture_boundary_ok` → `authority_accepted`.
- Workbook real em Git/CI.

## Schema

Migration `0014` · `ExpectedVersion=14` · grants [`deploy/postgres/grants-schema14-runtime-admin.sql`](../../deploy/postgres/grants-schema14-runtime-admin.sql).
