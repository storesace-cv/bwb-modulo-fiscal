# FE boundary + hub metadados fixture/HML/PRD (RM-FEFIX-005/006)

**Estado:** integração local até ao boundary BWB-MOCK (hardening RM-FEFIX-006).
**Não é** chamada real AGT; **não é** homologação; **não é** `accepted` fiscal (`DEC-PROD-009`).

## Definição canónica (ROADMAP)

> Outbox/estados até boundary + hub metadados fixture/HML/PRD
> Done: *Nunca rotular aceite AGT só com assinatura local*

Follow-up **RM-FEFIX-006**: hardening P1/P2 do PR #161 (slots, BaseURL loopback, concorrência, HTTP 200 semântico).

## Pacotes

| Pacote | Papel |
|---|---|
| [`internal/authority/fehub`](../../internal/authority/fehub/hub.go) | Hub metadados `fixture_agt` \| `homologation_agt` \| `production_agt` |
| [`internal/authority/feboundary`](../../internal/authority/feboundary/boundary.go) | Fila/estados → assinatura BWB-MOCK → [`femock`](../../internal/authority/femock/doc.go) |

Distinto de [`internal/persistence/outbox`](../../internal/persistence/outbox.go) → [`simulator`](../../internal/authority/simulator/simulator.go) (`fiscaljws` slice).

## Hub

| Kind | Transporte |
|---|---|
| `fixture_agt` | Permitido (BWB-MOCK local) |
| `homologation_agt` / `production_agt` | **Fail-closed** até credenciais/endpoints oficiais |

Slots (`endpoint_base_ref`, `credential_ref`, …) aceitam só refs opacas (charset restrito); rejeitam PEM, Bearer/Basic, DSN, JWT (`eyJ…`), `://`, userinfo, etc. Estado do hub é **não exportado**; `View()` revalida e **limpa** slots inválidos. `external_verified` **sempre** false.

## Boundary (RM-FEFIX-006)

- **BaseURL:** só `http` em host loopback autorizado (`127.0.0.1` / `localhost` / `::1`); sem userinfo, query, fragment ou path; redirects **negados**.
- **Concorrência:** só `queued` → `in_flight`; `in_flight` ou estados terminais → erro (`ErrInFlight` / `ErrInvalidTransition`); snapshot de credenciais sob lock.
- **HTTP 200 → `fixture_boundary_ok`:** exige `Content-Type: application/json`, envelope `simulated=true` + `mock=BWB-MOCK`, `status=ok`, `requestID` não vazio, e `operation` coerente quando presente. Corpo vazio/malformado → `fixture_boundary_transport_failed`.
- **`fixture_boundary_profile_blocked`:** ramo defensivo (Enqueue não inclui ops wire-blocked); exercitável via stub HTTP 422 + `BWB-MOCK-PROFILE-BLOCKED`.

## Estados até boundary

`queued_for_fixture_boundary` → `fixture_boundary_in_flight` →
`fixture_boundary_ok` \| `fixture_boundary_reject` \| `fixture_boundary_profile_blocked` \| `fixture_boundary_transport_failed`

- Assinatura local sozinha **não** avança para OK sem `Process` ao mock.
- `Submission.IsAGTAccepted()` **sempre** false.
- Sucesso mock **≠** `authority_accepted` AGT.
- Validação de input (payload nil) devolve `ErrInvalidInput` (não engole como transporte OK).

## Operações

As mesmas do mock activo: `obterEstado`, `consultarFactura`, e `softwareInfo` (**sintético BWB-MOCK**, ≠ endpoint AGT).

Persistência/worker: [`agt-fe-fixture-queue.md`](agt-fe-fixture-queue.md) (RM-FEFIX-007).

## Labels obrigatórios

`mock≠HML≠PRD` · `fixture_ok≠AGT_accepted` · fontes `pending_validation`.
