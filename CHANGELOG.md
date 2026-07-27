# Changelog

## 0.2.55-draft — 2026-07-27

- Backoffice: **RM-AGTPREP-003** — UI owner-only `/admin/ui/authority-profiles` (criar/editar metadados + readiness sanitizado: fingerprint/validade/alg/key-id/timestamps); OpenAPI admin `0.1.9-draft` (nota UI); ≠ AGT real / plaintext / deploy.

## 0.2.54-draft — 2026-07-27

- Backoffice: **RM-AGTPREP-002** — `AuthorityProfile` (migração `0007`, Admin API `/admin/v1/authority-profiles`, OpenAPI `0.1.8-draft`); operações FE catalogadas; `pending_external`; `external_verified=false`; ≠ AGT real / segredos / deploy.

## 0.2.53-draft — 2026-07-27

- Backoffice: **RM-AGTPREP-001** / **DEC-BO-004** — separação config pública `AuthorityProfile` vs SecAdm/`SecretStore`; roadmap `RM-AGTPREP-002`…`007`; ≠ AGT real / deploy / segredos.

## 0.2.52-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-024** — loader GeneralLedgerEntries (`GLEntriesLedgerSource`/`SyntheticGLEntriesLedger` + `MapGLEntriesLedgerToExport`); Store → `ErrUnsupported` (GAP-SAFT-GLE-PERSIST); ≠ AO-* / AGT / deploy.

## 0.2.51-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-023** — tipagem `GeneralLedgerEntries`/`Journal`/`Transaction` (TransactionType; Lines Debit+Credit) + export opcional (período/allowlist); fixture XSD; ≠ AO-* / AGT / deploy.

## 0.2.50-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-022** — UI admin read-only `/admin/ui/saft` (metadados estruturais + GAPs persistência; sem download XML/NIF/tokens); ≠ AO-* / AGT / deploy.

## 0.2.49-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-021** — loader WorkingDocuments (`WorkingLedgerSource`/`SyntheticWorkingLedger` + `MapWorkingLedgerToExport`); Store → `ErrUnsupported` (GAP-SAFT-WRK-PERSIST); ≠ AO-* / AGT / deploy.

## 0.2.48-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-020** — loader MovementOfGoods (`MovementLedgerSource`/`SyntheticMovementLedger` + `MapMovementLedgerToExport`); Store → `ErrUnsupported` (GAP-SAFT-MOV-PERSIST); ≠ AO-* / AGT / deploy.

## 0.2.47-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-019** — loader PurchaseInvoices (`PurchaseLedgerSource`/`SyntheticPurchaseLedger` + `MapPurchaseLedgerToExport`); Store → `ErrUnsupported` (GAP-SAFT-PUR-PERSIST); sem Line; ≠ AO-* / AGT / deploy.

## 0.2.46-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-018** — loader Payments (`PaymentLedgerSource`/`SyntheticPaymentLedger` + `MapPaymentsLedgerToExport`); `Store` → `ErrUnsupported` (GAP-SAFT-PAY-PERSIST); omissões sem dados sensíveis; ≠ AO-* / AGT / deploy.

## 0.2.45-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-017** — tipagem `GeneralLedgerAccounts`/`Account` (GroupingCategory; sem NumberOfEntries) + integração opcional no export; fixture XSD; ≠ AO-* / AGT / deploy.

## 0.2.44-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-016** — `BuildIncrementalExport` com `WorkingDocuments` (filtro período, allowlist WorkType, refs Customer/Product); 5 grupos L3 no export estrutural; ≠ AO-* / AGT / deploy.

## 0.2.43-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-015** — `BuildIncrementalExport` com `MovementOfGoods` (filtro período, allowlist MovementType, refs Customer/Supplier/Product); ≠ AO-* / AGT / deploy.

## 0.2.42-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-014** — `BuildIncrementalExport` com `PurchaseInvoices` (sem Line; Supplier; allowlist PurchaseType; período); ≠ AO-* / AGT / deploy.

## 0.2.41-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-013** — `BuildIncrementalExport` com `Payments` (filtro período, allowlist PaymentType, Customer refs, XML determinístico); ≠ AO-* / AGT / deploy.

## 0.2.40-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-012** — `BuildIncrementalExport` com `TaxTable` opcional (validação + refs TaxType/TaxCode das linhas; sem inventar taxas); ≠ AO-* / AGT / deploy.

## 0.2.39-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-011** — tipagem `TaxTable`/`TaxTableEntry` (TaxType IVA|IS|NS; TaxPercentage XOR TaxAmount; caps fail-closed); fixture XSD; ≠ AO-* / AGT / deploy.

## 0.2.38-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-010** — `ListSealedSalesForSAFT` (persistência→`SalesLedgerRecord`); escala qty 1/10000; sem inventar Hash/ProductCode; ≠ AGT/AO-*/deploy.

## 0.2.37-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-009** — mapeamento livro→`BuildIncrementalExport` (`MapSalesLedgerToExport`: scope/período, omissões, sem inventar Hash/imposto); ≠ AGT/AO-*/deploy.

## 0.2.36-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-008** — tipagem `PurchaseInvoices`/`PurchaseInvoice` + `Supplier` (XSD: sem Line; sem TotalDebit/Credit no contentor); fixture XSD; ≠ AO-* / AGT / deploy.

## 0.2.35-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-007** — tipagem `Payments`/`Payment`/`Line` (+ PaymentTax/PaymentMethod; enums XSD); caps fail-closed; fixture XSD; ≠ AO-* / AGT / deploy.

## 0.2.34-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-006** — tipagem `WorkingDocuments`/`WorkDocument`/`Line` (enums WorkType/Status); fixture XSD; WorkType semantics pending; ≠ AO-* / AGT / deploy.

## 0.2.33-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-005** — tipagem `MovementOfGoods`/`StockMovement`/`Line` (+ MovementTax; enums XSD); fixture sintética XSD; MovementType semantics pending; ≠ AO-* / AGT / deploy.

## 0.2.32-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-004** — export incremental estrutural (`BuildIncrementalExport`: período, allowlist InvoiceType, SHA-256 do XML, XSD opcional); ≠ livro fiscal/AGT/AO-* / deploy.

## 0.2.31-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-003** — tipagem `SalesInvoices`/`Invoice`/`Line` (+ Money2/datas; 5 grupos L3); XML determinístico; fixture sintética validada contra XSD embutido; Hash/InvoiceType semantics pendentes; ≠ conformidade legal / AO-* / AGT / deploy.

## 0.2.30-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-002** — inventário Header + stubs SourceDocuments/MasterFiles a partir do XSD (`pending_validation`); `NewSalesSkeleton`; ≠ export válido / AO-* / AGT / deploy.
- Admin ops: **RM-BO-007** — observabilidade backoffice (`internal/adminobs`): `GET /admin/v1/health|ready`, `GET /admin/v1/ops/metrics`; correlação `X-Request-Id`; logs/métricas sanitizados (sem tokens/cookies/DSN/NIF/subject); cardinalidade limitada; OpenAPI admin `0.1.7-draft`; docs operação/segurança; ≠ SecAdm plaintext / deploy.

## 0.2.29-draft — 2026-07-27

- SAF-T AO: **RM-SAFT-001** — fundação estrutural `internal/saftao` + embed XSD (`source_id` AO-SAFT-XSD-1.01_01, `pending_validation`); skeleton AuditFile + SHA-256; ≠ conformidade/certificação/AO-* / deploy.
- Backoffice UI: **RM-UI-005** — sessão browser opaca (`fiscal_admin_session` HttpOnly+SameSite+Secure fora de development); `POST /admin/ui/auth/session` (Bearer→cookie); logout CSRF; sem JWT/localStorage; redirect IdP interactivo ainda não ligado.
- Admin auth: **RM-BO-006** / **DEC-BO-003** — adaptador OIDC/JWT provider-neutral (`FISCAL_ADMIN_AUTH_MODE=oidc_jwt`); JWKS https; iss/aud exactos; alg allowlist; role map; owner subject allowlist; fail-closed; OpenAPI admin `0.1.6-draft`; JWKS local só em testes; ≠ fornecedor IdP / deploy.

## 0.2.28-draft — 2026-07-26

- Backoffice UI: **RM-UI-004** / **RM-ARCH-005** / **RM-M7** — SecAdm SSR owner-only (só metadados sanitizados); M7 UI mínima concluída; sem plaintext/Reveal/deploy.
- Backoffice UI: **RM-UI-003** — páginas SSR submissões/reconciliação + auditoria admin; sem corpos secretos/deploy.
- Backoffice UI: **RM-UI-002** — formulários SSR + CSRF (contribuintes/estabelecimentos/bindings/séries); sem JS/segredos/deploy.
- Backoffice UI: **RM-UI-001** — SSR Go em `/admin/ui/` (shell + dashboard read-only); CSP sem scripts; auth `adminauth` fail-closed + cookie dev opcional; sem SPA/IdP/segredos/deploy.
- Admin API: **RM-BO-005** — GET listagens taxpayers/establishments/scope-bindings + PATCH status; OpenAPI admin `0.1.5-draft`; audit; sem segredos/AGT/deploy.
- Segurança: **RM-SECADM-003** — HTTP SecAdm Put/Rotate/Revoke (`/admin/v1/secadm/*`); owner role+subject; audit append-only; resposta só metadados; OpenAPI admin `0.1.4-draft`; ≠ AGT real/deploy.
- Admin API: path canónico OpenAPI → [`specs/admin/openapi.yaml`](specs/admin/openapi.yaml) (DEC-BO-002 / RM-BO-001); deixa de usar `specs/openapi-admin/`; série desde `0.1.0-draft`; ≠ POS OpenAPI.
- Admin API: **RM-BO-004** — matriz RBAC tipada (`adminauth.Allows`); SecAdm write só `owner`; MFA adiado até IdP; sem ACL `secret.reveal`; OpenAPI admin `0.1.3-draft`.
- Admin API: **RM-BO-003** — GET `/admin/v1/ops/submissions`, `/audit-events`, `/secret-refs/metadata` (só metadados); OpenAPI admin `0.1.2-draft`; sem plaintext/AGT/deploy.
- Admin API: **RM-BO-002** — PATCH scope-bindings (séries/timezone/ambiente/status metadados); validação IANA; OpenAPI admin `0.1.1-draft`; audit append-only; sem segredos/AGT/deploy.
- Admin API: **DEC-BO-002** / **RM-BO-001** — `/admin/v1` + OpenAPI admin `0.1.0-draft`; RBAC injectável fail-closed (`adminauth`); auditoria append-only (`0006`); cadastros HTTP sobre `adminregistry`; ≠ POS OpenAPI; sem IdP/credenciais AGT/deploy.
- Segurança: **RM-SECADM-001** — gate owner-only (`internal/secadm`) sobre AdminView write-only; operadores comuns falham fechado; sem MFA/UI/HTTP ainda.
- Segurança: **RM-SECADM-002** — contrato write-only `internal/secretstore` (simulator AES-GCM em memória); metadados sanitizados; `AdminRevealDenied`; isolamento HML≠PRD; ≠ KMS/AGT real.
- Backoffice: **RM-BO-010** — fundação `taxpayers` / `establishments` / `scope_bindings` (`internal/adminregistry`, migration `0005`, `ExpectedVersion=5`); plano A sem segredos; sem UI/deploy.
- Arquitectura/produto: **DEC-BO-001** / **RM-ARCH-006** — backoffice funcional (plano A) vs zona admin integração/segredos owner write-only (plano B); metadados sanitizados; IDs `RM-BO-010`, `RM-SECADM-001`/`002`; sem credenciais reais/deploy.
- Persistência: **RM-TX-010** — VS-T09: ledger `authority_processing` antes do Submit (simulador lento); unavailable reverte a `sealed_locally`; ≠ AGT oficial.
- Config: **RM-TX-009** — `FISCAL_AUTHORITY` (`simulator` default; `agt-hml`/`agt-prd` reservados fail-closed); ≠ HML/PRD AGT; sem credenciais/deploy.
- Persistência: **RM-TX-008** — semântica at-least-once/dedup/reconciliação do outbox→simulador documentada; VS-T12 (reclaim `in_flight` + `authority_outcome_unknown`); ≠ AGT oficial / exactly-once.
- Docs/CI: «Estado revisto em» = `2026-07-26`; `verify_roadmap` rejeita alterações materiais ao ROADMAP com data desactualizada (diff vs base Git).
- Persistência/crypto: **RM-TX-007** — adaptador JWS RS256 com RSA efémera (`internal/fiscaljws`) no caminho outbox→simulador; envelope técnico; privado nunca persistido; **não** certificado; **≠** FE-RNG/`RM-FE-002`; sem credenciais/deploy.
- Persistência: **RM-TX-006** — migration `0004` (`authority_attempts`/`authority_responses`); worker outbox + simulador AGT interno (VS-T08/10/11); `ExpectedVersion=4`; simulador ≠ HML AGT; sem credenciais/deploy/servidor.
- Motor/produto: **DEC-REG-003** decidida — defaults do slice `invoice`/`credit_note` → `bwb.ao.vendas.ft`/`nc` (`activo=on`); restante seed `off`; registo fail-closed [`internal/doctype`](internal/doctype/); sem AO-* confirmados; sem deploy/servidor.
- Domínio/API: alinhamento a DEC-PROD-004/008/009/011 (enrollment, autoridade módulo, estados, Edge writer); sem declarar conformidade AGT.
- Compliance: dependências AGT ([`agt-dependencies.md`](docs/01-compliance/agt-dependencies.md)) + **DEC-DEL-002** — credenciais/confirmação externa = `BLOQUEADO_EXTERNO`/`ADIADO` **sem** travar catálogo, domínio, simulador, contratos, persistência nem testes; sem inventar respostas AGT nem declarar conformidade; `RM-FE-001`/`RM-CERT-001` → ADIADO (M6/M9).
- Produto: **DEC-PROD-001**–**015** (+ **DEC-OPS-001**) — modelo completo SAF-T/FE (5 grupos L3); esquema mínimo + seed [`DOCUMENT-CATALOG-RM-REQ-001.md`](compliance/derived/requirements/DOCUMENT-CATALOG-RM-REQ-001.md) + verifier CI; faseamento via **DEC-REG-003**; sem AO-* confirmados.
- Compliance RM-REQ-001: matriz de tipos — quatro camadas ortogonais L1 (documento legal) · L2 (tipo SAF-T) · L3 (estrutura SAF-T) · L4 (`documentType` FE); C-DOC-003 reformulado; sem bijecção; sem AO-* confirmados.
- Compliance RM-REQ-001: Citação H — docs FE HML (`registarFactura`/`solicitarSerie`/`listarSeries`/`listarFacturas`/`obterEstado`) + inventário `FE-RNG-*` extractado; AO-AGT-001 → `pending_validation`; AO-AGT-002 → `partial` (`requestID`); C-FE-001 (paths `/ws/` vs `/v1`); sem inventar códigos; sem AO-* confirmados.
- Compliance RM-REQ-001: Citação D expandida sobre `SAFTAO1.01_01.xsd` (`e9a938e1…`; p.ex. `AuditFile`, `InvoiceNo`, `InvoiceType`, `InvoiceStatus`, `References`, `Hash`/`HashControl`, `SAFTAOPaymentType`/`PaymentType`); AO-SAF-001/002 mantêm `pending_validation`; C-DOC-003 actualizado (`RC`/`RG` em Payments ≠ `InvoiceType`); sem AO-* confirmados.
- Compliance RM-REQ-001: DE 683/25 Art.1–6 + Anexos I–III + Tabelas 1–6 (Citação G @19164–19227); AO-TAX-001 → `partial` (`taxType` @19171 + Tabelas 2–6); SEQ-002/`solicitarSerie` @19183–19184; DOC-001 permanece `scaffold`; sem AO-* confirmados; sem inventar `FE-RNG-*`.
- Compliance RM-REQ-001: DE 74/19 + Rect. 10/19 (Citação F @1576–1584 / 1948–1949); C-SIGN-001 (SAF-T RSA ≠ JWS FE); OFF-002 → `partial`; DEC-REG-002 parcialmente mitigada; sem AO-* confirmados.
- Compliance RM-REQ-001: DP 71/25 Art.10 a)–j) @11908–11909 + excepções por tipo; SEQ-001/DOC-002/OFF-001 → `partial`; DOC-001 permanece `scaffold`; sem AO-* confirmados.
- Compliance RM-REQ-001: matriz de tipos documentais com citações DP 71/25 · DE 683/25 · FE HML · SAF-T XSD ([`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md)); conflitos C-DOC-001/002/003; AO-DOC-001 permanece `scaffold` (sem confirmação).
- Docs: dashboard `ROADMAP.md` alinhado a RM-SRC-004/RM-M2-C CONCLUÍDO (OCR `reviewed` ≠ AO-* confirmados).
- Compliance RM-SRC-004 / RM-M2-C: Rect. 10/19 **v2** (`b3db14e2…`, 3p) + DP 71/25 (`4931fd3c…`; citar 11902–11920) OCR `reviewed` no privado ([bwb-fiscal-sources-ao#3](https://github.com/storesace-cv/bwb-fiscal-sources-ao/pull/3) @ `c8a4e6e`); histórico Rect. incompleto `77b77f01…` preservado só em diagnóstico privado.
- Catálogo público: 21 fontes; `private_commit` alinhado a `c8a4e6e…`; verificador `EXPECTED_COLLECTION_COUNT=21` + page counts Rect=3 / DP71=21.
- Matriz provisória: admite Rect v2 + DP71; DOC/SEQ-001/TAX → `scaffold` (citação pendente); sem AO-* confirmados.
- ROADMAP: RM-SRC-004 e RM-M2-C → CONCLUÍDO; RM-REQ-001 permanece EM_CURSO; GAP-002 parcialmente mitigado.

## 0.2.27-draft — 2026-07-25

- Compliance RM-REQ-001: AO-SAF-001 — citação técnica do XSD ASSOFT (`AuditFile`, NS AO_1.01_01, sha256 `e9a938e1…`) mantendo `pending_validation`; sem AO-* confirmados.
- Compliance RM-REQ-001: AO-CRYPTO-001 → `partial` (`jwsDocumentSignature` @19168 + RS256 no snapshot FE `pending_validation`); JWS FE ≠ SAF-T; encadeamento não citado; sem AO-* confirmados; Rect.-dependent `blocked`.
- Compliance RM-REQ-001: AO-ID-001 → `partial` com citação DE 683/25 (`taxRegistrationNumber` @19166 / software @19167); lacuna estabelecimento/terminal explícita na linha; verificador exige lacuna na própria linha da tabela; sem AO-* confirmados; Rect.-dependent `blocked`.
- Compliance RM-REQ-001: AO-SEQ-002 → `partial` com citação DE 683/25 ART.4.º (gazeta 19164 / PDF p.2); critério do catálogo explicitamente não satisfeito só com ART.4; Rect.-dependent mantêm-se `blocked`; sem AO-* confirmados.
- Compliance RM-REQ-001: verificador endurecido (source_id/sha256/`Não confirmado` em AO-SEQ-002) + regressões contra desbloqueio Rect. e promoção indevida.
- Compliance RM-REQ-001: matriz provisória AO-* + verificador fail-closed (`verify_provisional_matrix.py`); linhas dependentes da Rect. 10/19 em `blocked`; sem requisitos confirmados.
- Compliance RM-SRC-004: DE 683/25 original correcto + OCR v2 `reviewed` no privado ([bwb-fiscal-sources-ao#2](https://github.com/storesace-cv/bwb-fiscal-sources-ao/pull/2)); Rect. 10/19 continua BLOQUEADA; RM-SRC-004/RM-M2-C fail-closed.
- Compliance RM-SRC-004 / RM-M2-C: pipeline OCR privado iniciado ([bwb-fiscal-sources-ao#2](https://github.com/storesace-cv/bwb-fiscal-sources-ao/pull/2)); **fail-closed** — itens `BLOQUEADOS` até os 3 diplomas correctos estarem `reviewed`.
- Achado: originais arquivados de Rect. 10/19 (`77b77f01…`) e DE 683/25 (`59a48189…`) **não** correspondem aos diplomas; derivados `rejected` removidos do catálogo público (só diagnóstico privado, não KB). DE 74/19 OCR `reviewed` permanece auxiliar.
- Sem merge; sem HTML/XSD/ZIP/runtime; sem `AO-*` novos.

## 0.2.26-draft — 2026-07-25

- Compliance SRC-B2: XSD `SAFTAO1.01_01.xsd` + LICENSE MIT ASSOFT + NOTICE/README/SHA256SUMS em `compliance/saft-ao/schemas/`; catálogo `AO-SAFT-XSD-1.01_01` → `git_public` / `pending_validation`; ZIP permanece `local_only`.
- Verificador/testes: invariantes genéricas de `versioned_path`/`git_public`; suite `tests/compliance/run-verify-catalog-tests.sh` na CI.
- CI/`tests/deploy`: `git diff --check` exclui o XSD upstream imutável (trailing whitespace ASSOFT por desenho).
- Docs activos + ROADMAP (`RM-SRC-005` CONCLUÍDO); GAP-004 parcialmente mitigado (sem confirmação AGT). Sem OCR, runtime, servidor ou alterações ao repo privado.

## 0.2.25-draft — 2026-07-25

- Compliance SRC-B1: repositório privado `storesace-cv/bwb-fiscal-sources-ao` inicializado; originais B1 (3 PDFs DR + 13 HTML FE + proveniência) sincronizados byte-for-byte com checksums; OCR **não** iniciado; XSD/ZIP permanecem SRC-B2.
- Docs: `ROADMAP.md` (RM-SRC-003/RM-M2-B CONCLUÍDOS; OCR desbloqueado mas PENDENTE); catálogo `sources.yaml` com `storage=private_sync` + ponteiros de commit/path privados; `docs/01-compliance/sources.md` actualizado.
- Sem runtime, migrations, OpenAPI, servidor ou dependência de `local/` em build/CI.

## 0.2.24-draft — 2026-07-25

- Governance: ruleset activo `Protect main and require project checks` (ID `19731202`) em `main` — PR obrigatório, checks `go-checks` / `predeploy-pg16-real` / `GitGuardian Security Checks`, approvals=0, bypass vazio, cubic opcional; RM-GOV-002 CONCLUÍDO.
- Docs: `ROADMAP.md` actualizado com evidência e configuração sanitizada do ruleset; verificador/testes/rule alinhados ao estado com ruleset activo. Sem runtime, OCR, servidor, `local/` ou branch protection clássica adicional.

## 0.2.23-draft — 2026-07-25

- Docs: `ROADMAP.md` canónico de estado/progresso; apontador em `docs/06-delivery/implementation-roadmap.md`; README/AGENTS/rules/template PR; verificador `scripts/verify_roadmap.py` + testes `tests/docs/run-verify-roadmap-tests.sh` na CI.
- Ops docs activos: `deployment.md` e `staging-runbook.md` alinhados ao sandbox `credential_store` / S3C2 CONFIRMED (deny-all = rollback); `local-dev.md` e `first-vertical-slice.md` com OpenAPI `0.1.6-draft`.
- Sem runtime, OCR, servidor, `local/` ou settings GitHub (RM-GOV-002 continua bloqueado).

## 0.2.22-draft — 2026-07-25

- CI (MAINT-CI-001): `actions/checkout@v4` → `@v7` e `actions/setup-go@v5` → `@v7` para eliminar as anotações de runtime Node.js 20 nos jobs CI. Sem alteração de código de aplicação.

## 0.2.21-draft — 2026-07-25

- Compliance B0 (docs only): auditoria cross-project SAF-T AO [`AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md`](docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md) contra `bwb-efatura-docs@629edde…`; matriz de reutilização; paridade XSD `e9a938e1…`; hipótese de namespaces só como inferência operacional; zero código/fixtures/dados copiados; `my-bwb-app` intacto. B1 continua bloqueado pela criação de `bwb-fiscal-sources-ao`.

## 0.2.20-draft — 2026-07-25

- Compliance PR A (metadados): catálogo versionado [`compliance/catalog/sources.yaml`](compliance/catalog/sources.yaml) da recolha `arquivo_fiscal_ao` (2026-07-25, 20 fontes); POLICY/README; schema JSON + validação PyYAML/jsonschema pinados; CI sem depender de `local/`; PDFs DR registados como image-only (12/2/16 p.) com OCR obrigatório no PR B; GAP-012 parcialmente fechado; GAP-001/002/004/005 abertos. Sem binários, sem OCR, sem novos `AO-*`.

## 0.2.19-draft — 2026-07-25

- Ops B2/B3 (docs): validação sandbox do gate pré-deploy; relatório [b2-predeploy-pg-dump-gate-report.md](docs/07-operations/b2-predeploy-pg-dump-gate-report.md); INC-S4-003 **RESOLVIDO**; INC-B2-001 aberto (prova poisoned não ensaiada); runbook: remediação humana de locks poisoned/stale/corrupt, sem auto-remoção. Sem novo SSH/deploy neste incremento documental.

## 0.2.18-draft — 2026-07-24

- Pre-deploy `pg_dump` gate (B1, repo only): helper `deploy-lock-acquire`/`deploy-lock-release`/`pre-deploy-pg-backup`; parser URI fechado + `pg_service.conf`/`PGPASSFILE`; instalação durável anti-TOCTOU; updater live com lock antes de upload e mutações só após `deploy_allowed=true`; `createdb`/`dropdb` via OS `postgres` sem `CREATEDB`/`SUPERUSER` em `fiscal_migrate`; mocks + CI PostgreSQL 16 real; runbook. Sem acesso ao servidor, sem B2/B3, sem Ready/merge.

## 0.2.17-draft — 2026-07-24

- S4 operacional (sandbox): validação do kit POS **APROVADA** (9/9); `rate_429` com 22×201 + 8×429 e zero 5xx/transporte; `fiscal-admin --output-file` raw 52 bytes sem CR/LF; release activa `5d7c14b…`; Nginx `10r/s`/`burst=20` e schema 3 inalterados; histórico da primeira reprovação preservado no relatório ops.
- Ops: fragmento versionado `deploy/sudoers/bwb-fiscal-deploy` fixo em `bwb-deploy` → só o helper fechado (sem placeholder `DEPLOY_USER`); runbook instala o ficheiro directamente; testes anti-regressão no suite de deploy.

## 0.2.16-draft — 2026-07-23

- S4 follow-up (repo only): `rate_429` do kit POS passa de ondas de cinco para **burst sincronizado de 30** (ready + gate local); `fiscal-admin --output-file` grava token raw de 52 bytes **sem** newline; mocks de deploy alinhados. Sem alteração de Nginx (`10r/s`/`burst=20`), OpenAPI, runtime `fiscal-api` ou schema. Validação real do sandbox permanece pendente pós-merge.

## 0.2.15-draft — 2026-07-23

- SSH ops (docs/snippet only): remoção definitiva do UFW LIMIT em TCP/22 → ALLOW 22; autenticação exclusivamente por chave; root só por chave (`PermitRootLogin prohibit-password`); `PasswordAuthentication` e `KbdInteractiveAuthentication` desativadas; `IdentitiesOnly` no snippet cliente; multiplexing apenas como otimização. Release/runtime/Nginx/API inalterados. Incidente S4 `rate_429` permanece aberto e fora deste PR.

## 0.2.14-draft — 2026-07-23

- S4 (repo only): kit e documentação de integração POS/software houses — quickstart, contrato, onboarding, checklist, publicação; OpenAPI `0.1.6-draft` (429 sem Problem/`Retry-After` garantidos; auth pública “Bearer credential issued by BWB for the sandbox”); `scripts/integration/pos-sandbox-kit.sh` com token fora de argv, URL sandbox exacta, fixtures sintéticas por run CSPRNG, testes mock HTTP+curl na CI. Sem SSH/deploy/runtime/Nginx/BD; sandbox real fica para validação operacional pós-merge.
- S4 hardening (Draft PR): validação semântica de respostas; rate_429 com exit/http_code separados; token Base64URL exacto com limite de leitura; env de teste só com loopback; harness INT/TERM com TMPDIR isolado (sem KILL/find global); guard OpenAPI 429 via bundle Redocly+jq; `git diff --check` sem mascarar falhas.
- S4 follow-up: CreateDocumentResponse sem `request_id` (additionalProperties:false); cleanup de filhos com prazo + KILL só nos PIDs tracked; teste com filho que ignora TERM.

## 0.2.13-draft — 2026-07-23

- S3C2 follow-up (I1): probe pós-reload com retry/deadline curto (401 só transitório; exige 403; 5xx/TLS/timeout → falha); `nginx-deny-all` e `nginx-open-rollback-fire` usam fail-closed rigoroso (`deny_restored` / `emergency_nginx_stop` comprovado / `emergency_stop_failed` CRITICAL); nunca `state=armed` com timer inactive após o fire. Relatório promoção sandbox ROLLED_BACK. Sem deploy/confirm/nova promoção no host.

## 0.2.12-draft — 2026-07-22

- S3C2 (repo only): Nginx open canónico (`rate=10r/s`, `burst=20`, `limit_req_status 429`, `location = /v1/documents`), HSTS, ACME-safe redirect, deny-all rollback, helper arm/confirm/deny + timer 5 min + boot-recovery `Before=nginx` + drop-in `nginx.service` `Requires=`/`After=` boot-recovery; flock; fail-closed com `deny_restored`, `emergency_nginx_stop` (só após `is-active != active`) ou `emergency_stop_failed` (CRITICAL); reload falhado no arm usa o mesmo fail-closed; confirm grava `confirmed` antes de cancelar o timer. Sem deploy/promoção no host.
- Hardening pré-merge (#16): prova de emergency stop; drop-in obrigatório bloqueia nginx se recovery armed falhar; reload ambíguo → fail-closed rigoroso.

## 0.2.11-draft — 2026-07-22

- S3C-tooling: `internal/buildinfo` + Health `revision` (required, SHA40\|dev); `fiscal-api version`; release ldflags + verificação `revision == COMMIT`; binário Go `fiscal-sandbox-measure` (perfis fechados sustained/burst/replay) substitui o medidor shell; helper `admin-sandbox-measure <sha> <profile>`; OpenAPI `0.1.5-draft`; runbook S3C1. Sem deploy/S3C1/S3C2/abertura pública.
- Hardening measure (Draft PR): pacing sustained sem catch-up; thresholds completos + `passed`/`failure_codes`; `attempted`/`http_responses`/`transport_errors`; replay tipado estrito; `revision=dev` só em `FISCAL_ENV=development`; token dir/ficheiro 0600 sem symlink; latência via Clock injectado.
- Fix sustained start-slot pacing e token open TOCTOU (`O_NOFOLLOW` + Fstat).

## 0.2.10-draft — 2026-07-22

- Pós-S3B (artefacto / Draft PR): Issue/Rotate/Revoke em PostgreSQL usam `pg_advisory_xact_lock` por `scope_id` + `SELECT` (sem `FOR UPDATE` nem UPDATE em `fiscal.scopes`); activate do helper com GNU `mv -T` + verificação `current-sha`; Nginx measure/candidate com `limit_req_status 429`; testes concorrentes estritos e regressão activate. Sem deploy/Ready/S3C neste incremento.

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
