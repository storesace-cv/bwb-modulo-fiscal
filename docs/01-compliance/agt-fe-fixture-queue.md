# Fila persistente FE fixture (RM-FEFIX-007)

**Estado:** integração mock-only concluída no repositório.
**Não é** homologação AGT; **não é** `authority_accepted`; **≠** outbox slice `fiscaljws`.

## Objectivo

Fechar o plano Excel/chaves (PR5–PR6) ligando identidades RSA de teste (`agttestkit`) à fila SQL `fe_fixture_submissions`, worker com retries, e boundary BWB-MOCK (`feboundary` → `femock`).

## Pacotes

| Pacote | Papel |
|---|---|
| [`internal/agttestkit`](../../internal/agttestkit/doc.go) | Inventário/validação workbook (read-only; CI sintético) |
| [`internal/authority/fefixqueue`](../../internal/authority/fefixqueue/doc.go) | Fila SQL + worker |
| [`internal/authority/feboundary`](../../internal/authority/feboundary/boundary.go) | Assinatura BWB-MOCK + HTTP loopback |
| [`internal/authority/femock`](../../internal/authority/femock/doc.go) | Mock FE local |
| [`internal/authority/prep`](../../internal/authority/prep/fixture_identities.go) | Catálogo sanitizado admin |

## Configuração operador

| Variável | Uso |
|---|---|
| `FISCAL_AGT_TEST_WORKBOOK` | Path absoluto ao workbook AGT em `local/` (fora Git). Opcional; default off em CI. |
| Admin `GET /admin/v1/authority/fixture-identities` | Lista refs sanitizadas quando workbook configurado (owner-only) |
| Admin `GET /admin/v1/authority/fixture-hub` | Metadados hub `fixture_agt` |

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
