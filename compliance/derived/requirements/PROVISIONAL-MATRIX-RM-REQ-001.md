# Matriz provisória RM-REQ-001

**Estado:** `EM_CURSO` — **não** é matriz AO-* confirmada (promoções normativas vivem em [`CONFIRMED-MATRIX-RM-REQ-001.md`](CONFIRMED-MATRIX-RM-REQ-001.md)).
**Data:** 2026-07-29
**Âmbito:** OCR `reviewed` (KB auxiliar) + XSD ASSOFT / snapshots FE `pending_validation`. **AO-DOC-002** + **AO-SEQ-001** promovidos (`confirmed_normative`). C-DOC-001: confronto visual DE 683 p.7 (`GF` ausente).

## Fontes admitidas neste rascunho

| source_id | Estado OCR / schema | Uso permitido aqui | Limite |
|---|---|---|---|
| `AO-LEG-DE-74-19-2019` | `reviewed` v1 (`5b63c80e…`, 12p) | Pesquisa auxiliar; citação futura com página do PDF original | Texto consolidado 74+Rect. exige citação página a página; **não** confirmado |
| `AO-LEG-RECT-10-19-2019` | `reviewed` v2 (`b3db14e2…`, 3p) | Pesquisa auxiliar; gazeta **1948–1949** (PDF p.2–3); capa PDF p.1 | v1 incorrecta `77b77f01…` só diagnóstico privado (≠ KB); **não** confirmar AO-* só com OCR |
| `AO-LEG-DE-683-25-2025` | `reviewed` v2 (`b01e4581…`, 66p) | Pesquisa auxiliar; citar **apenas** gazeta 19164–19227 (PDF p.2–65) | PDF p.66 = Aviso BNA 4/25 @19228 — **não** citar como DE 683 |
| `AO-LEG-DP-71-25-2025` | `reviewed` v1 (`4931fd3c…`, 21p) | Pesquisa auxiliar; citar **apenas** gazeta **11902–11920** (PDF p.2–20) | PDF p.21 = DE 372/25 @11921 — **não** citar como DP 71/25 |
| `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | HTML `pending_validation` (`50ba18e0…`) | Algoritmo JWS/RS256 FE (≠ SAF-T) | Snapshot HML; não validado AGT; sem inventar campos |
| `AO-FE-SNAP-HML-2026-07-25-REGISTAR` / `SOLICITAR` / `LISTAR` / `LISTAR-FATURAS` / `API` / `MODELO` | HTML `pending_validation` | Endpoints + `FE-RNG-*` + modelo assíncrono | Bytes no privado; sem inventar códigos/paths; ver Citação H |
| `AO-SAFT-XSD-1.01_01` | schema `pending_validation` (`e9a938e1…`) | Referência técnica ASSOFT MIT | Não afirmado como validado pela AGT |

## Bloqueios explícitos (fail-closed)

1. **RM-SRC-004 / RM-M2-C** — OCR do conjunto 74+Rect+683 `reviewed` (Rect. v2); DP 71/25 também `reviewed` — gate de fontes OCR fechado; **não** equivale a AO-* confirmados.
2. Esta matriz **não** inventa regras fiscais; promoções `confirmed_normative` registam-se **só** em [`CONFIRMED-MATRIX-RM-REQ-001.md`](CONFIRMED-MATRIX-RM-REQ-001.md) (estado `promoted` aqui).
3. `reviewed` OCR é KB auxiliar; **não** confirma AO-* por si; só citação página a página + revisão compliance (e promoção em CONFIRMED-MATRIX) sustenta `confirmed_normative`.
4. `partial` ≠ confirmado: ligação preliminar sujeita a revisão; o critério de aceitação do catálogo pode ser mais amplo do que a citação.
5. GAP-002 residual: URL oficial estável da Rect. / Imprensa Nacional ainda pode estar `pending_validation`; histórico rejected permanece excluído da KB.

## Legenda de estado da linha

| Estado | Significado |
|---|---|
| `scaffold` | ID do catálogo inicial; sem ligação normativa página a página; **não** confirmado |
| `partial` | Ligação preliminar a fonte `reviewed` com página candidata; sujeita a revisão; **não** confirmado |
| `promoted` | Norma em [`CONFIRMED-MATRIX-RM-REQ-001.md`](CONFIRMED-MATRIX-RM-REQ-001.md); residual engenharia/AGT pode permanecer |
| `blocked` | Dependente de fonte oficial ainda inacessível (ex.: GAP-013, credenciais AGT) |
| `pending_validation` | Artefacto técnico presente sem validação AGT |

Estados proibidos neste ficheiro: `confirmed`, `confirmado`, `confirmed_normative`, `validated_agt`.

## Citações candidatas (OCR `reviewed`, não normativas)

### Citação A — séries AGT (AO-SEQ-002)

Fonte: `AO-LEG-DE-683-25-2025` · original sha256 `b01e45813eccc54790ce23ae64bba4564731566476bb3dec0105c15ad4f223ca` · OCR v2 `reviewed` · PDF p.2 · gazeta **19164**.

Diploma / secção (OCR auxiliar, confrontar com PDF original): **ARTIGO 4.º (Série de facturas)**.

Trecho OCR (pesquisa; tipografia/OCR podem ter ruído — PDF original prevalece):

> Para efeitos do disposto na alínea b) do n.o 1 do artigo 10.oe alínea c) do n.o 1 do artigo 24.8, ambos do Decreto Presidencial n.o 71/25, de 20 de Março, que aprova o Regime Jurídico das Facturas, as séries utilizadas pelos contribuintes que emitem facturas em softwares de facturação electrónica são geradas pela Administração Geral Tributária.

Path privado: `derivatives/legislation/AO-LEG-DE-683-25-2025/v2/text.md` (`private_commit` `c8a4e6e8ec2772ff50ad1c8762842b983edbbbfd`).

### Citação B — identificação contribuinte / software (AO-ID-001)

Fonte: `AO-LEG-DE-683-25-2025` · original sha256 `b01e45813eccc54790ce23ae64bba4564731566476bb3dec0105c15ad4f223ca` · OCR v2 `reviewed`.

| Campo OCR (auxiliar) | PDF | Gazeta |
|---|---|---|
| `taxRegistrationNumber` (NIF contribuinte emissor) | p.4 | **19166** |
| `productId` / `productVersion` / `softwareValidationNumber` (software + versão/certificação) | p.5 | **19167** |

Limite explícito: estes campos cobrem **parte** do critério do catálogo (contribuinte + software/versão); **não** demonstram por si estabelecimento nem terminal. O critério completo do catálogo **não** fica satisfeito só com esta citação. PDF original prevalece; OCR pode ter ruído (`productld`, etc.).

### Citação C — assinatura JWS do documento (AO-CRYPTO-001)

**C.1** Fonte `AO-LEG-DE-683-25-2025` · sha256 `b01e4581…` · OCR v2 `reviewed` · PDF p.6 · gazeta **19168**.

Campo OCR auxiliar: `jwsDocumentSignature` — assinatura da factura com chave privada do emissor sobre campos incluindo `documentNo`, `taxRegistrationNumber`, `documentType`, `documentDate`, totais (OCR com ruído; PDF prevalece).

**C.2** Fonte `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` · sha256 `50ba18e0…` · `pending_validation` · secção «Algoritmo Utilizado (RS256)» — JWS com **RS256 (RSA + SHA-256)**; JWS FE ≠ mecanismos SAF-T.

Limites: **não** afirma encadeamento documental; **não** fecha lista exacta de campos sem confronto PDF+snapshot; critério do catálogo **não** fica satisfeito só com estas citações; **não** confirmado.

### Citação D — schema SAF-T AO (`SAFTAO1.01_01.xsd`)

Fonte: `AO-SAFT-XSD-1.01_01` · ficheiro versionado [`compliance/saft-ao/schemas/SAFTAO1.01_01.xsd`](../../saft-ao/schemas/SAFTAO1.01_01.xsd) · sha256 `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631` · estado catálogo **`pending_validation`** · upstream ASSOFT commit `ed86c7d5…` · `doc:Status` **Development** · MIT ASSOFT ([LICENSE](../../saft-ao/schemas/LICENSE) / [NOTICE](../../saft-ao/schemas/NOTICE.md)).

Identidade técnica (não normativa AGT): `targetNamespace` `urn:OECD:StandardAuditFile-Tax:AO_1.01_01` · `version`/`doc:Number` `1.01_01` · `id` `SAF-T_AO` · elemento raiz `AuditFile`.

| Tema (AO-SAF-*) | Elemento / tipo XSD | Linhas (aprox.) | Nota |
|---|---|---|---|
| Raiz / MasterFiles / SourceDocuments | `AuditFile` → `Header`, `MasterFiles`, `GeneralLedgerEntries?`, `SourceDocuments?` | L42–58 | Estrutura mínima do ficheiro |
| Vendas | `SourceDocuments/SalesInvoices/Invoice` | L445–542 | Inclui `InvoiceNo`, `DocumentStatus`, `Hash`, `HashControl`, `InvoiceType`, `Line` |
| Numeração (campo L3) | `InvoiceNo` pattern `[^ ]+ [^/^ ]+/[0-9]+` | L1974–2001 | Ex.: `FT S001/1`; **≠** L4 `documentType`; formato citado também em DE 683 `documentNo` |
| Tipo L2 (vendas) | `InvoiceType` sob L3 `SalesInvoices` | L2023–2065 | **Sem** FA/RC/RG — [C-DOC-003](../conflicts/C-DOC-003-fe-vs-saft-invoice-type.md); **≠** L4 FE |
| Estado / anulado | `InvoiceStatus` N/S/A/R | L2003–2021 | “A” = anulado (candidato AO-SAF-002) |
| Hash SAF-T | `Hash` (max 172) + `HashControl` | L1361–1367 / L1368–1374 / uso L491–492 | Algoritmo **não** está no XSD — DE 74 n.º34 @1582–1584; ≠ JWS FE ([C-SIGN-001](../conflicts/C-SIGN-001-saft-rsa-vs-fe-jws.md)) |
| Rectificativos | `References` (obrigatório quando `InvoiceType=NC`) | L1004–1023 / uso L523 | Candidato AO-SAF-002; DE 74 n.º4 e) @1577 |
| Tipo L2 + estrutura L3 (recibos) | `SAFTAOPaymentType` / `PaymentType` sob L3 `Payments` | L722–800 / L735 / **L2740–2754** | `RC`/`RG`/`AR` aqui **≠** L4 `documentType` e **≠** L2 `InvoiceType` (C-DOC-003) |
| Impostos linha | `TaxType` IVA/IS/NS | L2379–2395 | IEC aparece como `IECAmount` / produto tipo “E”, **não** como `TaxType` enum |
| Obrigação legal exportação | (fora do XSD) DE 74 Anexo I n.º1 | — | Gazeta **1576**; Rect. SAF-T(AO) @1948 |

Limites: XSD ASSOFT **não** equivale a validação AGT; `doc:Status=Development`; ausência de algoritmo Hash no schema; conflitos FE↔SAF-T abertos; critério do catálogo **não** fica satisfeito só com o schema versionado; **não** confirmado.

### Citação E — DP 71/25 (regime de facturas; tipos e requisitos)

Fonte: `AO-LEG-DP-71-25-2025` · sha256 `4931fd3ce711ef2b22e7316c3dd296d8c7c81993c88e29c70e09baeb0d0e7f76` · OCR v1 `reviewed` · citar só gazeta **11902–11920**.

| Tema | Norma | PDF | Gazeta |
|---|---|---|---|
| Definições **legais** L1 (Factura, Recibo, NC, ND, Talão, … — **não** códigos FE/SAF-T) | Art.3 | p.3–4 | 11903–11904 |
| Exclusões «não são factura» | Art.4 n.º9 | p.5 | 11905 |
| Requisitos obrigatórios a)–j) | Art.10 n.º1 | p.8–9 | 11908–11909 |
| NC / recibo: excepções ao Art.10 | Art.10 n.º5–6 | p.9 | 11909 |
| Numeração por tipo + séries | Art.10 n.º1 b) | p.8 | 11908 |
| Inelegibilidade de eliminação pós-emissão (software) | Art.3 n) | p.4 | 11904 |
| Contingência FE | Art.18 | p.11–12 | 11911–11912 |
| Estabelecimentos / séries a comunicar | Art.24 n.º1 | p.14 | 11914 |

Detalhe e cruzamento FE/SAF-T: [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md). Limite: OCR auxiliar; **não** confirma AO-*; Art.10 g) com tipografia OCR ruidosa («9)»).

### Citação F — DE 74/19 + Rect. 10/19 (conjunto; validação de software)

Fontes: `AO-LEG-DE-74-19-2019` · sha256 `5b63c80e358bd5eda60302a5f6d2adac3c23815de7fc8f496b0b8bdb909d9abd` · OCR v1 `reviewed` · gazeta **1576–1586** (PDF p.2–12); **não** citar p.12 Despacho 17/19 adjacente como 74/19.
`AO-LEG-RECT-10-19-2019` · sha256 `b3db14e2715541be00ac93032718e7a358b493796264e361d1f9d35a1a49e014` · OCR v2 `reviewed` · gazeta **1948–1949** (PDF p.2–3).

| Tema | Norma | PDF | Gazeta |
|---|---|---|---|
| Rect.: Art.1 inclui modelo requerimento; SAF-T(PT)→SAF-T(AO); Anexo III Modelo 08 | Rect. n.ºs 1–3 | Rect p.2–3 | 1948–1949 |
| Exportação SAF-T(AO) + XSD | Anexo I n.º1 | p.2 | 1576 |
| Sem alterar info fiscal sem evidência; doc assinado imutável | Anexo I n.º3 + n.º4 l) | p.2–3 | 1576–1577 |
| «Não serve de factura»; OrderReferences / References | Anexo I n.º4 c)–e) | p.3 | 1577 |
| Séries univocas por estabelecimento/programa; numeração contínua | Anexo I n.º4 g)–i) | p.3 | 1577 |
| Encadeamento Hash série/tipo + RSA/SHA-1 campos n.º34 | Anexo I n.º5 e) + n.º**34** | p.3–4 / p.8–10 | 1577–1578 / **1582–1584** |
| Contingência tipográfica / recuperação por séries | Anexo I n.º8–9 | p.6 | 1580 |

Limite: n.º34 = assinatura **SAF-T** (≠ JWS FE RS256) — [C-SIGN-001](../conflicts/C-SIGN-001-saft-rsa-vs-fe-jws.md). OCR auxiliar; **não** confirma AO-*.

### Citação G — DE 683/25 (estrutura + Anexos I–III + Tabelas)

Fonte: `AO-LEG-DE-683-25-2025` · sha256 `b01e45813eccc54790ce23ae64bba4564731566476bb3dec0105c15ad4f223ca` · OCR v2 `reviewed` · citar só gazeta **19164–19227** (PDF p.2–65); **não** citar p.66 / Aviso 4/25 @19228.

| Tema | Norma / secção | PDF | Gazeta |
|---|---|---|---|
| Objecto; Anexos I–III partes integrantes; actualizações AGT; séries AGT; vigência | Art.1–6 | p.2–3 | **19164–19165** |
| Anexo I — Estrutura de Dados (`registarFactura` e serviços) | Anexo I (início) | p.4 | **19166** |
| `documentNo` (formato alinhado SAF-T AO); software / lote ≤30 | Anexo I | p.5 | **19167** |
| `documentStatus` / anulação / `jwsDocumentSignature` | Anexo I | p.6 | **19168** |
| L4 FE `documentType` + regras por tipo (**≠** L2 `InvoiceType`) | Anexo I | p.7–8 | **19169–19170** |
| Impostos FE: `taxType` IVA/IS/IEC/NS; `taxCode`; isenções → Tabelas 4–6 | Anexo I | p.9–11 | **19171–19173** |
| Totais / retenções (`taxPayable`, `withholdingTax*`) | Anexo I | p.12–13 | **19174–19175** |
| `obterEstado` (V/I/P; atraso >24h sem contingência) | Anexo I | p.14–17 | **19176–19179** |
| `listarFacturas` / `consultarFactura` | Anexo I | p.17–21 | **19179–19183** |
| `solicitarSerie` (`seriesCode`/`seriesYear`/`documentType`) | Anexo I | p.21–22 | **19183–19184** |
| `listarSeries` + `seriesStatus` A/U/F; `invoicingMethod` FEPC/FESF/SF | Anexo I | p.23–27 | **19185–19189** |
| `validarDocumento` (adquirente) | Anexo I | p.27–30 | **19189–19192** |
| Anexo II — modelo validação **a posteriori**; lotes; rejeição ⇒ docs inexistentes fiscalmente | Anexo II | p.31 | **19193** |
| Anexo III — REST JSON (`registarFactura`, `obterEstado`, `listarFacturas`, `consultarFactura`, `solicitarSerie`, `listarSeries`, `validarDocumento`) | Anexo III | p.32 | **19194** |
| QR Code impresso (URL portal + `documentNo`; ECC 15%; logo AGT <20%) | Anexo III | p.32–33 | **19194–19195** |
| Tabela 1 CAE (referência INE) | Tabelas Anexo I | p.33–50 | **19195–19212** |
| Tabela 2 taxas IEC | Tabelas Anexo I | p.50–56 | **19212–19218** |
| Tabela 3 verbas/taxas IS | Tabelas Anexo I | p.56–61 | **19218–19223** |
| Tabela 4 isenções IVA | Tabelas Anexo I | p.61–64 | **19223–19226** |
| Tabelas 5–6 isenções IS / IEC | Tabelas Anexo I | p.64–65 | **19226–19227** |

Detalhe tipos FE: [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md). Limites: OCR auxiliar (ex.: «ANEXO \|» = Anexo I); **não** inventar `FE-RNG-*`; JWS FE ≠ Hash SAF-T (C-SIGN-001); tabelas fiscais **não** fecham arredondamento/cálculo MVP; **não** confirma AO-*.

### Citação H — documentação oficial FE HML (registar / listar / séries / FE-RNG)

Fontes (`pending_validation`, bytes no privado `c93db4f…`):

| source_id | SHA256 | Uso nesta citação |
|---|---|---|
| `AO-FE-SNAP-HML-2026-07-25-REGISTAR` | `eb430954…` | `POST …/v1/registarFactura`; inventário `FE-RNG-002`…`031` + `061`/`068`/`069`/`073` |
| `AO-FE-SNAP-HML-2026-07-25-SOLICITAR` | `f8fb22e7…` | `solicitarSerie` PRD; `FE-RNG-049`…`060` / `080`–`081`; refs `082`/`083` em `authorizedQuantity` |
| `AO-FE-SNAP-HML-2026-07-25-LISTAR` | `5729f02c…` | `listarSeries` HML+PRD `/v1/listarSeries` |
| `AO-FE-SNAP-HML-2026-07-25-LISTAR-FATURAS` | `c748caca…` | `listarFacturas` (HML com `/ws/` — C-FE-001) |
| `AO-FE-SNAP-HML-2026-07-25-CONSULTAR` | `1cac2844…` | `obterEstado` |
| `AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA` | `6d5cc1a0…` | `consultarFactura` HML+PRD `/v1/` |
| `AO-FE-SNAP-HML-2026-07-25-VALIDAR` | `7ab70629…` | `validarDocumento`; `FE-RNG-033`/`034`/`077` |
| `AO-FE-SNAP-HML-2026-07-25-API` | `06a9dbdf…` | Basic Auth; exemplo HML `registarFactura` |
| `AO-FE-SNAP-HML-2026-07-25-MODELO` | `f851f512…` | Assíncrono: `requestID` + consulta estado |
| `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | `50ba18e0…` | JWS RS256 (já Citação C) |
| `AO-FE-SNAP-HML-2026-07-25-GESTAO` | `de423e66…` | Distinção chaves software vs contribuinte (Citação J) |
| `AO-FE-SNAP-HML-2026-07-25-QRCODE` | `ccade20b…` | QR Model 2 / URL Quiosque AGT |
| `AO-FE-SNAP-HML-2026-07-25-INDEX` | `67fc40a6…` | Índice / arquitectura geral da API |

Inventário completo: [`FE-SERVICES-MATRIX-RM-REQ-001.md`](FE-SERVICES-MATRIX-RM-REQ-001.md). Conflito paths: [C-FE-001](../conflicts/C-FE-001-fe-endpoint-path-inconsistency.md).

Limites: HTML ≠ validação AGT; GAP-006 credenciais; **não** inventar `FE-RNG-001` isolado nem E-codes 082/083; **não** confirma AO-*.

### Citação I — «terminal informático» na identificação da série (DE 74/19)

Fonte: `AO-LEG-DE-74-19-2019` · original sha256 `5b63c80e…` · OCR v1 `reviewed` · PDF p.3 · gazeta **1577**.

Trecho OCR auxiliar (colunas misturadas no OCR; **PDF original prevalece**): menção a «o ano ou o numero do terminal informático, etc.) que, a existir, deverá sempre constar da identificação da série».

Path privado: `derivatives/legislation/AO-LEG-DE-74-19-2019/v1/text.md` (`private_commit` `c8a4e6e8…`).

Limites: isto é exemplo de conteúdo da **identificação da série**, **não** um campo FE de documento `terminal*`; **não** satisfaz o critério completo de `AO-ID-001` (contribuinte + estabelecimento + terminal + software + versão); tipografia/OCR com ruído; **não** confirmado.

### Citação J — gestão de chaves FE (snapshot HML; ≠ fecho GAP-013)

Fonte: `AO-FE-SNAP-HML-2026-07-25-GESTAO` · sha256 `de423e66…` · `pending_validation` · artefacto privado `…/gestao.html`.

Distinção extractada (auxiliar):

| Uso | HTML (resumo) | Implicação provisória |
|---|---|---|
| `jwsSoftwareSignature` | Par RSA gerado pelo **produtor**; privada **não** sai do ambiente do produtor; pública à AGT; RSA ≥ 2048 | Alinha preparação SecAdm/produtor; ≠ custódia contribuinte |
| `jwsDocumentSignature` / `jwsSignature` | HTML afirma emissão pela **AGT** e disponibilidade no portal do contribuinte | **Não** autoriza módulo externo a guardar privada do contribuinte |

Limites: snapshot HML ≠ escrito AGT de custódia; GAP-013 / `DEC-REG-KEY-CUSTODY` / `AO-KEY-001` permanecem abertos; **não** confirmado.

### Citação K — QR Code impresso (DE 683 Anexo III + FE HML)

Fontes:

| source_id | SHA/ref | Uso |
|---|---|---|
| `AO-LEG-DE-683-25-2025` | `b01e4581…` OCR v2 | Anexo III QR @**19194–19195** (ECC 15%; 33×33; URL portal OCR; `%20`; logo &lt;20%) |
| `AO-FE-SNAP-HML-2026-07-25-QRCODE` | `ccade20b…` | Model 2 / v4; URL Quiosque AGT + `emissor`/`document`; PNG 350×350 |

**Conflito de URL/params:** [C-FE-QR-001](../conflicts/C-FE-QR-001-qr-url-de683-vs-fe-hml.md). Mitigação `internal/feqr` (≠ fecho).

Limites: **não** implementa `RM-ENG-007`; OCR URL com ruído tipográfico; HTML ≠ aceite AGT; **não** confirmado.

## Linhas (rascunho)

| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | Premissa de produto; confirmação AGT em aberto (RM-FOUND-005) |
| AO-ID-001 | `partial` | DE 683 + DP 71 + DE 74/19 + FE HML | Ligação preliminar: NIF/software @**19166–19167**; estabelecimento Art.24 @**11914** + DE74 @**1577** + FE `establishmentNumber` / **FE-RNG-080** (Citação H). Terminal: OCR DE74 @**1577** menciona «terminal informático» na identificação da série (Citação I) — **não** é campo FE de documento; lacuna estabelecimento/terminal no critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-DOC-001 | `scaffold` | DP 71 + DE 683 + FE + SAF-T + DE 74/19 | L1 Art.10 + L4 `documentType` @**19169–19170** + L2/L3 SAF-T + classes 74/19 @**1576–1577**; **não** confundir camadas; falta DEC-REG-003 + C-DOC-* (incl. C-DOC-004…010); critério **não** fica satisfeito; **não** confirmado |
| AO-DOC-002 | `promoted` | DP 71 + DE 74/19 (+ Rect.) → [`CONFIRMED-MATRIX-RM-REQ-001.md`](CONFIRMED-MATRIX-RM-REQ-001.md) | **`confirmed_normative`** 2026-07-29 (Art.3 n) @**11904**; Art.8 n.º4–5/8–9 @**11907**; DE74 n.º3 @**1576** + n.º4 l) @**1577**). Residual: teste de imutabilidade engenharia. **≠** homologação AGT / JWS FE |
| AO-SEQ-001 | `promoted` | DP 71 + DE 74/19 (+ Rect.) → [`CONFIRMED-MATRIX-RM-REQ-001.md`](CONFIRMED-MATRIX-RM-REQ-001.md) | **`confirmed_normative`** 2026-07-29 (Art.3 n) @**11904**; Art.10 b) @**11908**; DE74 sequência contínua / unívoca @**1577**). Residual: unicidade concorrente engenharia. **≠** `AO-SEQ-002` / AGT |
| AO-SEQ-002 | `partial` | DE 683 + DP 71 + FE HML | Ligação preliminar: ART.4 @**19164** + `solicitarSerie` @**19183–19184** + FE `solicitarSerie`/`listarSeries` + FE-RNG-051/053/055–060 (Citação H); C-FE-001 paths abertos. Critério («POS não atribui o número fiscal final») **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-IDEM-001 | `scaffold` | — | Arquitectura de API / produto; não derivado só de legislação (`submissionGUID` FE @**19166** é candidato, não fecho) |
| AO-TAX-001 | `partial` | DE 683 Anexo I + Tabelas 2–6 | Ligação preliminar: `taxType`/`taxCode`/`taxExemptionCode` @**19171–19173** + Tabelas 2–6 @**19212–19227** (Citação G). Arredondamento/cálculo MVP e DEC-REG-003 **não** fechados; critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-CRYPTO-001 | `partial` | DE 683 + FE HML (+ DE 74/19 só como contraste) | Ligação preliminar FE: `jwsDocumentSignature` @**19168** + RS256 snapshot (`pending_validation`). Encadeamento Hash SAF-T n.º34 @**1582–1584** é **outro** mecanismo (C-SIGN-001 + `internal/signsep`); **não** misturar. Critério do catálogo **não** fica satisfeito só com estas fontes. **Não** confirmado |
| AO-KEY-001 | `blocked` | GAP-013 + Citação J | Snapshot GESTAO distingue chaves software vs contribuinte; **não** autoriza custódia por módulo externo; GAP-013 aberto |
| AO-AGT-001 | `pending_validation` | FE HML snapshots (Citação H) | Endpoints + FE-RNG + Basic Auth citados (`eb430954…`/`f8fb22e7…`/…); C-FE-001 mitigação `internal/fepath` (≠ fecho) + GAP-006 credenciais abertos; critério do catálogo **não** fica satisfeito só com snapshots; **não** confirmado |
| AO-AGT-002 | `partial` | FE MODELO + DE 683 | Ligação preliminar: assíncrono `requestID` + `obterEstado` (`f851f512…`, Citação H) + validação a posteriori Anexo II @**19193**. DEC-API-004 / estados completos **não** fechados; critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-OFF-001 | `partial` | DP 71 + DE 683 | Ligação preliminar: Art.18 @**11911–11912** + FE `obterEstado`/`validationStatus` P (atraso >24h sem contingência) @**19179**/**19183**. Regras Edge/produto e DEC-REG-004 **não** fechadas; critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-OFF-002 | `partial` | `AO-LEG-DE-74-19-2019` | Ligação preliminar: Anexo I n.º7–9 @**1580** (e **1579**; integração sem recalcular; séries de recuperação/contingência). Sync Edge/produto **não** fechado; critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-AUD-001 | `scaffold` | — | Produto: append-only + retenção `pending_norm` (`DEC-PROD-013`); **não** confirmado; critério do catálogo **não** fica satisfeito só com decisão de produto |
| AO-SAF-001 | `pending_validation` | XSD + DE 74/19 n.º1 | Citação D: `AuditFile`/`SalesInvoices`/`InvoiceNo`/`InvoiceType`/`Hash` (`e9a938e1…`, L42–58 / L445+ / L1974–2001 / L2023–2065 / L1361–1367) + exportação Anexo I n.º1 @**1576**; AGT pendente; critério do catálogo **não** fica satisfeito só com schema+citação; **não** confirmado |
| AO-SAF-002 | `pending_validation` | DE 74/19 + XSD | Citação D: `InvoiceStatus=A` (L2003–2021) + `References` p/ NC (L1004–1023) + DE74 @**1577**; L2 `PaymentType` RC/RG em L3 `Payments` ≠ L4 FE nem L2 `InvoiceType`; AGT pendente; critério **não** fica satisfeito só com estas citações; **não** confirmado |
| AO-OPS-001 | `scaffold` | — | Ops/DR |
| AO-UPD-001 | `scaffold` | — | Updates Edge assinados |

## Próximos passos (este item)

1. C-FE-001: mitigação fail-closed (`internal/fepath`) 2026-07-28 — **residual** confirmação AGT dos paths + GAP-006; manter `AO-AGT-001` `pending_validation` (VALIDAR/CONSULTAR-FATURA alinhados **não** fecham o conflito).
2. C-FE-QR-001: URL QR DE 683 vs FE HML — mitigação `internal/feqr` 2026-07-28; **não** fechar; **não** `RM-ENG-007`.
3. AO-SAF-001/002: validação AGT do XSD + vetores dourados — manter `pending_validation`.
4. C-DOC-003: mitigação fail-closed (seed + `doctype.CheckCDOC003Invariants`) documentada 2026-07-28 — **residual** `DEC-REG-003` / dual-stack; **não** marcar resolvido.
5. C-DOC-004: homónimo `AR` em `InvoiceType` **e** `PaymentType` — mitigação `CheckCDOC004Invariants` 2026-07-28; dual canónicos `off`; **residual** «grupo único por emissão»; **não** marcar resolvido.
6. C-DOC-005: homónimos segurador `RP`/`RE`/`CS`/`LD`/`RA` em `InvoiceType` **e** `WorkType` — mitigação `CheckCDOC005Invariants` 2026-07-28; seed FE só `InvoiceType`/`off`; **não** inventar FE→WorkType; **não** marcar resolvido.
7. C-DOC-006: homónimo `RC` em `PaymentType` **e** `PurchaseType` — mitigação `CheckCDOC006Invariants` 2026-07-28; dual canónicos `pagamentos.rc`/`compras.rc` `off`; **não** marcar resolvido.
8. C-DOC-007: homónimo `GR` em `MovementType` **e** `WorkType` — mitigação `CheckCDOC007Invariants` 2026-07-29; dual canónicos `movimentacao.gr`/`conferencia.gr` `off` (SAF-T-only); **não** inventar FE L4; **não** marcar resolvido.
9. C-DOC-008: homónimos `FT`/`NC` em `InvoiceType` **e** `PurchaseType` — mitigação `CheckCDOC008Invariants` 2026-07-29; vendas `on` (DEC-REG-003) ≠ compras `off`; seed `compras.nc`; **não** marcar resolvido.
10. C-DOC-009: `AR` também em `PurchaseType` (3.º L3) — mitigação `CheckCDOC009Invariants` 2026-07-29; seed `compras.ar`; **não** colapsar com C-DOC-004; **não** marcar resolvido.
11. C-DOC-010: homónimos restantes `FR`/`GF`/`FG`/`AC`/`AF`/`TV` em `InvoiceType` **e** `PurchaseType` — mitigação `CheckCDOC010Invariants` 2026-07-29; seeds compras; ambos `off`; **não** marcar resolvido.
12. Inventário `WorkType` (não dual / não segurador): seeds `conferencia.{cm,cc,nr,fo,ou,pp,gc}` + `CheckWorkTypeInventoryInvariants` 2026-07-29; **sem** FE L4; **sem** seed WorkType segurador (C-DOC-005); **sem** AO-*.
13. C-DOC-001: confronto visual p.7 (2026-07-28) confirma `GF` **ausente** no DE 683; presente HTML/XSD — `documentado_divergencia`; manter `GF` `conflito`/`off`.
14. C-DOC-002: rótulos `RG` documentados (mesmo código) — sem terceiro código.
15. C-SIGN-001: mitigação fail-closed (`internal/signsep`) 2026-07-28 — **residual** implementação n.º34 / JWS FE oficial + `AO-CRYPTO-001`; **não** marcar resolvido.
16. AO-DOC-001: fechar C-DOC-* residual + DEC-REG-003 — manter `scaffold`.
17. AO-DOC-002: **promovido** `confirmed_normative` (2026-07-29) — ver CONFIRMED-MATRIX; residual teste imutabilidade; **≠** AGT/JWS.
18. AO-SEQ-001: **promovido** `confirmed_normative` (2026-07-29) — sequência por série; residual concorrência; **≠** `AO-SEQ-002`.
19. AO-ID-001: Citação I (terminal na série) **não** fecha o critério; manter `partial`.
20. AO-KEY-001: Citação J (GESTAO) **não** fecha GAP-013 — manter `blocked`.
21. Não inventar `FE-RNG-*`; não alargar OpenAPI sem DEC-REG-003; não pôr `ConflictOpen=false` (fepath/feqr) sem fecho AGT; não promover AO-* sem evidência inequívoca.

## Referências

- Catálogo público: [`compliance/catalog/sources.yaml`](../../catalog/sources.yaml)
- Catálogo inicial de IDs: [`docs/01-compliance/requirements-catalog.md`](../../../docs/01-compliance/requirements-catalog.md)
- Gaps: [`docs/01-compliance/regulatory-gaps.md`](../../../docs/01-compliance/regulatory-gaps.md)
- Aquisição Rect. (privado): `storesace-cv/bwb-fiscal-sources-ao` → `docs/ACQUISITION-RECT-10-19.md`
- Tipos documentais: [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md)
- Serviços FE / FE-RNG: [`FE-SERVICES-MATRIX-RM-REQ-001.md`](FE-SERVICES-MATRIX-RM-REQ-001.md)
- Verificador provisório: [`compliance/scripts/verify_provisional_matrix.py`](../../scripts/verify_provisional_matrix.py)
- Matriz confirmada: [`CONFIRMED-MATRIX-RM-REQ-001.md`](CONFIRMED-MATRIX-RM-REQ-001.md) · [`verify_confirmed_matrix.py`](../../scripts/verify_confirmed_matrix.py)
