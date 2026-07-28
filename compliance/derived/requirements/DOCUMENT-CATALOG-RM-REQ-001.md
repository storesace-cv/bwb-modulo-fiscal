# Catálogo documental canónico (esquema mínimo)

**Estado:** rascunho rastreável — **não** confirma `AO-DOC-001` / `AO-DOC-002`.
**Data:** 2026-07-26
**Decisões:** `DEC-PROD-015` (esquema) · `DEC-PROD-001`–`014` · `DEC-REG-003` (defaults slice) · inventário em [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md)
**Regra:** células sem fonte = `pending` / `hipótese`; **não** inventar. PDF original prevalece.

## Esquema obrigatório (`DEC-PROD-015`)

| Campo | Obrigatório | Valores / notas |
|---|---|---|
| `grupo` | sim | `vendas` \| `movimentacao` \| `conferencia` \| `pagamentos` \| `compras` |
| `codigo_canonico` | sim | ID produto `bwb.ao.<grupo>.<slug>` (`DEC-PROD-007`) — **≠** código AGT |
| `designacao` | sim | Rótulo humano |
| `codigos_canal` | sim | `FE:` L4; `SAFT:` L2 (ou `∅`) |
| `estrutura_saft` | sim | L3 XSD ou `∅` (FE-only) |
| `elegibilidade` | sim | `SAF-T` \| `FE` \| `ambos` |
| `natureza_juridica` | sim | Ref. L1 / `n/a` / `pending` |
| `restricao_sectorial` | sim | `nenhuma` \| sector \| `pending` |
| `serie_necessaria` | sim | `sim` \| `nao` \| `condicional` \| `pending` |
| `requisitos` | sim | Citas / `pending` |
| `regras_rectificacao_anulacao` | sim | Citas / `pending` / `n/a` |
| `estado_normativo` | sim | `hipotese` \| `conflito` \| `pending_validation` \| `reviewed_aux` |
| `activo` | sim | `on` \| `off` \| `pending_dec_reg_003` |

Legenda `activo`: `on`/`off` = defaults do slice (`DEC-REG-003`); `pending_dec_reg_003` = histórico/legado (não usar em novas linhas).

## Entradas (seed — completar progressivamente)

Homónimos entre grupos = **linhas distintas** (canónicos distintos).

| grupo | codigo_canonico | designacao | codigos_canal | estrutura_saft | elegibilidade | natureza_juridica | restricao_sectorial | serie_necessaria | requisitos | regras_rectificacao_anulacao | estado_normativo | activo |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| vendas | `bwb.ao.vendas.ft` | Factura | FE:`FT`; SAFT:`InvoiceType=FT` | `SalesInvoices` | ambos | DP71 Art.3 f) @11903 | nenhuma | pending | Art.10 @11908–11909; FE lines @19169–19170 | pending (NC/rectif. Art.8) | hipotese | on |
| vendas | `bwb.ao.vendas.fr` | Factura-Recibo | FE:`FR`; SAFT:`InvoiceType=FR` | `SalesInvoices` | ambos | Art.3 k) @11903 | nenhuma | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.fg` | Factura Global | FE:`FG`; SAFT:`InvoiceType=FG` | `SalesInvoices` | ambos | Art.3 j) @11903 | nenhuma | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.gf` | Factura Genérica | FE:`GF` (HTML/XSD; ausente DE683 p.7 visual); SAFT:`InvoiceType=GF` | `SalesInvoices` | ambos | Art.3 i) @11903 | nenhuma | pending | C-DOC-001 documentado_divergencia | pending | conflito | off |
| vendas | `bwb.ao.vendas.fa` | Factura Adiantamento | FE:`FA`; SAFT:`∅` | `∅` | FE | Art.3 g) @11903 | nenhuma | pending | C-DOC-003; FE-only | Art.8 n.º10 via NC @11908 | conflito | off |
| vendas | `bwb.ao.vendas.ac` | Aviso de Cobrança | FE:`AC`; SAFT:`InvoiceType=AC` | `SalesInvoices` | ambos | Art.3 d) @11903 | nenhuma | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.ar` | Aviso Cobrança/Recibo (vendas) | FE:`AR`; SAFT:`InvoiceType=AR` | `SalesInvoices` | ambos | Art.3 d)+Art.6 n.º2 b) | nenhuma | pending | C-DOC-004 dual L3; ≠ `pagamentos.ar` | pending | hipotese | off |
| vendas | `bwb.ao.vendas.tv` | Talão de Venda | FE:`TV`; SAFT:`InvoiceType=TV` | `SalesInvoices` | ambos | Art.3 p) @11904 | nenhuma | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.nc` | Nota de Crédito | FE:`NC`; SAFT:`InvoiceType=NC` | `SalesInvoices` | ambos | Art.3 l) @11904 | nenhuma | pending | `referenceInfo` FE @19170; `References` XSD | Art.8 n.º4–5 @11907; Art.10 n.º5 | hipotese | on |
| vendas | `bwb.ao.vendas.nd` | Nota de Débito | FE:`ND`; SAFT:`InvoiceType=ND` | `SalesInvoices` | ambos | Art.3 m); Art.4 n.º9 ≠ factura | nenhuma | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.af` | Autofacturação | FE:`AF`; SAFT:`InvoiceType=AF` | `SalesInvoices` | ambos | Art.3 c)+k) | nenhuma | pending | Art.10 n.º3 série distinta | pending | hipotese | off |
| vendas | `bwb.ao.vendas.rp` | Prémio / Recibo de Prémio | FE:`RP`; SAFT:`InvoiceType=RP` | `SalesInvoices` | ambos | n/a Art.3 | segurador | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.re` | Estorno / Recibo de Estorno | FE:`RE`; SAFT:`InvoiceType=RE` | `SalesInvoices` | ambos | n/a Art.3 | segurador | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.cs` | Imputação Co-seguradoras | FE:`CS`; SAFT:`InvoiceType=CS` | `SalesInvoices` | ambos | n/a Art.3 | segurador | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.ld` | Imputação Co-seguradora Líder | FE:`LD`; SAFT:`InvoiceType=LD` | `SalesInvoices` | ambos | n/a Art.3 | segurador | pending | pending | pending | hipotese | off |
| vendas | `bwb.ao.vendas.ra` | Resseguro Aceite | FE:`RA`; SAFT:`InvoiceType=RA` | `SalesInvoices` | ambos | n/a Art.3 | segurador | pending | pending | pending | hipotese | off |
| pagamentos | `bwb.ao.pagamentos.rc` | Recibo Emitido | FE:`RC`; SAFT:`PaymentType=RC` | `Payments` | ambos | Art.3 o) @11904 | nenhuma | pending | FE: lines vazio + paymentReceipt @19169–19170; ≠ InvoiceType | pending | conflito | off |
| pagamentos | `bwb.ao.pagamentos.rg` | Outros Recibos Emitidos | FE:`RG`; SAFT:`PaymentType=RG` | `Payments` | ambos | Art.3 o) (rótulo FE C-DOC-002) | nenhuma | pending | C-DOC-002/003 | pending | conflito | off |
| pagamentos | `bwb.ao.pagamentos.ar` | Aviso Cobrança/Recibo (pagamentos) | FE:`AR`; SAFT:`PaymentType=AR` | `Payments` | ambos | Art.3 d)+Art.6 n.º2 b) | nenhuma | pending | C-DOC-004 dual L3; ≠ `vendas.ar` | pending | hipotese | off |
| movimentacao | `bwb.ao.movimentacao.gr` | Guia de Remessa | FE:`∅`; SAFT:`MovementType=GR` | `MovementOfGoods` | SAF-T | Art.4 n.º9 (≠ factura) | nenhuma | pending | pending | pending | pending_validation | off |
| movimentacao | `bwb.ao.movimentacao.gt` | Guia de Transporte | FE:`∅`; SAFT:`MovementType=GT` | `MovementOfGoods` | SAF-T | Art.4 n.º9 | nenhuma | pending | pending | pending | pending_validation | off |
| movimentacao | `bwb.ao.movimentacao.ga` | Guia Activos Fixos | FE:`∅`; SAFT:`MovementType=GA` | `MovementOfGoods` | SAF-T | pending | nenhuma | pending | pending | pending | pending_validation | off |
| movimentacao | `bwb.ao.movimentacao.gd` | Guia/Nota Devolução | FE:`∅`; SAFT:`MovementType=GD` | `MovementOfGoods` | SAF-T | pending | nenhuma | pending | pending | pending | pending_validation | off |
| conferencia | `bwb.ao.conferencia.pf` | Pró-forma | FE:`∅`; SAFT:`WorkType=PF` | `WorkingDocuments` | SAF-T | Art.4 n.º9 ≠ factura | nenhuma | pending | pending | n/a (não factura) | pending_validation | off |
| conferencia | `bwb.ao.conferencia.or` | Orçamento | FE:`∅`; SAFT:`WorkType=OR` | `WorkingDocuments` | SAF-T | Art.4 n.º9 | nenhuma | pending | pending | n/a | pending_validation | off |
| conferencia | `bwb.ao.conferencia.ne` | Nota de Encomenda | FE:`∅`; SAFT:`WorkType=NE` | `WorkingDocuments` | SAF-T | Art.4 n.º9 | nenhuma | pending | pending | n/a | pending_validation | off |
| conferencia | `bwb.ao.conferencia.dc` | Documento de Conferência | FE:`∅`; SAFT:`WorkType=DC` | `WorkingDocuments` | SAF-T | pending | nenhuma | pending | pending | pending | pending_validation | off |
| compras | `bwb.ao.compras.ft` | Factura (compras) | FE:`∅`; SAFT:`PurchaseType=FT` | `PurchaseInvoices` | SAF-T | pending | nenhuma | pending | ≠ `InvoiceType` vendas | pending | pending_validation | off |
| compras | `bwb.ao.compras.nl` | Nota de Liquidação | FE:`∅`; SAFT:`PurchaseType=NL` | `PurchaseInvoices` | SAF-T | pending | nenhuma | pending | pending | pending | pending_validation | off |
| compras | `bwb.ao.compras.rc` | Recibo (regime caixa compras) | FE:`∅`; SAFT:`PurchaseType=RC` | `PurchaseInvoices` | SAF-T | pending | regime caixa | pending | anotação XSD; ≠ PaymentType FE | pending | pending_validation | off |

### WorkType / PurchaseType restantes

Códigos L2 ainda não listados acima (ex. `CM`, `CC`, `FO`, `OU`, `GC`, `PP`, demais `PurchaseType`) entram no mesmo esquema; seed incremental **sem** inventar designações além do XSD.

## Lacunas explícitas

1. `serie_necessaria` / muitos `requisitos` / `regras_rectificacao_anulacao` = `pending` até citação por tipo.
2. Defaults do slice (`DEC-REG-003`): só `bwb.ao.vendas.ft` e `bwb.ao.vendas.nc` com `activo=on`; restantes seed `off`.
3. C-DOC-001/002/003 impedem promover linhas afectadas a estado normativo fechado.
4. POS usa `codigo_canonico` via mapping (`DEC-PROD-007`), nunca L4/L2 crus sem mapping.

## Referências

- Inventário por camadas: [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md)
- Decisões: [`docs/06-delivery/open-decisions.md`](../../../docs/06-delivery/open-decisions.md) `DEC-PROD-001`–`015`, `DEC-REG-003`
- XSD: [`compliance/saft-ao/schemas/SAFTAO1.01_01.xsd`](../../saft-ao/schemas/SAFTAO1.01_01.xsd) (`e9a938e1…`, `pending_validation`)
