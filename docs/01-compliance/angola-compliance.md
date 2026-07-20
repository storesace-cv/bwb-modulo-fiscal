# Estratégia de conformidade — Angola

## Premissa ativa

`ASM-REG-001`: a certificação do módulo fiscal externo dispensa a validação individual dos POS integrados, desde que o módulo seja a única autoridade fiscal de emissão.

## Método

Cada obrigação deve possuir:

- fonte legal e versão;
- identificador estável;
- interpretação aprovada;
- componente responsável;
- critérios de aceitação;
- testes e evidências;
- estado: rascunho, validado, implementado ou certificado.

## Domínios de conformidade

1. Identificação do produtor, contribuinte, estabelecimento e software.
2. Documentos e menções obrigatórias.
3. Impostos, taxas, isenções e arredondamentos.
4. Séries, numeração e cronologia.
5. Integridade, assinatura, hash e chaves.
6. Anulações, retificações e autofaturação.
7. Comunicação eletrónica e estados da AGT.
8. Contingência e reconciliação.
9. Arquivo, consulta, inspeção e auditoria.
10. SAF-T (AO), inventários e exportações.
11. Atualização do software e identificação de versões.
12. Segurança, privacidade, localização e retenção.

## Dossier de certificação

- descrição funcional e arquitetura;
- versões e componentes abrangidos;
- matriz de rastreabilidade;
- gestão de chaves;
- modelos de documentos;
- casos de teste e evidências;
- amostras SAF-T;
- procedimentos de instalação, contingência, backup, restauro e atualização;
- manuais de utilizador, integrador e auditor;
- lista de limitações conhecidas;
- declaração da fronteira POS/módulo.

## Regra de mudança

Uma atualização do pacote fiscal só entra em produção após análise regulatória, testes de regressão, assinatura do artefacto, plano de rollout e possibilidade de rollback técnico que não reverta dados fiscais já emitidos.
