# Relatório S3B — staging credential_store + medição loopback

**Data (UTC):** 2026-07-22T01:38Z (execução); actualizado na branch de correcção pós-S3B
**SHA instalado no host:** `96dfb441445851bc5b84f408565cac43cc8c5cd5`
**Schema no host:** `3` (`dirty=false`)
**Host:** `sandbox.fiscalmod.bwb.pt` / `194.9.62.239`

Este relatório **não** contém passwords, tokens, DSN, hashes nem NIF completos.

## Resultado

S3B concluído operacionalmente. HTTPS público `/v1/documents` manteve-se **deny-all (403)** durante toda a execução. S3C **não** foi executado; candidato Nginx aberto **não** foi activado. O servidor permanece em deny-all.

## SHA / schema / serviços

| Item | Valor |
|---|---|
| Release activa | `96dfb441445851bc5b84f408565cac43cc8c5cd5` |
| Schema | `3` / `dirty=false` |
| `bwb-fiscal-api` | active |
| Nginx | active |
| PostgreSQL | active |
| Auth runtime | `FISCAL_ENV=homologation` + `FISCAL_AUTH_MODE=credential_store` |
| `admin.env` | `root:root` `0600`; **ausente** da unit systemd |

## Portas

| Superfície | Estado |
|---|---|
| Público 80/443 | abertas (TLS + health) |
| `127.0.0.1:8080` | API loopback |
| `127.0.0.1:5432` | PG loopback |
| `127.0.0.1:18080` | activado só durante medição; **removido** após S3B |
| Externo 5432/8080/18080 | fechadas/timeout |
| HTTPS `/v1/documents` | **403** deny-all |
| HTTPS `/v1/health` | 200 `status=ok` |

## Privilégios validados

Positivos (após `grants-schema3-runtime-admin.sql` no plano aprovado):

- `fiscal_runtime`: SELECT scopes/credentials; INSERT/SELECT documentos e satélites; INSERT audit; UPDATE idempotency/outbox/series.
- `fiscal_admin`: SELECT/INSERT scopes (**sem** UPDATE de tabela); SELECT/INSERT credentials; UPDATE colunar `status,grace_until,revoked_at`; INSERT audit.

Serialização de Issue/Rotate/Revoke em PostgreSQL: `pg_advisory_xact_lock(namespace, hashtext(scope_id))` + `SELECT` normal do scope (sem `SELECT … FOR UPDATE`). Colisões de chave advisory só aumentam serialização. SQLite mantém `BEGIN IMMEDIATE`.

O advisory lock vive no **binário/API** (`internal/persistence`), não no helper de deploy. O helper só gere activate/releases; não implementa serialização de credenciais.

Negativos:

- runtime sem DDL (`CREATE TABLE` denied)
- runtime sem INSERT em scopes
- admin sem SELECT em documents
- admin sem UPDATE em qualquer coluna de `fiscal.scopes`
- admin sem UPDATE em `token_hash` (e demais colunas de credentials não grantadas)

## IDs sintéticos (permitidos no relatório)

| Tipo | ID |
|---|---|
| Scope ops (NIF alinhado à fixture) | `scope-s3b-ops-002` |
| Scope carga/medição | `scope-s3b-load-002` |
| Scopes iniciais (não usados nos gates finais) | `scope-s3b-ops-001`, `scope-s3b-load-001` |
| Credencial A (revogada no gate) | `dbf81fdd1e45bfd53a0d13127016d275` |
| Documento A | `2e17dc35f271640ac12b9ffe8fe28f8e` |
| Documento B (replay) | `d2f7aef19d9afc9ff148ab8b4ba50c8e` |
| Credencial medição (revogada) | `fb3f4d985785a2e8f0e025643a18b320` |
| `created-by` | `s3b-operator` / `s3b-measure` |

Tokens, hashes, DSN e NIF: **omitidos**.

## Gate A→B (loopback `:8080`)

Sucesso: A cria documento → revogar A → 401 com token A → emitir B → documento novo + replay; `document_id` A≠B.

## Medição — classificação: burst measurement

Zona provisória versionada: `rate=10r/s`, `burst=20`, `limit_req_status 429`.

A execução #3 sob tetos S3B (≤60 pedidos, ≤5 concorrentes, ≤60 s) é uma **medição de burst curto**, não uma prova de capacidade sustentada de 10 pedidos/segundo durante 60 segundos. A duração efectiva do envio ficou muito abaixo do teto de 60 s; os pares okish/429 reflectem absorção do `burst` e rejeições imediatas, não um perfil estável de taxa.

| Execução | Classificação | total_sent | concurrency | okish | 429 | other | exit |
|---|---|---:|---:|---:|---:|---:|---:|
| #1 (sem `limit_req_status 429`) | burst (inválida p/ 429) | 60 | 5 | 35 | 0 | 25 | 1 (503 como other) |
| #2 | burst (inválida p/ 429) | 60 | 5 | 18 | 0 | 42 | 1 |
| #3 (com `limit_req_status 429`) | **burst measurement** | 60 | 5 | **33** | **27** | **0** | **0** |

**Não recomendação:** estes números **não** definem sozinhos o `rate`/`burst` final de S3C.

**S3C deve medir separadamente:**

1. **Carga sustentada** — janela longa com taxa alvo constante; registar duração real, latência (p50/p95) e distribuição de códigos.
2. **Burst** — rajada curta para caracterizar `burst`/`nodelay` e `Retry-After` se aplicável.

Candidato aberto deve manter `limit_req_status 429`. Burst auxiliar (40 req paralelos) antes do fix: `503` (default Nginx) e `422` (fixture `external_id` repetido com idempotency distinta) — `422` não é sinal de rate-limit.

## Backups pré-alteração

- Config: `/etc/bwb-modulo-fiscal/backups/s3b-pre-20260722T012704Z/`
- PG dump protegido: `/var/backups/bwb-fiscal/fiscal-s3b-pre-20260722T012704Z.dump` (`root:root` `0600`) — **mantido**
- Restore smoke: dump pré-S3B com **schema 2** é **esperado** (criado antes da migration `0003`). A cópia temporária legível por `postgres` e a base `fiscal_s3b_restore_smoke` foram **removidas** após a validação; só permanece o backup protegido.

## Ajustes pós-execução (esta branch)

1. **Crítico — activate symlink:** `mv -f` sem `-T` aninhava `current.new` na release antiga. Produção exige GNU `mv -T` (Ubuntu 22.04); verificação `current-sha` fechada no helper/updater; teste de regressão no harness.
2. **Alto — privilégio scopes:** remover UPDATE amplo; lock via `pg_advisory_xact_lock` + SELECT no binário/API; testes negativos de colunas de scopes; concorrência Issue/Rotate/Revoke estrita (PG + SQLite).
3. **Médio — medição:** `limit_req_status 429`; relatório classifica corrida como burst measurement (sem recomendar rate final).

## CI do artefacto (Draft PR #14)

- HEAD verificado antes deste reforço: `97f53a6f50d9c001b64ad917c7df0c044624c673`
- Workflow CI desse HEAD: **SUCCESS** (`go-checks` + GitGuardian)
- Este documento e os testes estritos/activate reforçados são incrementos posteriores no mesmo Draft PR (sem Ready/merge).

## Incidentes

| Severidade | Fase | Causa | Impacto | Resolução | Estado | Risco residual |
|---|---|---|---|---|---|---|
| Crítico | Deploy activate | GNU `mv` sem `-T` + symlink dir | SHA novo instalado/migrado mas `current` ficou em N-1; `promote=ok` falso-positivo | `mv -Tf` + assert + `current-sha` no updater; helper reinstalado no host | Corrigido (artefacto; host já activado) | Host ainda corre helper com patch operacional; PR sincroniza artefactos |
| Alto | API pós-activate | `dev_static` sem NIF no novo binário | Crash-loop até `credential_store` | `fiscal.env` → homologation + credential_store | Corrigido | — |
| Alto | Admin Issue (temporário) | `FOR UPDATE` exigia UPDATE em scopes | Issue falhava; mitigação incorrecta com UPDATE de tabela | Advisory lock no binário/API; grants voltam a SELECT/INSERT | Corrigido no artefacto (aplicar grants + **novo binário** no host só em manutenção futura autorizada) | Host ainda tem **temporariamente UPDATE em scopes** e o **binário anterior** (sem advisory lock). **Não** revogar esse grant antes do novo binário ser mergeado e instalado |
| Médio | Medição | `limit_req`→503 vs script→429 | Gate falhava com throttling real | `limit_req_status 429` | Corrigido | S3C: medir sustentado ≠ burst |
| Baixo | Backup restore smoke | Dump `0600` + nome `bwb_schema_migrations` | Validação inicial falhou | Cópia temporária + DROP DB; schema 2 esperado | Contornado | — |
| Baixo | Scopes iniciais | NIF ≠ fixture | `nif_mismatch` | Scopes `*-002` | Contornado | Runbook: NIF da fixture |

## Não feito (conforme mandato)

- S3C / abertura pública de `/v1/documents`
- Aplicar neste passo novas alterações ao servidor (grants/helper/binário no host ficam para janela autorizada)
- Revogar o UPDATE temporário em scopes no host **antes** do merge+install do binário com advisory lock
- Medição sustentada de 60 s a 10 r/s

## Ficheiros desta branch

- `internal/persistence/credentials.go` — advisory lock (API/binário)
- `internal/persistence/credentials_test.go` — concorrência Issue/Rotate/Revoke estrita
- `internal/persistence/grants_schema3_postgres_test.go` — admin sem UPDATE em scopes
- `deploy/postgres/grants-schema3-runtime-admin.sql` — SELECT/INSERT scopes apenas
- `scripts/deploy/remote-deploy-helper.sh` — `mv -T`, `current-sha`, GNU dependency (activate; sem advisory lock)
- `scripts/deploy/update-staging.sh` — verificação pós-activate via `current-sha`
- `tests/deploy/run-tests.sh` — regressão activate + grants/advisory
- `deploy/nginx/measure/…` e `candidates/…` — `limit_req_status 429`
- `CHANGELOG.md` — secção `0.2.10-draft`
- `docs/07-operations/s3b-staging-report.md` — este relatório
