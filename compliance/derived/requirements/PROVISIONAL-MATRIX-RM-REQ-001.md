# Matriz provisória RM-REQ-001

**Estado:** `EM_CURSO` — **não** é matriz AO-* confirmada.
**Data:** 2026-07-26
**Âmbito:** OCR `reviewed` (KB auxiliar) + XSD ASSOFT / snapshots FE `pending_validation` — **não** há AO-* confirmados.

## Fontes admitidas neste rascunho

| source_id | Estado OCR / schema | Uso permitido aqui | Limite |
|---|---|---|---|
| `AO-LEG-DE-74-19-2019` | `reviewed` v1 (`5b63c80e…`, 12p) | Pesquisa auxiliar; citação futura com página do PDF original | Texto consolidado 74+Rect. exige citação página a página; **não** confirmado |
| `AO-LEG-RECT-10-19-2019` | `reviewed` v2 (`b3db14e2…`, 3p) | Pesquisa auxiliar; gazeta **1948–1949** (PDF p.2–3); capa PDF p.1 | v1 incorrecta `77b77f01…` só diagnóstico privado (≠ KB); **não** confirmar AO-* só com OCR |
| `AO-LEG-DE-683-25-2025` | `reviewed` v2 (`b01e4581…`, 66p) | Pesquisa auxiliar; citar **apenas** gazeta 19164–19227 (PDF p.2–65) | PDF p.66 = Aviso BNA 4/25 @19228 — **não** citar como DE 683 |
| `AO-LEG-DP-71-25-2025` | `reviewed` v1 (`4931fd3c…`, 21p) | Pesquisa auxiliar; citar **apenas** gazeta **11902–11920** (PDF p.2–20) | PDF p.21 = DE 372/25 @11921 — **não** citar como DP 71/25 |
| `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | HTML `pending_validation` (`50ba18e0…`) | Algoritmo JWS/RS256 FE (≠ SAF-T) | Snapshot HML; não validado AGT; sem inventar campos |
| `AO-SAFT-XSD-1.01_01` | schema `pending_validation` (`e9a938e1…`) | Referência técnica ASSOFT MIT | Não afirmado como validado pela AGT |

## Bloqueios explícitos (fail-closed)

1. **RM-SRC-004 / RM-M2-C** — OCR do conjunto 74+Rect+683 `reviewed` (Rect. v2); DP 71/25 também `reviewed` — gate de fontes OCR fechado; **não** equivale a AO-* confirmados.
2. Esta matriz **não** inventa regras fiscais e **não** promove linhas a «confirmado».
3. `reviewed` OCR é KB auxiliar; só após citação página a página + revisão compliance pode sustentar AO-* confirmados (ainda não neste PR).
4. `partial` ≠ confirmado: ligação preliminar sujeita a revisão; o critério de aceitação do catálogo pode ser mais amplo do que a citação.
5. GAP-002 residual: URL oficial estável da Rect. / Imprensa Nacional ainda pode estar `pending_validation`; histórico rejected permanece excluído da KB.

## Legenda de estado da linha

| Estado | Significado |
|---|---|
| `scaffold` | ID do catálogo inicial; sem ligação normativa página a página; **não** confirmado |
| `partial` | Ligação preliminar a fonte `reviewed` com página candidata; sujeita a revisão; **não** confirmado |
| `blocked` | Dependente de fonte oficial ainda inacessível (ex.: GAP-013, credenciais AGT) |
| `pending_validation` | Artefacto técnico presente sem validação AGT |

Estados proibidos neste ficheiro: `confirmed`, `confirmado`, `validated_agt`.

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

### Citação D — schema SAF-T AO (AO-SAF-001)

Fonte: `AO-SAFT-XSD-1.01_01` · ficheiro versionado [`compliance/saft-ao/schemas/SAFTAO1.01_01.xsd`](../../saft-ao/schemas/SAFTAO1.01_01.xsd) · sha256 `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631` · estado catálogo **`pending_validation`**.

Identidade técnica (não normativa AGT): `targetNamespace` `urn:OECD:StandardAuditFile-Tax:AO_1.01_01` · `version` `1.01_01` · elemento raiz `AuditFile`.

Limite: presença do XSD ASSOFT **não** equivale a validação AGT nem a conformidade de exportação; critério do catálogo **não** fica satisfeito só com o schema versionado; **não** confirmado.

## Linhas (rascunho)

| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | Premissa de produto; confirmação AGT em aberto (RM-FOUND-005) |
| AO-ID-001 | `partial` | `AO-LEG-DE-683-25-2025` | Ligação preliminar: `taxRegistrationNumber` @**19166** (PDF p.4) + `softwareValidationNumber`/`productVersion` @**19167** (PDF p.5). Estabelecimento/terminal **não** cobertos nesta citação; critério do catálogo **não** fica satisfeito só com estes campos. **Não** confirmado |
| AO-DOC-001 | `scaffold` | DP 71/25 + DE 683/25 + FE HML + SAF-T XSD | Inventário de tipos com citações em [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md); campos obrigatórios por tipo **ainda** incompletos; critério do catálogo **não** fica satisfeito; **não** confirmado |
| AO-DOC-002 | `scaffold` | DE 74/19 + Rect. 10/19 v2 | Imutabilidade/regras de emissão: citação consolidada pendente — **não** confirmado |
| AO-SEQ-001 | `scaffold` | DE 74/19 (+ Rect.) / DE 683/25 | Sequencialidade: split formal de fontes e citação página a página pendentes — **não** confirmado |
| AO-SEQ-002 | `partial` | `AO-LEG-DE-683-25-2025` | Ligação preliminar: ART. 4.º / gazeta **19164** (PDF p.2) — séries FE geradas pela AGT. O critério do catálogo («POS não atribui o número fiscal final») **não** fica satisfeito só com esta citação; cruzamento com DP 71/25 (`reviewed`, 11902–11920) ainda sem citação página a página. **Não** confirmado |
| AO-IDEM-001 | `scaffold` | — | Arquitectura de API / produto; não derivado só de legislação |
| AO-TAX-001 | `scaffold` | Rect. / 74/19 / fontes oficiais | Fontes OCR disponíveis; cálculo fiscal exige citação página a página — **não** confirmado |
| AO-CRYPTO-001 | `partial` | `AO-LEG-DE-683-25-2025` + `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | Ligação preliminar: `jwsDocumentSignature` @**19168** (PDF p.6) + RS256 no snapshot FE (`pending_validation`). JWS FE ≠ SAF-T; encadeamento **não** citado; critério do catálogo **não** fica satisfeito só com estas fontes. **Não** confirmado |
| AO-KEY-001 | `blocked` | GAP-013 | Custódia de chave contribuinte em aberto |
| AO-AGT-001 | `blocked` | FE HML/PRD oficiais | Credenciais/docs oficiais AGT pendentes |
| AO-AGT-002 | `scaffold` | — | Máquina de estados — DEC-API-004 |
| AO-OFF-001 | `blocked` | Contingência oficial | Fonte oficial de contingência pendente |
| AO-OFF-002 | `scaffold` | — | Sync sem renumerar — produto + legal |
| AO-AUD-001 | `scaffold` | — | Auditoria append-only — arquitectura |
| AO-SAF-001 | `pending_validation` | `AO-SAFT-XSD-1.01_01` | XSD versionado (`e9a938e1…`, `AuditFile`, NS AO_1.01_01); validação AGT pendente; critério do catálogo **não** fica satisfeito só com o schema; **não** confirmado |
| AO-SAF-002 | `pending_validation` | XSD + legislação completa | Anulados/retificativos — citação página a página + validação AGT pendentes |
| AO-OPS-001 | `scaffold` | — | Ops/DR |
| AO-UPD-001 | `scaffold` | — | Updates Edge assinados |

## Próximos passos (este item)

1. Completar AO-ID-001 com fontes que cubram estabelecimento/terminal (sem inventar; DE 74/19+Rect. v2 ou docs oficiais) — manter `partial`.
2. Cruzar AO-SEQ-002 com DP 71/25 (gazeta 11902–11920; não citar p.21/DE 372/25); manter `partial`.
3. AO-CRYPTO-001: confrontar lista de campos assinados PDF↔snapshot FE; manter `partial` até revisão compliance (sem inventar encadeamento).
4. AO-SAF-001: manter `pending_validation` até validação AGT; não promover a `partial`/`confirmado` só com XSD ASSOFT.
5. Fechar conflitos C-DOC-001/002/003 (GF OCR, rótulo RG, FE↔SAF-T) e completar Art.10 DP 71 página a página para campos por tipo — manter AO-DOC-001 `scaffold` até critérios do catálogo.
6. Extrair citações para AO-SEQ-001 / AO-TAX-001 / AO-DOC-002 (imutabilidade) sem promover a confirmado.
7. Não promover nenhuma linha a confirmado sem revisão de compliance + critérios testáveis; não alargar OpenAPI `document_type` sem DEC-REG-003.
8. Fechar RM-REQ-001 só com matriz rastreável e gate de revisão compliance (OCR fechado; confirmação AO-* ainda aberta).

## Referências

- Catálogo público: [`compliance/catalog/sources.yaml`](../../catalog/sources.yaml)
- Catálogo inicial de IDs: [`docs/01-compliance/requirements-catalog.md`](../../../docs/01-compliance/requirements-catalog.md)
- Gaps: [`docs/01-compliance/regulatory-gaps.md`](../../../docs/01-compliance/regulatory-gaps.md)
- Aquisição Rect. (privado): `storesace-cv/bwb-fiscal-sources-ao` → `docs/ACQUISITION-RECT-10-19.md`
- Tipos documentais: [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md)
- Verificador: [`compliance/scripts/verify_provisional_matrix.py`](../../scripts/verify_provisional_matrix.py)
