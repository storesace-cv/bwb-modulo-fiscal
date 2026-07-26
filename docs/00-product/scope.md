# Âmbito do produto

## Incluído no MVP Angola

- Multiempresa, multiestabelecimento, múltiplos terminais e integradores; adesão FE no contribuinte (NIF) com estado `not_enrolled|pending|active|suspended`; séries/config por estabelecimento (`DEC-PROD-004`).
- API REST JSON e webhooks assinados.
- Modelo de tipos: todos os legalmente aplicáveis em SAF-T e/ou FE (`DEC-PROD-014`); 5 grupos; activação POS; implementação faseada sem limitar o modelo (`DEC-REG-003` = ordem do slice).
- Validação fiscal, séries, numeração, assinatura/encadeamento e QR quando aplicável.
- Comunicação assíncrona com a AGT, polling e callbacks internos.
- Resiliência técnica offline: outbox, reenvio, idempotência (`DEC-PROD-010`); **sem** declarar emissão offline certificada até regra oficial (`DEC-REG-004` / `AO-OFF-*`).
- Arquivo fiscal e trilho de auditoria **append-only** (`DEC-PROD-013`); retenção final depende da norma consolidada.
- Geração e validação SAF-T (AO).
- Portal para configuração, documentos, falhas, exportações e auditoria.
- Fiscal Edge para Linux (um escritor fiscal por instalação; POS via API local — `DEC-PROD-011`), com instalação e atualização assinada.
- Sandbox, documentação de integração, coleção de exemplos e POS demo.

## Fora do MVP

- Funções gerais de POS: catálogo comercial, stock, caixa, fidelização ou contabilidade.
- Processamento de pagamentos.
- Edição visual completa de layouts de fatura.
- Aplicações móveis nativas.
- Cabo Verde e SAF-T (CV).
- Integrações específicas por ERP fora da API/SDK padrão.

## Fronteira de responsabilidade

O POS gere a operação comercial e apresenta/imprime o resultado. O módulo fiscal é a **única autoridade** de emissão e numeração (`DEC-PROD-008`, ADR-0001): valida a intenção, atribui identidade fiscal, produz artefactos e gere a relação técnica com a AGT. Vários POS acedem **só via API**.

O POS não reserva números fiscais nem considera um documento aceite pela AGT só porque recebeu HTTP 2xx ou `sealed_locally`. Só o estado `accepted` afirma aceitação (`DEC-PROD-009`). Deve respeitar o estado fiscal retornado.
