# Ciclo de integração de uma software house

1. Registo do parceiro e contactos técnicos.
2. Criação de credenciais sandbox **do módulo** (não credenciais AGT).
3. Implementação de `POST /v1/documents` (`createDocument`) com `Idempotency-Key` persistida antes da primeira tentativa (**AO-IDEM-001** — requisito de catálogo; conformidade ainda não validada).
4. Execução da suite de conformidade do integrador (quando publicada).
5. Testes de timeout, duplicação (replay) e conflitos `409`.
6. Validação de layouts/QR e documentos de amostra (fases posteriores).
7. Credenciais de produção por empresa/estabelecimento (módulo).
8. Piloto controlado.
9. Aprovação para rollout.
10. Monitorização e suporte.

## Checklist mínimo (este contrato)

- Não atribui número fiscal localmente (**AO-SEQ-002** — requisito de catálogo; conformidade ainda não validada).
- Não trata `requested_series` como série efetiva autorizada; o módulo resolve a série.
- Não envia `scope_id` no body (scope vem da identidade autenticada).
- Usa idempotência persistida antes da chamada (`Idempotency-Key` UUID).
- Em sucesso/`201`, espera `status: sealed_locally`; não assume `fiscal_number` nem `authority_request_id` neste incremento.
- Guarda `id`, `external_id`, `submission_id` (se presente) e `created_at` para correlação; não confunde `submission_id` com ID AGT.
- Em timeout, reenvia a **mesma** chave; não cria chave nova.
- Trata `409` com códigos distintos (`FISCAL_IDEMPOTENCY_CONFLICT` vs `FISCAL_EXTERNAL_ID_CONFLICT`).
- Apresenta mensagens de erro acionáveis ao operador (`Problem.code` / `errors[]`).
- Impede edição do documento após selagem local.
- Não depende de `GET /documents/{id}` neste draft (endpoint removido até implementação).

## Referências

- [api-guidelines.md](api-guidelines.md)
- [examples/create-document.http](examples/create-document.http)
- [OpenAPI 0.1.2-draft](../../specs/openapi/openapi.yaml)
