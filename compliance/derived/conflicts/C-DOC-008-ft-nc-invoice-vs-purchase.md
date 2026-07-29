# C-DOC-008 — Homónimos L2 `FT`/`NC` em `InvoiceType` **e** `PurchaseType` (L3 distintos)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed; residual DEC-REG-003 alargamento compras) |
| Data | 2026-07-29 |
| Severidade | alta (slice activo `FT`/`NC` vendas vs compras SAF-T) |
| Relacionado | [C-DOC-006](C-DOC-006-rc-payment-vs-purchase.md) · [DEC-REG-003](../../../docs/06-delivery/open-decisions.md) · [DEC-PROD-001](../../../docs/06-delivery/open-decisions.md) |

## Factos

1. **L4** FE `documentType` inclui **`FT`** e **`NC`** (DE 683 + HTML HML).
2. **L2** SAF-T `InvoiceType` sob L3 `SalesInvoices` inclui **`FT`** e **`NC`**.
3. **L2** SAF-T `PurchaseType` sob L3 `PurchaseInvoices` **também** inclui **`FT`** e **`NC`** — **outro** enum / estrutura (`≠` vendas).
4. Dual membership **não** prova bijecção L4↔L2 nem identidade venda/compra.
5. Seed produto:
   - `bwb.ao.vendas.ft` / `bwb.ao.vendas.nc` → FE + `InvoiceType` + `SalesInvoices` — **`activo=on`** (`DEC-REG-003` defaults)
   - `bwb.ao.compras.ft` / `bwb.ao.compras.nc` → FE=`∅` + `PurchaseType` + `PurchaseInvoices` — **`off`** / `pending_validation`

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Canónico | FE | Adaptador SAF-T | L3 | Política |
|---|---|---|---|---|
| `bwb.ao.vendas.ft` / `.nc` | `FT`/`NC` | `InvoiceType=*` | `SalesInvoices` | Defaults slice `on`; **≠** compras |
| `bwb.ao.compras.ft` / `.nc` | `∅` | `PurchaseType=*` | `PurchaseInvoices` | SAF-T-only; **off**; **≠** vendas |

## Mitigação de engenharia (2026-07-29)

- `doctype.CheckCDOC008Invariants` + testes; seed incremental `compras.nc` a partir do XSD.
- `ValidInvoiceType` ∧ `ValidPurchaseType` para `FT`/`NC`.

## Residual (não fechado)

- Activação / política de compras no MVP (`DEC-REG-003`); outros literais partilhados (FR, GF, …) quando seed existir.
- **Não** promover `AO-DOC-*`.

## Não fazer

- Não colapsar canónicos vendas/compras.
- Não inventar FE L4 no lado compras.
- Não desactivar `vendas.ft`/`vendas.nc` só por existirem homónimos PurchaseType.
- Não marcar C-DOC-008 como resolvido só porque os invariantes de seed passam.
