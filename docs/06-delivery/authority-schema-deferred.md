# Schema: authority attempts/responses

DDL criado na migration **`0004`** (`ExpectedVersion=4`) para o worker outbox + **simulador** AGT interno.

**Não** é integração HML/PRD AGT; **não** exige credenciais; **não** constitui evidência de conformidade.

## `fiscal.authority_attempts`

- `id` TEXT PK
- `document_id` TEXT NOT NULL FK → `fiscal.documents`
- `submission_id` TEXT NOT NULL (mesmo id estável da outbox)
- `attempt_no` BIGINT NOT NULL CHECK `> 0`
- UNIQUE `(submission_id, attempt_no)`
- `sent_at` TIMESTAMPTZ/TEXT UTC
- Append-only (triggers)

## `fiscal.authority_responses`

- `id` TEXT PK
- `attempt_id` TEXT NOT NULL FK → `authority_attempts`
- `authority_request_id` TEXT NULL
- `outcome` TEXT NOT NULL ∈ `authority_accepted` \| `authority_rejected` \| `authority_outcome_unknown`
- `received_at` TIMESTAMPTZ/TEXT UTC
- UNIQUE `(attempt_id)`
- Append-only (triggers)

Invariante: a resposta não apaga nem substitui o pedido/artefacto enviados.

Worker: [`internal/persistence/outbox.go`](../../internal/persistence/outbox.go) · simulador: [`internal/authority/simulator`](../../internal/authority/simulator).
