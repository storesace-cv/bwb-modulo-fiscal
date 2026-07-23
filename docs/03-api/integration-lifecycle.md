# Ciclo de integração de uma software house

1. Registo do parceiro e contactos técnicos (canal BWB no onboarding).
2. Criação de credenciais sandbox do módulo (Bearer BWB; não AGT); no sandbox a identidade fiscal é **sintéticas**.
3. Implementação de `POST /v1/documents` com `Idempotency-Key` persistida antes da primeira tentativa.
4. Execução do kit `scripts/integration/pos-sandbox-kit.sh` e checklist [pos-acceptance-checklist.md](pos-acceptance-checklist.md).
5. Testes de timeout, duplicação (replay), conflitos `409`, `403` scope mismatch, `422`, `429`.
6. Validação de layouts/QR (fases posteriores).
7. Credenciais de produção por empresa/estabelecimento — quando existirem (fora deste draft).
8. Piloto controlado.
9. Aprovação para rollout.
10. Monitorização e suporte.

## Checklist mínimo (este contrato)

- Não atribui número fiscal localmente.
- Não trata `requested_series` como série efetiva autorizada.
- Não envia `scope_id` no body.
- Usa idempotência persistida (`Idempotency-Key` UUID).
- Em `201`, espera `sealed_locally`; não assume `fiscal_number` nem `authority_request_id`.
- Em timeout, reenvia a **mesma** chave.
- Trata `409` com códigos distintos; `429` com backoff + mesma chave.
- Guarda `request_id` quando presente (pode faltar em 429).

## Referências

- [quickstart.md](quickstart.md)
- [api-guidelines.md](api-guidelines.md)
- [OpenAPI 0.1.6-draft](../../specs/openapi/openapi.yaml)
