# Changelog

## 0.2.9-draft — 2026-07-22

- Sandbox POS S3A (repo only): `fiscal-admin` + E2E/medição no build/manifesto Linux; `admin.env.allowlist` (DRIVER+URL); helper root→parser→`env -i`→`bwb-fiscal-admin`; backup/restore `admin.env`; grants PG explícitos fail-closed (sem CREATE ROLE; UPDATE colunar admin); Nginx público deny-all; candidato HTTPS aberto versionado (não activável); medição `127.0.0.1:18080`; gate A→B revoke + replay estável; runbook S3B/S3C. Sem deploy/SSH; `ExpectedVersion=3` inalterado.

## 0.2.8-draft — 2026-07-22

- Sandbox POS S2: `credential_store` auth com `ScopeBinding` (incl. Environment); `VerifyCredentialTokenHash` + ConstantTimeCompare em persistence; Issue/Rotate com `TokenSink` na mesma tx; `cmd/fiscal-admin`; 403 `FISCAL_SCOPE_MISMATCH`; OpenAPI `0.1.4-draft`. Sem migrations novas; Nginx/deny-all e staging intactos.
- Hardening S2: token Base64URL ASCII-only; `fiscal-admin` cria `--output-file` só após validação/BD; testes ErrInternal/500/timeout, audit/commit pós-Deliver (falha real diferida nos motores, sem hook de commit), Sync parcial; PostgreSQL de testes em BD temporária isolada (`dbtest`).

## 0.2.7-draft — 2026-07-21

- Sandbox POS S1: migration `0003` (PG/SQLite) — `scopes`, `api_credentials`, `audit_events`; `ExpectedVersion=3`; repositórios `issue`/`rotate`/`revoke` com audit co-transacional; token `bwb_sbox_` + SHA-256 sem pepper; sem HTTP/auth/CLI/Nginx/deploy.

## 0.2.6-draft — 2026-07-21

- D2 staging bootstrap report: host `sandbox.fiscalmod.bwb.pt` operacional (PG16, TLS, helper deploy, health ok). Sem segredos no relatório.

## 0.2.5-draft — 2026-07-21

- Staging deploy: migrate sob drop-priv (`bwb-fiscal-migrate`); runner removido da release; envs restorable logo após backup; falhas pós-activate (restart/health) com rotina N-1; health estrito a `"status":"ok"`; captura explícita do exit status do healthcheck sob `if`.
- PR D1 staging deploy foundation: allowlists, systemd (`fiscal.env` only), Nginx IPv4-only templates, closed remote helper + sudoers template (no `sudo bash`), transactional env backup/restore before activation, live health fixed to `http://127.0.0.1:8080/v1/health`, schema gate via `EXPECTED_SCHEMA_VERSION`, CI `git diff --check base...HEAD`. Sem acesso a servidor/DNS.

## 0.2.4-draft — 2026-07-21

- DEC-TIME-001 (PR #8): tempo fiscal vs técnico — `issued_at` com timezone IANA do scope (`Africa/Luanda`) e offset persistido; `created_at` UTC técnico (microssegundos, relógio injetável); `canonical_v2` activo com goldens imutáveis `canonical_v1`/`canonical_v2`; packages `fiscaltime`/`fiscaltz` (tzdata embutida, fail-closed); migration `0002` (PG/SQLite) aborta se houver `documents` ou `idempotency_records`; OpenAPI `0.1.3-draft` e exemplo Angola `+01:00`. Sem Cabo Verde runtime; sem recalculo de hashes; API sem migrate no arranque.
- Reforço SealInTx: `fiscaltime.ValidateNormalizedContext` no `prepareSealRequest` (timezone IANA + offset no instante + UTC micro); testes PG isolados para precondições da migration `0002`.

## 0.2.3-draft — 2026-07-21

- PR C2: `POST /v1/documents` (`createDocument`) sobre `SealInTx`; auth `dev_static` (só `FISCAL_ENV=development`, token ≥32 bytes, comparação constant-time); `SeriesResolver` estático; `SealResult.CreatedAt` persistido e estável no replay; Problem/códigos do contrato; fail-closed sem modo que aceite pedidos; testes HTTP dual-engine. Sem migrations no arranque da API, sem GET, sem AGT/JWS.

## 0.2.2-draft — 2026-07-21

- Contrato OpenAPI **`0.1.2-draft`**: `POST /documents` (`createDocument`) passa de **202 Accepted** para **201 Created** (criação + selagem local atómica).
- Removido do contrato o path **`GET /documents/{documentId}`** (estava declarado sem implementação); volta a entrar só com implementação correspondente.
- Resposta de sucesso deste fluxo: `status: sealed_locally`; **`fiscal_number` e `authority_request_id` ausentes** neste incremento (formato oficial de numeração e ID da autoridade ainda não confirmados / não atribuídos).
- Adicionado `submission_id` opcional (correlação **interna** do módulo; não é ID AGT).
- Schema `Problem` e respostas documentadas para 401, 403, 409, 413, 415, 422 e 500; `bearerAuth` descrito como autenticação POS/módulo (não AGT).
- `CreateDocumentResponse` com `status` const `sealed_locally` e campos obrigatórios incluindo `submission_id`/`created_at`; `SellerParty` com `tax_id`/`name` obrigatórios non-empty (pattern com ≥1 não-whitespace, alinhado à persistência; sem formato NIF); idem para `external_id` e campos de linha `line_id`/`description`/`tax_code`; URNs `urn:bwb:fiscal:error:…` (sem URLs fictícias); `info.license` MIT alinhado a `LICENSE`.
- Redocly: exceção **apenas** `GET /health` para `operation-4xx-response` (sem parâmetros/body; sem 4xx fictício); regra mantida globalmente.
- Docs POS/guidelines/lifecycle/slice/local-dev + exemplo [docs/03-api/examples/create-document.http](docs/03-api/examples/create-document.http). Sem implementação HTTP (PR C2).

## 0.2.1-draft — 2026-07-21

- SealInTx co-transacional (PR B): idempotência, série (PG `FOR UPDATE` / SQLite `BEGIN IMMEDIATE`), documento, ledger `sealed_locally`, outbox `authority_submission`; testes VS-T01–VS-T07 nos dois motores. Sem HTTP/worker/AGT.

## 0.2.0-draft — 2026-07-21

- Fundação de persistência (PR A): drivers pgx + modernc/sqlite; migrations forward-only embutidas; schema `fiscal` + `public.bwb_schema_migrations`; tipos money/quantity int64; canonical_v1; `cmd/fiscal-migrate` (`up`/`version`); CI com Postgres, imutabilidade de migrations, govulncheck e go-licenses. Sem SealInTx nem endpoints de documentos.

## 0.1.9-draft — 2026-07-21

- Default `FISCAL_HTTP_ADDR` em `127.0.0.1:8080` (cloud exige bind explícito); CI só em `push`/`pull_request` para `main` com `go vet` + `go test -race`; rejeição de overflow em timeouts em milissegundos.

## 0.1.8-draft — 2026-07-21

- Hardening do scaffold: `go.mod` 1.25.0 e CI/deploy em Go 1.26.x ([release policy](https://go.dev/doc/devel/release)); `ReadHeaderTimeout` configurável; `MaxHeaderBytes` 64 KiB; `Server.Serve(net.Listener)`; `TestLoadDefaults` hermético.

## 0.1.7-draft — 2026-07-21

- Scaffold Fase 1: módulo Go `github.com/storesace-cv/bwb-modulo-fiscal`, binário `cmd/fiscal-api` com `GET /v1/health` (stdlib), config por ambiente, timeouts HTTP, graceful shutdown, logs estruturados; CI mínima; guia local em `docs/06-delivery/local-dev.md`. Sem emissão fiscal, BD, Docker ou frameworks.

## 0.1.6-draft — 2026-07-21

- Tarefa zero OpenAPI (`0.1.1-draft`): `Money`/`DecimalQuantity` canónicos, `sealed_locally`, `authority_outcome_unknown`; `contingency_pending` reservado; diretrizes e máquina de estados harmonizadas; DEC-API-001/003 aplicadas no contrato.

## 0.1.5-draft — 2026-07-21

- Adicionados princípios obrigatórios de engenharia sénior (`ENGINEERING_PRINCIPLES.md`), ligação em `AGENTS.md`/`README.md` e regra Cursor `senior-engineering.mdc`.

## 0.1.4-draft — 2026-07-21

- Arquitetura do backoffice formalizada; DEC-REG-KEY-CUSTODY e DEC-SEC-EDGE-KEYS abertas (bloqueantes); GAP-013 (custódia externa da chave do contribuinte).

## 0.1.3-draft — 2026-07-21

- DEC-STACK-001 decidida: Go + PostgreSQL na cloud + SQLite WAL no Edge (condições XSD oficial, assinatura fiscal AGT e numeração preservadas).

## 0.1.2-draft — 2026-07-20

- Harmonização final do plano Fase 0: DEC-STACK-001 recomendada, `sealed_locally` único, OpenAPI tarefa zero, RSA efémero.
- Correção do plano técnico da Fase 0: at-least-once (sem exactly-once), JWS RS256 real com chaves de teste, estados neutros até DEC-API-004.
- Edge MVP com SQLite WAL (escritor único); PostgreSQL apenas na cloud.
- DEC-API-001, DEC-API-003 e DEC-DEL-001 decididas; DEC-API-004 aberta; prioridades de decisão reordenadas.
- Fase 0 interna reduzida a 2–4 semanas; vertical slice sem portal, webhooks nem frontend POS.
- Outbox distinta de logs operacionais; numeração sem promessa genérica de «zero buracos».

## 0.1.1-draft — 2026-07-20

- Plano executável da Fase 0 em `docs/06-delivery/phase-0-execution-plan.md`.
- Decisões técnicas e regulatórias em aberto em `docs/06-delivery/open-decisions.md`.
- Inventário de lacunas regulatórias em `docs/01-compliance/regulatory-gaps.md`.
- Proposta de stack (duas alternativas, sem implementação) em `docs/06-delivery/technical-stack-proposal.md`.
- Especificação do primeiro vertical slice (demo ponta a ponta) em `docs/06-delivery/first-vertical-slice.md`.
- Premissa `ASM-REG-001` mantida; OpenAPI e código de produção não alterados.
- Contradições documentais inventariadas (estados API, Money/quantity, proposta vs Decreto 74/19).

## 0.1.0-draft — 2026-07-20

- Documentação inicial do produto Angola-first.
- Registo da premissa `ASM-REG-001`.
- Arquitetura cloud/Edge e pacotes por país.
- Catálogo inicial de conformidade.
- Esqueleto OpenAPI.
- Baseline de segurança, testes, operações e roadmap.
- Portal do Contribuinte de Angola registado como fonte oficial prioritária.
- Guia Rápido de Emissão de Facturas e Portal institucional da AGT adicionados ao registo de fontes.
- Documentação técnica FE, Portal do Parceiro, Decreto 74/19 e área restrita de produtores registados no inventário de fontes.
- Criado plano de acesso, preservação e versionamento de artefactos oficiais.
- Definida `local/` como pasta exclusiva de consulta, integralmente excluída do GitHub.
