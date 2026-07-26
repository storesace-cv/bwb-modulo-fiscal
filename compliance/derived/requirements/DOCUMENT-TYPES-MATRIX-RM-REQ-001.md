# Matriz normativa de tipos documentais (provisória)

**Estado:** rascunho rastreável — **não** confirma `AO-DOC-001` / `AO-DOC-002` nem fecha `DEC-REG-003`.
**Data:** 2026-07-26
**Regra:** PDF original prevalece; OCR/`reviewed` e HTML FE são auxiliares. Códigos FE ≠ códigos SAF-T até mapeamento confirmado por compliance. Sem inventar tipos em falta.

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

## A. Definições legais (DP 71/25) — não são códigos FE

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

## B. Códigos FE `documentType` (DE 683/25 + HTML HML)

Campo JSON FE: **`documentType`** (string, minLength 2, maxLength 2), obrigatório no registo.

| Código | Significado (fonte) | DE 683 OCR | FE HTML HML | Alinha a DP 71 (hipótese) |
|---|---|---|---|---|
| FA | Factura de Adiantamento | PDF p.**7** · gazeta **19169** | `servico_registar.html` · campo `documentType` | Art.3 g) |
| FT | Factura | p.7 · 19169 | idem | Art.3 f) |
| FR | Factura/Recibo | p.7 · 19169 | idem | Art.3 k) |
| FG | Factura Global | p.7 · 19169 | idem | Art.3 j) |
| GF | Factura Genérica | **ausente no OCR p.7** (lacuna OCR) | **presente** em `servico_registar.html` | Art.3 i) — ver conflito C-DOC-001 |
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

Repetições do enum no mesmo diploma (OCR): PDF p.**22** · **19184** (pedido série); p.**24–26** · **19186–19188** (consultas) — mesma família de códigos (OCR parcial em algumas páginas).

Campo também assinado em JWS: `documentType` ∈ lista de campos de `jwsDocumentSignature` · DE 683 PDF p.**6** · gazeta **19168** · e snapshot `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA`.

### B.1 Regras de preenchimento FE ligadas ao tipo (não MVP completo)

| Regra | Campo FE | Tipos | Citação |
|---|---|---|---|
| `lines` **não** preenchido | `lines` | AR, RC, RG | DE 683 PDF p.**7–8** · **19169–19170** |
| `paymentReceipt` **obrigatório** | `paymentReceipt` | AR, RC, RG | DE 683 PDF p.**8** · **19170** |
| `referenceInfo` obrigatório (devolução) | `referenceInfo` | NC (indicar factura base) | DE 683 PDF p.**8** · **19170** |
| `documentCancelReason` se anulado | `documentCancelReason` | quando `documentStatus=A` | DE 683 PDF p.**6** · **19168**; remete Art.8 n.ºs 8–9 DP 71/25 |

## C. Códigos SAF-T `InvoiceType` (XSD ASSOFT)

Ficheiro: `SAFTAO1.01_01.xsd` · elemento `InvoiceType` · sha256 `e9a938e1…` · **`pending_validation`** (não validado AGT).

| Código | Documentação XSD | Enum XSD |
|---|---|---|
| FT | Factura | sim |
| FR | Factura/recibo | sim |
| GF | Factura genérica | sim |
| FG | Factura global | sim |
| AC | Aviso de cobrança | sim |
| AR | Aviso de cobrança/recibo | sim |
| ND | Nota de débito | sim |
| NC | Nota de crédito | sim |
| AF | Factura/recibo (autofacturação) | sim |
| TV | Talão de venda | sim |
| RP | Prémio ou recibo de prémio | sim (segurador) |
| RE | Estorno ou recibo de estorno | sim (segurador) |
| CS | Imputação a co-seguradoras | sim (segurador) |
| LD | Imputação a co-seguradora líder | sim (segurador) |
| RA | Resseguro aceite | sim (segurador) |

**Ausentes no XSD `InvoiceType` face ao enum FE:** `FA`, `RC`, `RG` (exportação SAF-T destes tipos **não** está demonstrada pelo schema versionado — conflito C-DOC-003).

## D. Cruzamento consolidado (legal ↔ FE ↔ SAF-T ↔ OpenAPI)

| Tipo (nome) | DP 71 | Código FE | Código SAF-T | OpenAPI BWB actual | Estado linha |
|---|---|---|---|---|---|
| Factura | Art.3 f) @11903 | FT | FT | `invoice` (**hipótese** de mapeamento) | `partial` citação; mapeamento OpenAPI **não** confirmado |
| Factura-Recibo | Art.3 k) @11903 | FR | FR | — | citado; fora MVP OpenAPI |
| Factura Global | Art.3 j) @11903 | FG | FG | — | citado |
| Factura Genérica | Art.3 i) @11903 | GF (HTML; OCR 683 incompleto) | GF | — | C-DOC-001 |
| Factura Adiantamento | Art.3 g) @11903 | FA | **∅** no `InvoiceType` | — | C-DOC-003 |
| Aviso de Cobrança | Art.3 d) @11903 | AC | AC | — | citado |
| Aviso Cobrança/Recibo | Art.6 n.º2 b) @11906 | AR | AR | — | citado |
| Talão de Venda | Art.3 p) @11904 | TV | TV | — | citado |
| Recibo | Art.3 o) @11904 | RC / RG | **∅** como InvoiceType | — | C-DOC-002/003 |
| Nota de Crédito | Art.3 l) @11904 | NC | NC | `credit_note` (**hipótese**) | `partial` citação; mapeamento **não** confirmado |
| Nota de Débito | Art.3 m) @11904 | ND | ND | — | citado; Art.4 n.º9: não é factura |
| Autofacturação (FR) | Art.3 c) @11903 | AF | AF | — | citado |
| Tipos segurador RE/RP/RA/CS/LD | — (não no Art.3) | RE, RP, RA, CS, LD | idem | — | só FE/SAF-T; fora Art.3 DP 71 |
| Pró-forma / guias / orçamentos | Art.4 n.º9 @11905 | — | — | — | **excluídos** como factura |

## E. DE 74/19 — classes para validação de software (não enum FE)

Citação OCR auxiliar: anexo requisitos · PDF p.**2–3** · `AO-LEG-DE-74-19-2019` (`5b63c80e…`).

| Classe (OCR) | PDF | Uso |
|---|---|---|
| Facturas e documentos rectificativos | p.2–3 | Assinatura / inalterabilidade / referências |
| Guias de transporte / remessa / docs de transporte | p.2–3 | Movimentação; menção «não serve de factura» para não-facturas |
| Documentos de conferência | p.2–3 | Conferência mercadorias/serviços |
| Referência a documento origem (`OrderReferences`) / facturas (`BillingReferences`) | p.3 | Rectificativos e docs não-factura |

Rect. 10/19 v2 (gazeta 1948–1949): Anexo III **Modelo 08** (requerimento) — **não** lista tipos de documento de emissão.

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

### F.5 Camadas FE / OpenAPI (não substituem Art.10)

| Âmbito | Fonte | Citação | Cobertura |
|---|---|---|---|
| Payload FE por tipo | DE 683 + HTML registar | p.7–8 · 19169–19170 | Parcial (B.1) |
| OpenAPI MVP | `invoice` / `credit_note` | OpenAPI draft | Subconjunto produto; **não** normativo AGT |

## G. Implicações explícitas (fail-closed)

1. **Não** alargar enums de domínio/OpenAPI além de `invoice`/`credit_note` sem `DEC-REG-003` + revisão compliance.
2. **Não** tratar OCR DE 683 p.7 como enum completo enquanto `GF` existir no HTML e faltar no OCR (C-DOC-001).
3. **Não** assumir bijecção FE↔SAF-T (`FA`/`RC`/`RG`).
4. `AO-DOC-001` permanece **não** confirmado; esta matriz é inventário citado.
5. Precedência: DP 71/25 (regime) + DE 683/25 (FE) + 74/19+Rect (validação software); conflitos → `compliance/derived/conflicts/`.

## Referências

- Matriz AO-* provisória: [`PROVISIONAL-MATRIX-RM-REQ-001.md`](PROVISIONAL-MATRIX-RM-REQ-001.md)
- Conflitos: [`../conflicts/C-DOC-001-fe-gf-ocr-gap.md`](../conflicts/C-DOC-001-fe-gf-ocr-gap.md), [`../conflicts/C-DOC-002-rg-label.md`](../conflicts/C-DOC-002-rg-label.md), [`../conflicts/C-DOC-003-fe-vs-saft-invoice-type.md`](../conflicts/C-DOC-003-fe-vs-saft-invoice-type.md)
- Decisão tipos MVP: `DEC-REG-003` em [`docs/06-delivery/open-decisions.md`](../../../docs/06-delivery/open-decisions.md)
