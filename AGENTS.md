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

1. Requisitos regulatórios em `docs/01-compliance/requirements-catalog.md`.
2. Decisões arquiteturais em `docs/02-architecture/adrs/`.
3. Contrato público em `specs/openapi/openapi.yaml`.
4. Modelo de domínio em `docs/04-domain/domain-model.md`.
5. Testes de conformidade e vetores aprovados.

Quando houver conflito, parar e registar o conflito. Não inventar uma interpretação fiscal.

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
