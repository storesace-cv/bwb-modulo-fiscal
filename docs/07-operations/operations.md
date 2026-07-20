# Operações

## SLOs iniciais para discussão

- API cloud mensal: 99,9% antes de compromisso comercial superior.
- Emissão local Edge: independente da cloud dentro das regras de contingência.
- Latência interna p95: objetivo a definir após protótipo e sem incluir espera AGT.
- RPO fiscal: próximo de zero para transações confirmadas.
- RTO: definido por cenário cloud/Edge e validado em exercício.

## Observabilidade

- Métricas por estado, sem expor conteúdo fiscal.
- Correlação POS → módulo → tentativa AGT → webhook.
- Alertas para backlog, rejeições, drift de relógio, disco, certificados e versões.
- Dashboard separado para saúde técnica e saúde fiscal.

## Runbooks prioritários

- AGT indisponível.
- Backlog de submissões.
- Certificado a expirar/comprometido.
- Colisão ou bloqueio de série.
- Edge offline por período prolongado.
- Disco cheio/corrupção local.
- Webhook de parceiro com falhas.
- Exportação SAF-T inválida.
- Incidente de segurança e acesso indevido.

## Mudanças

Rollout progressivo por versão e tenant. Atualizações fiscais têm aprovação de compliance. Nunca fazer rollback de dados emitidos; apenas de executáveis/configuração compatível.
