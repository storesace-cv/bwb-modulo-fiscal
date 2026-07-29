# ROADMAP — BWB Módulo Fiscal

**Fonte canónica de estado e progresso do projecto.**

**Estado revisto em:** 2026-07-29

**Inicialmente consolidado no PR:** [#28](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/28)

Glossário anti-colisão: **OPS-B1/B2/B3** = gate `pg_dump` pré-deploy; **SRC-A/B0/B1/B2/C/D** = trilho de fontes fiscais e SAF-T AO.

---

## 1. Visão executiva

O projecto já tem uma fundação técnica sólida e um sandbox funcional, mas ainda não é um módulo fiscal certificado nem está pronto para utilização comercial. Temos a plataforma de integração, persistência, segurança e operação; falta implementar e validar o núcleo fiscal oficial da AGT.

### Distinções obrigatórias

| Não confundir | Com |
|---|---|
| `sealed_locally` / SealInTx | emissão fiscal certificada pela AGT |
| Sandbox funcional (`sandbox.fiscalmod.bwb.pt`) | produção comercial |
| API tecnicamente operacional | conformidade fiscal / homologação AGT |
| Documentação arquivada ou catalogada | requisito `AO-*` validado |
| Implementação concluída no repositório | homologação ou certificação AGT |
| `FISCAL_ENV=homologation` | ambiente oficial de homologação da AGT, módulo homologado, ou certificação |

`FISCAL_ENV=homologation` é uma **designação técnica do ambiente sandbox BWB**. Não significa acesso ao ambiente oficial de homologação da AGT, não significa que o módulo tenha sido homologado e não significa certificação. A homologação oficial AGT continua **PENDENTE / NÃO INICIADA**.

### Fontes e derivados

Para cada derivado OCR de um diploma, o PDF oficial original permanece a representação autoritativa. OCR e Markdown são auxiliares de pesquisa. Só conteúdo com estado OCR `reviewed` e confrontado visualmente com o original pode sustentar requisitos `AO-*` confirmados. Aplica-se sempre a prioridade documental definida em [`AGENTS.md`](AGENTS.md) e [`compliance/POLICY.md`](compliance/POLICY.md). XSD, documentação técnica oficial e legislação mantêm os respectivos papéis. Não se inventam regras fiscais.

A pasta `local/` não é dependência do repositório: não copiar `local/` para build, testes, runtime ou CI.

---

## 2. Estado atual

| Campo | Valor |
|---|---|
| País activo | Angola |
| País futuro | Cabo Verde (ADIADO) |
| OpenAPI | POS `0.1.6-draft` · Admin `0.1.28-draft` ([specs/admin/openapi.yaml](specs/admin/openapi.yaml)) |
| Schema | `ExpectedVersion=13` |
| Sandbox | `https://sandbox.fiscalmod.bwb.pt` — S3C2 **CONFIRMED**, kit POS **9/9** |
| Auth sandbox | `credential_store` + `FISCAL_ENV=homologation` (técnico BWB ≠ AGT) |

### Onde estamos

| Área | Estado |
|---|---|
| Fundação técnica | CONCLUÍDO |
| Persistência transacional (slice) | CONCLUÍDO |
| API POS | Funcional (sandbox) |
| Auth / credenciais | Funcional |
| Sandbox BWB | Operacional |
| Deploy / rollback | Operacional |
| Segurança base | Implementada |
| Kit POS | Validado 9/9 |
| Backup pré-deploy (OPS-B2) | Gate validado; INC-B2-001 aberto |
| Catálogo fiscal (SRC-A) | CONCLUÍDO (metadados) |
| Armazenamento sincronizado (SRC-B1) | CONCLUÍDO (repo privado) |
| XSD SAF-T AO (SRC-B2) | CONCLUÍDO (Git público; `pending_validation`) |
| Auditoria B0 SAF-T | CONCLUÍDO |
| OCR das fontes | CONCLUÍDO (74/19 + Rect. v2 + 683/25 v2 + DP 71/25 `reviewed`; rejected ≠ KB; OCR ≠ AO-* confirmados) |
| Requisitos AO-* confirmados | EM_CURSO (3× `confirmed_normative`: AO-DOC-002, AO-SEQ-001, AO-OFF-001; resto provisório; ≠ AGT) |
| Fundação transacional fiscal | CONCLUÍDA (slice) |
| Motor regulamentar Angola | NÃO IMPLEMENTADO |
| Integração oficial AGT | NÃO IMPLEMENTADA |
| SAF-T AO | Tipagem 5 grupos L3 + export/map livro; produção/AGT NÃO |
| Assinatura / JWS AGT | Adaptador efémero de slice (outbox→simulador); **não** certificado / ≠ FE-RNG oficial |
| Faturação electrónica AGT | NÃO INTEGRADA |
| Homologação BWB (`FISCAL_ENV=homologation`) | Ambiente técnico sandbox — **não** é homologação AGT |
| Homologação oficial AGT | NÃO INICIADA |
| Certificação AGT | NÃO INICIADA |
| Produção comercial | NÃO INICIADA |
| Backoffice | Separado: plano A funcional vs plano B segredos (`DEC-BO-001`); UI SSR mínima |
| Edge | Planeado |
| Cabo Verde | ADIADO |

**Conclusão:** estamos no fim da fundação técnica e operacional e no início do produto fiscal propriamente dito.

---

## 3. O que já foi construído

### 3.1 Planeamento e conformidade

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-FOUND-001 | Angola como primeira fase; Cabo Verde como fase futura | CONCLUÍDO | [README.md](README.md) · [docs/00-product/scope.md](docs/00-product/scope.md) | — | Premissa de produto documentada |
| [x] | RM-FOUND-002 | Documentação fiscal catalogada (metadados SRC-A) | CONCLUÍDO | [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) · [CHANGELOG.md](CHANGELOG.md) | — | Catálogo versionado com 20 fontes |
| [x] | RM-FOUND-003 | Governação e proveniência de fontes (POLICY) | CONCLUÍDO | [compliance/POLICY.md](compliance/POLICY.md) · [compliance/README.md](compliance/README.md) | — | Política e CI de catálogo activos |
| [x] | RM-FOUND-004 | Auditoria B0 SAF-T AO cross-project | CONCLUÍDO | [docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md](docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md) | — | Experiência cross-project registada como não normativa |
| [ ] | RM-FOUND-005 | Confirmação formal ASM-REG-001 / DEC-REG-001 | BLOQUEADO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | Resposta AGT ou ata de risco aceite | Premissa validada juridicamente |
| [x] | RM-FOUND-006 | Lacunas regulatórias registadas (GAP-*) | CONCLUÍDO | [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | — | GAPs inventariados sem inventar regras |

### 3.2 Arquitectura

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-ARCH-001 | Stack Go + PostgreSQL cloud + SQLite WAL Edge | CONCLUÍDO | [docs/06-delivery/technical-stack-proposal.md](docs/06-delivery/technical-stack-proposal.md) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) | — | DEC-STACK-001 decidida |
| [x] | RM-ARCH-002 | API HTTP + Nginx + systemd | CONCLUÍDO | [docs/07-operations/staging-runbook.md](docs/07-operations/staging-runbook.md) · [docs/07-operations/d2-staging-bootstrap-report.md](docs/07-operations/d2-staging-bootstrap-report.md) | — | Sandbox operacional |
| [x] | RM-ARCH-003 | Migrations dual-engine PG/SQLite | CONCLUÍDO | [internal/platform/dbmigrate/migrate.go](internal/platform/dbmigrate/migrate.go) | — | ExpectedVersion=6 |
| [x] | RM-ARCH-004 | Scopes multi-tenant e separação de papéis | CONCLUÍDO | [docs/02-architecture/system-architecture.md](docs/02-architecture/system-architecture.md) · [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | — | Plataforma / integradores / contribuintes / estabelecimentos / credenciais POS / chaves AGT / HML-PRD |
| [x] | RM-ARCH-006 | Separação backoffice funcional vs zona segredos (`DEC-BO-001`) | CONCLUÍDO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) · [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md) · [CHANGELOG.md](CHANGELOG.md) | — | Plano A ops; plano B owner write-only; metadados sanitizados |
| [x] | RM-ARCH-005 | Backoffice operacional (UI) | CONCLUÍDO | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) · [internal/adminui/ui.go](internal/adminui/ui.go) · [CHANGELOG.md](CHANGELOG.md) | M7 + RM-BO-010 + RM-SECADM-002 + RM-UI-001…004 | SSR monólito `/admin/ui/`; slices UI-001…004 |

### 3.3 Motor transacional inicial

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TX-001 | Persistência PG/SQLite, documentos/linhas imutáveis, ledger, outbox | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) · [docs/06-delivery/first-vertical-slice.md](docs/06-delivery/first-vertical-slice.md) | — | Slice transacional persistido |
| [x] | RM-TX-002 | Numeração por série, idempotência, conflito external_id | CONCLUÍDO | [specs/openapi/openapi.yaml](specs/openapi/openapi.yaml) | — | Autoridade de numeração no módulo |
| [x] | RM-TX-003 | Hash canónico versionado, rollback, concorrência | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) | — | Testes dual-engine |
| [x] | RM-TX-004 | Timezone IANA, Africa/Luanda, hora fiscal vs técnica, microssegundos | CONCLUÍDO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) | — | DEC-TIME-001 |
| [x] | RM-TX-005 | Estados neutros + SealInTx (selagem local) | CONCLUÍDO | [specs/openapi/openapi.yaml](specs/openapi/openapi.yaml) | — | `sealed_locally` ≠ emissão certificada AGT |
| [x] | RM-TX-006 | Outbox worker + simulador AGT (VS-T08/10/11) | CONCLUÍDO | [PR #52](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/52) · [internal/persistence/outbox.go](internal/persistence/outbox.go) · [internal/authority/simulator/simulator.go](internal/authority/simulator/simulator.go) · [docs/06-delivery/authority-schema-deferred.md](docs/06-delivery/authority-schema-deferred.md) · [CHANGELOG.md](CHANGELOG.md) | — | Simulador ≠ HML AGT; sem credenciais/deploy |
| [x] | RM-TX-007 | JWS RS256 efémero no outbox→simulador | CONCLUÍDO | [internal/fiscaljws/fiscaljws.go](internal/fiscaljws/fiscaljws.go) · [internal/persistence/outbox.go](internal/persistence/outbox.go) · [CHANGELOG.md](CHANGELOG.md) | — | Envelope técnico; não certificado; ≠ FE-RNG / FE certificado oficial |
| [x] | RM-TX-008 | At-least-once, deduplicação e reconciliação (slice) | CONCLUÍDO | [docs/06-delivery/outbox-at-least-once.md](docs/06-delivery/outbox-at-least-once.md) · [internal/persistence/outbox_test.go](internal/persistence/outbox_test.go) · [CHANGELOG.md](CHANGELOG.md) | — | VS-T12; simulador ≠ AGT; sem exactly-once |
| [x] | RM-TX-009 | Config `FISCAL_AUTHORITY` (simulator vs AGT reservado) | CONCLUÍDO | [internal/platform/config/config.go](internal/platform/config/config.go) · [docs/06-delivery/local-dev.md](docs/06-delivery/local-dev.md) · [CHANGELOG.md](CHANGELOG.md) | — | Só `simulator`; `agt-hml`/`agt-prd` fail-closed; sem credenciais/deploy |
| [x] | RM-TX-010 | VS-T09: `authority_processing` sob simulador lento | CONCLUÍDO | [internal/persistence/outbox.go](internal/persistence/outbox.go) · [internal/persistence/outbox_test.go](internal/persistence/outbox_test.go) · [CHANGELOG.md](CHANGELOG.md) | — | Processing antes do Submit; unavailable reverte a `sealed_locally` |

### 3.4 API POS

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-API-001 | POST /v1/documents + OpenAPI 0.1.6-draft | CONCLUÍDO | [specs/openapi/openapi.yaml](specs/openapi/openapi.yaml) · [docs/03-api/api-guidelines.md](docs/03-api/api-guidelines.md) | — | Contrato público publicado |
| [x] | RM-API-002 | Bearer, Idempotency-Key, JSON estrito, limites, Problem | CONCLUÍDO | [docs/03-api/integration-contract.md](docs/03-api/integration-contract.md) | — | Erros e códigos HTTP documentados |
| [x] | RM-API-003 | Replay, isolamento por scope, NIF/TZ/série do binding | CONCLUÍDO | [docs/03-api/quickstart.md](docs/03-api/quickstart.md) | — | POS não escolhe livremente NIF/série |

### 3.5 Credenciais

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-CRED-001 | Scopes, issue/rotate/revoke, CSPRNG, SHA-256 | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) | — | Tokens só hash em BD |
| [x] | RM-CRED-002 | active/grace/revoked, expiração, auditoria append-only | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) | — | Ciclo de vida de credencial |
| [x] | RM-CRED-003 | fiscal-admin, TokenSink write-only, credential_store | CONCLUÍDO | [docs/06-delivery/local-dev.md](docs/06-delivery/local-dev.md) · [docs/07-operations/s3b-staging-report.md](docs/07-operations/s3b-staging-report.md) | — | Auth sandbox por credential_store |

### 3.6 Sandbox e segurança

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-SBOX-001 | Host sandbox Ubuntu 22.04 + PG16 + TLS/HSTS/Nginx | CONCLUÍDO | [docs/07-operations/d2-staging-bootstrap-report.md](docs/07-operations/d2-staging-bootstrap-report.md) | — | sandbox.fiscalmod.bwb.pt operacional |
| [x] | RM-SBOX-002 | Loopback API/PG, portas externas fechadas, SSH por chave, Fail2ban, firewall | CONCLUÍDO | [docs/07-operations/ssh-ufw-limit-remediation-report.md](docs/07-operations/ssh-ufw-limit-remediation-report.md) | — | Superfície pública reduzida |
| [x] | RM-SBOX-003 | Releases imutáveis, checksum/revision, rollback, multiplexing SSH | CONCLUÍDO | [docs/07-operations/staging-runbook.md](docs/07-operations/staging-runbook.md) | — | Deploy controlado |
| [x] | RM-SBOX-004 | Incidente UFW LIMIT resolvido + IdentitiesOnly | CONCLUÍDO | [docs/07-operations/ssh-ufw-limit-remediation-report.md](docs/07-operations/ssh-ufw-limit-remediation-report.md) | — | SSH estável |

### 3.7 Abertura autenticada

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-SBOX-010 | Health 200; sem token 401; inválido 401; NIF mismatch 403 | CONCLUÍDO | [docs/07-operations/s3c2-sandbox-promotion-report.md](docs/07-operations/s3c2-sandbox-promotion-report.md) · [docs/07-operations/s4-pos-kit-ops-validation-report.md](docs/07-operations/s4-pos-kit-ops-validation-report.md) | — | Gates de auth públicos |
| [x] | RM-SBOX-011 | Create/replay 201; rate limit 429; zero 5xx nos gates finais | CONCLUÍDO | [docs/07-operations/s4-pos-kit-ops-validation-report.md](docs/07-operations/s4-pos-kit-ops-validation-report.md) | — | Abertura confirmada |
| [x] | RM-SBOX-012 | Rollback temporizado, boot recovery, confirm, revogação efémera | CONCLUÍDO | [docs/07-operations/s3c2-sandbox-promotion-report.md](docs/07-operations/s3c2-sandbox-promotion-report.md) | — | S3C2 CONFIRMED |

### 3.8 Kit POS

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-KIT-001 | Onboarding, fixtures sintéticas, 9/9 casos, protecção token | CONCLUÍDO | [docs/03-api/quickstart.md](docs/03-api/quickstart.md) · [docs/07-operations/s4-pos-kit-ops-validation-report.md](docs/07-operations/s4-pos-kit-ops-validation-report.md) | — | Kit validado no sandbox |
| [x] | RM-KIT-002 | Execução macOS/Ubuntu + cleanup | CONCLUÍDO | [scripts/integration/pos-sandbox-kit.sh](scripts/integration/pos-sandbox-kit.sh) | — | Harness portável |

### 3.9 Deploy, operações e backups

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-OPS-001 | Deploy D1/D2, helpers fechados, sudoers | CONCLUÍDO | [docs/07-operations/staging-runbook.md](docs/07-operations/staging-runbook.md) · [docs/07-operations/d2-staging-bootstrap-report.md](docs/07-operations/d2-staging-bootstrap-report.md) | — | Caminho de deploy operacional |
| [x] | RM-OPS-002 | Gate pré-deploy: lock, dump, restore temp, schema, bloqueio | CONCLUÍDO | [docs/07-operations/b2-predeploy-pg-dump-gate-report.md](docs/07-operations/b2-predeploy-pg-dump-gate-report.md) | — | OPS-B1 merged + OPS-B2 APROVADO |
| [x] | RM-OPS-003 | INC-S4-003 resolvido (docs OPS-B3) | CONCLUÍDO | [docs/07-operations/b2-predeploy-pg-dump-gate-report.md](docs/07-operations/b2-predeploy-pg-dump-gate-report.md) · [docs/07-operations/s4-pos-kit-ops-validation-report.md](docs/07-operations/s4-pos-kit-ops-validation-report.md) | — | Incidente fechado com evidência |
| [ ] | RM-OPS-004 | INC-B2-001 prova lock poisoned | BLOQUEADO | [docs/07-operations/b2-predeploy-pg-dump-gate-report.md](docs/07-operations/b2-predeploy-pg-dump-gate-report.md) | Ensaio seguro de remediação humana no sandbox | Prova poisoned/stale/corrupt documentada |
| [x] | RM-OPS-005 | CI Node.js 24 (checkout@v7 + setup-go@v7) | CONCLUÍDO | [.github/workflows/ci.yml](.github/workflows/ci.yml) · [CHANGELOG.md](CHANGELOG.md) | — | Anotações Node 20 removidas |
| [x] | RM-OPS-006 | Promote sandbox `main` → ExpectedVersion=12 + grants schema12 | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/120 · [docs/07-operations/rm-ops-006-sandbox-schema12-promotion-report.md](docs/07-operations/rm-ops-006-sandbox-schema12-promotion-report.md) · [deploy/postgres/grants-schema12-runtime-admin.sql](deploy/postgres/grants-schema12-runtime-admin.sql) · 2026-07-27 | RM-OPS-002 + main@schema12 | current-sha=health.revision; dirty=false; API 401 sem token; admin fail_closed |
| [x] | RM-OPS-007 | Nginx open sandbox: proxy `/admin/v1/` + `/admin/ui` (fail-closed app; deny-all sem admin) | CONCLUÍDO | [docs/07-operations/rm-ops-006-sandbox-schema12-promotion-report.md](docs/07-operations/rm-ops-006-sandbox-schema12-promotion-report.md) · [deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf](deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf) · 2026-07-27 | RM-OPS-006 + S3C2 confirmed | HTTPS `/admin/ui/login` HTML; ready público; documents 401; timer confirmado |
| [x] | RM-OPS-008 | Fundação SMTP segura (implicit TLS 465; smtp.env 0600; teste owner-only; fake TLS nos testes) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/122 · [docs/07-operations/rm-ops-008-smtp-sandbox-report.md](docs/07-operations/rm-ops-008-smtp-sandbox-report.md) · INC-OPS-008-2 RESOLVIDO (465 `sent`) · 2026-07-27 | RM-OPS-007 + RM-BO-004 | Port 465; sem passwords no email; DeliveryStatus sanitizado; helper smtp-send-test |
| [x] | RM-OPS-009 | Digest SMTP de alertas ops sanitizados (owner-only; códigos allowlist; fake TLS) | CONCLUÍDO | [internal/notify/smtp/smtp.go](internal/notify/smtp/smtp.go) · [internal/adminapi/notification_handlers.go](internal/adminapi/notification_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [docs/07-operations/smtp-notifications.md](docs/07-operations/smtp-notifications.md) · 2026-07-27 | RM-OPS-008 + RM-BO-017 | POST alerts-digest; alert_count/codes; sem passwords/NIF; ≠ AGT |
| [x] | RM-OPS-010 | Landing HTML amigável em `GET /` no sandbox open (≠ sinal de disponibilidade) | CONCLUÍDO | [internal/landing/landing.go](internal/landing/landing.go) · [deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf](deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf) · [docs/07-operations/sandbox-root-landing.md](docs/07-operations/sandbox-root-landing.md) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-28 | RM-OPS-007 | 404 antigo na raiz era UX API-only; health continua em `/v1/health`; deny-all sem landing |

---

## 4. Caminho crítico para Angola

```mermaid
flowchart LR
  fontes[FontesOficiaisERevisao] --> reqs[RequisitosAO]
  reqs --> fluxo[PrimeiroFluxoFiscal]
  fluxo --> saft[SAFT_AO]
  saft --> fe[FE_HomologacaoAGT]
  fe --> evid[Evidencias]
  evid --> cert[CertificacaoAGT]
  cert --> piloto[Piloto]
  piloto --> prod[Producao]
```

Fluxo técnico alvo:

```text
POS → autenticação e binding → validação fiscal oficial → série/numeração
  → assinatura aplicável → persistência imutável → outbox
  → submissão AGT → consulta/reconciliação → SAF-T AO → auditoria/evidência
```

---

## 5. Roadmap detalhado por área

Índice das secções 6–14. Prefixo `RM-SRC-*` / `RM-ENG-*` / `RM-FE-*` etc. Evitar confundir OPS-B* com SRC-*.

---

## 6. Fontes fiscais e SAF-T AO

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-SRC-001 | Catálogo SRC-A (metadados) | CONCLUÍDO | [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) | — | 20 fontes catalogadas |
| [x] | RM-SRC-002 | Auditoria B0 (referência técnica não normativa) | CONCLUÍDO | [docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md](docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md) | — | Sem cópia de código/fixtures |
| [x] | RM-SRC-003 | Repo privado `storesace-cv/bwb-fiscal-sources-ao` + originais sincronizados | CONCLUÍDO | [https://github.com/storesace-cv/bwb-fiscal-sources-ao](https://github.com/storesace-cv/bwb-fiscal-sources-ao) · commit [`c93db4f89352eab884a4862943faa2ed874217fc`](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/c93db4f89352eab884a4862943faa2ed874217fc) · [catalog/sources.json](https://github.com/storesace-cv/bwb-fiscal-sources-ao/blob/c93db4f89352eab884a4862943faa2ed874217fc/catalog/sources.json) · [manifests/SHA256SUMS.txt](https://github.com/storesace-cv/bwb-fiscal-sources-ao/blob/c93db4f89352eab884a4862943faa2ed874217fc/manifests/SHA256SUMS.txt) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) | — | Repo privado criado; originais B1 sincronizados; integridade verificada |
| [x] | RM-SRC-004 | OCR + MD/TXT dos 3 PDFs image-only (74/19, Rect. 10/19, 683/25) | CONCLUÍDO | [bwb-fiscal-sources-ao@c8a4e6e](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/c8a4e6e8ec2772ff50ad1c8762842b983edbbbfd) · [PR #3](https://github.com/storesace-cv/bwb-fiscal-sources-ao/pull/3) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | — | 74/19 + Rect. v2 (`b3db14e2…`) + 683/25 v2 `reviewed`; DP 71/25 também `reviewed` (11902–11920); rejected ≠ KB |
| [x] | RM-SRC-005 | SRC-B2 XSD público/inventário no Git | CONCLUÍDO | [compliance/saft-ao/schemas/SAFTAO1.01_01.xsd](compliance/saft-ao/schemas/SAFTAO1.01_01.xsd) · [compliance/saft-ao/schemas/LICENSE](compliance/saft-ao/schemas/LICENSE) · [compliance/saft-ao/schemas/NOTICE.md](compliance/saft-ao/schemas/NOTICE.md) · [compliance/saft-ao/schemas/SHA256SUMS.txt](compliance/saft-ao/schemas/SHA256SUMS.txt) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) · [PR #31](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/31) | — | XSD+LICENSE+NOTICE versionados (`pending_validation`) |
| [x] | RM-VENDOR-001 | Catálogo vendor NET-BO + PT-CERT (≠ AGT); originais privados; audit NET-BO read-only | CONCLUÍDO | [compliance/catalog/vendor-integrations.yaml](compliance/catalog/vendor-integrations.yaml) · [docs/01-compliance/vendor-integrations/README.md](docs/01-compliance/vendor-integrations/README.md) · [docs/01-compliance/audits/AUD-VENDOR-NETBO-REUSE-MY-BWB-APP.md](docs/01-compliance/audits/AUD-VENDOR-NETBO-REUSE-MY-BWB-APP.md) · [bwb-fiscal-sources-ao@a889ef6](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/a889ef623c367e96cb7246ab42a274e54cbb2dc3) · 2026-07-28 | — | Sem conector live; sem AO-*; `DEC-VENDOR-001` |
| [x] | RM-SAFT-001 | Fundação estrutural gerador SAF-T AO (tipos/mapeamento a partir do XSD; ≠ conformidade) | CONCLUÍDO | [internal/saftao/schema.go](internal/saftao/schema.go) · [compliance/saft-ao/schemas/embed.go](compliance/saft-ao/schemas/embed.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SRC-005 | `source_id` AO-SAFT-XSD-1.01_01 (`pending_validation`); skeleton AuditFile; SHA-256; ≠ AO-* / certificação |
| [x] | RM-SAFT-002 | Skeleton Header + SourceDocuments (SalesInvoices/…) a partir do XSD | CONCLUÍDO | [internal/saftao/header_inventory.go](internal/saftao/header_inventory.go) · [internal/saftao/structure.go](internal/saftao/structure.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-001 | Inventário Header; tabelas SourceDocuments; ≠ export válido / AO-* / AGT |
| [x] | RM-SAFT-003 | Tipagem SalesInvoices/Invoice/Line (+ Money/datas) a partir do XSD | CONCLUÍDO | [PR #78](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/78) · [internal/saftao/sales.go](internal/saftao/sales.go) · [internal/saftao/money.go](internal/saftao/money.go) · [internal/saftao/fixture.go](internal/saftao/fixture.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-002 | 5 grupos L3; XML+XSD fixtures sintéticas; Hash/InvoiceType semantics pending; ≠ AO-* / AGT |
| [x] | RM-SAFT-004 | Export/validação SAF-T incremental (período + hash artefacto) | CONCLUÍDO | [PR #79](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/79) · [internal/saftao/export.go](internal/saftao/export.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-003 | Filtro período; allowlist InvoiceType; SHA-256 XML; XSD opcional; ≠ ledger/AGT/AO-* |
| [x] | RM-SAFT-005 | Tipagem MovementOfGoods/StockMovement/Line a partir do XSD | CONCLUÍDO | [PR #80](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/80) · [internal/saftao/movement.go](internal/saftao/movement.go) · [internal/saftao/fixture.go](internal/saftao/fixture.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-003 | Enums MovementType/Status; Customer XOR Supplier; fixture XSD; ≠ AO-* / AGT |
| [x] | RM-SAFT-006 | Tipagem WorkingDocuments/WorkDocument/Line a partir do XSD | CONCLUÍDO | [PR #81](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/81) · [internal/saftao/working.go](internal/saftao/working.go) · [internal/saftao/fixture.go](internal/saftao/fixture.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-003 | Enums WorkType/Status; fixture XSD; ≠ AO-* / AGT |
| [x] | RM-SAFT-007 | Tipagem Payments/Payment/Line a partir do XSD | CONCLUÍDO | [PR #82](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/82) · [internal/saftao/payments.go](internal/saftao/payments.go) · [internal/saftao/fixture.go](internal/saftao/fixture.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-003 | Enums PaymentType/Status; SourceDocumentID; caps fail-closed; ≠ AO-* / AGT |
| [x] | RM-SAFT-008 | Tipagem PurchaseInvoices/Invoice (+ Supplier) a partir do XSD | CONCLUÍDO | [PR #83](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/83) · [internal/saftao/purchase.go](internal/saftao/purchase.go) · [internal/saftao/fixture.go](internal/saftao/fixture.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-003 | Sem Line (XSD); sem TotalDebit/Credit no contentor; ≠ AO-* / AGT |
| [x] | RM-SAFT-009 | Mapeamento livro→BuildIncrementalExport (scope/período/omissões) | CONCLUÍDO | [PR #84](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/84) · [internal/saftao/mapledger.go](internal/saftao/mapledger.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-004 | SalesLedgerRecord; omissões fail-closed; sem inventar Hash/imposto; ≠ AGT/AO-* |
| [x] | RM-SAFT-010 | Loader persistência→SalesLedgerRecord (scope/período) | CONCLUÍDO | [PR #85](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/85) · [internal/persistence/saft_ledger.go](internal/persistence/saft_ledger.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-009 | ListSealedSalesForSAFT; sem inventar Hash/ProductCode; ≠ AGT/AO-* |
| [x] | RM-SAFT-011 | Tipagem TaxTable/TaxTableEntry a partir do XSD | CONCLUÍDO | [PR #86](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/86) · [internal/saftao/tax_table.go](internal/saftao/tax_table.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-003 | TaxType IVA/IS/NS; XOR TaxPercentage/Amount; caps fail-closed; ≠ AO-* / AGT |
| [x] | RM-SAFT-012 | TaxTable opcional em BuildIncrementalExport | CONCLUÍDO | [PR #87](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/87) · [internal/saftao/export.go](internal/saftao/export.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-011 + RM-SAFT-004 | Validação + refs TaxType/TaxCode; sem inventar taxas; ≠ AO-* / AGT |
| [x] | RM-SAFT-013 | Payments em BuildIncrementalExport (período/allowlist) | CONCLUÍDO | [PR #88](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/88) · [internal/saftao/export.go](internal/saftao/export.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-007 + RM-SAFT-004 | Filtro TransactionDate; AllowedPaymentTypes fail-closed; ≠ AO-* / AGT |
| [x] | RM-SAFT-014 | PurchaseInvoices em BuildIncrementalExport (sem Line) | CONCLUÍDO | [PR #89](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/89) · [internal/saftao/export.go](internal/saftao/export.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-008 + RM-SAFT-004 | Supplier refs; AllowedPurchaseTypes; sem Line; ≠ AO-* / AGT |
| [x] | RM-SAFT-015 | MovementOfGoods em BuildIncrementalExport | CONCLUÍDO | [PR #90](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/90) · [internal/saftao/export.go](internal/saftao/export.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-005 + RM-SAFT-004 | Filtro MovementDate; AllowedMovementTypes; refs; ≠ AO-* / AGT |
| [x] | RM-SAFT-016 | WorkingDocuments em BuildIncrementalExport | CONCLUÍDO | [PR #91](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/91) · [internal/saftao/export.go](internal/saftao/export.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-006 + RM-SAFT-004 | Filtro WorkDate; AllowedWorkTypes; 5 grupos L3; ≠ AO-* / AGT |
| [x] | RM-SAFT-017 | Tipagem GeneralLedgerAccounts/Account + export opcional | CONCLUÍDO | [PR #92](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/92) · [internal/saftao/general_ledger.go](internal/saftao/general_ledger.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-002 + RM-SAFT-004 | GroupingCategory; sem NumberOfEntries; ≠ AO-* / AGT |
| [x] | RM-SAFT-018 | Loader Payments (adapter/fixtures; GAP persistência) | CONCLUÍDO | [PR #93](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/93) · [internal/persistence/saft_payment_ledger.go](internal/persistence/saft_payment_ledger.go) · [internal/saftao/mappayments.go](internal/saftao/mappayments.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-013 + RM-SAFT-010 | SyntheticPaymentLedger; Store ErrUnsupported; ≠ AO-* / AGT |
| [x] | RM-SAFT-019 | Loader PurchaseInvoices (adapter/fixtures; GAP) | CONCLUÍDO | [PR #94](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/94) · [internal/persistence/saft_purchase_ledger.go](internal/persistence/saft_purchase_ledger.go) · [internal/saftao/mappurchase.go](internal/saftao/mappurchase.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-014 + RM-SAFT-010 | SyntheticPurchaseLedger; sem Line; ≠ AO-* / AGT |
| [x] | RM-SAFT-020 | Loader MovementOfGoods (adapter/fixtures; GAP) | CONCLUÍDO | [PR #95](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/95) · [internal/persistence/saft_movement_ledger.go](internal/persistence/saft_movement_ledger.go) · [internal/saftao/mapmovement.go](internal/saftao/mapmovement.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-015 + RM-SAFT-010 | SyntheticMovementLedger; ≠ AO-* / AGT |
| [x] | RM-SAFT-021 | Loader WorkingDocuments (adapter/fixtures; GAP) | CONCLUÍDO | [PR #96](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/96) · [internal/persistence/saft_working_ledger.go](internal/persistence/saft_working_ledger.go) · [internal/saftao/mapworking.go](internal/saftao/mapworking.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-016 + RM-SAFT-010 | SyntheticWorkingLedger; ≠ AO-* / AGT |
| [x] | RM-SAFT-022 | UI admin SAF-T read-only (estado estrutural) | CONCLUÍDO | [PR #97](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/97) · [internal/adminui/saft_ui.go](internal/adminui/saft_ui.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-021 + RM-UI-003 | `/admin/ui/saft`; sem XML/NIF/tokens; ≠ AO-* / AGT |
| [x] | RM-SAFT-023 | Tipagem GeneralLedgerEntries/Journal/Transaction + export opcional | CONCLUÍDO | [PR #98](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/98) · [internal/saftao/ledger_entries.go](internal/saftao/ledger_entries.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-017 + RM-SAFT-004 | TransactionType; Debit+Credit; filtro período; ≠ AO-* / AGT |
| [x] | RM-SAFT-024 | Loader GeneralLedgerEntries (adapter/fixtures; GAP) | CONCLUÍDO | [PR #99](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/99) · [internal/persistence/saft_gl_entries_ledger.go](internal/persistence/saft_gl_entries_ledger.go) · [internal/saftao/mapglentries.go](internal/saftao/mapglentries.go) · [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [CHANGELOG.md](CHANGELOG.md) | RM-SAFT-023 + RM-SAFT-010 | SyntheticGLEntriesLedger; ≠ AO-* / AGT |
| [ ] | RM-SRC-006 | Decreto 74/19 + Rect. 10/19 + 683/25 como requisitos confirmados | EM_CURSO | [compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md](compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · 2026-07-29 | OCR `reviewed` (RM-M2-C) + prioridade POLICY + revisão compliance | AO-DOC-002 + AO-SEQ-001 + AO-OFF-001 (DP71/74); outros AO-* provisórios; GAPs abertos |
| [ ] | RM-SRC-007 | Documentação FE / XSD SAF-T / contingência / chaves / HML-PRD oficiais | BLOQUEADO | [docs/01-compliance/agt-dependencies.md](docs/01-compliance/agt-dependencies.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | Acesso oficial AGT + SRC-B1/B2 (`BLOQUEADO_EXTERNO`; DEC-DEL-002) | Não trava catálogo/domínio/simulador/contratos/testes; fecho C exige AGT |
| [ ] | RM-REQ-001 | Requisitos AO-* confirmados a partir de fontes oficiais | EM_CURSO | [compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md](compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md) · [compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md](compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md) · 2026-07-29 | citação página a página + revisão compliance | **AO-DOC-002** + **AO-SEQ-001** + **AO-OFF-001** `confirmed_normative`; resto provisório; **≠** AGT |

---

## 7. Motor fiscal Angola

Fundação transacional fiscal: **CONCLUÍDA** para o slice inicial (`RM-TX-*`).

Motor regulamentar Angola: **NÃO IMPLEMENTADO**.

Integração oficial AGT: **NÃO IMPLEMENTADA**.

Homologação AGT: **NÃO INICIADA**.

Certificação: **NÃO INICIADA**.

SealInTx / `sealed_locally` **não** constituem emissão fiscal certificada.

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-ENG-001 | Tipos documentais oficiais | BLOQUEADO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [compliance/derived/requirements/DOCUMENT-CATALOG-RM-REQ-001.md](compliance/derived/requirements/DOCUMENT-CATALOG-RM-REQ-001.md) · [internal/doctype/doctype.go](internal/doctype/doctype.go) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | DEC-REG-003 (defaults) + fontes oficiais para fecho AO-* | Defaults FT+NC activos; resto off; AO-DOC-* não confirmados |
| [ ] | RM-ENG-002 | Séries e numeração conforme AGT | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-REQ-001 | Regras AO-SEQ implementadas |
| [ ] | RM-ENG-003 | Impostos, taxas, isenções, arredondamentos, descontos, retenções | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-REQ-001 | Cálculo fiscal completo testado |
| [ ] | RM-ENG-004 | NC/ND, anulação/retificação, referências, inalterabilidade | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-REQ-001 + DEC-API-002 | Documentos retificativos |
| [ ] | RM-ENG-005 | Assinatura aplicável + encadeamento só se confirmado | BLOQUEADO | [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) | Fontes oficiais + DEC-REG-KEY-CUSTODY | Assinatura sem inventar algoritmo |
| [ ] | RM-ENG-006 | Contingência + máquina de estados definitiva | BLOQUEADO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) | DEC-REG-004 + DEC-API-004 | Estados jurídicos fechados |
| [ ] | RM-ENG-007 | QR Code / FE-RNG quando aplicável | BLOQUEADO | [compliance/derived/conflicts/C-FE-QR-001-qr-url-de683-vs-fe-hml.md](compliance/derived/conflicts/C-FE-QR-001-qr-url-de683-vs-fe-hml.md) · [internal/feqr/feqr.go](internal/feqr/feqr.go) · [compliance/derived/requirements/FE-SERVICES-MATRIX-RM-REQ-001.md](compliance/derived/requirements/FE-SERVICES-MATRIX-RM-REQ-001.md) · 2026-07-28 | C-FE-QR-001 + Snapshot FE + RM-REQ-001 | Inventário+fail-closed; geração QR **não** iniciada |


---

## 8. Faturação electrónica AGT

`FISCAL_ENV=homologation` (sandbox BWB) **não** é a homologação oficial AGT.

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-FE-001 | Endpoints HML/PRD oficiais + autenticação documentada | ADIADO | [docs/01-compliance/agt-dependencies.md](docs/01-compliance/agt-dependencies.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | M6 + GAP-006 (`BLOQUEADO_EXTERNO` / DEC-DEL-002) | Credenciais AGT; não trava simulador/contratos/testes |
| [ ] | RM-FE-002 | JWS RS256 + campos exactos por operação | BLOQUEADO | [compliance/POLICY.md](compliance/POLICY.md) · [docs/01-compliance/sources.md](docs/01-compliance/sources.md) | Snapshot FE oficial (≠ assinatura SAF-T) | Assinatura FE verificável |
| [ ] | RM-FE-003 | Séries, registo, consulta, listagem, validação | PENDENTE | [docs/01-compliance/sources.md](docs/01-compliance/sources.md) | RM-FE-001 | Operações FE cobertas |
| [ ] | RM-FE-004 | Processamento assíncrono, FE-RNG, retries, reconciliação | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-FE-001 | Outbox/reconciliação AGT |
| [ ] | RM-FE-005 | Armazenamento de respostas + passagem a produção AGT | PENDENTE | [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | Homologação oficial AGT aceite | Produção FE autorizada |

---

## 9. Backoffice

Separação obrigatória (`DEC-BO-001` / `RM-ARCH-006`): **plano A** = backoffice funcional (cadastros, séries não secretas, estados, auditoria, metadados sanitizados); **plano B** = zona admin de integração/segredos (owner-only, write-only, cofre cifrado, HML≠PRD). Credenciais AGT, privadas, passwords, tokens e URLs privadas **nunca** no plano A.

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-BO-001 | Cadastros: contribuintes, estabelecimentos, scopes/bindings (plano A) | CONCLUÍDO | [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [internal/adminapi/handlers.go](internal/adminapi/handlers.go) · [internal/adminauth/auth.go](internal/adminauth/auth.go) · [internal/adminaudit/audit.go](internal/adminaudit/audit.go) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [CHANGELOG.md](CHANGELOG.md) | DEC-BO-002 + RM-BO-010 | `/admin/v1`; OpenAPI admin 0.1.0-draft; RBAC injectável fail-closed; audit append-only; sem segredos |
| [x] | RM-BO-002 | Séries e configuração não secreta; timezone; ambiente HML/PRD (metadados) | CONCLUÍDO | [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [internal/adminregistry/registry.go](internal/adminregistry/registry.go) · [internal/adminapi/handlers.go](internal/adminapi/handlers.go) · [CHANGELOG.md](CHANGELOG.md) | RM-BO-001 | PATCH `/admin/v1/scope-bindings/{id}`; IANA validado; audit; sem segredos |
| [x] | RM-BO-003 | Visibilidade ops: submissões, erros, reconciliação, auditoria; refs só metadados sanitizados | CONCLUÍDO | [internal/adminops/ops.go](internal/adminops/ops.go) · [internal/adminapi/ops_handlers.go](internal/adminapi/ops_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) | RM-BO-001 + RM-SECADM-002 | GET audit/ops/secret-refs metadata; sem corpos secretos |
| [x] | RM-BO-004 | Permissões e MFA do plano A; sem ACL de leitura de segredos | CONCLUÍDO | [internal/adminauth/rbac.go](internal/adminauth/rbac.go) · [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md) · [CHANGELOG.md](CHANGELOG.md) | DEC-BO-001 + DEC-BO-002 | Matriz RBAC tipada; MFA adiado até IdP; sem secret.reveal |
| [x] | RM-BO-005 | Listagens cadastro + PATCH status taxpayer/establishment | CONCLUÍDO | [internal/adminregistry/registry.go](internal/adminregistry/registry.go) · [internal/adminapi/handlers.go](internal/adminapi/handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) | RM-BO-001 | GET lists; PATCH status; audit; OpenAPI 0.1.5-draft; sem segredos |
| [x] | RM-UI-001 | UI shell + dashboard read-only (contribuintes/estabelecimentos/bindings) | CONCLUÍDO | [internal/adminui/ui.go](internal/adminui/ui.go) · [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) · [docs/06-delivery/local-dev.md](docs/06-delivery/local-dev.md) · [CHANGELOG.md](CHANGELOG.md) | RM-ARCH-005 + RM-BO-005 | SSR `/admin/ui/`; CSP; auth fail-closed; sem JS/segredos |
| [x] | RM-UI-002 | UI formulários/mutações + séries/config não secreta | CONCLUÍDO | [internal/adminui/forms.go](internal/adminui/forms.go) · [internal/adminui/csrf.go](internal/adminui/csrf.go) · [CHANGELOG.md](CHANGELOG.md) | RM-UI-001 + RM-BO-002 | CSRF double-submit; POST status/séries; sem segredos |
| [x] | RM-UI-003 | UI submissões, erros, reconciliação e auditoria | CONCLUÍDO | [internal/adminui/ops.go](internal/adminui/ops.go) · [CHANGELOG.md](CHANGELOG.md) | RM-UI-001 + RM-BO-003 | `/admin/ui/submissions` + `/admin/ui/audit`; acesso auditado; sem corpos secretos |
| [x] | RM-UI-004 | UI SecAdm owner-only (só metadados sanitizados) | CONCLUÍDO | [internal/adminui/secadm_ui.go](internal/adminui/secadm_ui.go) · [CHANGELOG.md](CHANGELOG.md) | RM-UI-001 + RM-SECADM-003 | GET/POST metadados; owner-only; sem plaintext |
| [x] | RM-BO-006 | Fundação OIDC/JWT provider-neutral (Admin API) | CONCLUÍDO | [internal/adminauth/oidc.go](internal/adminauth/oidc.go) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) | DEC-BO-002 + RM-BO-004 | Bearer JWT; JWKS https; alg allowlist; role map; owner subject allowlist; fail-closed; ≠ IdP fornecedor |
| [x] | RM-UI-005 | Sessão browser Secure+HttpOnly+SameSite + CSRF + logout (sem token no browser) | CONCLUÍDO | [internal/adminui/session.go](internal/adminui/session.go) · [docs/06-delivery/local-dev.md](docs/06-delivery/local-dev.md) · [CHANGELOG.md](CHANGELOG.md) | RM-BO-006 + RM-UI-001 | Cookie opaco servidor; POST Bearer→session; logout CSRF; sem JWT no browser; redirect IdP futuro |
| [x] | RM-BO-007 | Observabilidade/operabilidade do backoffice (métricas/health/audit ops) | CONCLUÍDO | [internal/adminobs/obs.go](internal/adminobs/obs.go) · [docs/07-operations/admin-observability.md](docs/07-operations/admin-observability.md) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md) · [CHANGELOG.md](CHANGELOG.md) | RM-BO-001 + RM-UI-001 | health/ready/metrics; request_id; logs/métricas sanitizados; ≠ tokens/NIF/DSN; SecAdm separado |
| [x] | RM-BO-010 | Fundação backend Taxpayer / Establishment / ScopeBinding | CONCLUÍDO | [internal/adminregistry/registry.go](internal/adminregistry/registry.go) · [migrations/postgres/0005_admin_registry.up.sql](migrations/postgres/0005_admin_registry.up.sql) · [migrations/sqlite/0005_admin_registry.up.sql](migrations/sqlite/0005_admin_registry.up.sql) · [CHANGELOG.md](CHANGELOG.md) | DEC-BO-001 | Persistência cadastros; sem UI; sem segredos; ExpectedVersion=5 (depois `0006` audit) |
| [x] | RM-SECADM-001 | Zona admin integração/segredos: owner-only, HML≠PRD, auditoria | CONCLUÍDO | [internal/secadm/gate.go](internal/secadm/gate.go) · [internal/secretstore/secretstore.go](internal/secretstore/secretstore.go) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [CHANGELOG.md](CHANGELOG.md) | DEC-BO-001 | Gate owner-only; HML≠PRD no store; HTTP em RM-SECADM-003 |
| [x] | RM-SECADM-002 | Contrato write-only SecretStore + simulator fail-closed + metadados | CONCLUÍDO | [internal/secretstore/secretstore.go](internal/secretstore/secretstore.go) · [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) · [CHANGELOG.md](CHANGELOG.md) | DEC-BO-001 | Sem GET admin; AES-GCM memória; HML≠PRD; sem credenciais reais |
| [x] | RM-SECADM-003 | HTTP SecAdm Put/Rotate/Revoke write-only + audit | CONCLUÍDO | [internal/adminapi/secadm_handlers.go](internal/adminapi/secadm_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) | RM-SECADM-001 + RM-SECADM-002 + RM-BO-004 | `/admin/v1/secadm/*`; owner role+subject; sem plaintext na resposta; sem AGT real |
| [x] | RM-AGTPREP-001 | DEC-BO-004: config AGT pública vs SecAdm segredos + roadmap prep | CONCLUÍDO | [PR #100](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/100) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) · [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md) · [CHANGELOG.md](CHANGELOG.md) | DEC-BO-001 + DEC-DEL-002 | Separação AuthorityProfile vs SecretStore; ≠ AGT real |
| [x] | RM-AGTPREP-002 | Modelo + API admin AuthorityProfile (HML/PRD; sem segredos) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/101 · [internal/adminregistry/authority_profile.go](internal/adminregistry/authority_profile.go) · [internal/adminapi/authority_handlers.go](internal/adminapi/authority_handlers.go) · [migrations/postgres/0007_authority_profiles.up.sql](migrations/postgres/0007_authority_profiles.up.sql) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-001 + RM-BO-001 | draft/validated/active/revoked; pending_external; ExpectedVersion=7; ≠ AGT |
| [x] | RM-AGTPREP-003 | UI owner-only perfis autoridade + readiness sanitizado | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/102 · [internal/adminui/authority_ui.go](internal/adminui/authority_ui.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-002 + RM-UI-005 | Fingerprint/validade/alg/key-id; sem plaintext |
| [x] | RM-AGTPREP-004 | SecAdm import/rotate/revoke cert+chave+credencial (SecretStore) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/103 · [internal/secretstore/material.go](internal/secretstore/material.go) · [internal/adminapi/secadm_material_handlers.go](internal/adminapi/secadm_material_handlers.go) · [internal/adminui/secadm_material_ui.go](internal/adminui/secadm_material_ui.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-001 + RM-SECADM-003 | Limites upload; password efémera; sem plaintext |
| [x] | RM-AGTPREP-005 | Validação offline par chave-cert / cadeia / fingerprint | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/104 · [internal/secretstore/offline_validate.go](internal/secretstore/offline_validate.go) · [internal/adminapi/secadm_offline_handlers.go](internal/adminapi/secadm_offline_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-004 | ≠ external_verified / AGT |
| [x] | RM-AGTPREP-006 | Adaptador simulador + teste ligação; prod fail-closed | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/105 · [internal/authority/prep/probe.go](internal/authority/prep/probe.go) · [internal/adminapi/authority_probe_handlers.go](internal/adminapi/authority_probe_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-002 + RM-TX-009 | simulator≠AGT; agt-hml/prd fail-closed |
| [x] | RM-AGTPREP-007 | Audit/RBAC/métricas + checklist readiness (4 flags) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/106 · [internal/adminapi/authority_readiness_handlers.go](internal/adminapi/authority_readiness_handlers.go) · [internal/adminobs/obs.go](internal/adminobs/obs.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-002…006 + RM-BO-007 | config/secrets/offline/external_verified=false |
| [x] | RM-BO-011 | UI checklist readiness AuthorityProfile (4 flags) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/107 · [internal/adminui/authority_ui.go](internal/adminui/authority_ui.go) · [internal/adminui/templates/authority_readiness.html](internal/adminui/templates/authority_readiness.html) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-007 + RM-AGTPREP-003 | Sem plaintext; external_verified=false |
| [x] | RM-AGTPREP-008 | Wizard owner-only preparação AuthorityProfile (3 passos; draft only) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/108 · [internal/adminui/authority_wizard.go](internal/adminui/authority_wizard.go) · [internal/adminui/templates/authority_wizard.html](internal/adminui/templates/authority_wizard.html) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-003 + RM-BO-011 | Sem activação; external_verified=false; ≠ AGT/plaintext |
| [x] | RM-AGTPREP-009 | Gate activação fail-closed (active exige config+secrets+offline; nunca external_verified) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/109 · [internal/adminregistry/authority_profile.go](internal/adminregistry/authority_profile.go) · [internal/adminregistry/authority_activation_gate_test.go](internal/adminregistry/authority_activation_gate_test.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-007 + RM-AGTPREP-008 | Simulator≠produção; ≠ AGT |
| [x] | RM-AGTPREP-010 | Lifecycle/rotação certificados (metadados SecretStore; sem plaintext) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/110 · [internal/authority/prep/lifecycle.go](internal/authority/prep/lifecycle.go) · [internal/adminapi/authority_lifecycle_handlers.go](internal/adminapi/authority_lifecycle_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-004 + RM-AGTPREP-005 | ≠ AGT |
| [x] | RM-AGTPREP-011 | Histórico append-only eventos perfil/material autoridade | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/111 · [internal/adminaudit/audit.go](internal/adminaudit/audit.go) · [internal/adminapi/authority_history_handlers.go](internal/adminapi/authority_history_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-007 + RM-BO-003 | Sem plaintext / NIF completo |
| [x] | RM-AGTPREP-012 | Alertas sanitizados readiness (expiração/refs/flags) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/112 · [internal/authority/prep/alerts.go](internal/authority/prep/alerts.go) · [internal/adminapi/authority_readiness_handlers.go](internal/adminapi/authority_readiness_handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-007 + RM-BO-011 | ≠ external_verified |
| [x] | RM-AGTPREP-013 | Testes de segurança superfície autoridade (RBAC/CSRF/leaks) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/113 · [internal/adminui/authority_security_test.go](internal/adminui/authority_security_test.go) · [internal/adminapi/authority_security_test.go](internal/adminapi/authority_security_test.go) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-AGTPREP-008…012 | Owner-only; sem PEM/password |
| [x] | RM-AGTPREP-014 | SecretStore durável cifrado em repouso (SQL + master key externa; HML/PRD fail-closed) | CONCLUÍDO | [internal/secretstore/sqlstore.go](internal/secretstore/sqlstore.go) · [internal/secretstore/open.go](internal/secretstore/open.go) · [migrations/postgres/0013_secret_store_entries.up.sql](migrations/postgres/0013_secret_store_entries.up.sql) · [deploy/postgres/grants-schema13-runtime-admin.sql](deploy/postgres/grants-schema13-runtime-admin.sql) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-29 | RM-SECADM-002 + RM-AGTPREP-004 | Ciphertext-only; write-only; ≠ AGT/KMS; ExpectedVersion=13 |
| [x] | RM-BO-012 | Cadastro cliente/loja + adesão FE enum (DEC-PROD-004) sem NIF em audit/logs | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/114 · [migrations/postgres/0009_taxpayer_fe_enrollment.up.sql](migrations/postgres/0009_taxpayer_fe_enrollment.up.sql) · [internal/adminregistry/fe_enrollment.go](internal/adminregistry/fe_enrollment.go) · [internal/adminapi/handlers.go](internal/adminapi/handlers.go) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-BO-001 + RM-BO-005 + DEC-PROD-004 | Enum por ambiente; active≡aderiu; ExpectedVersion=9; ≠ booleano / AGT |
| [x] | RM-BO-013 | Política calculada documentos disponíveis (5 grupos; SAF-T; FE se aderiu; pendências AGT indisponíveis) | CONCLUÍDO | [internal/doctype/availability.go](internal/doctype/availability.go) · [internal/adminregistry/doc_policy.go](internal/adminregistry/doc_policy.go) · [internal/adminapi/doc_policy_handlers.go](internal/adminapi/doc_policy_handlers.go) · [migrations/postgres/0010_establishment_doc_policy.up.sql](migrations/postgres/0010_establishment_doc_policy.up.sql) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-BO-012 + DEC-PROD-003/005 + RM-REQ-001 | Sem inventar códigos FE-RNG; só tipos activos/compatíveis |
| [x] | RM-BO-014 | Séries por loja/ambiente/tipo (draft/active/closed; código único; sem reutilizar/retroceder; audit) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/116 · [internal/adminregistry/series.go](internal/adminregistry/series.go) · [internal/adminapi/series_handlers.go](internal/adminapi/series_handlers.go) · [internal/adminui/series_ui.go](internal/adminui/series_ui.go) · [migrations/postgres/0011_establishment_series.up.sql](migrations/postgres/0011_establishment_series.up.sql) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-BO-002 + RM-BO-012 | Séries no estabelecimento; concorrência transaccional; ≠ numeração fiscal |
| [x] | RM-BO-015 | Painel filas/submissões (estados; tentativas; next_attempt; request_id; erro sanitizado) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/117 · [internal/adminops/ops.go](internal/adminops/ops.go) · [internal/adminapi/ops_handlers.go](internal/adminapi/ops_handlers.go) · [internal/adminui/templates/submissions.html](internal/adminui/templates/submissions.html) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-BO-003 + RM-UI-003 | Sem payload/JWS/segredo na UI/API |
| [x] | RM-BO-016 | Acções retry/cancel/manual-review (RBAC/CSRF/idempotência/audit); simulador ≠ produção | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/118 · [internal/adminops/actions.go](internal/adminops/actions.go) · [internal/adminapi/ops_action_handlers.go](internal/adminapi/ops_action_handlers.go) · [migrations/postgres/0012_ops_queue_actions.up.sql](migrations/postgres/0012_ops_queue_actions.up.sql) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-BO-015 + RM-BO-004 | Simulator só development/homologation interno; ExpectedVersion=12 |
| [x] | RM-BO-017 | Dashboards/filtros/paginação/alertas + testes backoffice ops | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/119 · [internal/adminops/dashboard.go](internal/adminops/dashboard.go) · [internal/adminapi/ops_handlers.go](internal/adminapi/ops_handlers.go) · [internal/adminui/templates/dashboard.html](internal/adminui/templates/dashboard.html) · [specs/admin/openapi.yaml](specs/admin/openapi.yaml) · [CHANGELOG.md](CHANGELOG.md) · 2026-07-27 | RM-BO-013…016 + RM-BO-007 | Filtros/paginação/alertas cobertos por testes |
| [x] | RM-BO-018 | Onboarding IdP provider-neutral + readiness diagnostics (sem IdP local inseguro) | CONCLUÍDO | https://github.com/storesace-cv/bwb-modulo-fiscal/pull/120 · [docs/07-operations/admin-idp-onboarding.md](docs/07-operations/admin-idp-onboarding.md) · [internal/adminauth/auth_readiness.go](internal/adminauth/auth_readiness.go) · [docs/07-operations/rm-ops-006-sandbox-schema12-promotion-report.md](docs/07-operations/rm-ops-006-sandbox-schema12-promotion-report.md) · 2026-07-27 | RM-BO-006 + RM-UI-005 | fail_closed até oidc_jwt; interactive login deferred |

---

## 10. Edge/offline

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-EDGE-001 | Serviço Edge + SQLite WAL + fila + sync at-least-once | ADIADO | [docs/02-architecture/edge-architecture.md](docs/02-architecture/edge-architecture.md) | M12 após estabilização Angola cloud | Edge emitindo offline nos limites legais |
| [ ] | RM-EDGE-002 | Deduplicação, reconciliação, chaves, updates assinados | ADIADO | [docs/02-architecture/edge-architecture.md](docs/02-architecture/edge-architecture.md) | M12 + DEC-SEC-EDGE-KEYS | Sync seguro |
| [ ] | RM-EDGE-003 | Instalador, observabilidade, suporte, falhas offline | ADIADO | [docs/02-architecture/edge-architecture.md](docs/02-architecture/edge-architecture.md) | M12 | Pacote suportável |

---

## 11. Integradores e software houses

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-INT-001 | Kit POS sandbox + docs onboarding | CONCLUÍDO | [docs/03-api/onboarding.md](docs/03-api/onboarding.md) · [docs/07-operations/s4-pos-kit-ops-validation-report.md](docs/07-operations/s4-pos-kit-ops-validation-report.md) | — | Integração sandbox |
| [ ] | RM-INT-002 | Portal: registo, quotas, versões, breaking changes, SDKs, suporte, termos | PENDENTE | [docs/03-api/publishing.md](docs/03-api/publishing.md) | M7–M11 | Onboarding comercial |
| [ ] | RM-INT-003 | Passagem sandbox → produção | PENDENTE | [docs/03-api/integration-lifecycle.md](docs/03-api/integration-lifecycle.md) | M11 + certificação | Processo de promoção |

---

## 12. Operações de produção

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-PROD-001 | Ambiente/domínio produção separados do sandbox | PENDENTE | [docs/07-operations/deployment.md](docs/07-operations/deployment.md) | M11 | Host produção dedicado |
| [ ] | RM-PROD-002 | HA, backups, retenção, restore periódico | PENDENTE | [docs/07-operations/operations.md](docs/07-operations/operations.md) | M11 | RPO/RTO ensaiados |
| [ ] | RM-PROD-003 | Monitorização, alertas, métricas, logs, incidentes, DR | PENDENTE | [docs/07-operations/operations.md](docs/07-operations/operations.md) | M11 | Observabilidade produção |
| [ ] | RM-PROD-004 | Segredos, WAF/DDoS, pentest, SBOM, patches, capacidade, privacidade | PENDENTE | [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md) | M8–M11 | Hardening produção |

---

## 13. Certificação AGT

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-CERT-001 | Registo produtor + Modelo 8 + manuais + tabelas + chave pública | ADIADO | [docs/01-compliance/agt-dependencies.md](docs/01-compliance/agt-dependencies.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | M9 + GAP-003 (`BLOQUEADO_EXTERNO` / DEC-DEL-002) | Acesso produtores AGT; não trava catálogo/domínio/CI |
| [ ] | RM-CERT-002 | SAF-T + ambiente de testes AGT + correcção NCs | PENDENTE | [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md) | M5–M6 + RM-CERT-001 | Testes AGT verdes |
| [ ] | RM-CERT-003 | Certificado + controlo da versão certificada | PENDENTE | [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md) | Aceitação AGT | Número de certificado publicado |

Homologação oficial AGT e certificação **não** são o sandbox BWB.

---

## 14. Cabo Verde

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-CV-001 | Pacote fiscal CV independente (sem reutilizar regras AO) | ADIADO | [docs/00-product/scope.md](docs/00-product/scope.md) · [AGENTS.md](AGENTS.md) | M13 após Angola estabilizada | Catálogo CV próprio aprovado |

---

## 15. Bloqueios, decisões e incidentes

### Governação documental

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-GOV-001 | Template PR + AGENTS/rules + verificador CI do roadmap | CONCLUÍDO | [ROADMAP.md](ROADMAP.md) · [.github/pull_request_template.md](.github/pull_request_template.md) · [AGENTS.md](AGENTS.md) · [.cursor/rules/roadmap-maintenance.mdc](.cursor/rules/roadmap-maintenance.mdc) · [scripts/verify_roadmap.py](scripts/verify_roadmap.py) · [tests/docs/run-verify-roadmap-tests.sh](tests/docs/run-verify-roadmap-tests.sh) · [.github/workflows/ci.yml](.github/workflows/ci.yml) · [PR #28](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/28) | — | Política assistida e verificável no repo |
| [x] | RM-GOV-002 | Ruleset activo em main com PR e checks obrigatórios | CONCLUÍDO | [Ruleset #19731202](https://github.com/storesace-cv/bwb-modulo-fiscal/rules/19731202) · [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | — | Ruleset activo em main; PR obrigatório; checks obrigatórios; sem bypass |

<a id="rm-gov-002-ruleset-de-main"></a>

#### RM-GOV-002 — ruleset de main

- Branch protection clássica em `main`: **ausente** (404); a proteção canónica é o ruleset de repositório.
- Ruleset activo: **Protect main and require project checks** (ID `19731202`), target `branch`, enforcement `active`, condição apenas `refs/heads/main`.
- URL estável: [https://github.com/storesace-cv/bwb-modulo-fiscal/rules/19731202](https://github.com/storesace-cv/bwb-modulo-fiscal/rules/19731202).
- Bypass actors: **vazio** (`current_user_can_bypass=never`).
- Regras: `deletion`, `non_fast_forward`, `pull_request`, `required_status_checks`.
- Pull request: `required_approving_review_count=0`; sem code owners; sem aprovação do último push; resolução de review threads obrigatória; merge permitido apenas via squash neste ruleset.
- Checks obrigatórios (strict): `go-checks` (integration `15368`), `predeploy-pg16-real` (integration `15368`), `GitGuardian Security Checks` (integration `46505`).
- `cubic · AI code reviewer`: **opcional** (não incluído nos required status checks).

### Incidentes e decisões abertas

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-BLK-001 | INC-B2-001 — prova remediação lock poisoned/stale/corrupt | BLOQUEADO | [docs/07-operations/b2-predeploy-pg-dump-gate-report.md](docs/07-operations/b2-predeploy-pg-dump-gate-report.md) | Procedimento seguro ensaiado | INC-B2-001 fechado |
| [ ] | RM-BLK-002 | DEC-REG-KEY-CUSTODY / GAP-013 custódia chave contribuinte | BLOQUEADO | [docs/01-compliance/agt-dependencies.md](docs/01-compliance/agt-dependencies.md) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | Orientação oficial AGT (`BLOQUEADO_EXTERNO`; DEC-DEL-002) | Custódia definitiva; não trava chave efémera/simulador/testes |
| [x] | RM-BLK-003 | INC-S4-003 resolvido | CONCLUÍDO | [docs/07-operations/b2-predeploy-pg-dump-gate-report.md](docs/07-operations/b2-predeploy-pg-dump-gate-report.md) | — | Gate pré-deploy comprovado |

---

## 16. Critérios de conclusão

Não existe uma única checkbox «projecto concluído». Níveis:

1. **Fundação concluída** — plataforma, sandbox, API POS, credenciais, deploy/backup (M0–M1).
2. **MVP fiscal Angola** — requisitos AO-*, motor regulamentar, primeiro fluxo oficial, SAF-T, FE em homologação **oficial** AGT (não apenas `FISCAL_ENV=homologation`).
3. **Certificação concluída** — dossier aceite, certificado, versão controlada.
4. **Produção comercial** — ambiente prod, segurança, observabilidade, backup/restore, suporte, piloto.
5. **Edge concluído** — M12.
6. **Expansão Cabo Verde** — M13.

Para Angola comercial, exigir no mínimo: regras oficiais rastreáveis; motor fiscal; SAF-T AO; integração FE; homologação oficial AGT; certificação; operação de produção; segurança; observabilidade; backup/restore; suporte; piloto; evidências.

---

## 17. Evidências e documentos relacionados

| Tipo | Documentos |
|---|---|
| Contrato | [specs/openapi/openapi.yaml](specs/openapi/openapi.yaml) |
| Compliance | [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml), [docs/01-compliance/](docs/01-compliance/) |
| Ops (históricos) | [docs/07-operations/](docs/07-operations/) — não reescritos por modernização |
| Decisões | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) |
| Apontador legado | [docs/06-delivery/implementation-roadmap.md](docs/06-delivery/implementation-roadmap.md) |
| Histórico de versões | [CHANGELOG.md](CHANGELOG.md) (não substitui este ROADMAP) |

---

## 18. Regras de manutenção do roadmap

1. Qualquer PR que conclua, introduza, bloqueie, adie ou altere um item `RM-*` deve actualizar este ficheiro no mesmo PR.
2. Alterações materiais a este ficheiro (qualquer conteúdo excepto a própria linha «Estado revisto em») exigem actualizar **Estado revisto em** para a data UTC do dia da alteração; a CI (`scripts/verify_roadmap.py`) rejeita data desactualizada quando detecta diff material vs a base.
3. `[x]` / `CONCLUÍDO` exige evidência verificável (link relativo, âncora interna ou URL HTTPS).
4. Plano aprovado, Draft PR ou PR Ready **não** são conclusão por si só; PR merged prova integração no Git, não deploy nem conformidade.
5. Deploy prova instalação, não conformidade; teste sandbox não prova homologação AGT.
6. Nenhum item fiscal é marcado concluído só por inferência técnica.
7. Um item criado atomicamente por este PR pode ficar `CONCLUÍDO` com links relativos aos artefactos do changeset; o URL do PR é acrescentado antes de Ready; se falhar artefacto/check, permanece `EM_CURSO`.
8. CI valida estrutura; a semântica de conclusão continua humana.
9. Template de PR: documentação atualizada **ou** não afetada com justificação.

### Marcos

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-M0 | M0 — Fundação técnica | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) · [internal/platform/dbmigrate/migrate.go](internal/platform/dbmigrate/migrate.go) | — | Stack + persistência + SealInTx |
| [x] | RM-M1 | M1 — Sandbox POS autenticado | CONCLUÍDO | [docs/07-operations/s3c2-sandbox-promotion-report.md](docs/07-operations/s3c2-sandbox-promotion-report.md) · [docs/07-operations/s4-pos-kit-ops-validation-report.md](docs/07-operations/s4-pos-kit-ops-validation-report.md) | — | Open autenticado + kit 9/9 |
| [x] | RM-M2-A | M2-A — Governação/catalogação SRC-A + B0 | CONCLUÍDO | [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) · [docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md](docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md) | — | Metadados e auditoria |
| [x] | RM-M2-B | M2-B — Armazenamento sincronizado fontes | CONCLUÍDO | [https://github.com/storesace-cv/bwb-fiscal-sources-ao](https://github.com/storesace-cv/bwb-fiscal-sources-ao) · commit [`c93db4f89352eab884a4862943faa2ed874217fc`](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/c93db4f89352eab884a4862943faa2ed874217fc) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) | — | Repo privado + originais B1 + integridade |
| [x] | RM-M2-C | M2-C — OCR e revisão visual | CONCLUÍDO | [bwb-fiscal-sources-ao@c8a4e6e](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/c8a4e6e8ec2772ff50ad1c8762842b983edbbbfd) · [PR #3](https://github.com/storesace-cv/bwb-fiscal-sources-ao/pull/3) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | — | 74+Rect v2+683 `reviewed`; DP 71/25 `reviewed` (11902–11920); rejected ≠ KB |
| [ ] | RM-M2-D | M2-D — Requisitos AO-* confirmados | EM_CURSO | [compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md](compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md) · [compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md](compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md) · 2026-07-29 | RM-M2-C + SRC-B2 | **AO-DOC-002** + **AO-SEQ-001** + **AO-OFF-001**; resto provisório; **≠** AGT |
| [ ] | RM-M2 | M2 — Fontes oficiais governadas (completo) | PENDENTE | [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) | RM-M2-A + RM-M2-B + RM-M2-C + RM-M2-D | Critérios de saída M2 |
| [ ] | RM-M3 | M3 — Requisitos Angola confirmados | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | M2 completo | AO-* confirmados |
| [ ] | RM-M4 | M4 — Primeiro documento fiscal completo | PENDENTE | [docs/06-delivery/first-vertical-slice.md](docs/06-delivery/first-vertical-slice.md) | M3 + motor regulamentar | Fluxo oficial |
| [ ] | RM-M5 | M5 — SAF-T AO válido | PENDENTE | [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) | M4 + XSD oficial | XML válido |
| [ ] | RM-M6 | M6 — FE em homologação oficial AGT | PENDENTE | [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | M4 + credenciais AGT | ≠ sandbox BWB |
| [x] | RM-M7 | M7 — Backoffice operacional mínimo | CONCLUÍDO | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) · [internal/adminui/ui.go](internal/adminui/ui.go) · [CHANGELOG.md](CHANGELOG.md) | M4 + RM-UI-001…004 | Ops mínimas SSR `/admin/ui/` |
| [ ] | RM-M8 | M8 — Readiness de certificação | PENDENTE | [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md) | M5–M7 | Dossier pronto |
| [ ] | RM-M9 | M9 — Certificação AGT | PENDENTE | [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md) | M8 + testes AGT | Certificado |
| [ ] | RM-M10 | M10 — Piloto | PENDENTE | [docs/00-product/vision.md](docs/00-product/vision.md) | M9 | Piloto assistido |
| [ ] | RM-M11 | M11 — Produção Angola | PENDENTE | [docs/07-operations/operations.md](docs/07-operations/operations.md) | M10 | Produção comercial |
| [ ] | RM-M12 | M12 — Edge | ADIADO | [docs/02-architecture/edge-architecture.md](docs/02-architecture/edge-architecture.md) | M11 | Edge certificado nos limites |
| [ ] | RM-M13 | M13 — Cabo Verde | ADIADO | [docs/00-product/scope.md](docs/00-product/scope.md) | M11 + catálogo CV | Expansão CV |
