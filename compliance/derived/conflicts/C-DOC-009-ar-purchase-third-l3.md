# C-DOC-009 — Homónimo L2 `AR` também em `PurchaseType` (3.º L3; extensão de C-DOC-004)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed; residual DEC-REG-003 / «grupo único») |
| Data | 2026-07-29 |
| Severidade | média-alta (três L3 distintos para o mesmo literal) |
| Relacionado | [C-DOC-004](C-DOC-004-ar-dual-l3.md) · [C-DOC-008](C-DOC-008-ft-nc-invoice-vs-purchase.md) |

## Factos

1. **C-DOC-004** já documenta `AR` em `InvoiceType` (`SalesInvoices`) **e** `PaymentType` (`Payments`) com FE=`AR`.
2. **L2** SAF-T `PurchaseType` sob L3 `PurchaseInvoices` **também** inclui **`AR`** (`e9a938e1…`, `pending_validation`).
3. Triple membership **não** prova bijecção L4↔L2 nem «um único grupo L3 por emissão».
4. Seed produto (todos `off` no lado compras; vendas/pagamentos `off` por C-DOC-004):
   - `bwb.ao.vendas.ar` → FE=`AR` + `InvoiceType=AR` + `SalesInvoices`
   - `bwb.ao.pagamentos.ar` → FE=`AR` + `PaymentType=AR` + `Payments`
   - `bwb.ao.compras.ar` → FE=`∅` + `PurchaseType=AR` + `PurchaseInvoices` (seed incremental XSD)

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Canónico | FE | Adaptador | L3 | Política |
|---|---|---|---|---|
| `bwb.ao.vendas.ar` | `AR` | `InvoiceType=AR` | `SalesInvoices` | C-DOC-004; **≠** pagamentos/compras |
| `bwb.ao.pagamentos.ar` | `AR` | `PaymentType=AR` | `Payments` | C-DOC-004; **≠** vendas/compras |
| `bwb.ao.compras.ar` | `∅` | `PurchaseType=AR` | `PurchaseInvoices` | SAF-T-only; **off**; **≠** vendas/pagamentos |

## Mitigação de engenharia (2026-07-29)

- `doctype.CheckCDOC009Invariants` + testes (inclui regressão C-DOC-004).
- `saftao.ValidPurchaseType(AR)`.

## Residual (não fechado)

- Escolha de L3 quando FE envia `AR` (`DEC-REG-003` / dual-stack / «grupo único»).
- **Não** promover `AO-DOC-*`.

## Não fazer

- Não colapsar os três canónicos.
- Não inventar FE L4 no lado compras.
- Não marcar C-DOC-009 (nem C-DOC-004) como resolvido só porque os invariantes passam.
