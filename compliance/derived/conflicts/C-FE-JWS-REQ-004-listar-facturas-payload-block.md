# C-FE-JWS-REQ-004 — `listarFacturas` `jwsSignature`: sem bloco Payload assinatura cruzável

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-08-16 |

## Factos

1. Tabela lista `taxRegistrationNumber`, `queryStartDate`, `queryEndDate`.
2. O HTML do serviço **não** apresenta bloco «Payload assinatura» alinhável (padrão usado noutros serviços FE).
3. Fonte: `AO-FE-SNAP-HML-2026-07-25-LISTAR-FATURAS` (`pending_validation`). Sem cruzamento tabela/bloco, o payload exacto não está fechado.

## Mitigação

- Perfil `listar_facturas_request` bloqueado.
