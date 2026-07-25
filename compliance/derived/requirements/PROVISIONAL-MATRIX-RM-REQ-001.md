# Matriz provisória RM-REQ-001

**Estado:** `EM_CURSO` — **não** é matriz AO-* confirmada.
**Data:** 2026-07-25
**Âmbito:** apenas fontes com OCR `reviewed` utilizável como KB auxiliar + XSD ASSOFT `pending_validation`.

## Fontes admitidas neste rascunho

| source_id | Estado OCR | Uso permitido aqui | Limite |
|---|---|---|---|
| `AO-LEG-DE-74-19-2019` | `reviewed` v1 | Pesquisa auxiliar; citação com página do PDF original | Conjunto normativo incompleto sem Rect. 10/19 |
| `AO-LEG-DE-683-25-2025` | `reviewed` v2 | Pesquisa auxiliar; citar **apenas** páginas de gazeta 19164–19227 (PDF p.2–65) | PDF p.66 = Aviso BNA 4/25 — **não** citar como DE 683 |
| `AO-SAFT-XSD-1.01_01` | N/A (schema) | Referência técnica `pending_validation` | Não afirmado como validado pela AGT |
| `AO-LEG-RECT-10-19-2019` | `rejected` | **Excluída** | Original incorrecto/incompleto; GAP-002 |

## Bloqueios explícitos

1. **Rectificação n.º 10/19** — sem original integral oficial; qualquer interpretação do DE 74/19 que dependa da rectificação permanece **não confirmada**.
2. **RM-SRC-004 / RM-M2-C** — continuam `BLOQUEADO` (não se afirma conclusão OCR do conjunto 74+Rect+683).
3. Esta matriz **não** inventa regras fiscais; linhas sem evidência `reviewed` + página ficam `blocked` / `hypothesis`.

## Legenda de estado da linha

| Estado | Significado |
|---|---|
| `scaffold` | ID existente no catálogo inicial; evidência normativa ainda não ligada página a página |
| `partial` | Ligação preliminar a fonte `reviewed`, sujeita a revisão jurídica |
| `blocked` | Dependente de Rect. 10/19 ou de fonte oficial ainda inacessível |
| `pending_validation` | Artefacto técnico presente mas sem validação AGT |

## Linhas (rascunho)

| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | Premissa de produto; confirmação AGT em aberto (RM-FOUND-005) |
| AO-ID-001 | `scaffold` | DE 74/19 / DE 683/25 | Campos de identificação — mapear em PR seguintes com páginas |
| AO-DOC-001 | `scaffold` | DE 74/19 | Tipos/campos — depende do conjunto 74+Rect |
| AO-DOC-002 | `scaffold` | DE 74/19 | Imutabilidade — hipótese até confirmação jurídica |
| AO-SEQ-001 | `scaffold` | DE 74/19 · DE 683/25 art. 4.º (série AGT) | Numeração: citar DE 683 p.19164 com cautela; Rect. pode afectar 74/19 |
| AO-SEQ-002 | `scaffold` | DE 683/25 | Série gerada pela AGT — evidência visual p.19164; formalizar página OCR |
| AO-IDEM-001 | `scaffold` | — | Arquitectura de API; não derivado só de legislação |
| AO-TAX-001 | `blocked` | — | Regras de cálculo exigem fontes oficiais completas / Rect. |
| AO-CRYPTO-001 | `blocked` | Snapshot FE / 74/19 | Distinguir JWS FE vs mecanismos SAF-T; FE oficial ainda incompleto |
| AO-KEY-001 | `blocked` | GAP-013 | Custódia de chave contribuinte em aberto |
| AO-AGT-001 | `blocked` | FE HML/PRD oficiais | Credenciais/docs oficiais AGT pendentes |
| AO-AGT-002 | `scaffold` | — | Máquina de estados — decisão DEC-API-004 |
| AO-OFF-001 | `blocked` | Contingência oficial | Fonte oficial de contingência pendente |
| AO-OFF-002 | `scaffold` | — | Sync sem renumerar — regra de produto + legal |
| AO-AUD-001 | `scaffold` | — | Auditoria append-only — arquitectura |
| AO-SAF-001 | `pending_validation` | `AO-SAFT-XSD-1.01_01` | XSD ASSOFT versionado; validação AGT pendente |
| AO-SAF-002 | `pending_validation` | XSD + legislação | Anulados/retificativos — sem fecho sem fontes completas |
| AO-OPS-001 | `scaffold` | — | Ops/DR |
| AO-UPD-001 | `scaffold` | — | Updates Edge assinados |

## Próximos passos (este item)

1. Para cada linha `partial`/`scaffold` candidata a DE 74/19 ou 683/25: citar `source_id`, secção, **página do PDF original**, e trecho OCR `reviewed` confrontado.
2. Manter linhas dependentes da Rect. 10/19 em `blocked` até GAP-002 fechar.
3. Não promover nenhuma linha a «requisito confirmado» sem revisão de compliance e critérios de aceitação testáveis.
4. Fechar RM-REQ-001 só quando a matriz AO-* estiver rastreável e o gate `RM-SRC-004` + revisão o permitirem (ou decisão explícita de scope reduzido documentada).

## Referências

- Catálogo público: [`compliance/catalog/sources.yaml`](../../catalog/sources.yaml)
- Catálogo inicial de IDs: [`docs/01-compliance/requirements-catalog.md`](../../../docs/01-compliance/requirements-catalog.md)
- Gaps: [`docs/01-compliance/regulatory-gaps.md`](../../../docs/01-compliance/regulatory-gaps.md)
- Aquisição Rect. (privado): `storesace-cv/bwb-fiscal-sources-ao` → `docs/ACQUISITION-RECT-10-19.md`
