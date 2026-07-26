# Directrizes da API (POS / software houses)

Contrato: [`specs/openapi/openapi.yaml`](../../specs/openapi/openapi.yaml) (`0.1.6-draft`).

- Autenticação: **Bearer credential issued by BWB for the sandbox** (não AGT).
- `POST /v1/documents` (`createDocument`) com `Idempotency-Key` UUID.
- Sucesso: `201` + `status: sealed_locally` — **não** significa emissão/aceitação AGT (`DEC-PROD-009`); sem `fiscal_number` / `authority_request_id`. Só `accepted` afirma aceitação pela AGT.
- Erros Problem (quando aplicável): 401/403/409/422/413/415/500. **429** (rate limit edge): status garantido; Problem/`Retry-After` **não** garantidos.
- Money / DecimalQuantity como strings; `issued_at` com offset Angola (`+01:00`).
- Tipo documental: código de produto / alias mapeado para conceito canónico (`DEC-PROD-007`); **não** enviar códigos FE/SAF-T crus sem mapping. Adaptadores por canal são do módulo.
- Autoridade: o módulo numera e emite; o POS **não** atribui número fiscal. N POS → mesma API (`DEC-PROD-008`).
- Offline: retries/idempotência/outbox são técnicos; **não** tratar selagem local offline como emissão certificada (`DEC-PROD-010`).

## Documentação S4

- [quickstart.md](quickstart.md)
- [integration-contract.md](integration-contract.md)
- [onboarding.md](onboarding.md)
- [pos-acceptance-checklist.md](pos-acceptance-checklist.md)
- [publishing.md](publishing.md)
- [integration-lifecycle.md](integration-lifecycle.md)
- Exemplos: [examples/](examples/)
