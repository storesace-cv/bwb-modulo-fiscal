# Matriz provisória RM-REQ-001

**Estado:** `EM_CURSO` — **não** é matriz AO-* confirmada.
**Data:** 2026-07-25
**Âmbito:** apenas fontes com OCR `reviewed` utilizável como KB auxiliar + XSD ASSOFT `pending_validation`.

## Fontes admitidas neste rascunho

| source_id | Estado OCR / schema | Uso permitido aqui | Limite |
|---|---|---|---|
| `AO-LEG-DE-74-19-2019` | `reviewed` v1 (`5b63c80e…`, 12p) | Pesquisa auxiliar; citação futura com página do PDF original | Conjunto normativo incompleto sem Rect. 10/19 — **não** confirmar texto consolidado |
| `AO-LEG-DE-683-25-2025` | `reviewed` v2 (`b01e4581…`, 66p) | Pesquisa auxiliar; citar **apenas** gazeta 19164–19227 (PDF p.2–65) | PDF p.66 = Aviso BNA 4/25 @19228 — **não** citar como DE 683 |
| `AO-SAFT-XSD-1.01_01` | schema `pending_validation` (`e9a938e1…`) | Referência técnica ASSOFT MIT | Não afirmado como validado pela AGT |
| `AO-LEG-RECT-10-19-2019` | `rejected` / excluída (`77b77f01…`) | **Não usar** | Original incorrecto/incompleto; GAP-002 |

## Bloqueios explícitos (fail-closed)

1. **Rectificação n.º 10/19** — sem original integral oficial; qualquer linha que dependa do texto consolidado DE 74/19 + Rect. permanece `blocked`.
2. **RM-SRC-004 / RM-M2-C** — continuam `BLOQUEADO` (conjunto 74+Rect+683 incompleto).
3. Esta matriz **não** inventa regras fiscais e **não** promove linhas a «confirmado».
4. `reviewed` OCR é KB auxiliar; só após citação página a página + revisão compliance pode sustentar AO-* confirmados (ainda não neste PR).

## Legenda de estado da linha

| Estado | Significado |
|---|---|
| `scaffold` | ID do catálogo inicial; sem ligação normativa página a página; **não** confirmado |
| `partial` | Ligação preliminar a fonte `reviewed` com página candidata; sujeita a revisão; **não** confirmado |
| `blocked` | Dependente da Rect. 10/19 e/ou de fonte oficial ainda inacessível |
| `pending_validation` | Artefacto técnico presente sem validação AGT |

Estados proibidos neste ficheiro: `confirmed`, `confirmado`, `validated_agt`.

## Linhas (rascunho)

| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | Premissa de produto; confirmação AGT em aberto (RM-FOUND-005) |
| AO-ID-001 | `scaffold` | DE 74/19 / DE 683/25 | Associação contribuinte/estabelecimento/software — mapear páginas depois; sem confirmação |
| AO-DOC-001 | `blocked` | DE 74/19 + Rect. 10/19 | Campos/tipos dependem do conjunto normativo 74+Rect (GAP-002) |
| AO-DOC-002 | `blocked` | DE 74/19 + Rect. 10/19 | Imutabilidade/regras de emissão exigem texto consolidado 74+Rect antes de confirmar |
| AO-SEQ-001 | `blocked` | DE 74/19 (+ Rect.) / DE 683/25 | Sequencialidade no âmbito 74/19 depende do conjunto com Rect.; não misturar com 683 sem split formal |
| AO-SEQ-002 | `scaffold` | `AO-LEG-DE-683-25-2025` | Candidato: ARTIGO 4.º (Série de facturas), gazeta **19164** (PDF p.2) — OCR menciona competência/séries AGT; **não** confirmado; formalizar citação em PR futuro |
| AO-IDEM-001 | `scaffold` | — | Arquitectura de API / produto; não derivado só de legislação |
| AO-TAX-001 | `blocked` | Rect. / fontes oficiais | Cálculo fiscal exige fontes oficiais completas e texto consolidado |
| AO-CRYPTO-001 | `blocked` | Snapshot FE oficial / 74+Rect | Distinguir JWS FE vs SAF-T; FE oficial incompleto |
| AO-KEY-001 | `blocked` | GAP-013 | Custódia de chave contribuinte em aberto |
| AO-AGT-001 | `blocked` | FE HML/PRD oficiais | Credenciais/docs oficiais AGT pendentes |
| AO-AGT-002 | `scaffold` | — | Máquina de estados — DEC-API-004 |
| AO-OFF-001 | `blocked` | Contingência oficial | Fonte oficial de contingência pendente |
| AO-OFF-002 | `scaffold` | — | Sync sem renumerar — produto + legal |
| AO-AUD-001 | `scaffold` | — | Auditoria append-only — arquitectura |
| AO-SAF-001 | `pending_validation` | `AO-SAFT-XSD-1.01_01` | XSD ASSOFT versionado; validação AGT pendente |
| AO-SAF-002 | `pending_validation` | XSD + legislação completa | Anulados/retificativos — sem fecho sem fontes completas / Rect. |
| AO-OPS-001 | `scaffold` | — | Ops/DR |
| AO-UPD-001 | `scaffold` | — | Updates Edge assinados |

## Próximos passos (este item)

1. Eleger linhas `scaffold` sem dependência Rect. (ex. AO-SEQ-002) e ligar `source_id` + página PDF + trecho OCR `reviewed` → estado `partial` (ainda não confirmado).
2. Manter AO-DOC-* / AO-SEQ-001 / AO-TAX-* e afins em `blocked` até GAP-002.
3. Não promover nenhuma linha a confirmado sem revisão de compliance + critérios de aceitação testáveis.
4. Fechar RM-REQ-001 só com matriz rastreável e gate `RM-SRC-004` (ou decisão explícita de scope reduzido documentada).

## Referências

- Catálogo público: [`compliance/catalog/sources.yaml`](../../catalog/sources.yaml)
- Catálogo inicial de IDs: [`docs/01-compliance/requirements-catalog.md`](../../../docs/01-compliance/requirements-catalog.md)
- Gaps: [`docs/01-compliance/regulatory-gaps.md`](../../../docs/01-compliance/regulatory-gaps.md)
- Aquisição Rect. (privado): `storesace-cv/bwb-fiscal-sources-ao` → `docs/ACQUISITION-RECT-10-19.md`
- Verificador: [`compliance/scripts/verify_provisional_matrix.py`](../../scripts/verify_provisional_matrix.py)
