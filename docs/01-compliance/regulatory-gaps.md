# Lacunas regulatórias e artefactos oficiais — Angola

**Data:** 2026-07-20 (actualização OCR Rect v2 + DP 71/25 2026-07-26; inventário FE HML + Citação J 2026-07-28)
**Estado:** inventário Fase 0 + catálogo + XSD SRC-B2 (`pending_validation`); RM-SRC-004/RM-M2-C **CONCLUÍDOS** (OCR `reviewed` 74/19 + Rect. 10/19 v2 + 683/25 v2; DP 71/25 também `reviewed`). Extração AO-* confirmados permanece aberta (RM-REQ-001 / RM-SRC-006 / RM-M2-D **EM_CURSO**).
**Regras:** preferir fontes oficiais; não tratar fontes comunitárias como normativas; não inventar regras fiscais; não versionar credenciais; OCR `rejected` não é base de conhecimento.

Documentos relacionados:

- [sources.md](sources.md)
- [official-access-plan.md](official-access-plan.md)
- [angola-compliance.md](angola-compliance.md)
- [requirements-catalog.md](requirements-catalog.md)
- [phase-0-execution-plan.md](../06-delivery/phase-0-execution-plan.md)
- [`compliance/catalog/sources.yaml`](../../compliance/catalog/sources.yaml)

## Resumo executivo

Inventário consolidado das dependências AGT: [`agt-dependencies.md`](agt-dependencies.md).

A documentação pública de Facturação Electrónica e os portais AGT/MINFIN estão parcialmente acessíveis. Existe catálogo versionado, originais B1 no privado, e XSD ASSOFT (`pending_validation`). OCR `reviewed` (KB auxiliar): DE 74/19, Rect. 10/19 **v2** (`b3db14e2…`, 3p; gazeta 1948–1949), DE 683/25 v2 (citar só 19164–19227), DP 71/25 (citar só 11902–11920; p.21 = DE 372/25 excluído). Histórico Rect. incorrecto `77b77f01…` + OCR v1 `rejected` só em diagnóstico privado (≠ KB). GAP-001/GAP-002 parcialmente mitigados (OCR); residual URL estável/Imprensa Nacional; GAP-014 parcialmente mitigado; GAP-005 parcialmente mitigado (inventário FE-RNG); GAP-004 parcialmente mitigado. Sem AO-* confirmados.

## Inventário de lacunas

| ID | Artefacto / diploma | Acesso atual | Bloqueia | Evidência para fechar |
|---|---|---|---|---|
| GAP-001 | Decreto Executivo n.º 74/19 (PDF oficial) | Original + OCR `reviewed` (`5b63c80e…`); Citação F provisória (Anexo I @1576–1584) | Extração `AO-*` **confirmados** | Revisão compliance + confronto PDF residual |
| GAP-002 | Rectificação do Decreto Executivo n.º 74/19 | **Parcialmente mitigado:** v2 `b3db14e2…` + OCR `reviewed`; correcções Art.1 / SAF-T(AO) / Anexo III @**1948–1949** citadas; v1 só diagnostics; URL estável residual | Interpretação consolidada 74+Rect. | URL estável; fecho AO-* sem inventar |
| GAP-003 | Modelo 8 (processo de produtores) | Área autenticada; acesso não demonstrado; **`BLOQUEADO_EXTERNO`** (`DEC-DEL-002`) — **não** trava catálogo/domínio/CI | Submissão/certificação, rotação de chaves comunicada à AGT | Cópia autorizada ou captura de requisitos; **sem** inventar conteúdo da área autenticada |
| GAP-004 | XSD oficial SAF-T (AO) | **Parcialmente mitigado:** XSD ASSOFT + LICENSE/NOTICE (`e9a938e1…`, `pending_validation`); Citação D (L2/L3: `InvoiceType` L2023–2065, `References` L1004–1023, `Hash` L1361–1367, `SAFTAOPaymentType` L2740–2754); `doc:Status=Development`; ZIP `local_only` | `AO-SAF-001`, `AO-SAF-002`, gerador/validador | Confirmação AGT + vetores dourados; fecho C-DOC-003 (L4≠L2≠L3) |
| GAP-005 | Especificação técnica FE versionada (snapshot) | **Parcial:** metadados no catálogo + inventário citado [`FE-SERVICES-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/FE-SERVICES-MATRIX-RM-REQ-001.md) (endpoints + FE-RNG); bytes HTML só no privado; C-FE-001 paths; redistribuição pública ainda `uncertain` | `AO-AGT-001`, conector, JWS/RSA, erros | Autorizar cópia pública e/ou fechar C-FE-001 + GAP-006 |
| GAP-014 | Decreto Executivo n.º 683/25 (PDF oficial) | Original **correcto** + OCR v2 `reviewed` (`b01e4581…`); Citação G provisória (Art.1–6 + Anexos I–III + Tabelas @**19164–19227**); p.66 = Aviso 4/25 @19228 (não citar como DE); incorrecto `59a48189…` só diagnostics; URL estável ainda `pending_validation` | Precedência legislativa posterior / `AO-TAX-001` / FE | URL estável; fecho AO-* sem inventar; confronto PDF residual |
| GAP-006 | Credenciais e ambiente de homologação | Pedido formal ainda não concluído; **`BLOQUEADO_EXTERNO`** (`DEC-DEL-002`) — **não** trava simulador/contratos/testes | Testes de integração **reais** com AGT | Credenciais só em cofre; HML sem segredos no Git; **não** inventar resposta AGT |
| GAP-007 | Confirmação processual de `ASM-REG-001` | Premissa de produto; sem evidência AGT; **`BLOQUEADO_EXTERNO`** — **não** trava engenharia | Narrativa «aceite AGT» | Resposta/ata AGT ou risco+plano B; **não** inventar confirmação |
| GAP-008 | Catálogo oficial completo de tipos documentais / impostos / isenções aplicáveis ao MVP | **Parcial:** modelo (`DEC-PROD-014`) + esquema/seed (`DEC-PROD-015` / `DOCUMENT-CATALOG-RM-REQ-001`); C-DOC-* abertos; muitos campos `pending`; OpenAPI slice estreito; `DEC-REG-003` aberta | `AO-DOC-001`, `AO-TAX-001`, `DEC-REG-003`, `DEC-PROD-001`–`015` | Completar seed; fechar conflitos + faseamento + cálculo; **sem** confirmar AO-* |
| GAP-009 | Regras oficiais de contingência / faturação offline | Não fechadas; produto permite outbox/reenvio/idempotência **sem** certificação (`DEC-PROD-010`) | `AO-OFF-001`, `AO-OFF-002`, Edge, `DEC-REG-004` | Texto oficial ou orientação AGT escrita |
| GAP-010 | Vetores / resultados de testes oficiais AGT | Não disponíveis | Declaração de conformidade | Relatórios oficiais ou harness alinhado aos testes publicados |
| GAP-011 | Portal do Contribuinte / guias operacionais estáveis | Manutenção / timeout em 2026-07-20 | Orientação operacional | Reconsulta + arquivo permitido de conteúdo/versão |
| GAP-012 | Manifesto de fontes versionado no repositório | **Parcialmente fechado:** `compliance/catalog/sources.yaml` + schema + CI (metadados; sem binários) | Rastreabilidade contínua | Manter catálogo actualizado; importar artefactos autorizados nos PRs B+ |
| GAP-013 | Confirmação oficial AGT sobre custódia/uso da chave privada do contribuinte por módulo fiscal externo | Não confirmada; **`BLOQUEADO_EXTERNO`**; `DEC-PROD-012` constraints ≠ fecho; snapshot FE GESTAO (`de423e66…`) distingue chave **software** (produtor) vs **contribuinte** (HTML: emitida AGT / portal) — **não** autoriza custódia por módulo externo; **não** trava chave efémera/testes | `TaxpayerKeyRef` definitivo; DEC-REG-KEY-CUSTODY; DEC-SEC-EDGE-KEYS | Escrito AGT / norma; **não** inventar autorização; ver [agt-dependencies.md](agt-dependencies.md) · Citação J em [`PROVISIONAL-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md) |

## Decreto Executivo n.º 74/19 e respetiva rectificação

### O que falta

- URL oficial estável (MINFIN/Imprensa Nacional) da Rect. 10/19 e do DP 71/25 — residual de proveniência.
- DE 683/25: Citação G (Anexos/Tabelas) provisória; falta URL oficial estável (GAP-014 residual) e fecho AO-* confirmados.
- Extração `AO-*` confirmados a partir do conjunto 74/19+Rect. + DP 71/25 + DE 683; OCR `reviewed` ≠ requisito confirmado.
- Histórico: manter `77b77f01…` / OCR v1 rejected apenas em diagnóstico privado (nunca reintroduzir como KB).

### O que **não** fecha esta lacuna

- Transcrições comunitárias.
- Resumos de blogs ou repositórios de terceiros.
- O ficheiro de consulta em `local/docs/minfin055809.pdf`.

### Nota sobre material em `local/` (consulta apenas)

| Item | Observação |
|---|---|
| Caminho | `local/docs/minfin055809.pdf` |
| Classificação | **Consulta local não versionada** (`.gitignore`) |
| Natureza observada | Título interno «Proposta de Decreto Executivo»; numeração em branco (`___/18`); **não** é o Decreto n.º 74/19 publicado |
| Hash SHA-256 (local) | `4bc3a781b72964dc52a604ad26edf8b857084172ccd892a727880cb41bd91f73` |
| Pode ser copiado para pasta versionada? | **Não automaticamente.** Só após confirmação de que (a) é documento publicável, (b) não é tratado como norma vigente, e (c) há autorização explícita. Mesmo autorizado, deve ser etiquetado como *proposta/rascunho histórico*, nunca como 74/19. |
| Credenciais / dados pessoais | Não identificados no uso consultivo; manter fora do Git por defeito |

**Contradição (CTX-005):** [sources.md](sources.md) regista «existência confirmada» do 74/19 com PDF oficial por arquivar; o único PDF presente em `local/` não satisfaz esse requisito.

## Modelo 8

| Aspeto | Estado |
|---|---|
| Disponibilidade | Esperada na área autenticada de produtores |
| Uso | Processo de validação/certificação; comunicação de chaves públicas conforme regras oficiais |
| Lacuna | Acesso e versão atual não demonstrados |
| Evidência de fecho | Confirmação do portal autenticado + checklist documental atual + procedimento interno de preenchimento (sem NIF/chaves no Git) |
| Requisitos relacionados | `AO-KEY-001`, dossier de certificação em [angola-compliance.md](angola-compliance.md) |

## XSD oficial SAF-T (AO)

| Aspeto | Estado |
|---|---|
| Disponibilidade | Cópia ASSOFT MIT versionada em `compliance/saft-ao/schemas/` (`pending_validation`); confirmação AGT pendente |
| Uso | Preparação de geração/validação (`AO-SAF-001`, `AO-SAF-002`); **não** declarar conformidade de produção |
| Lacuna | Oficialidade/vigência AGT e validação independente ainda em aberto (GAP-004 parcial) |
| Evidência de fecho | Confirmação AGT + hash + testes de validação + nota de vigência |
| Gate | Ver [official-access-plan.md](official-access-plan.md): sem confirmação AGT não declarar gerador SAF-T de produção |

O XSD ASSOFT redistribuído sob MIT serve diagnóstico e preparação; **não** substitui um schema autenticado pela AGT se este divergir.

## Especificações técnicas versionadas (Facturação Electrónica)

Fontes públicas já inventariadas em [sources.md](sources.md):

- Documentação técnica FE: `https://quiosqueagt.minfin.gov.ao/doc-agt/faturacao-electronica/1/`
- Portal do Parceiro (público + autenticado)
- Elementos técnicos observados (2026-07-20): arquitetura assíncrona, JSON, JWS, RS256, RSA ≥ 2048 bits, `requestID`, homologação/produção — **sempre sujeitos a confirmação no snapshot versionado**

**Lacuna:** falta o snapshot interno versionado e o mapeamento estável para `AO-AGT-001` / `AO-AGT-002`.

**Evidência de fecho:** export/arquivo permitido + `retrieved_at` + `sha256` + lista de endpoints/códigos de erro usados pelo conector.

## Credenciais e ambiente de homologação

| Item | Regra |
|---|---|
| Pedido | Canal oficial indicado na documentação AGT (ex.: contacto de produtores FE no inventário de fontes) |
| Armazenamento | Gestor de segredos / cofre; nunca Git, CHANGELOG, issues públicas ou prompts |
| Ambientes | Separar HML e produção ([deployment.md](../07-operations/deployment.md)) |
| Evidência de fecho | Confirmação de acesso HML + teste de autenticação bem-sucedido registado sem expor segredos |

Nesta Fase 0 é aceitável avançar com **simulador AGT** interno; isso **não** substitui GAP-006.

## Premissa `ASM-REG-001`

| Aspeto | Estado |
|---|---|
| No produto | Ativa (README, [angola-compliance.md](angola-compliance.md), ADR-0001) |
| Como lacuna | Falta evidência de aceitação no processo AGT |
| O que não fazer | Alterar a premissa; espalhar lógica POS como autoridade fiscal |
| Evidência de fecho | Confirmação AGT **ou** decisão de risco documentada com plano de registo por integrador |

## O que já se pode fazer sem fechar todas as lacunas

Permitido pelo [official-access-plan.md](official-access-plan.md):

- Infraestrutura, modelo canónico, idempotência (`AO-IDEM-001`), numeração interna (`AO-SEQ-001` / `AO-SEQ-002`).
- Simulador AGT e testes de estados (`AO-AGT-002` a nível de máquina de estados).
- Planeamento do vertical slice.

**Não declarar concluídos** até fechar lacunas respetivas: assinatura legal de produção, SAF-T de produção, conformidade certificável.

## Plano de fecho (ordem sugerida)

1. Entidade/NIF + registo produtor (GAP-003, GAP-006).
2. Arquivar 74/19 + rectificação (GAP-001, GAP-002).
3. Snapshot FE público (GAP-005).
4. Pedido/obtenção XSD SAF-T (AO) (GAP-004).
5. Perguntas formais `ASM-REG-001` e contingência (GAP-007, GAP-009).
6. Manifesto versionado (GAP-012) — metadados criados; importação de artefactos continua pendente.
7. Reconsulta portais operacionais (GAP-011).

## Critério de atualização

Rever este ficheiro:

- antes do gate da Fase 0;
- sempre que um artefacto oficial for obtido;
- antes de cada release do pacote fiscal Angola.
