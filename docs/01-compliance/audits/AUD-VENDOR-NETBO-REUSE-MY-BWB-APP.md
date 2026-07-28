# AUD-VENDOR-NETBO — Reutilização NET-BO (`my-bwb-app`)

| Campo | Valor |
|---|---|
| Audit ID | AUD-VENDOR-NETBO-REUSE-MY-BWB-APP |
| Data | 2026-07-28 |
| Repositório alvo | `storesace-cv/bwb-modulo-fiscal` (público) |
| Fonte consultada | checkout local `my-bwb-app` |
| Commit analisado | `629edde439b61aab1994e839415f19e086ce30a1` |
| Método | `git show` / `git ls-tree` + leitura read-only (`core/netbo_client.py` e relacionados) |
| Working tree mutações | **nenhuma** (`my-bwb-app` não modificado) |
| Código copiado | **nenhum** |
| Conector live | **não** neste incremento |

## Objectivo

Inventariar a integração NET-BO madura na app interna como **pista de engenharia** para um futuro adapter no módulo fiscal — sem tornar NET-BO fonte normativa AGT e sem transplantar Python.

## Resumo executivo

1. Cliente central: `core/netbo_client.py` — discovery `companies.api.net-bo.com`, auth login/password **ou** API token, retries/timeouts, cache de token ~1h.
2. Superfície madura: catálogo (products/units/stores/suppliers), encomendas, despesas, emails, eFatura PT→BO — **fora** do núcleo de emissão AGT Angola.
3. Auth: doc PDF pede header `token:`; cliente usa sobretudo `Authorization: Bearer` e, para `api_token`, query string (por vezes com `login`/`passwd`). Validar com vendor antes de portar.
4. `passwd` em query de `/api/auth` (doc + client) — risco de fuga em proxies/CDN; **não** espelhar em APIs públicas do módulo.
5. Endpoints de criação de documentos NET-BO (`create_invoice` no PDF; `create_doc_reception` no export da app) **não** são autoridade fiscal AO.
6. Money como `float` no export eFatura→NET-BO — **`do_not_port`** (política decimal do módulo).
7. Vendor técnico ≠ AGT; **zero** `AO-*` derivados desta auditoria.
8. Paridade com catálogo: PDF V2 `09bb09f8…` em `VENDOR-NETBO-API-V2` (privado `a889ef6…`).

## Matriz de reutilização

| Capacidade | Origem @ `629edde` | Uso futuro módulo | Ação |
|---|---|---|---|
| Server discovery | `NetBoClient` → `/companies/detailed` | Adapter HTTPS | `port_with_tests` (conceito Go) |
| Auth dual + cache ~1h | `auth_token`, `api/netbo_sync.get_netbo_client` | Segredos + redacção | `port_with_tests` (padrão); **`do_not_port`** passwd/token em query |
| Credenciais at-rest encriptadas | `api/netbo_sync.py` | Secret store | `port_with_tests` (conceito) |
| Retry/timeout/backoff | HTTPAdapter / timeouts longos | Cliente HTTP | `port_with_tests` |
| Envelope HTTP 200 + campo `error` | `raise_if_netbo_error_envelope` | Tratamento de erros | `port_with_tests` |
| Catalog GET products/units/stores/suppliers/… | `core/netbo_client.py`, sync | Sync ops | `port_with_tests` (HTTP); espelho Postgres app = `reference_only` |
| List documents `GET /api/data/documents` | client + receptions | Leitura ops | `port_with_tests` |
| Document types estáticos | `core/netbo_document_types.py` | Taxonomia vendor | `reference_only` |
| `POST …/create_doc_reception` | `api/efatura_netbo_export.py` | eFatura PT→BO | `reference_only` / **≠** emissão AO |
| `create_invoice` / NC / receipt (PDF) | doc vendor; pouco/ausente no client | — | **`do_not_port`** como autoridade AO |
| Movements/orders / POS tree / custos/OCR | client + despesas | Fora do núcleo AO | `reference_only` / `do_not_port` |
| Money `float` no export | `api/efatura_netbo_export.py` | — | **`do_not_port`** |
| Probes / UI / emails / FT | scripts, templates, `api/ft_netbo_*` | — | **`do_not_port`** |
| SAF-T export NET-BO (PDF) | `import_export/saft_*` | ≠ SAF-T AO | **`do_not_port`** |

## Segurança / observabilidade

- Nunca logar `passwd`, tokens completos, NIFs ou payloads de documentos.
- Redacção no client é **parcial**: alguns ramos debug redigem `passwd`/`token`; erros de auth podem incluir excertos de `response.text`; export logger sanitiza chaves mas ainda pode gravar payloads documentais em debug.
- Preferir secret store; isolar auth com query string do desenho do módulo fiscal.
- `scripts/check_netbo_token_logs.sh` — pista de fuga histórica; não copiar sem revisão.

## Endpoints: client vs doc (sanitizado)

**No client (+ export):** discovery, `/api/auth`, tables de catálogo, `/api/data/documents`, movements/orders, alguns POS/reports/costs/helpers, v5 recipes/import_export, `create_doc_reception`.

**No PDF, pouco/ausentes no client:** `create_invoice` / NC / receipt / transfer, `sales_*`, várias `configurations/*`, parte de tables (`clients`, `users`, …).

## Relação com fontes

- Normativo AO: [`compliance/catalog/sources.yaml`](../../../compliance/catalog/sources.yaml)
- Vendor: [`compliance/catalog/vendor-integrations.yaml`](../../../compliance/catalog/vendor-integrations.yaml)
- Matriz PDF: [`../vendor-integrations/NETBO-CAPABILITY-MATRIX.md`](../vendor-integrations/NETBO-CAPABILITY-MATRIX.md)

## Fora de âmbito

Implementação de conector, alteração a `my-bwb-app`, promoção a `AO-*`, cópia de fixtures/credenciais.
