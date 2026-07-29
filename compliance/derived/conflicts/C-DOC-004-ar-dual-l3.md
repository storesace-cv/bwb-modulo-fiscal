# C-DOC-004 — Homónimo L4 `AR` em L2 `InvoiceType` **e** `PaymentType` (L3 distintos)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed; residual «grupo único por emissão») |
| Data | 2026-07-28 |
| Severidade | média-alta (exportação / dual-stack; risco de colapso de canónicos) |
| Relacionado | [C-DOC-003](C-DOC-003-fe-vs-saft-invoice-type.md) (facto 7) |

## Factos

1. **L4** FE `documentType` inclui **`AR`** (DE 683 + HTML HML; ver matriz de tipos).
2. **L2** SAF-T `InvoiceType` sob L3 `SalesInvoices` inclui **`AR`** (`e9a938e1…`, L2023–2065, `pending_validation`).
3. **L2** SAF-T `PaymentType` / `SAFTAOPaymentType` sob L3 `Payments` inclui **`AR`** (L2740–2754).
4. O mesmo literal `AR` **não** prova bijecção L4↔L2 nem «um único grupo L3 por emissão».
5. Seed produto mantém **dois** canónicos distintos (ambos `off` / `hipotese`):
   - `bwb.ao.vendas.ar` → `InvoiceType=AR` + `SalesInvoices`
   - `bwb.ao.pagamentos.ar` → `PaymentType=AR` + `Payments`
6. **3.º L3:** `PurchaseType=AR` sob `PurchaseInvoices` — ver [C-DOC-009](C-DOC-009-ar-purchase-third-l3.md).

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Canónico | FE | Adaptador SAF-T | L3 | Política |
|---|---|---|---|---|
| `bwb.ao.vendas.ar` | `AR` | `InvoiceType=AR` | `SalesInvoices` | **Proibido** fundir com pagamentos; **proibido** `PaymentType` neste canónico |
| `bwb.ao.pagamentos.ar` | `AR` | `PaymentType=AR` | `Payments` | **Proibido** fundir com vendas; **proibido** `InvoiceType` neste canónico |

## Mitigação de engenharia (2026-07-28)

- `doctype.CheckCDOC004Invariants` + testes: dual seed obrigatório; L3 alinhado ao enum L2; sem colapso de `CodigoCanonico`.
- `saftao.ValidInvoiceType(AR)` e `ValidPaymentType(AR)` ambos verdadeiros no XSD — **não** autoriza um único adaptador.
- Documentação: [DOCUMENT-TYPES-MATRIX-RM-REQ-001.md](../requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md) (nota «grupo único por emissão — aberto»).

## Residual (não fechado)

- Decisão normativa/produto de **qual** L3 usar por emissão quando FE envia `AR` (`DEC-REG-003` / dual-stack).
- **Não** promover `AO-DOC-*`; ambos os seeds permanecem `off`.

## Não fazer

- Não colapsar `vendas.ar` e `pagamentos.ar` num único canónico.
- Não colapsar com `compras.ar` (ver [C-DOC-009](C-DOC-009-ar-purchase-third-l3.md)).
- Não marcar C-DOC-004 como resolvido só porque os invariantes de seed passam.
- Não promover `AO-DOC-*` a partir desta mitigação.
- Não inventar bijecção L4=`AR` → um só L2.
- Não activar `AR` no OpenAPI slice sem fecho do residual.
- Não marcar C-DOC-004 como resolvido só porque os invariantes de seed passam.
