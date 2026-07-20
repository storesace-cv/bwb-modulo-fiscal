# Baseline de segurança

## Identidade e acesso

- Autenticação forte para portal e credenciais distintas para máquinas.
- Autorização por tenant, empresa, estabelecimento e capacidade.
- Privilégio mínimo e segregação entre suporte, operação e gestão de chaves.
- Credenciais de sandbox nunca funcionam em produção.

## Criptografia

- TLS em trânsito e cifra em repouso.
- Chaves fiscais em HSM/KMS quando disponível; no Edge, keystore protegido pelo sistema.
- Rotação, revogação e inventário de chaves.
- Artefactos e atualizações Edge assinados.

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

Eventos devem incluir ator, ação, recurso, instante, origem, correlação e resultado, evitando dados excessivos. Acesso de suporte a informação fiscal é justificado e auditado.
