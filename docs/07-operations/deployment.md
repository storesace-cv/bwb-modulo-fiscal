# Estratégia de deployment

## Ambientes

- `local`: desenvolvimento (`FISCAL_AUTH_MODE=dev_static` apenas com `FISCAL_ENV=development` — ver [local-dev.md](../06-delivery/local-dev.md)).
- `test`: testes automáticos e simuladores.
- `sandbox` / **staging hostname**: `sandbox.fiscalmod.bwb.pt` — operacional; auth actual **`credential_store`** com `FISCAL_ENV=homologation` (**designação técnica BWB**; não é homologação oficial AGT, nem certificação).
- `production` hostname: `fiscalmod.bwb.pt` — **reservado** até aprovação operacional de produção.
- Não partilhar credenciais, chaves ou bases de dados entre ambientes.

Ver runbook de staging: [staging-runbook.md](staging-runbook.md). Progresso do projecto: [ROADMAP.md](../../ROADMAP.md).

## Cloud (staging D1/D2)

- **D1 (repo):** systemd, Nginx templates (open + deny-all de rollback), allowlists, scripts fail-closed, OpenAPI sandbox URL — sem acesso ao servidor.
- **D2 (host):** DNS A, TLS ACME, hardening, roles PostgreSQL, install — só após merge D1.
- Artefactos Linux por commit SHA + `SHA256SUMS`; migrate separado do restart; rollback de binário pós-migration só com prova N-1.
- API em `127.0.0.1`; TLS no Nginx.
- **Estado live actual do sandbox:** `/v1/documents` está **publicamente aberto com autenticação** (`credential_store`); sem token → **401**; Nginx `limit_req` **10r/s**, **burst=20**, `limit_req_status 429`. Isto **não** significa homologação AGT.
- **Deny-all** (`tls.deny.conf` / helpers `nginx-deny-all`) é artefacto de **rollback / fail-safe**, não o estado público live actual após S3C2 **CONFIRMED**.

## Edge

Artefactos por arquitetura suportada, manifesto de compatibilidade, assinatura e canal de atualização. Atualização: descarregar → verificar → preparar → parar com segurança → migrar → testar saúde → promover ou recuperar executável anterior (com as mesmas regras de compatibilidade de schema).

## Versionamento

Separar versão da aplicação, versão da API, versão do pacote fiscal, versão do schema de dados e versão do conector AGT. Todas ficam visíveis em diagnóstico e auditoria.
