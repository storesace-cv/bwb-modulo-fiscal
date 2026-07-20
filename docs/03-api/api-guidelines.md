# Diretrizes da API para POS

## Princípios

- REST/JSON sobre TLS; OpenAPI 3.1 como contrato.
- Versão principal no caminho (`/v1`).
- Resposta 2xx não implica necessariamente aceitação final pela AGT.
- Todas as criações usam `Idempotency-Key`.
- `external_id` é único no âmbito do integrador e empresa.
- Erros são estruturados, estáveis e acionáveis.
- Webhooks são assinados, repetíveis e consultáveis por polling.

## Estados conceptuais

`received → validated → fiscally_issued → queued_for_authority → authority_processing → authority_accepted`

Saídas alternativas: `rejected`, `authority_rejected`, `cancelled` ou `contingency_pending`, conforme regras do documento.

## Semântica de timeout

Após timeout, o cliente repete o mesmo pedido com a mesma chave. Nunca cria uma nova chave até consultar o estado do pedido original.

## Erro padrão

```json
{
  "type": "https://docs.example/errors/validation",
  "title": "Documento inválido",
  "status": 422,
  "code": "FISCAL_VALIDATION_FAILED",
  "request_id": "req_...",
  "errors": [{"field": "customer.tax_id", "code": "INVALID_TAX_ID", "message": "NIF inválido"}]
}
```

## Compatibilidade

Adicionar campos opcionais é compatível. Remover, renomear, mudar semântica, tornar obrigatório ou alterar enum exige nova versão principal ou período formal de migração.
