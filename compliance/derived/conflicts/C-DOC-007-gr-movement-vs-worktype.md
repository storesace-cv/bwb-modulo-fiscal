# C-DOC-007 — Homónimo L2 `GR` em `MovementType` **e** `WorkType` (L3 distintos)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed; residual DEC-REG-003) |
| Data | 2026-07-29 |
| Severidade | média (exportação dual-stack; risco de colapso canónico) |
| Relacionado | [DEC-PROD-001](../../../docs/06-delivery/open-decisions.md) · [C-DOC-005](C-DOC-005-insurer-invoice-vs-worktype.md) · [C-DOC-006](C-DOC-006-rc-payment-vs-purchase.md) |

## Factos

1. **L2** SAF-T `MovementType` sob L3 `MovementOfGoods` inclui **`GR`** (guia de remessa; `e9a938e1…`, `pending_validation`).
2. **L2** SAF-T `WorkType` sob L3 `WorkingDocuments` **também** inclui **`GR`** (mesmo literal; outro enum / estrutura).
3. **L4** FE: **sem** `documentType=GR` catalogado — ambos os seeds são **SAF-T-only** (`FE=∅`).
4. Dual membership no XSD **não** prova identidade entre guia de movimentação e guia em conferência (`DEC-PROD-001`).
5. Seed produto (ambos `off` / `pending_validation`):
   - `bwb.ao.movimentacao.gr` → `MovementType=GR` + `MovementOfGoods`
   - `bwb.ao.conferencia.gr` → `WorkType=GR` + `WorkingDocuments` (seed incremental a partir do XSD; **não** inventa L4 FE)

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Canónico | FE | Adaptador SAF-T | L3 | Política |
|---|---|---|---|---|
| `bwb.ao.movimentacao.gr` | `∅` | `MovementType=GR` | `MovementOfGoods` | **≠** `conferencia.gr`; **proibido** inventar FE L4 |
| `bwb.ao.conferencia.gr` | `∅` | `WorkType=GR` | `WorkingDocuments` | **≠** `movimentacao.gr`; **proibido** inventar FE L4 |

## Mitigação de engenharia (2026-07-29)

- `doctype.CheckCDOC007Invariants` + testes; `ParseSAFTTypeAdapter` reconhece `MovementType`.
- `saftao.ValidMovementType(GR)` ∧ `ValidWorkType(GR)`.

## Residual (não fechado)

- Qual L3 usar quando o domínio de produto fala em «guia de remessa» (`DEC-REG-003`).
- **Não** promover `AO-DOC-*`; seeds permanecem `off`.

## Não fazer

- Não colapsar os dois canónicos.
- Não inventar `documentType` FE `GR`.
- Não activar no OpenAPI slice sem fecho do residual.
- Não marcar C-DOC-007 como resolvido só porque os invariantes de seed passam.
