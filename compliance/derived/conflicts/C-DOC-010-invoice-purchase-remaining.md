# C-DOC-010 — Homónimos L2 restantes em `InvoiceType` **e** `PurchaseType` (`FR`/`GF`/`FG`/`AC`/`AF`/`TV`)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed; residual DEC-REG-003 alargamento compras) |
| Data | 2026-07-29 |
| Severidade | média-alta (completar cobertura Invoice∩Purchase além de C-DOC-008/009) |
| Relacionado | [C-DOC-008](C-DOC-008-ft-nc-invoice-vs-purchase.md) · [C-DOC-001](C-DOC-001-fe-gf-ocr-gap.md) (GF FE) · [DEC-REG-003](../../../docs/06-delivery/open-decisions.md) |

## Factos

1. **L2** SAF-T `InvoiceType` e `PurchaseType` partilham literais **`FR`**, **`GF`**, **`FG`**, **`AC`**, **`AF`**, **`TV`** (além de `FT`/`NC` em C-DOC-008 e `AR` em C-DOC-009).
2. Dual membership **não** prova bijecção L4↔L2 nem identidade venda/compra.
3. Seed produto (todos `off` neste lote; ≠ DEC-REG-003 defaults `FT`/`NC`):
   - `bwb.ao.vendas.{fr,gf,fg,ac,af,tv}` → FE + `InvoiceType` + `SalesInvoices`
   - `bwb.ao.compras.{fr,gf,fg,ac,af,tv}` → FE=`∅` + `PurchaseType` + `PurchaseInvoices`

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Canónico | FE | Adaptador SAF-T | L3 | Política |
|---|---|---|---|---|
| `bwb.ao.vendas.*` (códigos acima) | L4 = L2 | `InvoiceType=*` | `SalesInvoices` | `off`; **≠** compras |
| `bwb.ao.compras.*` (códigos acima) | `∅` | `PurchaseType=*` | `PurchaseInvoices` | SAF-T-only; `off`; **≠** vendas |

## Mitigação de engenharia (2026-07-29)

- `doctype.CheckCDOC010Invariants` + testes; seeds incrementais compras a partir do XSD.
- `ValidInvoiceType` ∧ `ValidPurchaseType` para cada literal.
- GF vendas mantém `conflito` (C-DOC-001); compras GF = `pending_validation` (sem inventar FE).

## Residual (não fechado)

- Activação / política de compras e alargamento do slice (`DEC-REG-003`).
- Fecho C-DOC-001 para `GF` no canal FE.
- **Não** promover `AO-DOC-*`.

## Não fazer

- Não colapsar canónicos vendas/compras.
- Não inventar FE L4 no lado compras.
- Não activar estes literais no OpenAPI slice sem DEC-REG-003.
- Não marcar C-DOC-010 como resolvido só porque os invariantes de seed passam.
