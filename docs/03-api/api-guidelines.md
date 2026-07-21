# Diretrizes da API para POS

Contrato: [`specs/openapi/openapi.yaml`](../../specs/openapi/openapi.yaml) (`0.1.2-draft`).

## Princípios

- REST/JSON sobre TLS; OpenAPI 3.1 como contrato.
- Versão principal no **base path** do servidor (`/v1`); paths no OpenAPI não repetem `/v1`.
- Endpoint canónico de criação: **`POST /v1/documents`** (`operationId: createDocument`) — cria e sela localmente de forma **atómica** (`201 Created`).
- Resposta 2xx **não** implica aceitação final pela AGT (DEC-API-004 aberta).
- Todas as criações usam `Idempotency-Key` (UUID) — alinhado a **AO-IDEM-001** (requisito de catálogo; conformidade certificável ainda não validada neste draft).
- `external_id` é único no âmbito do integrador/empresa (scope da identidade autenticada).
- O POS **não** atribui número fiscal nem escolhe a série efetiva — **AO-SEQ-002** (requisito de catálogo; conformidade ainda não validada).
- `scope_id` **não** vai no body; vem exclusivamente da identidade autenticada.
- `requested_series` é apenas referência; a `SeriesCode` efetiva é resolvida pelo módulo.
- Erros são estruturados (`Problem`), estáveis e acionáveis.
- Autenticação `bearerAuth` é **POS/módulo**, não credenciais AGT.
- Webhooks (quando existirem) serão assinados, repetíveis e consultáveis por polling.

## createDocument — sucesso e replay

- Sucesso: **`201 Created`** com corpo `CreateDocumentResponse` (`status` const `sealed_locally`; `submission_id` e `created_at` obrigatórios).
- Neste incremento **não** existem `fiscal_number` nem `authority_request_id` na resposta de createDocument.
- `submission_id` é **correlação interna** do módulo — **não** é ID AGT.
- `seller.tax_id` e `seller.name` são obrigatórios com pelo menos um carácter não-whitespace (sem formato NIF neste draft); `customer` permanece opcional sem a mesma obrigatoriedade.
- `external_id` e campos de linha `line_id` / `description` / `tax_code` exigem pelo menos um carácter não-whitespace (formato fiscal de `tax_code` não confirmado).
- Replay com a mesma `Idempotency-Key` e o mesmo pedido: **`201`** com os mesmos `id`, `external_id`, `status`, `submission_id` e `created_at` originais.
- Mesma chave com pedido semanticamente diferente: **`409`** `FISCAL_IDEMPOTENCY_CONFLICT`.
- `external_id` já usado noutro documento: **`409`** `FISCAL_EXTERNAL_ID_CONFLICT`.

## Estados técnicos (contrato)

Sequência feliz (ciclo completo futuro):

`received → validated → sealed_locally → queued_for_authority → authority_processing → authority_accepted`

Neste incremento, `createDocument` devolve apenas **`sealed_locally`** (estado derivado do ledger). A existência de mensagem na outbox **não** altera o estado HTTP para `queued_for_authority` sem transição de ledger correspondente.

- `sealed_locally` é estado **técnico**; **não** afirma emissão jurídica perante a AGT.
- `contingency_pending` permanece no enum como **reservado**.
- `cancelled` **não** faz parte do contrato (DEC-API-002 aberta).
- Não existe `GET /documents/{documentId}` neste draft (removido até haver implementação).

## Semântica de timeout

Após timeout, o cliente repete o mesmo pedido com a **mesma** `Idempotency-Key`. Nunca cria uma nova chave até obter a resposta original ou um conflito explícito.

## Content-Type e corpo

- `Content-Type` deve ser `application/json` (charset válido permitido).
- Propriedades desconhecidas e JSON adicional após o primeiro objeto são rejeitados (`422`).
- Corpo acima do limite: `413`.

## Autenticação e autorização

- `401` — credencial ausente ou inválida; inclui `WWW-Authenticate`.
- `403` — identidade autenticada sem autorização para o recurso.
- Produção sem validador real: fail-closed (detalhe de implementação no PR C2).

## Erro padrão (`Problem`)

```json
{
  "type": "urn:bwb:fiscal:error:validation",
  "title": "Documento inválido",
  "status": 422,
  "code": "FISCAL_VALIDATION_FAILED",
  "request_id": "req_01EXAMPLE000000000000000",
  "errors": [{"field": "currency", "code": "INVALID_ENUM", "message": "Valor não permitido"}]
}
```

`request_id` é gerado pelo servidor. O campo `type` usa URNs estáveis `urn:bwb:fiscal:error:…` (não URLs fictícias). Códigos estáveis incluem `FISCAL_UNAUTHORIZED`, `FISCAL_FORBIDDEN`, `FISCAL_IDEMPOTENCY_CONFLICT`, `FISCAL_EXTERNAL_ID_CONFLICT`, `FISCAL_PAYLOAD_TOO_LARGE`, `FISCAL_UNSUPPORTED_MEDIA_TYPE`, `FISCAL_VALIDATION_FAILED`, `FISCAL_INTERNAL_ERROR`.

## Exemplo

Coleção mínima sem segredos: [examples/create-document.http](examples/create-document.http).

## Compatibilidade

Adicionar campos opcionais é compatível. Remover, renomear, mudar semântica, tornar obrigatório ou alterar enum exige nova versão principal ou período formal de migração. Alterações deste draft (`0.1.2-draft`: `201`, remoção do GET, campos de resposta) estão no [CHANGELOG](../../CHANGELOG.md).
