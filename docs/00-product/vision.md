# Visão do produto

## Problema

Software houses de POS precisam cumprir regras fiscais e integrar com a AGT, mas não devem duplicar em cada produto a complexidade de assinatura, transmissão, contingência, auditoria e SAF-T.

## Proposta

O BWB Módulo Fiscal oferece uma API estável e simples. Recebe a intenção comercial do POS, valida-a, atribui identidade fiscal, preserva evidências, comunica com a AGT e devolve o resultado necessário para impressão ou entrega do documento.

## Utilizadores

- Software houses e integradores de POS.
- Comerciantes/contribuintes e respetivos operadores.
- Equipas de suporte e conformidade da BWB.
- Auditores e autoridades, dentro das permissões legais.

## Princípios

- Compliance por desenho e rastreabilidade artigo → requisito → teste → evidência.
- API pública desacoplada da API da autoridade tributária.
- Paridade funcional entre cloud e Edge, respeitando limites de contingência.
- Imutabilidade de resultados fiscais.
- Integração simples, observabilidade segura e recuperação automática.
- Pacotes fiscais independentes por país e versionados.

## Métricas iniciais

- Integração sandbox por uma software house em até 5 dias úteis.
- Zero duplicações fiscais causadas por reenvio.
- 100% das regras críticas ligadas a testes automatizados.
- Recuperação automática de transmissões após falha de rede.
- Rastreabilidade completa de cada transição de estado.
