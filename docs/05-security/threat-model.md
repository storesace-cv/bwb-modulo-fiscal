# Modelo inicial de ameaças

| Ameaça | Impacto | Controlo inicial |
|---|---|---|
| POS repete pedido após timeout | Fatura duplicada | Idempotência durável |
| POS tenta escolher número | Quebra de sequência | Numeração exclusiva no módulo |
| Alteração de documento emitido | Fraude/inconformidade | Livro imutável e retificação formal |
| Roubo de chave | Documentos falsos | HSM/KMS, rotação e segregação |
| Webhook falsificado | Estado incorreto no POS | Assinatura e proteção anti-replay |
| Dois Edge usam a mesma série | Colisão fiscal | Propriedade/partição formal de séries |
| Relógio do terminal manipulado | Cronologia inválida | Fonte de tempo confiável e deteção de drift |
| Atualização Edge adulterada | Compromisso local | Pacotes assinados e verificação pré-instalação |
| Operador de suporte consulta dados sem motivo | Violação de privacidade | RBAC, justificação e auditoria |
| AGT indisponível | Paragem de vendas | Contingência legal e outbox durável |
| Disco Edge cheio/corrompido | Perda ou bloqueio | Monitorização, reservas e recuperação testada |

O modelo deve evoluir com diagramas de fluxo de dados e sessões STRIDE antes do piloto.
