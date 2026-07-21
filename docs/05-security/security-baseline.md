# Baseline de segurança

## Identidade e acesso

- Autenticação forte para portal e credenciais distintas para máquinas.
- Autorização por tenant, contribuinte, estabelecimento e capacidade.
- Privilégio mínimo e segregação entre suporte, operação e gestão de chaves/credenciais.
- Credenciais de homologação nunca funcionam em produção; ambientes HML e PRD rigorosamente isolados.

## Segredos e chaves

- Abstração `SecretStore` (Secret Manager / KMS / HSM conforme fornecedor ainda por decidir).
- Em produção, provisionamento de segredos por **bootstrap fora da UI** (CLI, agente ou vault): TLS autenticado, write-only, gravação direta no `SecretStore`, sem persistência intermédia, sem logs do segredo, sem retorno nem visualização posterior.
- A UI do backoffice não recebe, armazena nem exibe material secreto; mostra apenas metadados seguros (fingerprint derivado de chave pública ou metadados do provisionamento, estado, validade, origem, ambiente, rotação/revogação).
- **Proibida** cópia automática de chaves privadas cloud→Edge ou Edge→cloud; qualquer provisionamento é explícito, individual, autenticado e auditado.
- Custódia da chave privada do **contribuinte** no `SecretStore` da plataforma condicionada a autorização do contribuinte **e** a DEC-REG-KEY-CUSTODY (permissão oficial AGT).
- Chaves de teste do vertical slice: par RSA efémero, privada nunca persistida nem no Git.
- Rotação, revogação, expiração e inventário de refs auditados.
- Artefactos e atualizações Edge assinados.

## Criptografia

- TLS em trânsito e cifra em repouso.
- Edge: keystore próprio e política separada do cloud.
- Assinatura interna da API (se aplicável) distinta da assinatura fiscal AGT (esta depende de fontes oficiais).

## Aplicação

- Validação estrita de schema e limites de payload.
- Proteção contra replay e abuso.
- Dependências fixadas, SBOM e análise de vulnerabilidades.
- Segredos fora do repositório e das imagens.
- Logs sem dados fiscais completos ou segredos.

## Disponibilidade

- Backups cifrados, testados e com retenção definida.
- RPO/RTO aprovados antes do piloto.
- Testes de recuperação, perda de rede, disco cheio, relógio incorreto e corrupção.

## Auditoria

Eventos devem incluir ator, ação, recurso, instante, origem, correlação e resultado, evitando dados excessivos. Acesso de suporte a informação fiscal é justificado e auditado; suporte não acede a material secreto nem a dados de outro tenant.
