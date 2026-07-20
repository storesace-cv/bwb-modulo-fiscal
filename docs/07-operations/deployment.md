# Estratégia de deployment

## Ambientes

- `local`: desenvolvimento.
- `test`: testes automáticos e simuladores.
- `sandbox`: integração de software houses.
- `staging`: réplica operacional sem dados reais.
- `production`: dados e credenciais reais.

Não partilhar credenciais, chaves ou bases de dados entre ambientes.

## Cloud

Infraestrutura como código, imagens imutáveis, migrações verificadas, rollout canário e healthchecks. Base de dados com backup point-in-time e restauração ensaiada.

## Edge

Artefactos por arquitetura suportada, manifesto de compatibilidade, assinatura e canal de atualização. Atualização: descarregar → verificar → preparar → parar com segurança → migrar → testar saúde → promover ou recuperar executável anterior.

## Versionamento

Separar versão da aplicação, versão da API, versão do pacote fiscal, versão do schema de dados e versão do conector AGT. Todas ficam visíveis em diagnóstico e auditoria.
