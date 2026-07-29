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

## Ainda não confirmados

Todos os restantes IDs do [catálogo](../../../docs/01-compliance/requirements-catalog.md) permanecem na [matriz provisória](PROVISIONAL-MATRIX-RM-REQ-001.md) (`scaffold` / `partial` / `blocked` / `pending_validation`).

## Verificador

[`compliance/scripts/verify_confirmed_matrix.py`](../../scripts/verify_confirmed_matrix.py)
