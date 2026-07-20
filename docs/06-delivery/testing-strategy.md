# Estratégia de testes

## Pirâmide

- Unitários: cálculos, validações, estados e mapeamentos.
- Propriedades: dinheiro, arredondamento, sequências e idempotência.
- Contrato: OpenAPI, SDKs, webhooks e compatibilidade.
- Integração: base de dados, outbox, AGT sandbox/simulador e SAF-T.
- Conformidade: vetores ligados a requisitos `AO-*`.
- Sistema: cloud e Edge, falhas e recuperação.
- Segurança: SAST, SCA, secrets, DAST e abuso.
- Desempenho: picos POS, concorrência por série e exportações grandes.

## Cenários obrigatórios

- chamadas simultâneas para a mesma série;
- repetição antes/depois de commit e após timeout;
- falha entre emissão e publicação na outbox;
- reinício do Edge durante emissão e sincronização;
- AGT lenta, indisponível, rejeição e callback duplicado;
- relógio incorreto;
- restauro de backup sem duplicar números;
- SAF-T grande e com documentos anulados;
- atualização de pacote fiscal com documentos pendentes.

## Evidências

CI deve produzir relatório por requisito, versões, hashes dos artefactos e vetores. Evidências de certificação são imutáveis e reproduzíveis.
