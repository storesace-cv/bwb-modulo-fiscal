# Outbox: at-least-once, deduplicação e reconciliação (slice)

**Âmbito:** worker outbox → **simulador** AGT interno (`RM-TX-006`/`RM-TX-007`/`RM-TX-008`).
**Não** é integração HML/PRD AGT; **não** constitui evidência de conformidade nem exactly-once.

## Entrega at-least-once

1. Após `SealInTx`, a outbox fica `pending` com `submission_id` estável (UUID).
2. O worker faz claim → `in_flight`, chama a autoridade, e só então marca `succeeded` **ou** devolve a `pending` com backoff.
3. Crash a meio (`in_flight` stale): reclaim automático após ~30s → nova tentativa (at-least-once).
4. Indisponibilidade do simulador (`ErrUnavailable`): outbox volta a `pending`; ledger permanece `sealed_locally` (nunca `authority_accepted`).

## Deduplicação

- Chave estável: `submission_id` (igual na outbox e em `authority_attempts`).
- `(submission_id, attempt_no)` UNIQUE; uma resposta por `attempt_id`.
- Se o ledger já está em estado terminal de autoridade (`authority_accepted` / `authority_rejected` / `authority_outcome_unknown`) e a outbox é reprocessada, o worker **não** cria nova tentativa (VS-T11).

## Reconciliação (slice)

| Situação | Comportamento |
|---|---|
| Claim / antes do Submit | Ledger → `authority_processing` (VS-T09); outbox `in_flight` |
| `authority_accepted` / `authority_rejected` | Terminal no slice; número fiscal **não** é reutilizado |
| `authority_outcome_unknown` | Persistido; exige reconciliação operacional futura — **não** antecipa aceite AGT |
| Indisponibilidade / erro de transporte | Outbox `pending`; ledger reverte a `sealed_locally` se estava em processing |
| Resposta/callback duplicado | Idempotência por `submission_id` + estado terminal |
| Worker reinicia | Reclaim `in_flight` + nova tentativa; sem exactly-once |

Reconciliação **oficial** AGT (consulta FE, FE-RNG, credenciais) permanece fora de âmbito até `RM-FE-*` / acesso AGT.

## Referências

- Worker: [`internal/persistence/outbox.go`](../../internal/persistence/outbox.go)
- Simulador: [`internal/authority/simulator`](../../internal/authority/simulator)
- Schema: [`authority-schema-deferred.md`](authority-schema-deferred.md)
- Spec do slice: [`first-vertical-slice.md`](first-vertical-slice.md) (VS-T08/10/11/12)
