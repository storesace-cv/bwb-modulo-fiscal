# C-FE-JWS-REQ-003 — `validarDocumento` `jwsSignature`: tabela ≠ Payload assinatura

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-08-16 |

## Factos

1. Tabela: `taxRegistrationNumber`, `documentNo`.
2. Bloco «Payload assinatura Validar Documento»: inclui também `action`, `deductibleVATPercentage`, `nonDeductibleAmount`.
3. Fonte: `AO-FE-SNAP-HML-2026-07-25-VALIDAR` (`pending_validation`).

## Mitigação

- Perfil `validar_documento_request` bloqueado.
