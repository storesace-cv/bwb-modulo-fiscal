# C-FE-JWS-REQ-001 — `solicitarSerie` `jwsSignature`: tabela ≠ Payload assinatura

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-08-16 |

## Factos

1. Tabela («campos … na assinatura»): `taxRegistrationNumber`, `establishmentNumber`, `seriesYear`, `documentType`.
2. Bloco «Payload assinatura Solicitar Serie»: acrescenta `seriesContingencyIndicator` (5 campos).
3. Fonte: `AO-FE-SNAP-HML-2026-07-25-SOLICITAR` (`pending_validation`).

## Mitigação

- Perfil `solicitar_serie_request` bloqueado.
