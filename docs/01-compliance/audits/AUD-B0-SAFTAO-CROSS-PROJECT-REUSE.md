# AUD-B0 — Auditoria de reutilização SAF-T AO (cross-project)

| Campo | Valor |
|---|---|
| Audit ID | AUD-B0-SAFTAO-CROSS-PROJECT-REUSE |
| Data | 2026-07-25 |
| Repositório alvo | `storesace-cv/bwb-modulo-fiscal` (público) |
| Fonte consultada | checkout local `my-bwb-app` → remoto `storesace-cv/bwb-efatura-docs` (privado) |
| Commit analisado | `629edde439b61aab1994e839415f19e086ce30a1` |
| Método | apenas `git show <commit>:<path>` / `git ls-tree -r <commit>` |
| Working tree da fonte | **não** utilizado; alterações locais não relacionadas ignoradas |
| Código/fixtures copiados | **nenhum** |
| Servidor / OCR / repo privado de fontes | **não** neste PR |

## Objectivo

Inventariar capacidades SAF-T AO existentes na aplicação privada para orientar, com segurança, os PRs B1–D do módulo fiscal — **sem** tornar a app numa fonte normativa e **sem** transplantar código Python.

## Resumo executivo

1. O XSD `SAFTAO1.01_01.xsd` no commit analisado tem SHA-256 **idêntico** ao registado no catálogo e no arquivo fiscal local (`e9a938e1…`).
2. O repositório da app **não** tem LICENSE de topo que autorize cópia automática para o Git público; a secção «Licença» do README está por definir.
3. Capacidades reutilizáveis como **conceito** (validação XSD, ZIP seguro, proveniência, lifecycle imutável) devem ser portadas via testes/vetores em Go — não por cópia linha a linha.
4. Autofix, limpeza de namespaces, extractors/stats e regras de negócio da app **não** são requisitos AGT sem citação oficial.
5. A hipótese sobre namespaces redundantes vs rejeição AGT permanece **inferência operacional** (ver secção dedicada).
6. Fixtures de teste parecem sintéticas; ficam `pending_validation` e **não** são copiadas neste PR.
7. B1 continua bloqueado apenas pela decisão de criar `storesace-cv/bwb-fiscal-sources-ao` (não reutilizar `bwb-efatura-docs` para PDFs/OCR/HTML).

## Roadmap formal

| Fase | Conteúdo | Estado |
|---|---|---|
| A | Governação + catálogo de fontes | Concluído (`714ed19…`) |
| **B0** | Esta auditoria (docs only) | Este PR |
| B1 | Repo privado `bwb-fiscal-sources-ao` + PDFs + OCR + HTML | Bloqueado (decisão humana) |
| B2 | XSD + LICENSE/NOTICE MIT ASSOFT + inventários FE no Git público | Após B1 |
| C | Requisitos `AO-*` + rastreabilidade (fontes oficiais; B0 só como pista) | Após B2 |
| D | Implementação/testes SAF-T com vetores aprovados | Após C / matriz |

A app privada e o futuro repo de fontes **não** são fontes normativas. Fontes normativas: Diário da República, documentação oficial AGT/MINFIN, XSD ASSOFT com estado `pending_validation` até confirmação AGT. Ver [compliance/POLICY.md](../../../compliance/POLICY.md) e [catalog/sources.yaml](../../../compliance/catalog/sources.yaml).

---

## Paridade do XSD (prova de hash)

| Origem | Path / registo | SHA-256 |
|---|---|---|
| Catálogo módulo fiscal | `AO-SAFT-XSD-1.01_01` em `compliance/catalog/sources.yaml` | `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631` |
| Arquivo fiscal local (consulta) | `local/.../02_saft_ao/SAFTAO1.01_01.xsd` | `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631` |
| App @ `629edde` | `saftao_core/schemas/SAFTAO1.01_01.xsd` | `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631` |
| Upstream ASSOFT (já registado no catálogo) | repositório público ASSOFT / ZIP da recolha | mesmo hash no XSD da recolha |

**Conclusão:** paridade bit-a-bit confirmada. **Este PR não copia o XSD.** A importação pública (B2) deve usar o artefacto do arquivo/ASSOFT com `LICENSE` MIT + `NOTICE`, citando a paridade com `629edde`. XSD válido localmente **não** prova aceitação pela AGT.

---

## Capacidades reutilizáveis (catálogo)

| Capacidade | Origem típica @ `629edde` | Uso futuro no módulo | Ação |
|---|---|---|---|
| Validação XSD | `saftao_core/validate.py` (+ XSD) | `xmllint`/validador isolado + testes (D) | `port_with_tests` |
| Ingestão ZIP segura | `api/integracao_manual/saft_ao/zip_loader.py` | Limites, 1 XML, anti-traversal (D/futuro) | `port_with_tests` |
| Validação / diagnóstico read-only | `xml_diagnostics.py`, modos validate | Relatórios sem mutação | `port_with_tests` |
| Hashes / proveniência | `provenance.py` | Alinhar a catálogo POLICY | `port_with_tests` (conceito) |
| Versões imutáveis | lifecycle ADR + `versioning.py` | Modelo de domínio (ADR próprio) | `reference_only` → `port_with_tests` |
| Jobs append-only | lifecycle docs / serviço | Auditoria append-only | `reference_only` → `port_with_tests` |
| Diagnósticos de estrutura/NS | `xml_structure.py` | Testes/vetores | `reference_only` |
| Vetores de teste futuros | `tests/fixtures/saft*.xml` + testes | `specs/test-vectors/` após autorização | `pending_validation` |

---

## Itens não normativos (não viram `AO-*` sem fonte oficial)

| Item | Motivo |
|---|---|
| Regras de negócio em `validate.py` sem citação XSD/diploma | Engenharia da app |
| Autofixes (`autofix_apply.py`) | Mutação automática; proibida até requisito + teste |
| Arredondamentos/`ROUND_HALF_UP` específicos da app | Precisão deve vir de regra oficial |
| Limpeza / normalização de namespaces redundantes | Hipótese operacional AGT |
| Extractors / tax_stats / price_stats / Excel compare | Analytics / integração app |
| Estados UI, routes HTTP, templates, migrations Postgres da app | Acoplados à app |
| Scripts oneoff / working tree | Fora do commit autorizado |

---

## Hipótese de namespaces (obrigatório: não normativa)

### Facto observado (sanitizado)

- Em investigação operacional documentada na app, um XML SAF-T AO **passava** validação XSD local e foi **rejeitado** pela AGT com mensagens relacionadas com identificadores de cliente / documentos de trabalho.
- O mesmo tipo de ficheiro apresentava declarações de namespace no elemento raiz que incluíam um prefixo redundante igual ao namespace por defeito, sem uso lexical correspondente desse prefixo em elementos.

**Não se registam aqui** NIF, nomes, IDs, paths de storage, hashes de ficheiros operacionais nem detalhes de ambiente de produção.

### Inferência operacional

É **compatível** com um cenário em que o validador AGT interpreta declarações `xmlns` redundantes de forma diferente do validador XSD local (lxml/`xmllint`).

### Estado

- **Não confirmada** por fonte oficial (diploma, XSD AGT autenticado, ou resposta escrita AGT).
- **Proibida** como requisito `AO-*`, como autofix, ou como regra de exportação no módulo fiscal até existir evidência oficial.
- Uso permitido: pista para testes de regressão **informativos** e para perguntas à AGT.

---

## Inventário por grupo

Método: `git ls-tree -r 629edde` filtrado a caminhos SAF-T AO. Lista de paths (sem conteúdo).

### Schema

| Path | Notas |
|---|---|
| `saftao_core/schemas/SAFTAO1.01_01.xsd` | Hash paritário; import B2 via ASSOFT/arquivo |

### Validação

| Path | Notas |
|---|---|
| `saftao_core/validate.py` | XSD + regras internas + Excel log |
| `tests/test_saftao_validate_readonly.py` | Casos read-only |
| `tests/test_saftao_validate_screen.py` | UI validate — não portar UI |

### Autofix

| Path | Notas |
|---|---|
| `saftao_core/autofix_apply.py` | Correcções automáticas estruturais/montantes |
| `tests/test_saftao_autofix_apply.py` | Cobertura autofix |

### Diagnóstico / proveniência / metadados

| Path | Notas |
|---|---|
| `saftao_core/xml_diagnostics.py` | Diagnóstico read-only |
| `saftao_core/xml_structure.py` | Estrutura MasterFiles / NS |
| `saftao_core/provenance.py` | SHA-256, fingerprint |
| `saftao_core/metadata.py` | Identidade/metadados ficheiro |
| `saftao_core/operations.py` | Orquestração app — rejeitar cópia |
| `tests/test_saftao_provenance.py` / `test_saftao_metadata.py` | Testes |

### ZIP seguro

| Path | Notas |
|---|---|
| `api/integracao_manual/saft_ao/zip_loader.py` | max bytes, ZIP inválido, exactamente 1 XML, path traversal (`..`, absolutos) |
| `api/integracao_manual/saft_ao/constants.py` | Constantes (incl. limites) |
| `tests/test_saft_ao_zip_loader.py` | Casos de segurança |

### Extracção / análise

| Path | Notas |
|---|---|
| `api/integracao_manual/saft_ao/sales_extractor.py` | Extracção vendas |
| `api/integracao_manual/saft_ao/tax_stats.py` | Stats impostos |
| `api/integracao_manual/saft_ao/price_stats.py` | Stats preços |
| `api/integracao_manual/saft_ao/xml_common.py` | Helpers XML |
| `api/integracao_manual/saft_ao/product_group.py` | Grupos produto |
| `saftao_core/vendas_excel_saft_compare.py` | Comparação Excel — fora do núcleo certificado |
| Testes `tests/test_saft_ao_*.py` correspondentes | Checklist |

### Lifecycle / versioning

| Path | Notas |
|---|---|
| `docs/adr/ADR-0003-SAFTAO-LIFECYCLE.md` | ADR da **app** (não confundir com ADR-0003 country-packages do módulo) |
| `docs/SAFTAO_LIFECYCLE.md` | Documentação lifecycle |
| `saftao_core/lifecycle.py` | Records → versões → jobs → artefactos |
| `saftao_core/versioning.py` | Nomenclatura / versões |
| `saftao_core/status.py` | Estados calculados |
| Testes lifecycle/versioning/status | Checklist |
| Migrations / UI / routes HTTP da app | **Não reutilizar** |

### Testes (ficheiros)

`tests/test_saft_ao_price_stats.py`, `test_saft_ao_product_group.py`, `test_saft_ao_sales_extractor.py`, `test_saft_ao_xml_common.py`, `test_saft_ao_zip_loader.py`, `test_saftao_architecture_freeze.py`, `test_saftao_artifact_download.py`, `test_saftao_autofix_apply.py`, `test_saftao_customer_ns_prefix.py`, `test_saftao_lifecycle_migration.py`, `test_saftao_lifecycle_pipeline.py`, `test_saftao_lifecycle_service.py`, `test_saftao_metadata.py`, `test_saftao_provenance.py`, `test_saftao_report_totals.py`, `test_saftao_status.py`, `test_saftao_unused_root_namespaces.py`, `test_saftao_validate_readonly.py`, `test_saftao_validate_screen.py`, `test_saftao_versioning.py`.

### Fixtures (inventário sem payloads)

| Ficheiro | Finalidade aparente | Parece sintético? | Ação B0 |
|---|---|---|---|
| `tests/fixtures/saft_ao_venda_minimal.xml` | Venda mínima | Sim (marcadores «Test» / produto teste) | `pending_validation` — não copiar |
| `tests/fixtures/saftao_masterfiles_flat_customer_id.xml` | CustomerID flat em MasterFiles | Sim («Teste…», «BWB Test») | `pending_validation` |
| `tests/fixtures/saftao_masterfiles_ns_prefix_customer.xml` | Customer com prefixo NS | Sim | `pending_validation` |
| `tests/fixtures/saftao_totals_imputed_tax.xml` | Totais / imposto imputado | A confirmar | `pending_validation` |
| `tests/fixtures/saftao_unused_root_namespaces.xml` | Namespaces raiz não usados | Sim | `pending_validation` |
| `tests/fixtures/saftao_workdocument_missing_customer.xml` | WorkDocument sem customer | Sim | `pending_validation` |

**Política:** não incluir valores, NIF nem payloads neste relatório público. Cópia futura exige autorização explícita + confirmação de que são 100% sintéticas.

### Evidência operacional

| Path | Tratamento |
|---|---|
| `docs/audits/AUD-2026-07-14-SAFTAO-AGT-NAMESPACE-COMPAT.md` | Referência sanitizada apenas (secção namespaces). Contém dados de produção — **não** republicar conteúdo integral. |

### Explicitamente fora de âmbito (não reutilizar)

UI (`templates/`, `static/js/`), `api/saftao/routes/*`, migrations `086+` / `338_*`, scripts `oneoff`, pycache, working tree sujo, ficheiros reais de clientes.

---

## Matriz de reutilização

`source_repo` = `storesace-cv/bwb-efatura-docs`  
`source_commit` = `629edde439b61aab1994e839415f19e086ce30a1`  
`licença/autorização` da app = **sem LICENSE de topo**; cópia pública de código/fixtures exige autorização humana explícita. O XSD em si redistribui-se sob **MIT ASSOFT**, não sob licença da app.

| source_path | categoria | comportamento | fundamento oficial | confiança | risco dados | destino futuro | ação |
|---|---|---|---|---|---|---|---|
| `saftao_core/schemas/SAFTAO1.01_01.xsd` | schema | Schema 1.01_01 | XSD ASSOFT/MIT; ≠ aceite AGT | alta | baixo | `compliance/saft-ao/schemas/` (B2 via arquivo+LICENSE) | `reuse` |
| `saftao_core/validate.py` | validação | Validar XML vs XSD + regras app | XSD (parcial) | média | baixo | validador Go/`xmllint` (D) | `port_with_tests` |
| `saftao_core/autofix_apply.py` | autofix | Mutar XML antes de validar | nenhum oficial | baixa | baixo | — | `pending_validation` / default `reject` |
| `saftao_core/xml_structure.py` | diagnóstico | Estrutura MasterFiles / NS | parcial XSD | média | baixo | testes/vetores (D) | `reference_only` |
| `saftao_core/xml_diagnostics.py` | diagnóstico | Diagnóstico read-only | — | média | baixo | tooling diagnóstico | `reference_only` |
| `saftao_core/operations.py` | app | Orquestração | — | — | baixo | — | `reject` |
| `saftao_core/provenance.py` | proveniência | SHA-256 / fingerprint | engenharia | alta | baixo | alinhamento POLICY | `port_with_tests` |
| `saftao_core/metadata.py` | metadados | Identidade ficheiro | — | média | médio | — | `reference_only` |
| `api/.../zip_loader.py` | ZIP seguro | 1 XML, limites, traversal | engenharia/segurança | alta | baixo | ingestão (D) | `port_with_tests` |
| `sales_extractor.py` / `tax_stats.py` / `price_stats.py` / `xml_common.py` / `product_group.py` | extracção | Analytics | não normativo | média | médio | opcional fora do núcleo | `reference_only` |
| `vendas_excel_saft_compare.py` | extracção | Diff Excel | não normativo | média | médio | — | `reject` (núcleo) |
| `docs/adr/ADR-0003-SAFTAO-LIFECYCLE.md` / `SAFTAO_LIFECYCLE.md` | lifecycle | Record/versão/job/artefacto | engenharia | média | baixo | ADR próprio do módulo (nome distinto) | `reference_only` |
| `lifecycle.py` / `versioning.py` / `status.py` | lifecycle | Imutabilidade / estados | engenharia | média | baixo | domínio Go | `port_with_tests` |
| `tests/test_saftao_*.py` / `test_saft_ao_*.py` | testes | Casos | — | média | baixo | testes Go | `reference_only` → `port_with_tests` |
| `tests/fixtures/saft*.xml` | fixtures | XML de teste | XSD | média | a confirmar | `specs/test-vectors/` se autorizado | `pending_validation` |
| `docs/audits/AUD-2026-07-14-…NAMESPACE…` | evidência ops | NS vs rejeição AGT | **inferência** | baixa–média | **alto** (prod) | só citações sanitizadas | `reference_only` |

---

## Licença e autorização

| Artefacto | Situação |
|---|---|
| Código/fixtures `bwb-efatura-docs` | Sem LICENSE de topo em `629edde`; README «Licença — [Definir conforme política interna]». **Não** copiar para o público sem autorização explícita e rastreio. |
| XSD | MIT ASSOFT (upstream). Importar no B2 com LICENSE+NOTICE a partir do arquivo/ASSOFT; citar paridade de hash com este commit. |
| Experiência da app | Partilha de conhecimento pretendida pelo proprietário; **não** equivale a licença de redistribuição de código. |

---

## Controlo de integridade da fonte (working tree)

O working tree de `my-bwb-app` continha alterações locais não relacionadas no início desta auditoria. Este PR **não** as modificou. O relatório de entrega do PR confirma `git status --porcelain` inicial = final.

## Referências no módulo fiscal

- [compliance/POLICY.md](../../../compliance/POLICY.md)
- [compliance/catalog/sources.yaml](../../../compliance/catalog/sources.yaml) (`AO-SAFT-XSD-1.01_01`)
- [saft-ao.md](../saft-ao.md)
- [implementation-roadmap.md](../../06-delivery/implementation-roadmap.md)
