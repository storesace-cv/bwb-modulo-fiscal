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

### Citação E — DP 71/25 (regime de facturas; tipos e requisitos)

Fonte: `AO-LEG-DP-71-25-2025` · sha256 `4931fd3ce711ef2b22e7316c3dd296d8c7c81993c88e29c70e09baeb0d0e7f76` · OCR v1 `reviewed` · citar só gazeta **11902–11920**.

| Tema | Norma | PDF | Gazeta |
|---|---|---|---|
| Definições de tipos (Factura, FR, FG, FA, NC, ND, Recibo, Talão, …) | Art.3 | p.3–4 | 11903–11904 |
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

## Linhas (rascunho)

| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | Premissa de produto; confirmação AGT em aberto (RM-FOUND-005) |
| AO-ID-001 | `partial` | DE 683 + DP 71 + DE 74/19 | Ligação preliminar: NIF/software @**19166–19167**; estabelecimento Art.24 @**11914** + séries por estabelecimento Anexo I n.º4 i) @**1577**. Terminal **não** coberto; critério do catálogo **não** fica satisfeito só com estes campos. **Não** confirmado |
| AO-DOC-001 | `scaffold` | DP 71 + DE 683 + FE + SAF-T + DE 74/19 | Art.10 DP71 + classes 74/19 n.º4 @**1576–1577**; falta mapeamento validador + C-DOC-*; critério do catálogo **não** fica satisfeito; **não** confirmado |
| AO-DOC-002 | `partial` | DP 71 + DE 74/19 (+ Rect.) | Ligação preliminar: DP71 Art.3 n)/Art.8 @**11904**/**11907** + DE74 Anexo I n.º3 e n.º4 l) @**1577** (sem alterar doc assinado; ver também **1576**). Critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-SEQ-001 | `partial` | DP 71 + DE 74/19 | Ligação preliminar: DP71 Art.10 b) @**11908** + DE74 n.º4 g)–i) @**1577** (séries univocas / numeração contínua). Unicidade concorrente **não** demonstrada; critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-SEQ-002 | `partial` | `AO-LEG-DE-683-25-2025` + `AO-LEG-DP-71-25-2025` | Ligação preliminar: DE 683 ART.4 @**19164** (séries AGT) + DP 71 Art.10 b)/Art.24 @**11908**/**11914** (séries por tipo / comunicação). O critério («POS não atribui o número fiscal final») **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-IDEM-001 | `scaffold` | — | Arquitectura de API / produto; não derivado só de legislação |
| AO-TAX-001 | `scaffold` | Rect. / 74/19 / fontes oficiais | Fontes OCR disponíveis; cálculo fiscal exige citação página a página — **não** confirmado |
| AO-CRYPTO-001 | `partial` | DE 683 + FE HML (+ DE 74/19 só como contraste) | Ligação preliminar FE: `jwsDocumentSignature` @**19168** + RS256 snapshot (`pending_validation`). Encadeamento Hash SAF-T n.º34 @**1582–1584** é **outro** mecanismo (C-SIGN-001); **não** misturar. Critério do catálogo **não** fica satisfeito só com estas fontes. **Não** confirmado |
| AO-KEY-001 | `blocked` | GAP-013 | Custódia de chave contribuinte em aberto |
| AO-AGT-001 | `blocked` | FE HML/PRD oficiais | Credenciais/docs oficiais AGT pendentes |
| AO-AGT-002 | `scaffold` | — | Máquina de estados — DEC-API-004 |
| AO-OFF-001 | `partial` | `AO-LEG-DP-71-25-2025` | Ligação preliminar: Art.18 @**11911–11912** (offline / tipografia Art.7 n.º6) + menção contingência. Regras Edge/produto e DEC-REG-004 **não** fechadas; critério do catálogo **não** fica satisfeito só com Art.18. **Não** confirmado |
| AO-OFF-002 | `partial` | `AO-LEG-DE-74-19-2019` | Ligação preliminar: Anexo I n.º7–9 @**1580** (e **1579**; integração sem recalcular; séries de recuperação/contingência). Sync Edge/produto **não** fechado; critério do catálogo **não** fica satisfeito só com estas citações. **Não** confirmado |
| AO-AUD-001 | `scaffold` | — | Auditoria append-only — arquitectura |
| AO-SAF-001 | `pending_validation` | XSD + DE 74/19 n.º1 | XSD (`e9a938e1…`) + obrigação exportação Anexo I n.º1 @**1576** (Rect. corrige SAF-T AO); validação AGT pendente; critério do catálogo **não** fica satisfeito só com schema+citação; **não** confirmado |
| AO-SAF-002 | `pending_validation` | DE 74/19 n.º4 e) + XSD | Rectificativos com `References` @**1577**; validação AGT / regras anulação completas pendentes; **não** confirmado |
| AO-OPS-001 | `scaffold` | — | Ops/DR |
| AO-UPD-001 | `scaffold` | — | Updates Edge assinados |

## Próximos passos (este item)

1. Confrontar PDF DE 74 p.2–10 e Rect. p.2–3 nas citações F (OCR ruidoso).
2. AO-DOC-001 / AO-TAX-001: mapear validadores — manter `scaffold`.
3. Separar implementação crypto FE vs SAF-T (C-SIGN-001) — manter `AO-CRYPTO-001` `partial`.
4. AO-ID-001: terminal ainda em falta — manter `partial`.
5. Não promover nenhuma linha a confirmado; não alargar OpenAPI sem DEC-REG-003.
6. Actualizar DEC-REG-002 (arquivo obtido; fecho AO-* ainda aberto).

## Referências

- Catálogo público: [`compliance/catalog/sources.yaml`](../../catalog/sources.yaml)
- Catálogo inicial de IDs: [`docs/01-compliance/requirements-catalog.md`](../../../docs/01-compliance/requirements-catalog.md)
- Gaps: [`docs/01-compliance/regulatory-gaps.md`](../../../docs/01-compliance/regulatory-gaps.md)
- Aquisição Rect. (privado): `storesace-cv/bwb-fiscal-sources-ao` → `docs/ACQUISITION-RECT-10-19.md`
- Tipos documentais: [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md)
- Verificador: [`compliance/scripts/verify_provisional_matrix.py`](../../scripts/verify_provisional_matrix.py)
