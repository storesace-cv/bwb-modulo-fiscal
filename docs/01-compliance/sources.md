# Fontes regulatórias e técnicas

## Fonte oficial prioritária

### Portal do Contribuinte — Ministério das Finanças de Angola / AGT

- URL: https://portaldocontribuinte.minfin.gov.ao
- Classificação: fonte oficial primária.
- Âmbito: avisos da AGT, obrigações dos contribuintes, faturação, IVA, software validado, procedimentos e acesso aos serviços tributários.
- Prioridade: alta para informação operacional e comunicados; diplomas publicados no Diário da República continuam a ser a fonte normativa aplicável.
- Estado verificado em 2026-07-20: o endereço apresentava uma página temporária de manutenção («Sítio não encontrado / Tente novamente mais tarde»). Manter a fonte registada e voltar a verificar antes de cada análise regulatória.

### Guia Rápido — Emissão de Facturas

- URL: https://portaldocontribuinte.minfin.gov.ao/guia-rapido-emissao-de-facturas
- Classificação: fonte oficial primária de orientação operacional.
- Âmbito: procedimentos e instruções práticas relacionados com a emissão de facturas no Portal do Contribuinte.
- Prioridade: alta para compreender o fluxo operacional esperado; confirmar sempre os requisitos jurídicos nos diplomas e especificações técnicas vigentes.
- Estado verificado em 2026-07-20: a consulta automática terminou por timeout. Rever manualmente e arquivar conteúdo/versão durante a Fase 0.

### Portal institucional da Administração Geral Tributária

- URL: https://agt.minfin.gov.ao/PortalAGT/#!/
- URL canónica observada: https://agt.minfin.gov.ao/PortalAGT/
- Classificação: fonte oficial primária.
- Âmbito: informação institucional da AGT, legislação, comunicados, serviços, formulários, contactos e orientações tributárias.
- Prioridade: alta para localizar publicações e confirmar orientações oficiais. As rotas após `#!` pertencem à aplicação web e podem não ser preservadas por ferramentas automáticas.
- Estado verificado em 2026-07-20: o endereço respondeu como aplicação web, sem conteúdo textual extraível na consulta automática. Rever manualmente durante a Fase 0.

### Documentação técnica oficial — Facturação Electrónica

- URL raiz: https://quiosqueagt.minfin.gov.ao/doc-agt/faturacao-electronica/1/
- Classificação: fonte oficial técnica primária.
- Âmbito verificado: arquitetura assíncrona, JSON, JWS, polling, callbacks, autenticação, gestão de chaves, QR Code, serviços de registo, consulta, listagem e validação de facturas, payloads e códigos de erro.
- Elementos técnicos observados em 2026-07-20: RS256 (RSA + SHA-256), chaves RSA com mínimo indicado de 2048 bits, `requestID`/identificadores de submissão, endpoints de homologação e produção e assinatura de dados do software/documento.
- Regra: arquivar uma versão datada antes de implementar; a documentação encontra-se em evolução e contém referências a funcionalidades futuras.

### Portal do Parceiro — MINFIN/AGT

- URL: https://portaldoparceiro.minfin.gov.ao/
- Homologação referenciada pela AGT: https://portaldoparceiro.hml.minfin.gov.ao/
- Classificação: fonte oficial técnica e operacional primária.
- Âmbito verificado: modelos de documentos, guia do utilizador, chaves e NIF de testes e especificação técnica de serviços de facturação electrónica.
- Limitação: aplicação dependente de JavaScript e áreas autenticadas; alguns ficheiros exigem sessão/credenciais de produtor.
- Credenciais técnicas: a documentação oficial indica pedido à AGT com nome e NIF da empresa através de `produtores.dfe.dcrr.agt@minfin.gov.ao`.

### Decreto Executivo n.º 74/19

- Classificação: fonte normativa primária quando obtida do Diário da República/Ministério das Finanças/AGT.
- Âmbito: regras e requisitos de validação de sistemas de processamento electrónico de facturação.
- Ação obrigatória: obter o PDF oficial e respetiva rectificação, registar hash e extrair requisitos para a matriz. Não usar como fonte normativa final uma transcrição comunitária.

### Área autenticada de Produtores de Software

- Classificação: repositório oficial restrito.
- Âmbito esperado, sujeito a confirmação após autenticação: Modelo 8, submissão técnica, ficheiro XSD oficial SAF-T (AO), manuais, estados do processo e certificado.
- Estado: acesso ainda não demonstrado, por depender do registo/NIF e credenciais da empresa produtora.
- Segurança: credenciais, chaves privadas e certificados nunca entram no Git, na documentação ou em prompts. Usar gestor de segredos e diretório de evidências com controlo de acesso.

## Regras de utilização das fontes

1. Preferir Diário da República, Ministério das Finanças e AGT a resumos de terceiros.
2. Não converter um aviso operacional em requisito legal sem identificar a respetiva base normativa.
3. Para cada requisito `AO-*`, registar URL, título, entidade, data de publicação, data de consulta e versão/estado.
4. Guardar uma cópia ou hash do documento consultado quando permitido, porque páginas e anexos podem mudar.
5. Se uma fonte oficial estiver indisponível, marcar a evidência como pendente; não substituir silenciosamente por uma interpretação não oficial.
6. Rever fontes antes de cada release do pacote fiscal Angola.

## Registo de consulta

| Fonte | Última consulta | Estado | Próxima ação |
|---|---|---|---|
| Portal do Contribuinte | 2026-07-20 | Página em manutenção | Verificar novamente durante a Fase 0 |
| Guia Rápido — Emissão de Facturas | 2026-07-20 | Timeout na consulta automática | Consultar e arquivar manualmente |
| Portal institucional da AGT | 2026-07-20 | Aplicação web acessível; conteúdo não extraído | Rever secções e anexos manualmente |
| Documentação técnica de Facturação Electrónica | 2026-07-20 | Pública e acessível | Criar snapshot versionado na Fase 0 |
| Portal do Parceiro | 2026-07-20 | Conteúdo público indexado; aplicação requer JavaScript | Obter downloads e credenciais de homologação |
| Decreto Executivo n.º 74/19 + rectificação | 2026-07-20 | Existência confirmada; PDF oficial ainda por arquivar | Descarregar de fonte oficial e calcular hash |
| Área de Produtores / Modelo 8 / XSD SAF-T (AO) | 2026-07-20 | Restrita; acesso não demonstrado | Registar empresa/NIF e obter credenciais |

## Fontes comunitárias

Fontes como a ASSOFT/Projeto XSD Angola podem apoiar diagnóstico e interoperabilidade, mas não substituem o XSD, diplomas, instruções ou resultados de validação emitidos pela AGT. Qualquer diferença deve ser resolvida a favor da versão oficial aplicável.
