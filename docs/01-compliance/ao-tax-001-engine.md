# AO-TAX-001 — motor de cálculo IVA (RM-ENG-003)

**Estado:** residual de engenharia fechado por testes (RM-ENG-003). **≠** `confirmed_normative`.

**Fonte provisória:** [`AO-LEG-DE-683-25-2025`](../../compliance/catalog/sources.yaml) — Citação G `taxType`/`taxCode` @19171–19173; Tabelas 2–6 @19212–19227 (`pending_validation`). Matriz: [`PROVISIONAL-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md) — `AO-TAX-001` permanece **`partial`**.

**Não é:** catálogo oficial completo de isenções (IS/IEC); descontos comerciais; retenções na fonte; arredondamento documental AGT confirmado; homologação AGT.

## Âmbito MVP (`internal/taxao`)

| Capacidade | Comportamento |
|---|---|
| Catálogo `tax_code` | `NOR` (14%), `RED` (5%), `INT` (7%), `ISE` (0%) — alinhado com fixtures SAFT sintéticos; taxas **provisórias** |
| Linha | `net = round_half_up(qty_scaled × unit_cents / 10⁴)` |
| IVA linha | `tax = round_half_up(net × rate_bp / 10⁴)`; sem `float` |
| Documento | Soma determinística de net/tax/gross por linha |
| Desconhecido | Fail-closed (`ErrUnknownTaxCode`) — ex.: `OUT`, `NS`, `NA` válidos em XSD SAFT mas **não** no MVP Seal |

## Integração

- `prepareSealRequest` (`internal/persistence/seal.go`) valida cada linha via `taxao.CalculateLine` antes de selar.
- Totais **não** persistidos em BD (schema actual); cálculo disponível para testes/export futuro.

## Evidência de teste

- `AO_TAX_001_*` em [`internal/taxao/calc_test.go`](../../internal/taxao/calc_test.go)
- Vetor SAFT sintético: 1 × 100,00 NOR → net 100,00 / tax 14,00 / gross 114,00
- Rejeição `tax_code` desconhecido na validação Seal (prepare)

## Fora de âmbito (bloqueado até fonte/decisão)

- IS / IEC / `taxExemptionCode` (Tabelas 3–4)
- Descontos e retenções (DEC-REG-003)
- Promoção de taxas a norma confirmada sem revisão OCR `reviewed` + confronto PDF
- Aceitação ou certificado AGT
