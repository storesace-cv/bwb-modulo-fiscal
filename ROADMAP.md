# ROADMAP — BWB Módulo Fiscal

**Fonte canónica de estado e progresso do projecto.**

**Estado revisto em:** 2026-07-25

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
| OpenAPI | `0.1.6-draft` ([specs/openapi/openapi.yaml](specs/openapi/openapi.yaml)) |
| Schema | `ExpectedVersion=3` |
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
| Requisitos AO-* confirmados | EM_CURSO (matriz provisória; citação página a página + revisão compliance; sem confirmados) |
| Fundação transacional fiscal | CONCLUÍDA (slice) |
| Motor regulamentar Angola | NÃO IMPLEMENTADO |
| Integração oficial AGT | NÃO IMPLEMENTADA |
| SAF-T AO | NÃO IMPLEMENTADO |
| Assinatura / JWS AGT | NÃO IMPLEMENTADA |
| Faturação electrónica AGT | NÃO INTEGRADA |
| Homologação BWB (`FISCAL_ENV=homologation`) | Ambiente técnico sandbox — **não** é homologação AGT |
| Homologação oficial AGT | NÃO INICIADA |
| Certificação AGT | NÃO INICIADA |
| Produção comercial | NÃO INICIADA |
| Backoffice | Planeado |
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
| [x] | RM-ARCH-003 | Migrations dual-engine PG/SQLite | CONCLUÍDO | [internal/platform/dbmigrate/migrate.go](internal/platform/dbmigrate/migrate.go) | — | ExpectedVersion=3 |
| [x] | RM-ARCH-004 | Scopes multi-tenant e separação de papéis | CONCLUÍDO | [docs/02-architecture/system-architecture.md](docs/02-architecture/system-architecture.md) · [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | — | Plataforma / integradores / contribuintes / estabelecimentos / credenciais POS / chaves AGT / HML-PRD |
| [ ] | RM-ARCH-005 | Backoffice operacional | PENDENTE | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | M7 | UI/ops mínimos em produção |

### 3.3 Motor transacional inicial

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TX-001 | Persistência PG/SQLite, documentos/linhas imutáveis, ledger, outbox | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) · [docs/06-delivery/first-vertical-slice.md](docs/06-delivery/first-vertical-slice.md) | — | Slice transacional persistido |
| [x] | RM-TX-002 | Numeração por série, idempotência, conflito external_id | CONCLUÍDO | [specs/openapi/openapi.yaml](specs/openapi/openapi.yaml) | — | Autoridade de numeração no módulo |
| [x] | RM-TX-003 | Hash canónico versionado, rollback, concorrência | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) | — | Testes dual-engine |
| [x] | RM-TX-004 | Timezone IANA, Africa/Luanda, hora fiscal vs técnica, microssegundos | CONCLUÍDO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) | — | DEC-TIME-001 |
| [x] | RM-TX-005 | Estados neutros + SealInTx (selagem local) | CONCLUÍDO | [specs/openapi/openapi.yaml](specs/openapi/openapi.yaml) | — | `sealed_locally` ≠ emissão certificada AGT |

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
| [ ] | RM-SRC-006 | Decreto 74/19 + Rect. 10/19 + 683/25 como requisitos confirmados | BLOQUEADO | [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | OCR `reviewed` + prioridade POLICY | GAPs legais fechados sem inventar |
| [ ] | RM-SRC-007 | Documentação FE / XSD SAF-T / contingência / chaves / HML-PRD oficiais | BLOQUEADO | [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | Acesso oficial AGT + SRC-B1/B2 | Fontes oficiais suficientes para C |
| [ ] | RM-REQ-001 | Requisitos AO-* confirmados a partir de fontes oficiais | EM_CURSO | [compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md](compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md) · [compliance/derived/requirements/FE-SERVICES-MATRIX-RM-REQ-001.md](compliance/derived/requirements/FE-SERVICES-MATRIX-RM-REQ-001.md) · [compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md](compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md) · [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | citação página a página + revisão compliance; DEC-REG-003 | Citação H FE/FE-RNG; AGT-001 `pending_validation`; AGT-002/TAX/SEQ/ID/CRYPTO/DOC-002/OFF `partial`; DOC-001 `scaffold`; SAF `pending_validation`; C-FE-001/C-DOC-003/C-SIGN-001; sem confirmados |

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
| [ ] | RM-ENG-001 | Tipos documentais oficiais | BLOQUEADO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | DEC-REG-003 + fontes oficiais | Tipos MVP confirmados |
| [ ] | RM-ENG-002 | Séries e numeração conforme AGT | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-REQ-001 | Regras AO-SEQ implementadas |
| [ ] | RM-ENG-003 | Impostos, taxas, isenções, arredondamentos, descontos, retenções | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-REQ-001 | Cálculo fiscal completo testado |
| [ ] | RM-ENG-004 | NC/ND, anulação/retificação, referências, inalterabilidade | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-REQ-001 + DEC-API-002 | Documentos retificativos |
| [ ] | RM-ENG-005 | Assinatura aplicável + encadeamento só se confirmado | BLOQUEADO | [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) | Fontes oficiais + DEC-REG-KEY-CUSTODY | Assinatura sem inventar algoritmo |
| [ ] | RM-ENG-006 | Contingência + máquina de estados definitiva | BLOQUEADO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) | DEC-REG-004 + DEC-API-004 | Estados jurídicos fechados |
| [ ] | RM-ENG-007 | QR Code / FE-RNG quando aplicável | PENDENTE | [docs/01-compliance/sources.md](docs/01-compliance/sources.md) | Snapshot FE oficial + RM-REQ-001 | Campos FE-RNG correctos |

---

## 8. Faturação electrónica AGT

`FISCAL_ENV=homologation` (sandbox BWB) **não** é a homologação oficial AGT.

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-FE-001 | Endpoints HML/PRD oficiais + autenticação documentada | BLOQUEADO | [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | Credenciais e docs oficiais AGT | Cliente FE em HML AGT |
| [ ] | RM-FE-002 | JWS RS256 + campos exactos por operação | BLOQUEADO | [compliance/POLICY.md](compliance/POLICY.md) · [docs/01-compliance/sources.md](docs/01-compliance/sources.md) | Snapshot FE oficial (≠ assinatura SAF-T) | Assinatura FE verificável |
| [ ] | RM-FE-003 | Séries, registo, consulta, listagem, validação | PENDENTE | [docs/01-compliance/sources.md](docs/01-compliance/sources.md) | RM-FE-001 | Operações FE cobertas |
| [ ] | RM-FE-004 | Processamento assíncrono, FE-RNG, retries, reconciliação | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-FE-001 | Outbox/reconciliação AGT |
| [ ] | RM-FE-005 | Armazenamento de respostas + passagem a produção AGT | PENDENTE | [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | Homologação oficial AGT aceite | Produção FE autorizada |

---

## 9. Backoffice

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-BO-001 | Gestão integradores, contribuintes, estabelecimentos, NIF, scopes | PENDENTE | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | M7 | Cadastros mínimos |
| [ ] | RM-BO-002 | Séries, timezone, credenciais POS, ambientes HML/PRD | PENDENTE | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | M7 | Ops de scope sem ler segredos |
| [ ] | RM-BO-003 | Referências AGT, submissões, erros, reconciliação, auditoria, SAF-T | PENDENTE | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | M6–M7 | Visibilidade operacional |
| [ ] | RM-BO-004 | Permissões e MFA; segredos não legíveis na UI | PENDENTE | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | DEC-REG-KEY-CUSTODY | UI sem privadas |

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
| [ ] | RM-CERT-001 | Registo produtor + Modelo 8 + manuais + tabelas + chave pública | BLOQUEADO | [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) · [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | GAP-003 + acesso oficial | Dossier submetível |
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
| [ ] | RM-BLK-002 | DEC-REG-KEY-CUSTODY / GAP-013 custódia chave contribuinte | BLOQUEADO | [docs/06-delivery/open-decisions.md](docs/06-delivery/open-decisions.md) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | Orientação oficial AGT | Modelo de custódia autorizado |
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
2. `[x]` / `CONCLUÍDO` exige evidência verificável (link relativo, âncora interna ou URL HTTPS).
3. Plano aprovado, Draft PR ou PR Ready **não** são conclusão por si só; PR merged prova integração no Git, não deploy nem conformidade.
4. Deploy prova instalação, não conformidade; teste sandbox não prova homologação AGT.
5. Nenhum item fiscal é marcado concluído só por inferência técnica.
6. Um item criado atomicamente por este PR pode ficar `CONCLUÍDO` com links relativos aos artefactos do changeset; o URL do PR é acrescentado antes de Ready; se falhar artefacto/check, permanece `EM_CURSO`.
7. CI valida estrutura; a semântica de conclusão continua humana.
8. Template de PR: documentação atualizada **ou** não afetada com justificação.

### Marcos

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-M0 | M0 — Fundação técnica | CONCLUÍDO | [CHANGELOG.md](CHANGELOG.md) · [internal/platform/dbmigrate/migrate.go](internal/platform/dbmigrate/migrate.go) | — | Stack + persistência + SealInTx |
| [x] | RM-M1 | M1 — Sandbox POS autenticado | CONCLUÍDO | [docs/07-operations/s3c2-sandbox-promotion-report.md](docs/07-operations/s3c2-sandbox-promotion-report.md) · [docs/07-operations/s4-pos-kit-ops-validation-report.md](docs/07-operations/s4-pos-kit-ops-validation-report.md) | — | Open autenticado + kit 9/9 |
| [x] | RM-M2-A | M2-A — Governação/catalogação SRC-A + B0 | CONCLUÍDO | [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) · [docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md](docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md) | — | Metadados e auditoria |
| [x] | RM-M2-B | M2-B — Armazenamento sincronizado fontes | CONCLUÍDO | [https://github.com/storesace-cv/bwb-fiscal-sources-ao](https://github.com/storesace-cv/bwb-fiscal-sources-ao) · commit [`c93db4f89352eab884a4862943faa2ed874217fc`](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/c93db4f89352eab884a4862943faa2ed874217fc) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) | — | Repo privado + originais B1 + integridade |
| [x] | RM-M2-C | M2-C — OCR e revisão visual | CONCLUÍDO | [bwb-fiscal-sources-ao@c8a4e6e](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/c8a4e6e8ec2772ff50ad1c8762842b983edbbbfd) · [PR #3](https://github.com/storesace-cv/bwb-fiscal-sources-ao/pull/3) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) · [docs/01-compliance/regulatory-gaps.md](docs/01-compliance/regulatory-gaps.md) | — | 74+Rect v2+683 `reviewed`; DP 71/25 `reviewed` (11902–11920); rejected ≠ KB |
| [ ] | RM-M2-D | M2-D — Requisitos AO-* confirmados | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | RM-M2-C + SRC-B2 | Matriz AO-* |
| [ ] | RM-M2 | M2 — Fontes oficiais governadas (completo) | PENDENTE | [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) · [compliance/catalog/sources.yaml](compliance/catalog/sources.yaml) | RM-M2-A + RM-M2-B + RM-M2-C + RM-M2-D | Critérios de saída M2 |
| [ ] | RM-M3 | M3 — Requisitos Angola confirmados | PENDENTE | [docs/01-compliance/requirements-catalog.md](docs/01-compliance/requirements-catalog.md) | M2 completo | AO-* confirmados |
| [ ] | RM-M4 | M4 — Primeiro documento fiscal completo | PENDENTE | [docs/06-delivery/first-vertical-slice.md](docs/06-delivery/first-vertical-slice.md) | M3 + motor regulamentar | Fluxo oficial |
| [ ] | RM-M5 | M5 — SAF-T AO válido | PENDENTE | [docs/01-compliance/saft-ao.md](docs/01-compliance/saft-ao.md) | M4 + XSD oficial | XML válido |
| [ ] | RM-M6 | M6 — FE em homologação oficial AGT | PENDENTE | [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md) | M4 + credenciais AGT | ≠ sandbox BWB |
| [ ] | RM-M7 | M7 — Backoffice operacional mínimo | PENDENTE | [docs/02-architecture/backoffice-architecture.md](docs/02-architecture/backoffice-architecture.md) | M4 | Ops mínimas |
| [ ] | RM-M8 | M8 — Readiness de certificação | PENDENTE | [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md) | M5–M7 | Dossier pronto |
| [ ] | RM-M9 | M9 — Certificação AGT | PENDENTE | [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md) | M8 + testes AGT | Certificado |
| [ ] | RM-M10 | M10 — Piloto | PENDENTE | [docs/00-product/vision.md](docs/00-product/vision.md) | M9 | Piloto assistido |
| [ ] | RM-M11 | M11 — Produção Angola | PENDENTE | [docs/07-operations/operations.md](docs/07-operations/operations.md) | M10 | Produção comercial |
| [ ] | RM-M12 | M12 — Edge | ADIADO | [docs/02-architecture/edge-architecture.md](docs/02-architecture/edge-architecture.md) | M11 | Edge certificado nos limites |
| [ ] | RM-M13 | M13 — Cabo Verde | ADIADO | [docs/00-product/scope.md](docs/00-product/scope.md) | M11 + catálogo CV | Expansão CV |
