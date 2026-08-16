# C-FE-JWS-REQ-002 — `listarSeries` `jwsSignature`: tabela ≠ Payload assinatura

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-08-16 |

## Factos

1. Tabela: `taxRegistrationNumber`, `documentNo`.
2. Bloco «Payload assinatura Listar Serie»: apenas `taxRegistrationNumber`.
3. Fonte: `AO-FE-SNAP-HML-2026-07-25-LISTAR` (`pending_validation`).

## Mitigação

- Perfil `listar_series_request` bloqueado.
