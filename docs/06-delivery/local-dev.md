# Desenvolvimento local (scaffold Fase 1)

Comandos mínimos para o binário `fiscal-api` (apenas `GET /v1/health`).

## Pré-requisitos

- Baseline de linguagem no `go.mod`: **Go 1.25.0**.
- Toolchain de CI/deploy: **Go 1.26.x** mais recente (corrigida; ver [Go Release Policy](https://go.dev/doc/devel/release)).
- Go 1.24 já não recebe correções de segurança após o lançamento de Go 1.26.

## Variáveis de ambiente

| Variável | Default | Descrição |
| --- | --- | --- |
| `FISCAL_HTTP_ADDR` | `127.0.0.1:8080` | Endereço de listen (loopback por omissão; cloud/outras interfaces exigem valor explícito) |
| `FISCAL_APP_VERSION` | `0.0.0-dev` | Campo `version` do health |
| `FISCAL_PACKAGE` | `AO-UNDECLARED` | Campo `fiscalPackage` do health |
| `FISCAL_HTTP_READ_TIMEOUT` | `5s` | Timeout de leitura completa do pedido (ms inteiros ou duração Go) |
| `FISCAL_HTTP_READ_HEADER_TIMEOUT` | `5s` | Timeout da leitura dos headers (proteção contra slowloris; ms ou duração Go) |
| `FISCAL_HTTP_WRITE_TIMEOUT` | `10s` | Timeout de escrita |
| `FISCAL_HTTP_IDLE_TIMEOUT` | `60s` | Timeout idle |
| `FISCAL_HTTP_SHUTDOWN_TIMEOUT` | `10s` | Timeout de graceful shutdown |

Limite técnico fixo: `MaxHeaderBytes` = **64 KiB** (não configurável por ambiente).

O default de `FISCAL_HTTP_ADDR` escuta apenas em loopback. Em cloud (ou qualquer exposição não local), definir explicitamente o endereço desejado (ex.: `0.0.0.0:8080` atrás de um proxy/rede controlada).

## Comandos

```bash
# Testes
go test ./...
go test -race ./...

# Formatação e análise estática
gofmt -w .
go vet ./...

# Executar API
go run ./cmd/fiscal-api

# Health check
curl -sS http://127.0.0.1:8080/v1/health
```

Encerramento gracioso: `SIGINT` / `SIGTERM`.

Este incremento não inclui base de dados, Docker, ORM nem emissão fiscal.
