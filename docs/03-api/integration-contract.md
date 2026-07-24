# Contrato de integração (createDocument)

Fonte: `specs/openapi/openapi.yaml` (`0.1.6-draft`).

## Campos obrigatórios (DocumentIntent)

`external_id`, `document_type` (`invoice` \| `credit_note`), `currency` (`AOA`), `issued_at` (RFC3339 com offset), `seller` (`tax_id`, `name`), `lines[]` (`line_id`, `description`, `quantity`, `unit_price`, `tax_code`).

Opcional: `requested_series` (não autoriza a série; o módulo resolve), `customer`.

## Money e DecimalQuantity

- `Money`: string com exactamente 2 casas decimais (sem float).
- `DecimalQuantity`: string canónica positiva (limites no OpenAPI).

## Scope e seller

- O scope vem da identidade autenticada (não enviar `scope_id` no body).
- `seller.tax_id` deve coincidir com o NIF/identidade do scope; mismatch → `403` `FISCAL_SCOPE_MISMATCH`.
- **Sandbox:** identidade fiscal **sintéticas** (não oficiais), alinhadas ao scope provisionado pela BWB.
- **Produção futura:** dados reais só após processo formal; regras ainda abertas.

## Série, timezone e estado

- Timezone fiscal do scope (Angola: `Africa/Luanda`, tipicamente `+01:00`).
- Sucesso: `status` constante `sealed_locally`.
- **Ausentes:** `fiscal_number`, `authority_request_id`.
- `submission_id` é correlação interna — **não** é ID AGT.

## Idempotência e external_id

| Situação | Resultado |
|---|---|
| Mesma `Idempotency-Key` + mesmo pedido | `201` replay estável |
| Mesma key + body semanticamente diferente | `409` `FISCAL_IDEMPOTENCY_CONFLICT` |
| Nova key + mesmo `external_id` | `409` `FISCAL_EXTERNAL_ID_CONFLICT` |

Em timeout: reenviar com a **mesma** chave; não gerar chave nova.

## Limites, timeouts e 429

- `413` / `415` conforme OpenAPI.
- Timeouts: o cliente deve definir timeouts HTTP finitos e retries seguros.
- Sandbox rate-limit (edge): tipicamente `10r/s`, `burst=20` → `429`. Sem garantia de Problem JSON nem `Retry-After`.
- Kit POS (`pos-sandbox-kit.sh`): caso `rate_429` usa rajada sincronizada de 30 pedidos concorrentes (não ondas de 5) para exercer o limiter.

## Não afirmado neste contrato

Emissão/aceitação AGT, certificação, produção pronta, Cabo Verde.
