# C-DOC-006 — Homónimo L2 `RC` em `PaymentType` **e** `PurchaseType` (L3 distintos)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed; residual DEC-REG-003) |
| Data | 2026-07-28 |
| Severidade | média-alta (exportação dual-stack; risco de colapso canónico) |
| Relacionado | [C-DOC-003](C-DOC-003-fe-vs-saft-invoice-type.md) (RC ≠ `InvoiceType`) · [C-DOC-004](C-DOC-004-ar-dual-l3.md) |

## Factos

1. **L4** FE `documentType` inclui **`RC`** (recibo emitido; DE 683 + HTML HML).
2. **L2** SAF-T `PaymentType` sob L3 `Payments` inclui **`RC`** (`e9a938e1…`, `pending_validation`).
3. **L2** SAF-T `PurchaseType` sob L3 `PurchaseInvoices` **também** inclui **`RC`** (anotação XSD: regime de caixa) — **outro** enum / estrutura.
4. O mesmo literal `RC` **não** prova bijecção L4↔L2 nem identidade entre recibo FE e recibo de compras.
5. Seed produto (ambos `off`):
   - `bwb.ao.pagamentos.rc` → FE=`RC` + `PaymentType=RC` + `Payments` (`conflito`)
   - `bwb.ao.compras.rc` → FE=`∅` + `PurchaseType=RC` + `PurchaseInvoices` (`pending_validation`)

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Canónico | FE | Adaptador SAF-T | L3 | Política |
|---|---|---|---|---|
| `bwb.ao.pagamentos.rc` | `RC` | `PaymentType=RC` | `Payments` | **Proibido** `PurchaseType` / `InvoiceType`; **≠** `compras.rc` |
| `bwb.ao.compras.rc` | `∅` (SAF-T-only) | `PurchaseType=RC` | `PurchaseInvoices` | **Proibido** FE=`RC` neste canónico; **≠** `pagamentos.rc` |

## Mitigação de engenharia (2026-07-28)

- `doctype.CheckCDOC006Invariants` + testes; `ParseSAFTTypeAdapter` reconhece `PurchaseType`.
- `saftao.ValidPaymentType(RC)` ∧ `ValidPurchaseType(RC)`; `ValidInvoiceType(RC)` = false (C-DOC-003).

## Residual (não fechado)

- Semântica «regime de caixa» em compras vs recibo FE (`DEC-REG-003` / dual-stack).
- **Não** promover `AO-DOC-*`; seeds permanecem `off`.

## Não fazer

- Não colapsar `pagamentos.rc` e `compras.rc`.
- Não tratar L4=`RC` como `PurchaseType=RC` nem como `InvoiceType`.
- Não activar no OpenAPI slice sem fecho do residual.
- Não marcar C-DOC-006 como resolvido só porque os invariantes de seed passam.
