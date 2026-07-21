# Desenvolvimento local (scaffold Fase 1)

Comandos mínimos para o binário `fiscal-api` (apenas `GET /v1/health`).

## Pré-requisitos

- Go **1.24+** (toolchain local testada: Go 1.26.x via Homebrew).

## Variáveis de ambiente

| Variável | Default | Descrição |
| --- | --- | --- |
| `FISCAL_HTTP_ADDR` | `:8080` | Endereço de listen |
| `FISCAL_APP_VERSION` | `0.0.0-dev` | Campo `version` do health |
| `FISCAL_PACKAGE` | `AO-UNDECLARED` | Campo `fiscalPackage` do health |
| `FISCAL_HTTP_READ_TIMEOUT` | `5s` | Timeout de leitura (ms inteiros ou duração Go) |
| `FISCAL_HTTP_WRITE_TIMEOUT` | `10s` | Timeout de escrita |
| `FISCAL_HTTP_IDLE_TIMEOUT` | `60s` | Timeout idle |
| `FISCAL_HTTP_SHUTDOWN_TIMEOUT` | `10s` | Timeout de graceful shutdown |

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
