# C-DOC-005 — Homónimos segurador L4/`InvoiceType` **e** `WorkType` (`RP`/`RE`/`CS`/`LD`/`RA`)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed; residual DEC-REG-003) |
| Data | 2026-07-28 |
| Severidade | média (exportação dual-stack; risco de colapso L3) |
| Relacionado | [C-DOC-003](C-DOC-003-fe-vs-saft-invoice-type.md) · [C-DOC-004](C-DOC-004-ar-dual-l3.md) |

## Factos

1. **L4** FE `documentType` inclui **`RP`**, **`RE`**, **`CS`**, **`LD`**, **`RA`** (DE 683 + HTML HML; sector segurador — **sem** Art.3 DP 71).
2. **L2** SAF-T `InvoiceType` sob L3 `SalesInvoices` inclui os mesmos literais (`e9a938e1…`, `pending_validation`).
3. **L2** SAF-T `WorkType` sob L3 `WorkingDocuments` **também** inclui `RP|RE|CS|LD|RA` (XSD WorkingDocuments).
4. Dual membership no XSD **não** prova bijecção L4↔L2 nem «um único grupo L3 por emissão».
5. Seed produto actual (todos `off` / `hipotese`):
   - `bwb.ao.vendas.{rp,re,cs,ld,ra}` → `InvoiceType=*` + `SalesInvoices`
   - **sem** canónicos `conferencia.*` FE→`WorkType` para estes códigos (não inventar)

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Canónico | FE | Adaptador SAF-T | L3 | Política |
|---|---|---|---|---|
| `bwb.ao.vendas.rp` … `ra` | `RP`…`RA` | `InvoiceType=*` | `SalesInvoices` | **Proibido** `WorkType` neste seed FE até `DEC-REG-003` |
| (futuro SAF-T-only / dual) | `∅` ou decisão | `WorkType=*` | `WorkingDocuments` | Canónico **distinto**; **não** colapsar com `vendas.*` |

## Mitigação de engenharia (2026-07-28)

- `doctype.CheckCDOC005Invariants` + testes: seeds `vendas.*` obrigatórios; `InvoiceType`+`SalesInvoices`+`off`; rejeita FE→`WorkType` para estes códigos.
- `ParseSAFTTypeAdapter` reconhece `WorkType`.
- `saftao.ValidInvoiceType` ∧ `ValidWorkType` ambos aceitam os literais — **não** autoriza um único adaptador.

## Residual (não fechado)

- Decisão normativa/produto de quando FE `RP`…`RA` mapeia para `WorkingDocuments` vs `SalesInvoices` (`DEC-REG-003`).
- **Não** promover `AO-DOC-*`; seeds permanecem `off`.

## Não fazer

- Não inventar `bwb.ao.conferencia.rp` (etc.) só para «fechar» o XSD.
- Não activar códigos segurador no OpenAPI slice sem fecho do residual.
- Não marcar C-DOC-005 como resolvido só porque os invariantes de seed passam.
