# Diretrizes da API para POS

## Princípios

- REST/JSON sobre TLS; OpenAPI 3.1 como contrato.
- Versão principal no caminho (`/v1`).
- Resposta 2xx não implica necessariamente aceitação final pela AGT.
- Todas as criações usam `Idempotency-Key`.
- `external_id` é único no âmbito do integrador e empresa.
- Erros são estruturados, estáveis e acionáveis.
- Webhooks são assinados, repetíveis e consultáveis por polling.

## Estados técnicos (contrato)

Sequência feliz:

`received → validated → sealed_locally → queued_for_authority → authority_processing → authority_accepted`

Saídas alternativas no contrato: `rejected`, `authority_rejected`, `authority_outcome_unknown`.

- `sealed_locally` é estado **técnico** (número + persistência + artefacto local); **não** fecha DEC-API-004 nem afirma emissão jurídica perante a AGT.
- `contingency_pending` permanece no enum como **reservado**; o primeiro vertical slice **não** transita para este estado enquanto as regras oficiais de contingência estiverem abertas.
- `cancelled` **não** faz parte do contrato (DEC-API-002 aberta).

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
