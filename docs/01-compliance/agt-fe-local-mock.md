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

## FE-RNG

Códigos simuláveis só da allowlist no código (`AllowlistedFERNG`), com `source_id` do snapshot (`pending_validation`). Códigos desconhecidos são rejeitados no harness. Respostas marcam `simulated=true`.

## Segurança

Basic Auth injectado no construtor; sem credenciais versionadas; sem paths `/sigt/fe/...`; sem listener público obrigatório; sem rede AGT.
