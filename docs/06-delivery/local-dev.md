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
- **Migration `0002` (DEC-TIME-001):** aborta se existirem linhas em `documents` **ou** `idempotency_records` (hashes `canonical_v1` / sem contexto temporal). Em desenvolvimento, **recriar a BD**; **não** recalcular hashes existentes.

```bash
# SQLite local
export FISCAL_DATABASE_DRIVER=sqlite
export FISCAL_DATABASE_URL=./tmp/fiscal.db
# Se a BD foi criada antes de 0002: rm ./tmp/fiscal.db
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

## Variáveis do endpoint de documentos (`fiscal-api`)

A API **não** executa migrations no arranque — usar `fiscal-migrate` antes.

| Variável | Obrigatório | Descrição |
| --- | --- | --- |
| `FISCAL_ENV` | sim | Deve ser `development` para `dev_static` |
| `FISCAL_AUTH_MODE` | sim | Apenas `dev_static` neste incremento (fail-closed se ausente) |
| `FISCAL_AUTH_DEV_TOKEN` | sim | Bearer do módulo; mínimo 32 bytes; nunca em logs |
| `FISCAL_AUTH_DEV_SCOPE_ID` | sim | `scope_id` injetado na identidade (não vem do body) |
| `FISCAL_SCOPE_TIMEZONE` | sim | IANA do scope; neste incremento apenas `Africa/Luanda` (DEC-TIME-001) |
| `FISCAL_AUTH_DEV_FORBIDDEN_TOKEN` | não | Token válido que devolve 403 (testes) |
| `FISCAL_SERIES_MODE` | sim | Apenas `static` neste incremento |
| `FISCAL_SERIES_EFFECTIVE_CODE` | sim | `SeriesCode` efetiva autorizada (POS não controla) |
| `FISCAL_DATABASE_DRIVER` | sim | `postgres` ou `sqlite` |
| `FISCAL_DATABASE_URL` | sim | DSN Postgres ou path SQLite |

Exemplo local (token fictício ≥32 bytes):

```bash
export FISCAL_ENV=development
export FISCAL_AUTH_MODE=dev_static
export FISCAL_AUTH_DEV_TOKEN='0123456789abcdef0123456789abcdef'
export FISCAL_AUTH_DEV_SCOPE_ID=scope-dev
export FISCAL_SCOPE_TIMEZONE=Africa/Luanda
export FISCAL_SERIES_MODE=static
export FISCAL_SERIES_EFFECTIVE_CODE=A
export FISCAL_DATABASE_DRIVER=sqlite
export FISCAL_DATABASE_URL=./tmp/fiscal.db
go run ./cmd/fiscal-migrate up
go run ./cmd/fiscal-api
```

Contrato: [api-guidelines.md](../03-api/api-guidelines.md). Sem ficheiros `.env` versionados.

Persistência: `SealInTx` (API interna) e testes VS-T01–VS-T07.

Contrato público `0.1.3-draft` + implementação HTTP `POST /v1/documents` (PR C2) + DEC-TIME-001 (`canonical_v2`, migration `0002`).

**Não** inclui worker AGT, JWS, ficheiros `.env` nem `GET /documents/{id}`. Cabo Verde runtime **não** está implementado.
