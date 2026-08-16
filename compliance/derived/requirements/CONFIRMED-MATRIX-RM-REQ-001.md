# Matriz de requisitos AO-* confirmados (RM-REQ-001)

**Estado:** `EM_CURSO` — contém **apenas** requisitos cuja obrigação normativa foi confirmada por citação página a página sobre OCR `reviewed` confrontado com o PDF original.
**Data:** 2026-07-29
**Precedência:** [`compliance/POLICY.md`](../../POLICY.md) · arquivo privado `storesace-cv/bwb-fiscal-sources-ao`
**Não é:** aceitação AGT, homologação oficial, autorização de produção, nem fecho de `DEC-REG-003` / `C-DOC-001` / custódia JWS.

## Regras de promoção

1. Promover só quando o texto normativo (PDF original; OCR `reviewed` como auxiliar) sustenta **inequivocamente** o critério do catálogo **no âmbito declarado**.
2. Distinguir sempre: **regime jurídico facturas (DP 71/25)** · **validação de software / SAF-T (DE 74/19 + Rect.)** · **facturação electrónica (DE 683/25 + FE)** · **homologação** · **produção**.
3. Ambiguidade ou conflito → manter fail-closed na [matriz provisória](PROVISIONAL-MATRIX-RM-REQ-001.md); **não** inventar regra.
4. Confirmação normativa **≠** Definition of Done de engenharia (testes, API, Edge) e **≠** certificado AGT.

## Legenda de estado

| Estado | Significado |
|---|---|
| `confirmed_normative` | Obrigação normativa confirmada no âmbito declarado; residual de engenharia/AGT pode permanecer |

Estados proibidos aqui: `validated_agt`, `homologado`, `produção_autorizada`.

## Requisitos confirmados

### AO-DOC-002 — Impedir alteração destrutiva após emissão

| Campo | Valor |
|---|---|
| Estado | `confirmed_normative` |
| Critério do catálogo | Impedir alteração destrutiva após emissão |
| Âmbito confirmado | Softwares de facturação sujeitos ao Regime Jurídico das Facturas (DP 71/25) e aos requisitos de validação de software do DE 74/19 (+ Rect. 10/19 como conjunto) |
| Canais | **Regime jurídico** (DP 71) + **validação software / SAF-T** (DE 74) |
| Fora de âmbito | Homologação AGT; produção FE; inventário de tipos (`DEC-REG-003`); custódia de chaves JWS; XSD ASSOFT como aceite AGT |
| Residual engenharia | Teste de imutabilidade / append-only no módulo (evidência esperada do catálogo) — **não** bloqueia a confirmação normativa |

#### Evidência inequívoca

| # | source_id | Original sha256 | OCR | PDF p. | Gazeta | Norma | Trecho (OCR auxiliar; PDF prevalece) |
|---|---|---|---|---:|---:|---|---|
| 1 | `AO-LEG-DP-71-25-2025` | `4931fd3ce711ef2b22e7316c3dd296d8c7c81993c88e29c70e09baeb0d0e7f76` | v1 `reviewed` (`reviewer_kind=ai_assisted`, `reviewed_against_original=true`) | 4 | **11904** | Art.3 n) definição «Software de Facturação» | «…garante a numeração sequencial e cronológica dos documentos e que **não permite a respectiva eliminação após a sua emissão**» |
| 2 | `AO-LEG-DP-71-25-2025` | idem | idem | 7 | **11907** | Art.8 n.º4–5 | «As facturas devem ser **anuladas ou rectificadas por notas de crédito**…»; NC com «anulação»/«rectificação» + identificação do documento |
| 3 | `AO-LEG-DE-74-19-2019` | `5b63c80e358bd5eda60302a5f6d2adac3c23815de7fc8f496b0b8bdb909d9abd` | v1 `reviewed` | 2 | **1576** | Anexo I n.º**3** | Sistemas **não** dispõem de funções que permitam **alterar** informação de natureza fiscal, directa ou indirectamente, **sem gerar evidência** agregada à informação original |
| 4 | `AO-LEG-DE-74-19-2019` | idem | idem | 3 | **1577** | Anexo I n.º4 **l)** | A aplicação **não pode permitir** que num documento **já assinado** seja **alterada qualquer informação** |
| 5 | `AO-LEG-RECT-10-19-2019` | `b3db14e2715541be00ac93032718e7a358b493796264e361d1f9d35a1a49e014` | v2 `reviewed` | 2–3 | **1948–1949** | Conjunto 74+Rect. | Rect. integra o conjunto normativo POLICY; **não** altera o sentido de n.º3 / n.º4 l) acima |

Paths privados (consulta; **não** dependência de build):
`derivatives/legislation/AO-LEG-DP-71-25-2025/v1/text.md` · `…/AO-LEG-DE-74-19-2019/v1/text.md` · `…/AO-LEG-RECT-10-19-2019/v2/text.md`.

#### Excepção normativa explícita (não é edição destrutiva livre)

DP 71/25 Art.8 n.º**8–9** (@**11907**): em casos de rectificação dos requisitos da alínea a) do Art.10 n.º1 (e facturas não enviadas ao adquirente), a factura **pode ser anulada** pelas funcionalidades do software validado **sem** emissão de nota de crédito.
**Fail-closed:** isto **não** autoriza edição livre do conteúdo após assinatura/emissão; permanece anulação/rectificação pelos meios legalmente previstos.

#### Distinções obrigatórias

| Tema | Relação com AO-DOC-002 |
|---|---|
| JWS FE (`jwsDocumentSignature`, RS256) | Mecanismo **FE** (DE 683 / snapshots HML) — **não** usado nesta confirmação; ver `AO-CRYPTO-001` (ainda não confirmado) e [C-SIGN-001](../conflicts/C-SIGN-001-saft-rsa-vs-fe-jws.md) |
| Hash / RSA-SHA1 SAF-T (DE 74 n.º34) | Encadeamento **SAF-T** — distinto; **não** misturar com imutabilidade pós-emissão aqui confirmada |
| Homologação / produção AGT | **Não** confirmados |
| `documentStatus` FE A/C (@19168) | Candidato auxiliar FE; **não** necessário à confirmação (DP71+DE74 bastam) |

#### Conclusão

**Facto:** o critério «impedir alteração destrutiva após emissão» está normativamente sustentado: proibição de eliminação pós-emissão (DP 71 Art.3 n)), proibição de alteração de documento assinado / de informação fiscal sem evidência (DE 74 n.º3 e n.º4 l)), e via de correcção por NC (DP 71 Art.8 n.º4–5), com a excepção estreita Art.8 n.º8–9 documentada.
**Não se afirma:** conformidade do produto implementado, aceitação AGT, nem fecho de outros `AO-*`.

### AO-SEQ-001 — Numeração sequencial (e identificação unívoca) por série

| Campo | Valor |
|---|---|
| Estado | `confirmed_normative` |
| Critério do catálogo | Garantir numeração única e sequencial por série |
| Âmbito confirmado | Obrigação de **numeração sequencial e cronológica** por tipo de documento / série (DP 71) e de numeração **progressiva e contínua** dentro de cada série, com identificação **unívoca** do documento (DE 74) |
| Canais | **Regime jurídico** (DP 71) + **validação software / SAF-T** (DE 74 + Rect.) |
| Fora de âmbito | Homologação AGT; produção FE; atribuição de séries pela AGT (`AO-SEQ-002` / DE 683 Art.4 — **não** misturar); `DEC-REG-003` |
| Residual engenharia | Unicidade sob **concorrência** / recuperação — **fechado** por testes RM-ENG-002 (`AO-SEQ-001_concurrent_continuous_sequence`, VS-T06/T07); ver [`docs/01-compliance/ao-seq-001-engine.md`](../../../docs/01-compliance/ao-seq-001-engine.md). **≠** conformidade AGT |

#### Evidência inequívoca

| # | source_id | Original sha256 | OCR | PDF p. | Gazeta | Norma | Trecho (OCR auxiliar; PDF prevalece) |
|---|---|---|---|---:|---:|---|---|
| 1 | `AO-LEG-DP-71-25-2025` | `4931fd3c…` | v1 `reviewed` | 4 | **11904** | Art.3 n) | Software certificado «garante a **numeração sequencial e cronológica** dos documentos…» |
| 2 | `AO-LEG-DP-71-25-2025` | idem | idem | 8 | **11908** | Art.10 n.º1 **b)** | «**Numeração sequencial e cronológica** por tipo de documento e o respectivo ano económico, podendo ser utilizadas uma ou mais **séries** devidamente identificadas» |
| 3 | `AO-LEG-DE-74-19-2019` | `5b63c80e…` | v1 `reviewed` | 3 | **1577** | Anexo I n.º4 (séries / sequência) | Documentos «**numerados de forma progressiva e contínua**, dentro de cada série»; identificação para «identificar **univocamente** cada documento emitido» |
| 4 | `AO-LEG-RECT-10-19-2019` | `b3db14e2…` | v2 `reviewed` | 2–3 | **1948–1949** | Conjunto 74+Rect. | Rect. integra o conjunto; **não** derroga a sequência contínua por série |

Paths privados: `derivatives/legislation/AO-LEG-DP-71-25-2025/v1/text.md` · `…/AO-LEG-DE-74-19-2019/v1/text.md` · `…/AO-LEG-RECT-10-19-2019/v2/text.md`.

#### Distinções obrigatórias

| Tema | Relação com AO-SEQ-001 |
|---|---|
| Séries geradas pela AGT (DE 683 Art.4 @19164; `solicitarSerie`) | **`AO-SEQ-002`** / FE — **não** confirmado aqui |
| Unicidade concorrente no módulo | Residual de **engenharia** (testes); a norma exige sequência contínua / identificação unívoca, não um algoritmo de locking |
| Homologação / produção AGT | **Não** confirmados |
| Facturação electrónica `documentNo` (@19167) | Alinhamento de formato FE/SAF-T — auxiliar; **não** necessário à confirmação DP71+DE74 |

#### Conclusão

**Facto:** a obrigação de numeração sequencial/cronológica por tipo e série (DP 71) e progressiva/contínua com identificação unívoca por série (DE 74) está normativamente sustentada.
**Fail-closed:** **não** se afirma que o POS/AGT atribuam o número (`AO-SEQ-002` permanece provisório). Residual de concorrência: evidência em RM-ENG-002 / [`ao-seq-001-engine.md`](../../../docs/01-compliance/ao-seq-001-engine.md).
**Não se afirma:** conformidade do produto perante a AGT nem aceitação/homologação.

### AO-OFF-001 — Contingência apenas nas condições autorizadas

| Campo | Valor |
|---|---|
| Estado | `confirmed_normative` |
| Critério do catálogo | Emitir em contingência apenas nas condições autorizadas |
| Âmbito confirmado | Contingência do **Regime Jurídico das Facturas** (DP 71/25 Art.18 + remissão Art.7 n.º6): só perante inoperacionalidade que impossibilite facturação electrónica, nos modos a)/b) e com deveres de menção, submissão posterior e comunicação à AGT |
| Canais | **Regime jurídico** (DP 71) — facturação electrónica / contingência |
| Fora de âmbito | Homologação AGT; produção; regras Edge/`DEC-REG-004`; inventário FE `validationStatus` P (DE 683) como fecho autónomo; SAF-T Hash |
| Residual engenharia | Operacionalização Edge/cloud, reconciliação e testes de falha (evidência do catálogo) — **não** alarga as condições legais |

#### Evidência inequívoca

| # | source_id | Original sha256 | OCR | PDF p. | Gazeta | Norma | Trecho (OCR auxiliar; PDF prevalece) |
|---|---|---|---|---:|---:|---|---|
| 1 | `AO-LEG-DP-71-25-2025` | `4931fd3c…` | v1 `reviewed` | 11–12 | **11911–11912** | Art.18 n.º1–4 | Contingência **só** se inoperacionalidade impossibilitar FE: (a) offline por falta de comunicação com a Plataforma; (b) tipográfica nos termos do Art.7 n.º6 (energia/avaria/acesso). Deveres: menção «emitido em contingência…»; submissão posterior à AGT; informar AGT imediatamente |
| 2 | `AO-LEG-DP-71-25-2025` | idem | idem | 7 | **11907** | Art.7 n.º6 (remissão) | Blocos tipográficos mediante **autorização** AGT, prazo ≤45 dias, requisitos do Regime — base da alínea b) do Art.18 |

Path privado: `derivatives/legislation/AO-LEG-DP-71-25-2025/v1/text.md`.

#### Distinções obrigatórias

| Tema | Relação com AO-OFF-001 |
|---|---|
| DE 683 `obterEstado` / `validationStatus` P (>24h) | Camada **FE** auxiliar (Citação G); **não** substitui Art.18; **não** usada como fundamento único desta promoção |
| Séries de recuperação DE 74 n.º8–9 | Candidato **`AO-OFF-002`** — **não** confirmado aqui |
| Homologação / produção AGT | **Não** confirmados |

#### Conclusão

**Facto:** a norma delimita **quando** e **como** a contingência é permitida (Art.18 + Art.7 n.º6); emissão fora desses casos **não** está autorizada por este artigo.
**Fail-closed:** **não** inventar modos de contingência adicionais; **não** afirmar fecho Edge/`DEC-REG-004`.
**Não se afirma:** conformidade do produto nem aceitação AGT.

## Ainda não confirmados

Todos os restantes IDs do [catálogo](../../../docs/01-compliance/requirements-catalog.md) permanecem na [matriz provisória](PROVISIONAL-MATRIX-RM-REQ-001.md) (`scaffold` / `partial` / `blocked` / `pending_validation` / `promoted` residual).

## Verificador

[`compliance/scripts/verify_confirmed_matrix.py`](../../scripts/verify_confirmed_matrix.py)
