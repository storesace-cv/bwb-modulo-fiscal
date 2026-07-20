# Âmbito do produto

## Incluído no MVP Angola

- Multiempresa, multiestabelecimento, múltiplos terminais e integradores.
- API REST JSON e webhooks assinados.
- Emissão de faturas e documentos retificativos definidos no catálogo aprovado.
- Validação fiscal, séries, numeração, assinatura/encadeamento e QR quando aplicável.
- Comunicação assíncrona com a AGT, polling e callbacks internos.
- Contingência e sincronização posterior.
- Arquivo fiscal e trilho de auditoria imutável.
- Geração e validação SAF-T (AO).
- Portal para configuração, documentos, falhas, exportações e auditoria.
- Fiscal Edge para Linux, com instalação e atualização assinada.
- Sandbox, documentação de integração, coleção de exemplos e POS demo.

## Fora do MVP

- Funções gerais de POS: catálogo comercial, stock, caixa, fidelização ou contabilidade.
- Processamento de pagamentos.
- Edição visual completa de layouts de fatura.
- Aplicações móveis nativas.
- Cabo Verde e SAF-T (CV).
- Integrações específicas por ERP fora da API/SDK padrão.

## Fronteira de responsabilidade

O POS gere a operação comercial e apresenta/imprime o resultado. O módulo fiscal decide se a intenção é fiscalmente válida, atribui a identidade fiscal, produz os artefactos fiscais e gere a relação técnica com a AGT.

O POS não pode considerar um documento definitivamente aceite apenas porque recebeu HTTP 2xx. Deve respeitar o estado fiscal retornado.
