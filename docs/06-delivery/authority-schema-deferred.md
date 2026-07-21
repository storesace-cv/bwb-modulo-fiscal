# Schema previsto: authority attempts/responses (adiado)

DDL **não** criado nos PRs de fundação/selagem. Criar no PR do worker outbox/simulador.

## `fiscal.authority_attempts` (previsto)

- `id` TEXT PK
- `document_id` TEXT NOT NULL FK → `fiscal.documents`
- `submission_id` TEXT NOT NULL (mesmo id estável da outbox)
- `attempt_no` BIGINT NOT NULL CHECK `> 0`
- UNIQUE `(submission_id, attempt_no)`
- `sent_at` TIMESTAMPTZ/TEXT UTC

## `fiscal.authority_responses` (previsto)

- `id` TEXT PK
- `attempt_id` TEXT NOT NULL FK → `authority_attempts`
- `authority_request_id` TEXT NULL
- `outcome` TEXT NOT NULL (enum técnico alinhado ao contrato)
- `received_at` TIMESTAMPTZ/TEXT UTC

Invariante: a resposta não apaga nem substitui o pedido/artefacto enviados.
