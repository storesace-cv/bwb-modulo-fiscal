# Fiscal Edge — arquitetura local Linux

## Objetivos

- API em loopback ou rede local explicitamente autorizada.
- Continuidade durante falhas de Internet.
- Persistência após reinício abrupto.
- Sincronização segura e observável.
- Instalação, diagnóstico e atualização simples.

## Componentes

- `edge-api`: endpoint compatível com a API pública relevante.
- `fiscal-core`: regras, cálculo, séries e geração de artefactos.
- `edge-store`: base transacional e journal durável.
- `sync-worker`: comunicação com cloud/AGT e backoff controlado.
- `edge-admin`: CLI local para estado, exportação diagnóstica e manutenção.
- `updater`: verifica assinatura, compatibilidade e saúde antes de promover versão.

## Restrições

- Um único proprietário de cada série num dado instante.
- Não permitir dois Edge independentes a emitir na mesma série sem protocolo formal de partição.
- Relógio monitorizado; desvios relevantes bloqueiam ou degradam conforme regra aprovada.
- Base de dados e chaves cifradas em repouso.
- Credenciais únicas por instalação, nunca imagens clonadas com o mesmo segredo.

## Instalação inicial

Suportar Ubuntu LTS/Debian e distribuição por pacote ou contentor conforme validação. Serviço gerido por `systemd`, healthcheck local, logs estruturados e rotação segura.
