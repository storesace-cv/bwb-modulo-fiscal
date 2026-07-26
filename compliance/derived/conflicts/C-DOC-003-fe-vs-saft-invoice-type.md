# C-DOC-003 — códigos FE sem `InvoiceType` SAF-T (`FA`, `RC`, `RG`)

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-07-26 |
| Severidade | alta (exportação / dual-stack) |

## Factos

1. Enum FE (`documentType` em DE 683/25 + HTML HML) inclui **`FA`**, **`RC`**, **`RG`**.
2. Elemento SAF-T `InvoiceType` em `SAFTAO1.01_01.xsd` (`e9a938e1…`, `pending_validation`) **não** enumera `FA`, `RC` nem `RG`.
3. DP 71/25 reconhece Factura Adiantamento e Recibo como figuras legais (Art.3 g)/o)).

## Não fazer

- Não assumir que todo `documentType` FE exporta 1:1 para `InvoiceType`.
- Não inventar mapeamento SAF-T para `FA`/`RC`/`RG` sem fonte oficial ou decisão compliance.
