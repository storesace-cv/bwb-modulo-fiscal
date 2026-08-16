# C-FE-JWS-DOC-001 — `jwsDocumentSignature`: exemplo vs campos documentados

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-08-16 |
| Severidade | alta (registarFactura) |

## Factos

1. `AO-FE-SNAP-HML-2026-07-25-REGISTAR` / `ESTRUTURA` (`pending_validation`): bloco «Payload assinatura Registar Factura» e tabela de campos de `jwsDocumentSignature` incluem `documentTotals` (`taxPayable` / `netTotal` / `grossTotal`).
2. O exemplo compacto JWS no mesmo HTML de registo decodifica um payload **sem** `documentTotals`.
3. Não se aplica intersecção mínima entre exemplo e tabela.

## Mitigação

- Perfil `registar_document` = `blocked_conflict` (`SignRegistarDocumentBlocked`).

## Não fazer

- Não emitir `jwsDocumentSignature` de produção até fecho compliance/AGT.
