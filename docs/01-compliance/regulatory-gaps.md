# Lacunas regulatórias e artefactos oficiais — Angola

**Data:** 2026-07-20 (actualização OCR Rect v2 + DP 71/25 2026-07-26)
**Estado:** inventário Fase 0 + catálogo + XSD SRC-B2 (`pending_validation`); RM-SRC-004/RM-M2-C **CONCLUÍDOS** (OCR `reviewed` 74/19 + Rect. 10/19 v2 + 683/25 v2; DP 71/25 também `reviewed`). Extração AO-* confirmados permanece aberta (RM-REQ-001 / RM-SRC-006).
**Regras:** preferir fontes oficiais; não tratar fontes comunitárias como normativas; não inventar regras fiscais; não versionar credenciais; OCR `rejected` não é base de conhecimento.

Documentos relacionados:

- [sources.md](sources.md)
- [official-access-plan.md](official-access-plan.md)
- [angola-compliance.md](angola-compliance.md)
- [requirements-catalog.md](requirements-catalog.md)
- [phase-0-execution-plan.md](../06-delivery/phase-0-execution-plan.md)
- [`compliance/catalog/sources.yaml`](../../compliance/catalog/sources.yaml)

## Resumo executivo

A documentação pública de Facturação Electrónica e os portais AGT/MINFIN estão parcialmente acessíveis. Existe catálogo versionado, originais B1 no privado, e XSD ASSOFT (`pending_validation`). OCR `reviewed` (KB auxiliar): DE 74/19, Rect. 10/19 **v2** (`b3db14e2…`, 3p; gazeta 1948–1949), DE 683/25 v2 (citar só 19164–19227), DP 71/25 (citar só 11902–11920; p.21 = DE 372/25 excluído). Histórico Rect. incorrecto `77b77f01…` + OCR v1 `rejected` só em diagnóstico privado (≠ KB). GAP-001/GAP-002 parcialmente mitigados (OCR); residual URL estável/Imprensa Nacional; GAP-014 parcialmente mitigado; GAP-005 aberto; GAP-004 parcialmente mitigado. Sem AO-* confirmados.

## Inventário de lacunas

| ID | Artefacto / diploma | Acesso atual | Bloqueia | Evidência para fechar |
|---|---|---|---|---|
| GAP-001 | Decreto Executivo n.º 74/19 (PDF oficial) | Original correcto + OCR `reviewed` no privado (`5b63c80e…`); metadados públicos com `derivatives` reviewed; conjunto com Rect. v2 disponível | Extração `AO-*` confirmados (citação página a página) | Citações rastreáveis + revisão compliance |
| GAP-002 | Rectificação do Decreto Executivo n.º 74/19 | **Parcialmente mitigado:** original v2 correcto (`b3db14e2…`, 3p) + OCR `reviewed` no privado/público; gazeta **1948–1949** (PDF p.2–3); v1 incorrecta `77b77f01…` + OCR rejected só em `diagnostics/` privado (≠ KB); URL oficial estável / Imprensa Nacional ainda residual | Interpretação consolidada 74+Rect. para AO-* | Confirmar URL MINFIN/Imprensa Nacional estável; extrair `AO-*` com citação página a página (sem inventar) |
| GAP-003 | Modelo 8 (processo de produtores) | Área autenticada; acesso não demonstrado | Submissão/certificação, rotação de chaves comunicada à AGT | Cópia autorizada ou captura de requisitos atuais + referência de versão; **sem** dados pessoais desnecessários no Git |
| GAP-004 | XSD oficial SAF-T (AO) | **Parcialmente mitigado:** XSD ASSOFT + LICENSE/NOTICE em `compliance/saft-ao/schemas/` (`AO-SAFT-XSD-1.01_01`, MIT, `pending_validation`); **não** afirmado como validado pela AGT; ZIP permanece `local_only` | `AO-SAF-001`, `AO-SAF-002`, gerador/validador | Confirmação AGT + validação independente |
| GAP-005 | Especificação técnica FE versionada (snapshot) | 13 HTML em `local/` + metadados no catálogo; snapshot **não** no Git público (bytes no privado B1) | `AO-AGT-001`, conector, JWS/RSA, erros | Snapshot autorizado no Git público ou ponteiros + inventários HML/PRD |
| GAP-014 | Decreto Executivo n.º 683/25 (PDF oficial) | Original **correcto** + OCR v2 `reviewed` no privado/público (`b01e4581…`, 66p); PDF p.66 = início Aviso 4/25 @19228 (mesmo DR — não citar como DE); incorrecto `59a48189…` em `diagnostics/`; URL oficial estável ainda `pending_validation` | Precedência legislativa posterior | Confirmar URL MINFIN/Imprensa Nacional estável; ao citar DE 683 usar páginas 19164–19227 |
| GAP-006 | Credenciais e ambiente de homologação | Pedido formal ainda não concluído (conforme inventário) | Testes de integração reais com AGT | Credenciais apenas em gestor de segredos; registo de ambiente (HML) sem segredos no Git |
| GAP-007 | Confirmação processual de `ASM-REG-001` | Premissa de produto; sem evidência AGT | Modelo de certificação comercial | Resposta/ata AGT ou aceite formal de risco + plano B (ADR-0001) |
| GAP-008 | Catálogo oficial completo de tipos documentais / impostos / isenções aplicáveis ao MVP | Parcial via FE pública; incompleto sem 74/19 + manuais | `AO-DOC-001`, `AO-TAX-001` | Extrato aprovado por compliance a partir de fontes oficiais |
| GAP-009 | Regras oficiais de contingência / faturação offline | Não fechadas | `AO-OFF-001`, `AO-OFF-002`, Edge | Texto oficial ou orientação AGT escrita |
| GAP-010 | Vetores / resultados de testes oficiais AGT | Não disponíveis | Declaração de conformidade | Relatórios oficiais ou harness alinhado aos testes publicados |
| GAP-011 | Portal do Contribuinte / guias operacionais estáveis | Manutenção / timeout em 2026-07-20 | Orientação operacional | Reconsulta + arquivo permitido de conteúdo/versão |
| GAP-012 | Manifesto de fontes versionado no repositório | **Parcialmente fechado:** `compliance/catalog/sources.yaml` + schema + CI (metadados; sem binários) | Rastreabilidade contínua | Manter catálogo actualizado; importar artefactos autorizados nos PRs B+ |
| GAP-013 | Confirmação oficial AGT sobre custódia/uso da chave privada do contribuinte por módulo fiscal externo | Não confirmada; contrato privado insuficiente | Provisionamento de `TaxpayerKeyRef` na plataforma; DEC-REG-KEY-CUSTODY; DEC-SEC-EDGE-KEYS | Orientação/escrito oficial AGT ou regra em diploma/manual versionado; ver [open-decisions.md](../06-delivery/open-decisions.md) e [backoffice-architecture.md](../02-architecture/backoffice-architecture.md) |

## Decreto Executivo n.º 74/19 e respetiva rectificação

### O que falta

- URL oficial estável (MINFIN/Imprensa Nacional) da Rect. 10/19 e do DP 71/25 — residual de proveniência.
- DE 683/25: original correcto v2 + OCR `reviewed` já no arquivo; falta URL oficial estável (GAP-014 residual).
- Extração `AO-*` com citação página a página a partir do conjunto 74/19+Rect. v2 (+ DP 71/25 onde aplicável); OCR `reviewed` ≠ requisito confirmado.
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
