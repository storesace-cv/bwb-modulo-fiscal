# Início rápido — integração POS (sandbox)

Contrato: OpenAPI `0.1.6-draft` (`specs/openapi/openapi.yaml`).  
Sandbox: `https://sandbox.fiscalmod.bwb.pt/v1` (host confirmado S3C2; não é produção).

**Isto não é emissão fiscal perante a AGT.** Um `201` com `status: sealed_locally` significa apenas selagem local no módulo. Não existe `fiscal_number` nem `authority_request_id` neste draft.

## Autenticação

- Header: `Authorization: Bearer <token>`
- Credencial: **Bearer credential issued by BWB for the sandbox** (não AGT).
- O contacto operacional e o canal seguro de credenciais são fornecidos pela BWB durante o onboarding.

## Pedido mínimo

1. `Idempotency-Key`: UUID (obrigatório).
2. `POST /v1/documents` com JSON `DocumentIntent`.
3. `issued_at` com offset fiscal Angola (ex. `+01:00`); não usar `Z` como atalho.
4. `currency`: `AOA`; `seller.tax_id` = identidade fiscal do **scope** (no sandbox: identificador **sintético** provisionado pela BWB).

Ver exemplos em [examples/](examples/).

## Respostas

| Código | Significado | Acção típica |
|---|---|---|
| 201 | Criado/replay idempotente; `sealed_locally` | Guardar `id`, `external_id`, `submission_id`, `created_at` |
| 401 | Token ausente/inválido/revogado | Reautenticar; não inventar chave nova |
| 403 | Incl. `FISCAL_SCOPE_MISMATCH` | Corrigir seller vs scope |
| 409 | `FISCAL_IDEMPOTENCY_CONFLICT` ou `FISCAL_EXTERNAL_ID_CONFLICT` | Não reutilizar chave/body incorrectos |
| 422 | `FISCAL_VALIDATION_FAILED` | Corrigir campos (`errors[]`) |
| 429 | Rate limit (edge) | Backoff; **mesma** Idempotency-Key no retry; body Problem/`Retry-After` **não** garantidos |
| 5xx | Erro interno | Retry seguro com a **mesma** chave; abrir suporte com `request_id` se existir |

`request_id` pode estar ausente (ex. 429 Nginx) — não falhar o cliente por isso.

## Kit E2E

Ver [scripts/integration/README.md](../../scripts/integration/README.md).

## Referências

- [integration-contract.md](integration-contract.md)
- [onboarding.md](onboarding.md)
- [pos-acceptance-checklist.md](pos-acceptance-checklist.md)
