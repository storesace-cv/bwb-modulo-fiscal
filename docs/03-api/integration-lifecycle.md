# Ciclo de integração de uma software house

1. Registo do parceiro e contactos técnicos.
2. Criação de credenciais sandbox.
3. Implementação da criação, consulta e webhooks.
4. Execução da suite de conformidade do integrador.
5. Testes de timeout, duplicação e contingência.
6. Validação de layouts/QR e documentos de amostra.
7. Credenciais de produção por empresa/estabelecimento.
8. Piloto controlado.
9. Aprovação para rollout.
10. Monitorização e suporte.

## Checklist mínimo

- Não atribui número fiscal localmente.
- Usa idempotência persistida antes da chamada.
- Guarda o ID fiscal devolvido.
- Trata estados assíncronos.
- Verifica assinatura de webhooks.
- Não reemite após timeout.
- Sabe operar com Edge e cloud.
- Apresenta mensagens de erro acionáveis ao operador.
- Impede edição do documento após emissão.
