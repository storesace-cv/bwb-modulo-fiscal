# FE boundary + hub metadados fixture/HML/PRD (RM-FEFIX-005)

**Estado:** integração local até ao boundary BWB-MOCK.  
**Não é** chamada real AGT; **não é** homologação; **não é** `accepted` fiscal (`DEC-PROD-009`).

## Definição canónica (ROADMAP)

> Outbox/estados até boundary + hub metadados fixture/HML/PRD  
> Done: *Nunca rotular aceite AGT só com assinatura local*

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

Slots (`endpoint_base_ref`, `credential_ref`, …) aceitam só refs; rejeitam PEM/password/Basic. `external_verified` **sempre** false.

## Estados até boundary

`queued_for_fixture_boundary` → `fixture_boundary_in_flight` →  
`fixture_boundary_ok` \| `fixture_boundary_reject` \| `fixture_boundary_profile_blocked` \| `fixture_boundary_transport_failed`

- Assinatura local sozinha **não** avança para OK sem `Process` ao mock.
- `Submission.IsAGTAccepted()` **sempre** false.
- Sucesso mock **≠** `authority_accepted` AGT.

## Operações

As mesmas do mock activo: `obterEstado`, `consultarFactura`, e `softwareInfo` (**sintético BWB-MOCK**, ≠ endpoint AGT).

## Labels obrigatórios

`mock≠HML≠PRD` · `fixture_ok≠AGT_accepted` · fontes `pending_validation`.
