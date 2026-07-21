# Máquina de estados documental

Sequência técnica alinhada ao OpenAPI `0.1.1-draft` (tarefa zero). DEC-API-004 permanece aberta para a semântica jurídica de emissão/aceitação.

```mermaid
stateDiagram-v2
  [*] --> Received
  Received --> Rejected: validação falha
  Received --> Validated: validação passa
  Validated --> SealedLocally: número + artefacto local + persistência
  SealedLocally --> QueuedForAuthority
  QueuedForAuthority --> AuthorityProcessing: autoridade recebe ou requestID
  QueuedForAuthority --> AuthorityOutcomeUnknown: timeout / resultado da entrega incerto
  AuthorityProcessing --> AuthorityAccepted
  AuthorityProcessing --> AuthorityRejected
  AuthorityProcessing --> AuthorityOutcomeUnknown: resultado final incerto
  AuthorityOutcomeUnknown --> AuthorityProcessing: autoridade confirma processamento
  AuthorityOutcomeUnknown --> AuthorityAccepted: reconciliação confirma aceitação
  AuthorityOutcomeUnknown --> AuthorityRejected: reconciliação confirma rejeição
  AuthorityOutcomeUnknown --> QueuedForAuthority: retry seguro com o mesmo identificador
```

## Regras

- Transições são append-only e auditadas.
- `sealed_locally` não afirma por si emissão fiscal jurídica perante a AGT (DEC-API-004).
- Estados finais fiscais não são revertidos por atualização direta.
- Rejeição da autoridade não autoriza automaticamente reutilização do número.
- `authority_outcome_unknown` cobre resultado incerto na entrega (ex.: VS-T12 — worker perde o resultado durante a submissão) e no processamento; exige reconciliação.
- Retry a partir de `authority_outcome_unknown` para `queued_for_authority` usa o **mesmo identificador estável de submissão**; **não** cria novo documento nem nova selagem.
- Reprocessamento/reconciliação cria nova **tentativa** de submissão (mesmo documento e mesma selagem), não um novo documento fiscal.
- `contingency_pending` existe no contrato OpenAPI como **reservado**; o primeiro vertical slice **não** implementa transição para este estado; regras oficiais de contingência (`AO-OFF-*`, DEC-REG-004) permanecem abertas — não inventar o fluxo aqui.
- Retificação/anulação é um comando legal separado, com referência ao original; `cancelled` não está no contrato (DEC-API-002).
