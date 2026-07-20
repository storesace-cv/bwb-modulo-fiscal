# Lacunas regulatórias e artefactos oficiais — Angola

**Data:** 2026-07-20  
**Estado:** inventário inicial da Fase 0  
**Regras:** preferir fontes oficiais; não tratar fontes comunitárias como normativas; não inventar regras fiscais; não versionar credenciais.

Documentos relacionados:

- [sources.md](sources.md)
- [official-access-plan.md](official-access-plan.md)
- [angola-compliance.md](angola-compliance.md)
- [requirements-catalog.md](requirements-catalog.md)
- [phase-0-execution-plan.md](../06-delivery/phase-0-execution-plan.md)

## Resumo executivo

A documentação pública de Facturação Electrónica e os portais AGT/MINFIN estão parcialmente acessíveis, mas **faltam artefactos oficiais versionados** necessários para fechar requisitos críticos (`AO-CRYPTO-*`, `AO-SAF-*`, certificação). O acesso a Modelo 8, XSD SAF-T (AO) e homologação depende de registo/credenciais de produtora.

## Inventário de lacunas

| ID | Artefacto / diploma | Acesso atual | Bloqueia | Evidência para fechar |
|---|---|---|---|---|
| GAP-001 | Decreto Executivo n.º 74/19 (PDF oficial) | Existência referida; PDF oficial **não** arquivado no projeto | Matriz normativa, assinatura, menções, séries | PDF de fonte oficial (DR/MINFIN/AGT), data, URL/origem, SHA-256, responsável |
| GAP-002 | Rectificação do Decreto Executivo n.º 74/19 | Não arquivada | Interpretação correta do 74/19 | PDF oficial da rectificação + hash + ligação ao diploma base |
| GAP-003 | Modelo 8 (processo de produtores) | Área autenticada; acesso não demonstrado | Submissão/certificação, rotação de chaves comunicada à AGT | Cópia autorizada ou captura de requisitos atuais + referência de versão; **sem** dados pessoais desnecessários no Git |
| GAP-004 | XSD oficial SAF-T (AO) | Restrito / não obtido | `AO-SAF-001`, `AO-SAF-002`, gerador/validador | Ficheiro oficial + hash + data de vigência; validação independente documentada |
| GAP-005 | Especificação técnica FE versionada (snapshot) | Pública e acessível; snapshot do projeto ainda por criar | `AO-AGT-001`, conector, JWS/RSA, erros | Snapshot datado (HTML/PDF/export permitido) + hash + versão observada |
| GAP-006 | Credenciais e ambiente de homologação | Pedido formal ainda não concluído (conforme inventário) | Testes de integração reais com AGT | Credenciais apenas em gestor de segredos; registo de ambiente (HML) sem segredos no Git |
| GAP-007 | Confirmação processual de `ASM-REG-001` | Premissa de produto; sem evidência AGT | Modelo de certificação comercial | Resposta/ata AGT ou aceite formal de risco + plano B (ADR-0001) |
| GAP-008 | Catálogo oficial completo de tipos documentais / impostos / isenções aplicáveis ao MVP | Parcial via FE pública; incompleto sem 74/19 + manuais | `AO-DOC-001`, `AO-TAX-001` | Extrato aprovado por compliance a partir de fontes oficiais |
| GAP-009 | Regras oficiais de contingência / faturação offline | Não fechadas | `AO-OFF-001`, `AO-OFF-002`, Edge | Texto oficial ou orientação AGT escrita |
| GAP-010 | Vetores / resultados de testes oficiais AGT | Não disponíveis | Declaração de conformidade | Relatórios oficiais ou harness alinhado aos testes publicados |
| GAP-011 | Portal do Contribuinte / guias operacionais estáveis | Manutenção / timeout em 2026-07-20 | Orientação operacional | Reconsulta + arquivo permitido de conteúdo/versão |
| GAP-012 | Manifesto de fontes versionado no repositório | Estrutura recomendada; ainda não criada | Rastreabilidade contínua | `compliance/sources-manifest.yaml` (metadados públicos) |

## Decreto Executivo n.º 74/19 e respetiva rectificação

### O que falta

- PDF oficial do Decreto Executivo n.º 74/19.
- PDF oficial da respetiva rectificação.
- Registo de hash, origem, data de publicação e data de consulta.
- Extração controlada de requisitos para a matriz (`AO-*`), com interpretação aprovada.

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
| Disponibilidade | Restrita segundo inventário do projeto |
| Uso | Geração e validação (`AO-SAF-001`, `AO-SAF-002`) |
| Lacuna | Ficheiro oficial não obtido; schemas comunitários **não** são norma |
| Evidência de fecho | XSD oficial + hash + testes de validação + nota de vigência |
| Gate | Ver [official-access-plan.md](official-access-plan.md): sem XSD oficial não declarar gerador SAF-T de produção |

Qualquer XSD obtido de fonte comunitária (ex.: projetos de interoperabilidade) pode servir apenas diagnóstico interno e deve ser marcado como **não normativo**.

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
6. Manifesto versionado (GAP-012).
7. Reconsulta portais operacionais (GAP-011).

## Critério de atualização

Rever este ficheiro:

- antes do gate da Fase 0;
- sempre que um artefacto oficial for obtido;
- antes de cada release do pacote fiscal Angola.
