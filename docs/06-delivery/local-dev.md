# Desenvolvimento local (scaffold Fase 1 + fundação de persistência)

## Pré-requisitos

- Baseline de linguagem no `go.mod`: **Go 1.25.0**.
- Toolchain de CI/deploy: **Go 1.26.x** (ver [Go Release Policy](https://go.dev/doc/devel/release)).
- PostgreSQL 16+ para testes cloud (`FISCAL_TEST_DATABASE_URL`); SQLite (pure Go) para Edge/local.

## Variáveis HTTP (`fiscal-api`)

| Variável | Default | Descrição |
| --- | --- | --- |
| `FISCAL_HTTP_ADDR` | `127.0.0.1:8080` | Listen (loopback por omissão) |
| `FISCAL_APP_VERSION` | `0.0.0-dev` | Health `version` |
| `FISCAL_PACKAGE` | `AO-UNDECLARED` | Health `fiscalPackage` |
| `FISCAL_HTTP_READ_TIMEOUT` | `5s` | Timeout de leitura |
| `FISCAL_HTTP_READ_HEADER_TIMEOUT` | `5s` | Timeout de headers |
| `FISCAL_HTTP_WRITE_TIMEOUT` | `10s` | Timeout de escrita |
| `FISCAL_HTTP_IDLE_TIMEOUT` | `60s` | Timeout idle |
| `FISCAL_HTTP_SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown |

## Variáveis de base de dados (`fiscal-migrate` / testes)

| Variável | Descrição |
| --- | --- |
| `FISCAL_DATABASE_DRIVER` | `postgres` ou `sqlite` |
| `FISCAL_DATABASE_URL` | DSN Postgres ou path do ficheiro SQLite |
| `FISCAL_TEST_DATABASE_URL` | DSN Postgres para testes de integração |

## Migrations

- SQL embutido no binário (`embed`); forward-only (`*.up.sql`).
- Controlo Postgres: `public.bwb_schema_migrations`.
- Tabelas da aplicação: schema `fiscal.*`.
- CLI produção: só `up` e `version` — ver [migrate-runbook.md](migrate-runbook.md).

```bash
# SQLite local
export FISCAL_DATABASE_DRIVER=sqlite
export FISCAL_DATABASE_URL=./tmp/fiscal.db
go run ./cmd/fiscal-migrate up
go run ./cmd/fiscal-migrate version

# Postgres
export FISCAL_DATABASE_DRIVER=postgres
export FISCAL_DATABASE_URL='postgres://fiscal:fiscal@127.0.0.1:5432/fiscal?sslmode=disable'
go run ./cmd/fiscal-migrate up
```

## Comandos

```bash
go test ./...
go test -race ./...
gofmt -w .
go vet ./...
bash scripts/check-migrations.sh

# Tools (módulo separado tools/)
cd tools && go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
cd tools && go install github.com/google/go-licenses@v1.6.0
govulncheck ./...
```

```bash
go run ./cmd/fiscal-api
curl -sS http://127.0.0.1:8080/v1/health
```

Este incremento inclui `SealInTx` (API interna) e testes VS-T01–VS-T07. **Não** inclui `POST /documents`, worker AGT, JWS nem ficheiros `.env`.
