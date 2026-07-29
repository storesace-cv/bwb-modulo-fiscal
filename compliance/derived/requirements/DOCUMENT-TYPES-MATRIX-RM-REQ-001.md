# Matriz normativa de tipos documentais (provisória)

**Estado:** rascunho rastreável — **não** confirma `AO-DOC-001` / `AO-DOC-002`; `DEC-REG-003` = defaults slice FT+NC (faseamento, não perímetro).
**Data:** 2026-07-26
**Modelo:** todos os legalmente aplicáveis com canal SAF-T e/ou FE (`DEC-PROD-014`); implementação faseada sem truncar.
**Defaults slice (`DEC-REG-003`):** OpenAPI `invoice`/`credit_note` → `bwb.ao.vendas.ft`/`bwb.ao.vendas.nc` (`activo=on`); resto do seed `off`. Sem promover AO-DOC-*.
**Catálogo L3:** exactamente **5** grupos (`DEC-PROD-001`).
**Inclusão:** só tipos com canal SAF-T e/ou FE (`DEC-PROD-002`); excluir o que não sirva nenhum.
**Config POS:** activar/desactivar grupo inteiro e tipos dentro do grupo (`DEC-PROD-003`) — produto; **não** confirma AO-*.
**Adesão FE:** estado no contribuinte/NIF (`DEC-PROD-004`: `not_enrolled|pending|active|suspended`); séries no estabelecimento.
**Disponibilidade:** conjunção `DEC-PROD-005` (canónico + activo + regime/NIF + FE + série + ambiente + sector).
**Routing:** `DEC-PROD-006` — sem FE activa só SAF-T; SAF-T-only ∉ endpoint FE; FE-only só FE.
**Canónico:** um conceito interno + adaptadores por canal; POS só com mapping (`DEC-PROD-007`).
**Autoridade:** módulo emite/numera; N POS só API (`DEC-PROD-008`).
**Estados:** `sealed_locally`…`accepted`/`rejected` — sem aceitação AGT antes de `accepted` (`DEC-PROD-009`).
**Offline:** outbox/reenvio/idempotência; sem emissão offline certificada (`DEC-PROD-010`).
**Edge:** um escritor fiscal; sem multi-instância na mesma série sem coordenação (`DEC-PROD-011`).
**Chaves:** por contribuinte; non-exportable quando possível; nunca POS; rotação auditada; definitivo AGT (`DEC-PROD-012`).
**Auditoria:** append-only; retenção final = norma consolidada (`DEC-PROD-013`).
**Perímetro modelo:** completo por canal (`DEC-PROD-014`); slice ≠ modelo.
**Catálogo canónico:** esquema mínimo + seed em [`DOCUMENT-CATALOG-RM-REQ-001.md`](DOCUMENT-CATALOG-RM-REQ-001.md) (`DEC-PROD-015`).
**Regra:** PDF original prevalece; OCR/`reviewed` e HTML FE são auxiliares. Sem inventar tipos em falta.

## Camadas — não confundir

Estas quatro camadas são **ortogonais**. Homónimos (ex.: «FT», «RC», «Recibo») **não** implicam equivalência.

| # | Camada | O que é | Onde vive | Exemplos | **Não** é |
|---|---|---|---|---|---|
| L1 | **Documento legal** | Figura jurídica / obrigação de emissão | DP 71/25 (Art.3–4, 10, …); DE 74/19 (classes) | «Factura», «Nota de crédito», «Recibo» | Código JSON FE; enum XSD |
| L2 | **Tipo SAF-T** | Enumeração de tipo **dentro** de um bloco XSD | `InvoiceType`, `SAFTAOPaymentType`, `PurchaseType`, … | `InvoiceType=FT`; `PaymentType=RC` | Estrutura XML; `documentType` FE |
| L3 | **Estrutura SAF-T** | Bloco / tabela do ficheiro `AuditFile` | `SalesInvoices`, `Payments`, `MovementOfGoods`, `WorkingDocuments`, … | Recibo em `Payments/Payment`; venda em `SalesInvoices/Invoice` | Tipo legal; campo FE |
| L4 | **`documentType` FE** | Campo JSON da API de facturação electrónica | DE 683 Anexo I + HTML HML (`registarFactura` / séries) | `documentType=FA\|FT\|RC\|…` | `InvoiceType`; estrutura SAF-T |

**Regras fail-closed**

1. L1 **não** enumera códigos de dois caracteres — só rótulos legais.
2. L4 (`documentType`) **não** é L2 (`InvoiceType` / `PaymentType`).
3. L2 **depende** de L3: o mesmo código pode existir em enums diferentes (`AR` em `InvoiceType` e em `SAFTAOPaymentType`) sem ser o mesmo conceito operacional.
4. Cruzamentos L1↔L4↔L2↔L3 são **hipóteses** até revisão compliance — ver C-DOC-001/002/003; defaults de activação = `DEC-REG-003` (não confirma AO-*).
5. API POS usa **canónico** / código próprio mapeado (`DEC-PROD-007`) — **não** L4/L2 crus sem mapping.

## Fontes cruzadas

| Camada | source_id | Ficheiro / artefacto | SHA256 (prefixo) | Estado | Intervalo citável |
|---|---|---|---|---|---|
| Regime jurídico | `AO-LEG-DP-71-25-2025` | privado `derivatives/.../v1/text.md` ← original `Diario_Republica_I_52_2025_03_20.pdf` | `4931fd3c…` | OCR `reviewed` | Gazeta **11902–11920** (PDF p.2–20); **não** citar p.21 / DE 372/25 |
| Spec FE (diploma) | `AO-LEG-DE-683-25-2025` | privado `derivatives/.../v2/text.md` ← original v2 | `b01e4581…` | OCR `reviewed` | Gazeta **19164–19227** (PDF p.2–65); **não** citar p.66 / Aviso 4/25 |
| Spec FE (HTML HML) | `AO-FE-SNAP-HML-2026-07-25-REGISTAR` | privado `originals/.../servico_registar.html` | (catálogo) | `pending_validation` | Secção campo `documentType` |
| Spec FE (estrutura) | `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | privado `originals/.../estrutura.html` | `50ba18e0…` | `pending_validation` | Campo `documentType` no payload JWS |
| SAF-T XSD | `AO-SAFT-XSD-1.01_01` | [`compliance/saft-ao/schemas/SAFTAO1.01_01.xsd`](../../saft-ao/schemas/SAFTAO1.01_01.xsd) | `e9a938e1…` | `pending_validation` | elemento `InvoiceType` |
| Validação software | `AO-LEG-DE-74-19-2019` | privado OCR v1 | `5b63c80e…` | OCR `reviewed` | Classes de documentos no anexo (não enum FE) |
| Rect. Modelo 08 | `AO-LEG-RECT-10-19-2019` | privado OCR v2 | `b3db14e2…` | OCR `reviewed` | Gazeta **1948–1949** — **não** enumera tipos de emissão |
| Contrato BWB | OpenAPI | [`specs/openapi/openapi.yaml`](../../../specs/openapi/openapi.yaml) | — | draft produto | `document_type`: `invoice` \| `credit_note` |

`private_commit` alinhado: `c8a4e6e8ec2772ff50ad1c8762842b983edbbbfd` (legislação).

## A. Camada L1 — documento legal (DP 71/25)

**Não** são códigos FE nem enums SAF-T.

Citação base: ART. **3.º** (Definições) · PDF p.**3–4** · gazeta **11903–11904** · `AO-LEG-DP-71-25-2025`.

| ID legal (rótulo) | Alínea OCR | PDF | Gazeta | Natureza (OCR auxiliar) | Notas / artigos relacionados |
|---|---|---|---|---|---|
| Factura | Art.3 f) | p.3 | 11903 | Factura (documento comercial oneroso) | Art.4 (emissão); Art.10 (requisitos) |
| Factura Adiantamento | Art.3 g) | p.3 | 11903 | Adiantamento / antecipação | Art.4 n.º1; Art.8 n.º10 (regularização) |
| Factura Electrónica | Art.3 h) | p.3 | 11903 | Forma electrónica / software | Modalidade, não tipo autónomo de operação |
| Factura Genérica | Art.3 i) | p.3 | 11903 | Instituição financeira, mensal | Art.6 n.º2 c) (dispensa recibo) |
| Factura Global | Art.3 j) | p.3 | 11903 | Periodicidade máx. mensal | Art.8 n.º2–3 |
| Factura-Recibo | Art.3 k) | p.3 | 11903 | Factura + pagamento total | Art.6 n.º2 a) |
| Nota de Crédito | Art.3 l) | p.4 | 11904 | Anulação / rectificação | Art.8 n.º4–6; Art.10 n.º5 |
| Nota de Débito | Art.3 m) | p.4 | 11904 | Débito sem obrigação de factura; sem liquidação imposto | Art.4 n.º3, n.º7 |
| Recibo | Art.3 o) | p.4 | 11904 | Pagamento parcial/total | Art.6 |
| Talão de Venda ou Prestação de Serviço | Art.3 p) | p.4 | 11904 | Transmissão + pagamento | Art.5 n.º2–3 |
| Aviso de Cobrança | Art.3 d) | p.3 | 11903 | Seguradoras | Art.6 n.º2 b) (aviso-recibo) |
| Auto-Facturação | Art.3 c) | p.3 | 11903 | Factura-recibo pelo adquirente | Art.10 n.º3 |
| Documento Fiscalmente Relevante | Art.3 e) | p.3 | 11903 | Inclui recibos, ND, NC, despacho, talões | Género residual |

### A.1 Explicitamente **não** são factura (DP 71/25)

Citação: ART. **4.º** n.º**9** · PDF p.**5** · gazeta **11905**.

| Documento (OCR) | PDF | Gazeta | Implicação para domínio BWB |
|---|---|---|---|
| Bordereaux bancários | p.5 | 11905 | Fora de `document_type` fiscal de emissão |
| Factura pró-forma | p.5 | 11905 | Não factura |
| Guia de remessa ou transporte | p.5 | 11905 | Suporte / circulação — ver também DE 74/19 |
| Nota de crédito | p.5 | 11905 | **Não** é factura; é documento rectificativo |
| Nota de débito | p.5 | 11905 | **Não** é factura |
| Nota de encomenda / preço / pagamento / remessa | p.5 | 11905 | Não factura |
| Requisição de fundos | p.5 | 11905 | Não factura |
| Orçamento de venda ou serviço | p.5 | 11905 | Não factura |
| Qualquer outro não regulado | p.5 | 11905 | Fail-closed: não inventar |

## B. Camada L4 — `documentType` da API FE (DE 683/25 + HTML HML)

Campo JSON FE: **`documentType`** (string, minLength 2, maxLength 2), obrigatório no registo / séries.
Isto é a **API FE**, não o XSD SAF-T e não o rótulo legal do DP 71.

| Código L4 | Significado (fonte FE) | DE 683 OCR | FE HTML HML | L1 DP 71 (hipótese — **não** equivalência) |
|---|---|---|---|---|
| FA | Factura de Adiantamento | PDF p.**7** · gazeta **19169** | `servico_registar.html` · campo `documentType` | Art.3 g) |
| FT | Factura | p.7 · 19169 | idem | Art.3 f) |
| FR | Factura/Recibo | p.7 · 19169 | idem | Art.3 k) |
| FG | Factura Global | p.7 · 19169 | idem | Art.3 j) |
| GF | Factura Genérica | **ausente** no DE 683 p.7 (OCR + searchable + **visual 2026-07-28**) | **presente** em `servico_registar.html` + XSD `InvoiceType` | Art.3 i) — [C-DOC-001](../conflicts/C-DOC-001-fe-gf-ocr-gap.md) `documentado_divergencia` |
| AC | Aviso de Cobrança | p.7 · 19169 | idem | Art.3 d) |
| AR | Aviso de Cobrança/Recibo | p.7 · 19169 | idem | Art.3 d) + Art.6 n.º2 b) |
| TV | Talão de Venda | p.7 · 19169 | idem | Art.3 p) |
| RC | Recibo Emitido | p.7 · 19169 | idem | Art.3 o) / Art.6 — rótulo FE ≠ DP literal |
| RG | Recibo / Outros Recibos Emitidos | p.7 · 19169 (OCR «Recibo»); p.**22** · **19184** («Outros Recibos Emitidos») | HTML: «Recibo» | Art.3 o) — ver conflito C-DOC-002 |
| RE | Estorno ou Recibo de Estorno | p.7 · 19169 | idem | Sector segurador (não definido no Art.3 DP 71) |
| ND | Nota de Débito | p.7 · 19169 | idem | Art.3 m) |
| NC | Nota de Crédito | p.7 · 19169 | idem | Art.3 l) |
| AF | Factura/Recibo de Autofacturação | p.7 · 19169 | idem | Art.3 c)+k) |
| RP | Prémio ou Recibo de Prémio | p.7 · 19169 | idem | Segurador |
| RA | Resseguro Aceite | p.7 · 19169 | idem | Segurador |
| CS | Imputação a Co-seguradoras | p.7 · 19169 | idem | Segurador |
| LD | Imputação a Co-seguradora Líder | p.7 · 19169 | idem | Segurador |

Campo também assinado em JWS: `documentType` ∈ lista de campos de `jwsDocumentSignature` · DE 683 PDF p.**6** · gazeta **19168** · e snapshot `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA`.

Repetições do enum no mesmo diploma (OCR): PDF p.**22** · **19184** (pedido série); p.**24–26** · **19186–19188** (consultas) — mesma família de códigos (OCR parcial em algumas páginas; p.22/26 usam rótulo RG «Outros Recibos Emitidos»).

### B.0 Estrutura DE 683/25 (Artigos + Anexos + Tabelas)

Citação consolidada: **Citação G** em [`PROVISIONAL-MATRIX-RM-REQ-001.md`](PROVISIONAL-MATRIX-RM-REQ-001.md).

| Bloco | Conteúdo (OCR auxiliar) | PDF | Gazeta |
|---|---|---|---|
| Art.1–6 | Objecto; Anexos I–III integrantes; séries geradas pela AGT; vigência | p.2–3 | **19164–19165** |
| Anexo I | Estrutura JSON FE + serviços (`registarFactura` … `validarDocumento`) | p.4–30 | **19166–19192** |
| Anexo II | Modelo validação **a posteriori**; lotes; rejeição ⇒ inexistência fiscal | p.31 | **19193** |
| Anexo III | REST/JSON; QR Code impresso (`documentNo` na URL portal) | p.32–33 | **19194–19195** |
| Tabelas 1–6 | CAE; taxas IEC/IS; isenções IVA/IS/IEC | p.33–65 | **19195–19227** |

**Não** inventar códigos `FE-RNG-*` a partir do OCR do QR; URL/portal ≠ especificação RNG completa.

### B.1 Regras de preenchimento FE ligadas ao tipo (não MVP completo)

| Regra | Campo FE | Tipos | Citação |
|---|---|---|---|
| `lines` **não** preenchido | `lines` | AR, RC, RG | DE 683 PDF p.**7–8** · **19169–19170** |
| `paymentReceipt` **obrigatório** | `paymentReceipt` | AR, RC, RG | DE 683 PDF p.**8** · **19170** |
| `referenceInfo` obrigatório (devolução) | `referenceInfo` | NC (indicar factura base) | DE 683 PDF p.**8** · **19170** |
| `documentCancelReason` se anulado | `documentCancelReason` | quando `documentStatus=A` | DE 683 PDF p.**6** · **19168**; remete Art.8 n.ºs 8–9 DP 71/25 |

## C. Camadas L2 + L3 — catálogo em 5 grupos funcionais SAF-T

Ficheiro: [`SAFTAO1.01_01.xsd`](../../saft-ao/schemas/SAFTAO1.01_01.xsd) · sha256 `e9a938e1…` · **`pending_validation`** · Citação D em [`PROVISIONAL-MATRIX-RM-REQ-001.md`](PROVISIONAL-MATRIX-RM-REQ-001.md).

**Decisão de produto:** [`DEC-PROD-001`](../../../docs/06-delivery/open-decisions.md) — o catálogo organiza-se **exactamente** nestes **5** grupos (L3). Homónimos L2 entre grupos **não** são o mesmo conceito (`GR` em `MovementType` ≠ `GR` em `WorkType`; `RC` em `PaymentType` ≠ `RC` em `PurchaseType`).

### C.0 Índice canónico — 5 grupos (L3)

| # | Grupo funcional | Estrutura L3 | Elemento | Tipo L2 | Linhas XSD (tipo) | ≠ |
|---|---|---|---|---|---|---|
| 1 | **Vendas** | `SalesInvoices` | `Invoice` | `InvoiceType` | L2023–2065 | L4 `documentType` |
| 2 | **Movimentação de mercadorias** | `MovementOfGoods` | `StockMovement` | `MovementType` | L2089–2107 | L4 FE |
| 3 | **Conferência / trabalho** | `WorkingDocuments` | `WorkDocument` | `WorkType` | L2482–2525 | L4 FE |
| 4 | **Pagamentos / recibos** | `Payments` | `Payment` | `SAFTAOPaymentType` / `PaymentType` | L2740–2754 | L4 FE; ≠ `InvoiceType` |
| 5 | **Compras** | `PurchaseInvoices` | `Invoice` | `PurchaseType` | L2230–2265 | ≠ `InvoiceType` vendas |

**Facto crítico:** `RC`/`RG` do grupo 4 vivem em L3 **`Payments`**, **não** em `SalesInvoices`. Isso **não** mapeia L4 FE → L2 `InvoiceType` ([C-DOC-003](../conflicts/C-DOC-003-fe-vs-saft-invoice-type.md)).

### C.1 Grupo 1 — Vendas (`InvoiceType`)

| Código L2 | Documentação XSD |
|---|---|
| FT | Factura |
| FR | Factura/recibo |
| GF | Factura genérica |
| FG | Factura global |
| AC | Aviso de cobrança |
| AR | Aviso de cobrança/recibo |
| ND | Nota de débito |
| NC | Nota de crédito |
| AF | Factura/recibo (autofacturação) |
| TV | Talão de venda |
| RP | Prémio ou recibo de prémio (segurador) |
| RE | Estorno ou recibo de estorno (segurador) |
| CS | Imputação a co-seguradoras (segurador) |
| LD | Imputação a co-seguradora líder (segurador) |
| RA | Resseguro aceite (segurador) |

**Ausentes face a L4 FE:** `FA`, `RC`, `RG` — [C-DOC-003](../conflicts/C-DOC-003-fe-vs-saft-invoice-type.md).

### C.2 Grupo 2 — Movimentação de mercadorias (`MovementType`)

| Código L2 | Documentação XSD |
|---|---|
| GR | Guia de remessa |
| GT | Guia de transporte (inclui guias globais) |
| GA | Guia de movimentação de activos fixos próprios |
| GD | Guia ou nota de devolução |

### C.3 Grupo 3 — Conferência / trabalho (`WorkType`)

| Código L2 | Documentação XSD |
|---|---|
| CM | Consultas de mesa |
| CC | Crédito de consignação |
| GR | Guia de remessa |
| NR | Nota de remessa |
| FO | Folhas de obra |
| NE | Nota de encomenda |
| OU | Outros |
| OR | Orçamentos |
| PF | Pró-forma |
| DC | Conferência de mercadorias / serviços |
| RP / RE / CS / LD / RA | Segurador (quando aplicável; ver XSD) |
| PP | (enum XSD; documentação OCR/XSD a confrontar) |
| GC | Guia de consignação |

### C.4 Grupo 4 — Pagamentos / recibos (`SAFTAOPaymentType`)

| Código L2 | Documentação XSD | Estrutura L3 |
|---|---|---|
| RC | Recibo emitido | `Payments` |
| RG | Outros recibos emitidos | `Payments` |
| AR | Aviso de cobrança/recibo | `Payments` |

### C.5 Grupo 5 — Compras (`PurchaseType`)

| Código L2 | Documentação XSD |
|---|---|
| FT | Factura |
| FR | Factura/recibo |
| GF | Factura genérica |
| FG | Factura global |
| AC | Aviso de cobrança |
| AR | Aviso de cobrança/recibo |
| AF | Factura/recibo (autofacturação) |
| TV | Talão de venda |
| NL | Nota de liquidação |
| NC | Nota de crédito |
| RC | Recibo (só regime de caixa — anotação XSD) |

**≠** grupo 1: mesmo código L2 sob `PurchaseType` **não** é `InvoiceType` de vendas.

### C.6 Outros elementos SAF-T (campos — não são L4 nem grupo funcional)

| Elemento | Linhas | Camada | Uso |
|---|---|---|---|
| `InvoiceNo` | L1974–2001 | L3 campo | Pattern tipo+série/n.º **no XML SAF-T** |
| `InvoiceStatus` | L2003–2021 | L3 campo | N/S/A/R |
| `References` | L1004–1023 | L3 | Obrigatório quando `InvoiceType=NC` |
| `Hash` / `HashControl` | L1361–1374 | L3 crypto SAF-T | ≠ JWS FE (C-SIGN-001) |

### C.7 Canal de utilização — inclusão / exclusão (`DEC-PROD-002`)

**Regra:** incluir só se canal ∈ {`SAF-T`, `FE`, `ambos`}. Excluir se canal = `nenhum`.

| Canal | Critério (facto de fonte) |
|---|---|
| `FE` | Código em L4 `documentType` (DE 683 + HTML HML) |
| `SAF-T` | Código em enum L2 de um dos 5 grupos (`DEC-PROD-001`) |
| `ambos` | `FE` ∧ `SAF-T` (mesmo código pode viver em L2/L3 distintos — **não** bijecção) |
| `nenhum` → **excluído** | Sem L4 e sem L2 — fora do catálogo de tipos |

#### Incluídos — canal `ambos` (homónimo L4 ∩ algum L2; grupo L3 pode diferir)

| Código | L4 FE | L2 (grupo) | Nota |
|---|---|---|---|
| FT, FR, FG, GF, AC, ND, NC, AF, TV | sim | `InvoiceType` (1) | GF: C-DOC-001 (divergência DE 683 vs HTML/XSD) |
| AR | sim | `InvoiceType` (1) e/ou `PaymentType` (4) | dual canónicos seed; grupo único por emissão — aberto ([C-DOC-004](../conflicts/C-DOC-004-ar-dual-l3.md)) |
| RE, RP, RA, CS, LD | sim | `InvoiceType` (1); também `WorkType` (3) | segurador; dual L3 — [C-DOC-005](../conflicts/C-DOC-005-insurer-invoice-vs-worktype.md); seed só `InvoiceType`/`off` |
| RC | sim | `PaymentType` (4); também `PurchaseType` (5) regime caixa | **≠** `InvoiceType` (C-DOC-003); dual L3 — [C-DOC-006](../conflicts/C-DOC-006-rc-payment-vs-purchase.md) |
| RG | sim | `PaymentType` (4) | C-DOC-002/003 |

#### Incluídos — canal `FE` apenas

| Código | L4 FE | L2 SAF-T | Nota |
|---|---|---|---|
| FA | sim | **∅** `InvoiceType` / `PaymentType` / … | C-DOC-003 — **incluir** (serve FE) |

#### Incluídos — canal `SAF-T` apenas (sem L4 FE catalogado)

| Grupo L3 | Códigos L2 | Nota |
|---|---|---|
| 2 Movimentação | GR, GT, GA, GD | `MovementType` | GR dual `WorkType` — [C-DOC-007](../conflicts/C-DOC-007-gr-movement-vs-worktype.md) |
| 3 Conferência | CM, CC, GR, NR, FO, NE, OU, OR, PF, DC, PP, GC (+ segurador se só em WorkType) | `WorkType`; PF/OR/NE ≠ «factura» L1; GR ≠ `MovementType` (C-DOC-007) |
| 5 Compras | NL (+ demais `PurchaseType` já cobertos noutros canais como homónimos) | `NL` sem L4 FE |

#### Excluídos — canal `nenhum` (não entram no catálogo de tipos)

| Figura | Motivo |
|---|---|
| «Factura Electrónica» (Art.3 h) | Modalidade L1, não código FE/SAF-T |
| «Documento Fiscalmente Relevante» (Art.3 e) | Género residual L1, não código |
| Bordereaux bancários; requisição de fundos; «nota de preço» / «nota de pagamento» sem enum | Art.4 n.º9 / sem L4 nem L2 |
| Qualquer inventário comercial / OpenAPI alias sem código de canal | Produto ≠ canal fiscal |
| Códigos inventados para «fechar» C-DOC-* | Proibido |

**Fail-closed:** ausência de código num canal **não** autoriza inventar o outro.

#### Routing operacional (`DEC-PROD-006`)

| Canal do tipo | Sem FE `active` | Com FE `active` |
|---|---|---|
| `SAF-T` | fluxos SAF-T aplicáveis | SAF-T aplicável; **nunca** endpoint FE |
| `FE` | **indisponível** para FE | só se elegível endpoint + contribuinte/série |
| `ambos` | só lado SAF-T aplicável | FE se elegível; SAF-T conforme L2/L3 (≠ bijecção) |

### C.8 Configuração POS — activação (`DEC-PROD-003`)

| Nível | Operação | Efeito |
|---|---|---|
| Grupo (1–5) | activar / desactivar | Desactivar grupo ⇒ todos os tipos do grupo indisponíveis nesse POS |
| Tipo (catálogo) | activar / desactivar | Tipo activo ⇒ grupo pai activo; fora do catálogo ⇒ impossível |

Enforcement: módulo fiscal (não o POS). Activação **não** alarga OpenAPI nem fecha `DEC-REG-003`. Defaults de activação no MVP = decisão futura.

### C.9 Disponibilidade efectiva (`DEC-PROD-005`)

`available = canonical ∧ group_type_active ∧ regime_nif ∧ fe_ok ∧ series_ok_when_required ∧ environment_ok ∧ sector_ok`

Qualquer gate falso ⇒ indisponível. Activação POS ≠ disponibilidade completa. Detalhe: [`open-decisions.md`](../../../docs/06-delivery/open-decisions.md) `DEC-PROD-005`.

### C.10 Conceito canónico + adaptadores (`DEC-PROD-007`)

```
POS code  --mapping-->  CanonicalDocumentType  --adapter-->  FE L4 / SAF-T L2+L3
```

- Um canónico por entrada de catálogo; adaptadores **por canal** (não o POS escolhe o enum fiscal).
- Sem mapping ⇒ rejeitar. Código fiscal cru no pedido POS ⇒ rejeitar.
- OpenAPI `invoice`/`credit_note` = aliases de produto (candidatos a mapping), **não** códigos AGT.

## D. Cruzamento consolidado (hipótese — L1/L4/L2 + grupo L3)

Colunas **não** são bijecções. Grupo L3 = um dos **5** de DEC-PROD-001. Células vazias / ∅ = lacuna — **não** inventar. MVP = `DEC-REG-003` (ainda aberta).

| L1 documento legal | L4 FE | L2 tipo | Grupo L3 (1–5) | OpenAPI BWB | Estado |
|---|---|---|---|---|---|
| Factura (Art.3 f) | FT | `InvoiceType=FT` | 1 Vendas | `invoice` (**hipótese**) | **não** confirmado |
| Factura-Recibo (Art.3 k) | FR | `InvoiceType=FR` | 1 Vendas | — | hipótese |
| Factura Global (Art.3 j) | FG | `InvoiceType=FG` | 1 Vendas | — | hipótese |
| Factura Genérica (Art.3 i) | GF (HTML/XSD; **ausente** DE 683 p.7 visual) | `InvoiceType=GF` | 1 Vendas | — | C-DOC-001 |
| Factura Adiantamento (Art.3 g) | FA | **∅** `InvoiceType` | **∅** | — | C-DOC-003 |
| Aviso de Cobrança (Art.3 d) | AC | `InvoiceType=AC` | 1 Vendas | — | hipótese |
| Aviso Cobrança/Recibo (Art.3 d)+Art.6 n.º2 b) | AR | `InvoiceType` e/ou `PaymentType` | **1 e/ou 4** | — | sem decisão de grupo único |
| Talão de Venda (Art.3 p) | TV | `InvoiceType=TV` | 1 Vendas | — | hipótese |
| Recibo (Art.3 o) | RC / RG | `PaymentType=RC/RG` | **4** Pagamentos | — | C-DOC-002/003 |
| Nota de Crédito (Art.3 l) | NC | `InvoiceType=NC` | 1 Vendas | `credit_note` (**hipótese**) | **não** confirmado |
| Nota de Débito (Art.3 m) | ND | `InvoiceType=ND` | 1 Vendas | — | hipótese |
| Auto-Facturação (Art.3 c) | AF | `InvoiceType=AF` | 1 Vendas | — | hipótese |
| (sem Art.3) segurador | RE, RP, RA, CS, LD | `InvoiceType` | 1 Vendas | — | só L2/L4 |
| Guias / transporte (Art.4 n.º9 exclui como factura) | — | `MovementType` | **2** Movimentação | — | inventário L2; fora MVP até DEC-REG-003 |
| Conferência / pró-forma / etc. | — | `WorkType` | **3** Conferência | — | inventário L2; fora MVP até DEC-REG-003 |
| Compras (lado fornecedor) | — | `PurchaseType` | **5** Compras | — | inventário L2; fora emissão FE típica |
| Excluídos Art.4 n.º9 | — | — | — | — | fora emissão factura |

## E. DE 74/19 + Rect. 10/19 — validação de software (conjunto normativo)

Fontes: `AO-LEG-DE-74-19-2019` (`5b63c80e…`, 12p, OCR `reviewed`) + `AO-LEG-RECT-10-19-2019` v2 (`b3db14e2…`, 3p, OCR `reviewed`).
Gazeta DE 74: **1576–1586** (PDF p.2–12; p.1 = capa/sumário do fascículo). Gazeta Rect.: **1948–1949** (PDF p.2–3; p.1 = capa).
POLICY: requisitos derivados devem citar **ambos** quando aplicável. PDF original prevalece.

### E.1 Rectificação n.º 10/19 (o que altera o 74/19)

| # | Correcção (OCR auxiliar) | PDF | Gazeta |
|---|---|---|---|
| 1 | Art.1.º: aprova regras/requisitos **e** o modelo do requerimento (anexos parte integrante) — referência n.º1 Art.8 DP 312/18 | p.2 | 1948 |
| 2 | Anexo II, 3.º passo: «SAF-T(**PT**)» → «SAF-T(**AO**)» | p.2 | 1948 |
| 3 | Inclui **Anexo III** — Modelo do Requerimento (Modelo 08 / declaração produtor + chave pública) | p.2–3 | 1948–1949 |

Anexo III **não** enumera tipos de emissão; é requerimento de validação de software.

### E.2 DE 74/19 — classes de documentos (Anexo I)

| Classe (OCR) | Norma anexo | PDF | Gazeta | Uso |
|---|---|---|---|---|
| Facturas e documentos rectificativos | n.º4 a) | p.2–3 | 1576–1577 | Assinatura / eficácia externa |
| Guias transporte/remessa / docs transporte | n.º4 a) | p.2 | 1576 | Movimentação |
| Docs conferência / consultas de mesa | n.º4 a) | p.2 | 1576 | Conferência cliente |
| Não-factura: menção «Este documento não serve de factura» | n.º4 c) | p.3 | 1577 | Tabelas SAF-T 4.2–4.4 |
| `OrderReferences` / `References` (Billing) | n.º4 d)–e) | p.3 | 1577 | Origem / rectificativos |
| Tipos FT/NC/ND (exemplo estrutura) | anexo tabelar | p.11 | 1585 | OCR ruidoso; cruzar com FE/SAF-T enums |

### E.3 DE 74/19 — invariantes críticas (Anexo I)

| Tema | Norma | PDF | Gazeta | AO-* candidato |
|---|---|---|---|---|
| Exportação SAF-T(AO) + XSD | n.º1 | p.2 | 1576 | `AO-SAF-001` |
| Chave assimétrica fabricante | n.º2 | p.2 | 1576 | `AO-KEY-*` / crypto SAF-T |
| Sem alterar info fiscal sem evidência | n.º3 | p.2 | 1576 | `AO-DOC-002` |
| Assinar docs eficácia externa (excepto recibos) | n.º4 a)–b) → n.º34 | p.2–3 | 1576–1577 | crypto SAF-T (**≠** FE JWS) |
| Séries por estabelecimento/programa; sem repetir; univocidade | n.º4 g)–i) | p.3 | 1577 | `AO-SEQ-001` / `AO-ID-001` |
| Série descontinuada: inibir, **não apagar** | n.º4 j) | p.3 | 1577 | `AO-DOC-002` / seq |
| Doc assinado: **não alterar** informação | n.º4 l) | p.3 | 1577 | `AO-DOC-002` |
| Encadeamento Hash por série/tipo | n.º5 e) | p.3–4 | 1577–1578 | crypto SAF-T; C-SIGN-001 |
| Integração multi-sistema sem recalcular | n.º7 | p.5–6 | 1579–1580 | `AO-OFF-002` (parcial) |
| Contingência tipográfica / recuperação séries | n.º8–9 | p.6 | 1580 | `AO-OFF-*` |
| RSA + SHA-1 + campos assinados (InvoiceDate, SystemEntryDate, InvoiceNo, GrossTotal, Hash ant.) | n.º**34** | p.8–10 | 1582–1584 | crypto SAF-T; **≠** `jwsDocumentSignature` FE |

### E.4 Limites

- Mecanismo n.º34 é **SAF-T / validação de software**, não o JWS FE (ver [C-SIGN-001](../conflicts/C-SIGN-001-saft-rsa-vs-fe-jws.md)).
- OCR com ruído tipográfico forte (colunas); confrontar PDF antes de fechar AO-*.
- p.12 do fascículo inclui início de Despacho 17/19 adjacente — **não** citar como DE 74/19.

## F. DP 71/25 — requisitos e ciclo de vida (citações página a página)

Critério de `AO-DOC-001` («campos obrigatórios por tipo» + validação operacional) **não** fica satisfeito só com este inventário. PDF original prevalece; OCR `ai_assisted`.

### F.1 Índice de artigos relevantes (só 11902–11920)

| Artigo | Tema | PDF | Gazeta |
|---|---|---|---|
| 1–2 | Objecto / âmbito | p.2 | 11902 |
| 3 | Definições (tipos) | p.3–4 | 11903–11904 |
| 4 | Emissão; exclusões «não factura» | p.4–5 | 11904–11905 |
| 5 | Dispensa + talão | p.5–6 | 11905–11906 |
| 6 | Recibos | p.6 | 11906 |
| 7 | Processamento / software / tipografia | p.6–7 | 11906–11907 |
| 8 | Prazo, rectificação, anulação (NC) | p.7–8 | 11907–11908 |
| 9 | Portal do Contribuinte | p.8 | 11908 |
| **10** | **Requisitos obrigatórios + excepções por tipo** | **p.8–9** | **11908–11909** |
| 11–15 | Auto-facturação | p.9–11 | 11909–11911 |
| 16–17 | FE sujeição / emissão tempo real | p.11 | 11911 |
| 18 | Contingência FE | p.11–12 | 11911–11912 |
| 19–20 | Autenticidade; remete Art.10 | p.12 | 11912 |
| 21 | Manifestação NC electrónica | p.12 | 11912 |
| 24 | Estabelecimentos / softwares / séries | p.14–15 | 11914–11915 |
| 25 | Comunicação SAF-T | p.15 | 11915 |
| 36 | Spec técnica → Decreto Executivo (683/25) | p.19 | 11919 |
| 42 | Entrada em vigor (6 meses após publicação) | p.20 | 11920 |

### F.2 Art.10.º n.º1 — requisitos obrigatórios da factura

Fonte: `AO-LEG-DP-71-25-2025` · sha256 `4931fd3c…` · OCR v1 `reviewed`.

| Alínea | Conteúdo (OCR auxiliar) | PDF | Gazeta |
|---|---|---|---|
| a) | Nome/firma/NIF/sede do fornecedor **e** do adquirente (quando no exercício de actividade) | p.8 | 11908 |
| b) | Numeração sequencial e cronológica **por tipo de documento** e ano económico; uma ou mais séries | p.8 | 11908 |
| c) | Discriminação bens/serviços, quantidades; embalagens não transaccionáveis | p.8 | 11908 |
| d) | Preço unitário e total em moeda nacional (+ extenso); excepções import/export | p.8 | 11908 |
| e) | Taxas de imposto e montante, quando devido | p.8 | 11908 |
| f) | Motivo de não liquidação + norma legal | p.8 | 11908 |
| g) | Data, hora e local de colocação/prestação; pagamentos antecipados se aplicável (OCR tipografa «9)» — confrontar PDF) | p.9 | 11909 |
| h) | Redacção em português | p.9 | 11909 |
| i) | Data da emissão | p.9 | 11909 |
| j) | Software AGT + código hash + gráfica/tipografia + n.º certificação/validação | p.9 | 11909 |

Art.10 n.º2: taxas diferentes → descrição separada · p.**9** · **11909**.

### F.3 Art.10.º — aplicação / excepções por tipo

| Tipo / figura | Norma | Excepção vs n.º1 | PDF | Gazeta |
|---|---|---|---|---|
| Factura genérica, factura-recibo, aviso de cobrança, factura global | Art.10 n.º4 | Devem respeitar n.º1 completo | p.9 | 11909 |
| Nota de crédito | Art.10 n.º5 | Excepção **alíneas g)** | p.9 | 11909 |
| Recibo | Art.10 n.º6 | Excepção **alíneas c) e g)** | p.9 | 11909 |
| Auto-facturação (factura/recibo) | Art.10 n.º3 + Art.12 | Série **diferente** da factura de vendas (alíena b); menção «Auto-Facturação» | p.9 | 11909 |
| Factura entidade estrangeira | Art.10 n.º7 | Tradução PT | p.9 | 11909 |
| NC (conteúdo adicional) | Art.8 n.º4–5 | Motivo; «anulação»/«rectificação»; doc. anulado; prova conhecimento adquirente | p.7 | 11907 |
| FA regularização | Art.8 n.º10 | Via notas de crédito | p.8 | 11908 |

### F.4 Outras citações DP 71 ligadas a tipos / emissão

| Tema | Norma | PDF | Gazeta | Nota |
|---|---|---|---|---|
| Software não permite eliminação após emissão | Art.3 n) | p.4 | 11904 | Candidato `AO-DOC-002` (parcial) |
| Contingência offline / tipografia | Art.18 + Art.7 n.º6 | p.11–12 / p.7 | 11911–11912 / 11907 | Candidato `AO-OFF-001` (parcial) |
| Comunicação estabelecimento + séries | Art.24 n.º1 a)–c) | p.14 | 11914 | Lacuna terminal em `AO-ID-001` |
| SAF-T comunicação facturas/recibos/outros | Art.25 | p.15 | 11915 | Liga a `AO-SAF-*` |
| Spec FE por Decreto Executivo | Art.36 | p.19 | 11919 | Ponte para DE 683/25 |

### F.5 Outras camadas (não substituem Art.10 / L1)

| Camada | Âmbito | Fonte | Citação | Cobertura |
|---|---|---|---|---|
| L4 | Payload FE `documentType` | DE 683 + HTML registar | p.7–8 · 19169–19170 | Parcial (B.1) |
| L4 | Anexos II–III / Tabelas | DE 683 Citação G | p.31–65 · 19193–19227 | Modelo a posteriori + REST/QR |
| L2/L3 | Tipo + estrutura SAF-T | XSD ASSOFT | Citação D / secção C | **≠** L4 |
| Produto | OpenAPI MVP | `invoice` / `credit_note` | OpenAPI draft | Subconjunto; **não** normativo AGT |

## G. Implicações explícitas (fail-closed)

1. **Não** confundir L1 (rótulo legal) · L2 (enum SAF-T) · L3 (estrutura SAF-T) · L4 (`documentType` FE).
2. Catálogo = **exactamente 5** grupos L3 (`DEC-PROD-001`); **não** inventar 6.º grupo.
3. Incluir só canal SAF-T e/ou FE (`DEC-PROD-002`); **excluir** canal `nenhum`; **não** inventar códigos para o outro canal.
4. Config POS (`DEC-PROD-003`): grupo e tipos activáveis; pedido inactivo ⇒ rejeição pelo módulo; **não** o POS numera.
5. Disponibilidade efectiva (`DEC-PROD-005`): **todos** os gates; activação sozinha **não** basta.
6. Routing (`DEC-PROD-006`): SAF-T-only ∉ FE; FE-only só FE; sem FE `active` ⇒ só SAF-T aplicável.
7. Canónico + adaptadores (`DEC-PROD-007`): POS só código mapeado; **nunca** código fiscal cru sem mapping.
8. Módulo = única autoridade de emissão/numeração; N POS via API (`DEC-PROD-008`).
9. Estados doc. (`DEC-PROD-009`): não afirmar aceitação AGT antes de `accepted`.
10. Offline (`DEC-PROD-010`): outbox/reenvio/idempotência; **não** declarar emissão offline certificada.
11. Edge (`DEC-PROD-011`): um processo fiscal escritor; sem multi-instância na mesma série sem coordenação.
12. Chaves (`DEC-PROD-012`): segregadas; non-exportable quando possível; nunca POS; rotação auditada; definitivo AGT.
13. Auditoria (`DEC-PROD-013`): append-only; retenção final só com norma consolidada.
14. Modelo completo SAF-T∪FE (`DEC-PROD-014`); **não** truncar catálogo ao OpenAPI slice; faseamento = `DEC-REG-003`.
15. **Não** alargar OpenAPI público sem revisão; slice actual `invoice`/`credit_note` ≠ perímetro do modelo.
16. **Não** activar `GF` nem tratar DE 683 p.7 como enum L4 alinhado ao HTML enquanto C-DOC-001 (`documentado_divergencia`) permanecer sem fecho compliance.
17. **Não** assumir bijecção L4↔L2↔L3 (`FA`/`RC`/`RG`; `PaymentType` ≠ `InvoiceType` ≠ `PurchaseType`).
18. `AO-DOC-001` permanece **não** confirmado; esta matriz é inventário citado por camada/grupo/canal.
19. Precedência: DP 71/25 (L1) + DE 683/25 (L4) + XSD (L2/L3) + 74/19+Rect (validação); conflitos → `compliance/derived/conflicts/`.

## Referências

- Matriz AO-* provisória: [`PROVISIONAL-MATRIX-RM-REQ-001.md`](PROVISIONAL-MATRIX-RM-REQ-001.md)
- Conflitos: [`../conflicts/C-DOC-001-fe-gf-ocr-gap.md`](../conflicts/C-DOC-001-fe-gf-ocr-gap.md), [`../conflicts/C-DOC-002-rg-label.md`](../conflicts/C-DOC-002-rg-label.md), [`../conflicts/C-DOC-003-fe-vs-saft-invoice-type.md`](../conflicts/C-DOC-003-fe-vs-saft-invoice-type.md)
- Modelo/catálogo: `DEC-PROD-001`–`015`; catálogo: [`DOCUMENT-CATALOG-RM-REQ-001.md`](DOCUMENT-CATALOG-RM-REQ-001.md); `DEC-REG-003` = ordem slice
