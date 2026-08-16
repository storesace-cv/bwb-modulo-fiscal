# Instruções permanentes para agentes de desenvolvimento

## Leitura obrigatória antes de qualquer ação

Antes de analisar, planear, executar comandos, editar ficheiros ou propor código, ler integralmente:

1. este `AGENTS.md`;
2. `ENGINEERING_PRINCIPLES.md`;
3. as regras aplicáveis em `.cursor/rules/`;
4. os requisitos e ADRs relacionados com a tarefa.

Não iniciar trabalho enquanto esta leitura não estiver concluída.

## Missão

Construir um módulo fiscal externo, certificável pela AGT, que seja a autoridade fiscal de emissão para vários POS, disponível como serviço cloud e como serviço local Linux.

## Fontes de verdade

1. Estado e progresso do projecto em `ROADMAP.md` (roadmap canónico).
2. Requisitos regulatórios em `docs/01-compliance/requirements-catalog.md`.
3. Decisões arquiteturais em `docs/02-architecture/adrs/`.
4. Contrato público em `specs/openapi/openapi.yaml`.
5. Modelo de domínio em `docs/04-domain/domain-model.md`.
6. Testes de conformidade e vetores aprovados.
7. Catálogo versionado de fontes fiscais em `compliance/catalog/sources.yaml` e política em `compliance/POLICY.md`.

Quando houver conflito, parar e registar o conflito. Não inventar uma interpretação fiscal.

## Manutenção do ROADMAP

- Qualquer PR que conclua, introduza, bloqueie, adie ou altere um item `RM-*` deve actualizar `ROADMAP.md` no mesmo PR.
- Usar o formato de tabela canónico e os estados `CONCLUÍDO` | `PENDENTE` | `EM_CURSO` | `BLOQUEADO` | `ADIADO`.
- `[x]` / `CONCLUÍDO` exige evidência verificável; a CI valida estrutura (`scripts/verify_roadmap.py`), não a semântica fiscal.
- `docs/06-delivery/implementation-roadmap.md` é apenas apontador de compatibilidade.
- Ver também `.cursor/rules/roadmap-maintenance.mdc`.

## Fontes fiscais versionadas

Antes de tarefas fiscais (regras, SAF-T, FE, requisitos `AO-*`):

1. Consultar `compliance/catalog/sources.yaml` e `compliance/POLICY.md`.
2. Citar `source_id`, diploma/secção, **página** (quando PDF), endpoint ou `FE-RNG-*` conforme a fonte.
3. Alertar se a fonte estiver `pending_validation`, `superseded`, `withdrawn`, ou se o derivado OCR não estiver `reviewed`.
4. Tratar PDFs image-only como **não legíveis** para requisitos até existir OCR com páginas `reviewed`; renderizar páginas não equivale a texto pesquisável. Derivados e revisão (`reviewer_kind=ai_assisted`) vivem no repo privado — nunca depender de `local/`.
5. Para cada derivado OCR de um diploma, o PDF oficial original permanece a representação autoritativa; OCR/Markdown são auxiliares de pesquisa. Só conteúdo `reviewed` e confrontado visualmente com o original pode sustentar requisitos `AO-*` confirmados.
6. JWS/RS256 da faturação electrónica é distinto de mecanismos SAF-T.
7. Nunca depender de `local/` em build, testes, runtime ou CI.

## Pasta local de consulta

- A pasta `local/` contém materiais fornecidos apenas para consulta pelo Cursor/agentes.
- Todo o conteúdo de `local/` é local, não versionado e não deve ser sincronizado com GitHub.
- Nunca remover `local/` do `.gitignore` nem forçar a inclusão dos seus ficheiros no Git.
- Um ficheiro em `local/` não pode ser usado como dependência, schema, fixture, fonte de build ou artefacto de runtime a partir dessa localização.
- Se um ficheiro for necessário ao projeto, copiar (preferencialmente) ou mover para uma pasta versionada adequada, após verificar licença, confidencialidade e autorização para o versionar.
- Ao copiar, indicar origem, versão/data e hash quando o ficheiro for regulatório ou técnico.
- Não copiar credenciais, chaves privadas, certificados secretos, dados pessoais ou materiais cuja distribuição seja restrita.
- Se não for claro se o ficheiro pode ser sincronizado, mantê-lo em `local/` e pedir decisão ao responsável do projeto.

## Regras obrigatórias

- Nunca usar `float`/`double` para dinheiro. Usar decimal ou inteiros na menor unidade, conforme a decisão de domínio.
- Um documento fiscal emitido é imutável. Correções usam documentos retificativos ou operações legalmente permitidas.
- Não apagar documentos fiscais, sequências, eventos de auditoria ou tentativas de transmissão.
- Toda criação de documento exige `Idempotency-Key` e identificador externo único por integrador/empresa.
- Não reservar um número fiscal num POS. O módulo fiscal é a autoridade de numeração.
- Não aceitar datas sem timezone nem confiar no relógio do POS como fonte única.
- Não colocar chaves, tokens, NIF completos ou conteúdo integral de faturas em logs operacionais.
- Toda alteração de regra fiscal deve indicar pelo menos um requisito `AO-*` e incluir testes.
- Toda alteração do contrato público requer compatibilidade retroativa ou nova versão da API.
- Toda migração de dados fiscais deve ser reversível e testada sobre cópia anonimizada.
- Edge e cloud devem produzir o mesmo resultado fiscal para o mesmo vetor de entrada e pacote de regras.
- Código específico de Angola permanece no pacote de país; não espalhar condicionais de país pelo núcleo.
- Cabo Verde não deve ser implementado até existir catálogo próprio aprovado.

## Definition of Done

Uma tarefa fiscal só está concluída quando:

1. requisito e critérios de aceitação estão identificados;
2. implementação está coberta por testes unitários e de conformidade;
3. auditoria e erros estão tratados;
4. documentação pública foi atualizada quando aplicável;
5. comportamento cloud/edge foi verificado;
6. não existem segredos ou dados pessoais nos artefactos.

## Fluxo recomendado no Cursor

1. Ler o requisito e ADR relevante.
2. Propor plano pequeno e listar ficheiros afetados.
3. Criar/ajustar primeiro o teste ou vetor fiscal.
4. Implementar a alteração mínima.
5. Executar testes, validação OpenAPI e verificações de segurança.
6. Atualizar matriz de rastreabilidade e changelog.

## Relatórios finais de execução (obrigatório)

Todo relatório final de execução (Cursor/agente) deve incluir **sempre** estas duas secções, nesta ordem:

### 1. Problemas/incidentes internos encontrados

Para cada item: causa; impacto; resolução; estado; risco residual.

Se nenhum: escrever exactamente `Nenhum`.

Inclui falhas de CI locais/remotas, whitespace, ShellCheck, mocks, bugs de código, etc. **Não** misturar com questões AGT.

### 2. Problemas/questões AGT encontrados ou actualizados

Para cada item: IDs `AGT-Q-*` (ver [`docs/01-compliance/agt-clarifications-register.md`](docs/01-compliance/agt-clarifications-register.md)); observação; impacto; mitigação; estado; pergunta para a AGT.

Se nenhum: escrever exactamente `Nenhum`.

Se não houve comunicação real com a AGT nesta execução: escrever exactamente `Sem chamadas reais à AGT nesta execução`.

**Proibido:** inventar erros de API HML/PRD sem observação; colocar NIF/PEM/credenciais no relatório; omitir estas secções.
