# Catálogo inicial de requisitos

> Rascunho para validação jurídico-fiscal. Matriz confirmada (norma): [`compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md). Matriz provisória (resto): [`PROVISIONAL-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md).

| ID | Requisito | Criticidade | Evidência esperada | Estado normativo |
|---|---|---:|---|---|
| ASM-REG-001 | O módulo certificado é a autoridade fiscal; POS integrados não são certificados individualmente. | Bloqueante | Confirmação/aceitação no processo AGT | provisório |
| AO-ID-001 | Associar documento a contribuinte, estabelecimento, terminal, software e versão. | Alta | Teste e registo de auditoria | provisório (`partial`) |
| AO-DOC-001 | Validar campos obrigatórios por tipo documental antes da emissão. | Alta | Vetores positivos/negativos | provisório (`scaffold`) |
| AO-DOC-002 | Impedir alteração destrutiva após emissão. | Crítica | Teste de imutabilidade | **`confirmed_normative`** (DP 71 + DE 74; ≠ AGT) |
| AO-SEQ-001 | Garantir numeração única e sequencial por série. | Crítica | Testes concorrentes e recuperação | provisório (`partial`) |
| AO-SEQ-002 | Impedir que o POS atribua o número fiscal final. | Crítica | Teste de API e autorização | provisório (`partial`) |
| AO-IDEM-001 | Repetir pedido com a mesma chave sem nova emissão. | Crítica | Teste de timeout/reenvio | provisório (`scaffold`) |
| AO-TAX-001 | Calcular e validar impostos com precisão decimal e regras versionadas. | Crítica | Vetores de cálculo e arredondamento | provisório (`partial`) |
| AO-CRYPTO-001 | Assinar/encadear documentos conforme especificação vigente. | Crítica | Verificação criptográfica independente | provisório (`partial`) |
| AO-KEY-001 | Proteger chaves, rotação e acesso com segregação de funções. | Crítica | Auditoria e teste de rotação | provisório (`blocked`) |
| AO-AGT-001 | Transmitir no formato exigido e preservar requestID/resposta. | Crítica | Teste de integração AGT | provisório (`pending_validation`) |
| AO-AGT-002 | Tratar receção e aceitação fiscal como estados distintos. | Crítica | Máquina de estados testada | provisório (`partial`) |
| AO-OFF-001 | Emitir em contingência apenas nas condições autorizadas. | Crítica | Testes de falha e reconciliação | provisório (`partial`) |
| AO-OFF-002 | Sincronizar sem renumerar ou duplicar documentos. | Crítica | Teste de recuperação | provisório (`partial`) |
| AO-AUD-001 | Manter trilho append-only de ações e transições. | Alta | Consulta e prova de integridade | provisório (`scaffold`) |
| AO-SAF-001 | Exportar SAF-T (AO) completo e conforme schema aplicável. | Crítica | Validação XSD e vetores dourados | provisório (`pending_validation`) |
| AO-SAF-002 | Incluir anulados/retificativos preservando sequencialidade. | Crítica | Amostra SAF-T validada | provisório (`pending_validation`) |
| AO-OPS-001 | Permitir backup, restauro e disaster recovery sem colisões. | Alta | Ensaio documentado | provisório (`scaffold`) |
| AO-UPD-001 | Aceitar apenas versões Edge assinadas e compatíveis. | Alta | Teste de atualização adulterada | provisório (`scaffold`) |
