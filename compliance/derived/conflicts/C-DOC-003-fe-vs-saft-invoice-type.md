# C-DOC-003 — L4 `documentType` FE ≠ L2 `InvoiceType` ≠ L3 estrutura (`FA`, `RC`, `RG`)

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-07-26 |
| Severidade | alta (exportação / dual-stack) |

## Factos (quatro camadas)

1. **L4** FE `documentType` (DE 683/25 + HTML HML) inclui **`FA`**, **`RC`**, **`RG`**.
2. **L2** SAF-T `InvoiceType` (`SalesInvoices`, L2023–2065, `e9a938e1…`, `pending_validation`) **não** enumera `FA`, `RC` nem `RG`.
3. **L3** SAF-T: recibos tipicamente em `Payments/Payment`, **não** em `SalesInvoices/Invoice`.
4. **L2** `SAFTAOPaymentType` (**L2740–2754**) sob L3 `Payments` enumera **`RC`**, **`RG`**, **`AR`** — **outro** enum e **outra** estrutura que `InvoiceType`.
5. **L1** DP 71/25 reconhece Factura Adiantamento e Recibo (Art.3 g)/o)) — rótulos legais, **não** códigos.
6. `FA` ausente de `InvoiceType` e de `SAFTAOPaymentType`.

## Não fazer

- Não tratar L4=`RC` como L2=`InvoiceType` nem como prova de mapeamento.
- Não tratar L2=`PaymentType=RC` como fecho de L4→SAF-T (são camadas distintas).
- Não inventar estrutura L3 para `FA` sem fonte oficial / `DEC-REG-003`.
- Não confundir L1 «Recibo» com L4 `RG`/`RC` nem com L2 `PaymentType`.
