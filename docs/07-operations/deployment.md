# Estratégia de deployment

## Ambientes

- `local`: desenvolvimento.
- `test`: testes automáticos e simuladores.
- `sandbox` / **staging hostname**: `sandbox.fiscalmod.bwb.pt` (operacional staging; auth actual `dev_static`).
- `production` hostname: `fiscalmod.bwb.pt` — **reservado** até autenticação real e aprovação operacional.
- Não partilhar credenciais, chaves ou bases de dados entre ambientes.

Ver runbook de staging: [staging-runbook.md](staging-runbook.md).

## Cloud (staging D1/D2)

- **D1 (repo):** systemd, Nginx templates, allowlists, scripts fail-closed, OpenAPI sandbox URL — sem acesso ao servidor.
- **D2 (host):** DNS A, TLS ACME, hardening, roles PostgreSQL, install — só após merge D1.
- Artefactos Linux por commit SHA + `SHA256SUMS`; migrate separado do restart; rollback de binário pós-migration só com prova N-1.
- API em `127.0.0.1`; TLS no Nginx; `/v1/documents` deny-all no Git.

## Edge

Artefactos por arquitetura suportada, manifesto de compatibilidade, assinatura e canal de atualização. Atualização: descarregar → verificar → preparar → parar com segurança → migrar → testar saúde → promover ou recuperar executável anterior (com as mesmas regras de compatibilidade de schema).

## Versionamento

Separar versão da aplicação, versão da API, versão do pacote fiscal, versão do schema de dados e versão do conector AGT. Todas ficam visíveis em diagnóstico e auditoria.
