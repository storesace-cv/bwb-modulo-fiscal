# Máquina de estados documental

```mermaid
stateDiagram-v2
  [*] --> Received
  Received --> Rejected: validação falha
  Received --> Validated: validação passa
  Validated --> FiscallyIssued: número + assinatura + persistência
  FiscallyIssued --> ContingencyPending: sem comunicação autorizada
  FiscallyIssued --> QueuedForAuthority
  ContingencyPending --> QueuedForAuthority: comunicação recuperada
  QueuedForAuthority --> AuthorityProcessing: AGT recebe/requestID
  AuthorityProcessing --> AuthorityAccepted
  AuthorityProcessing --> AuthorityRejected
```

## Regras

- Transições são append-only e auditadas.
- Estados finais fiscais não são revertidos por atualização direta.
- Rejeição da AGT não autoriza automaticamente reutilização do número.
- Reprocessamento cria nova tentativa de submissão, não novo documento.
- Retificação/anulação é um comando legal separado, com referência ao original.
