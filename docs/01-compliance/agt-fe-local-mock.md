# Mock FE AGT local (RM-FEFIX-004)

**Estado:** harness de teste estritamente local.

**Não é** HML AGT, **não é** PRD AGT, **não é** homologação nem aceitação oficial.

## Pacote

[`internal/authority/femock`](../../internal/authority/femock/doc.go) — HTTP handler / `httptest` sob `/mock/agt-fe/v1/...`.

Distinto de [`internal/authority/simulator`](../../internal/authority/simulator/simulator.go) (stub in-process de submissão via `fiscaljws`).

## JWS

| | AGT (aberto) | BWB mock |
|---|---|---|
| `typ` | JWT vs JOSE — [C-FE-JWS-TYP-001](../../compliance/derived/conflicts/C-FE-JWS-TYP-001-typ-jwt-vs-jose.md) / [AGT-Q-005](agt-clarifications-register.md#agt-q-005) | **`BWB-MOCK` apenas** |
| Perfis wire | zero activos | assinatura mock helper; `feprofile.Sign*` permanece bloqueado |
| Sucesso | — | **≠** homologação AGT |

## Operações

Suportadas (payload builders + JWS BWB-MOCK): `softwareInfo`, `obterEstado`, `consultarFactura`.

Bloqueadas (`BWB-MOCK-PROFILE-BLOCKED` + conflito): `registarFactura`, `solicitarSerie`, `listarSeries`, `validarDocumento`, `listarFacturas`.

## FE-RNG (catálogo contextual)

`ScriptFERNG(op, code)` só aceita entradas `emit_active` para essa operação exacta, com `source_id` da operação:

| Operação | Códigos emitíveis | source_id |
|---|---|---|
| `softwareInfo` | FE-RNG-010, FE-RNG-011 | `AO-FE-SNAP-HML-2026-07-25-REGISTAR` |
| `obterEstado` | FE-RNG-010, FE-RNG-011, FE-RNG-032 | `AO-FE-SNAP-HML-2026-07-25-CONSULTAR` |
| `consultarFactura` | FE-RNG-010, FE-RNG-011, FE-RNG-032 | `AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA` |

Códigos REGISTAR/SOLICITAR (ex. FE-RNG-031, 051, 080) ficam `reference_blocked` — catalogados, **nunca** emitidos pelas rotas HTTP activas. Ver [AGT-Q-019](agt-clarifications-register.md#agt-q-019) (031 vs 032).

## Transporte / idempotência

- Content-Type: `application/json` obrigatório (`mime.ParseMediaType`; charset utf-8 opcional).
- Ordem: Basic Auth → parse → JWS/role/binding/payload → depois replay.
- Replay: resultado funcional estável; **requestID novo** por chamada; `clientRequestID` ignorado.

## Segurança

Basic Auth injectado no construtor (limpo em `Close`); sem credenciais versionadas; sem paths `/sigt/fe/...`; sem listener público; sem rede AGT.

## Boundary / hub (RM-FEFIX-005)

Ver [`agt-fe-boundary-hub.md`](agt-fe-boundary-hub.md): estados `fixture_boundary_*` + hub `fixture_agt`/`homologation_agt`/`production_agt`. Sucesso mock **≠** aceite AGT.
