# AUD-VENDOR-NETBO — Reutilização NET-BO (`my-bwb-app`)

| Campo | Valor |
|---|---|
| Audit ID | AUD-VENDOR-NETBO-REUSE-MY-BWB-APP |
| Data | 2026-07-28 |
| Repositório alvo | `storesace-cv/bwb-modulo-fiscal` (público) |
| Fonte consultada | checkout local `my-bwb-app` |
| Commit analisado | `629edde439b61aab1994e839415f19e086ce30a1` |
| Método | `git show` / leitura read-only de `core/netbo_client.py` e módulos relacionados |
| Working tree mutações | **nenhuma** (`my-bwb-app` não modificado) |
| Código copiado | **nenhum** |
| Conector live | **não** neste incremento |

## Objectivo

Inventariar a integração NET-BO madura na app interna como **pista de engenharia** para um futuro adapter no módulo fiscal — sem tornar NET-BO fonte normativa AGT e sem transplantar Python.

## Resumo executivo

1. Cliente central: `core/netbo_client.py` — discovery `companies.api.net-bo.com`, auth login/password **ou** API token, retries/timeouts, cache de token ~1h.
2. Superfície madura: catálogo (products/units/stores/suppliers), encomendas, despesas, emails operacionais, adapters de contabilidade — **fora** do núcleo de emissão AGT.
3. Auth doc/cliente: `passwd` em query de `/api/auth`; token em header `token:` (login) ou query (api_token). Risco de fugas em logs/proxies.
4. Endpoints de criação de documentos NET-BO (PDF V2) **não** devem ser portados como autoridade fiscal AO.
5. Vendor técnico ≠ AGT; **zero** `AO-*` derivados desta auditoria.
6. Paridade com catálogo: PDF V2 `09bb09f8…` em `VENDOR-NETBO-API-V2` (privado `a889ef6…`).

## Matriz de reutilização

| Capacidade | Origem @ `629edde` | Uso futuro módulo | Ação |
|---|---|---|---|
| Server discovery | `NetBoClient` → `/companies/detailed` | Adapter HTTPS | `port_with_tests` (conceito Go) |
| Auth dual (login / api_token) + cache | `auth_token` | Segredos + redacção | `port_with_tests` |
| Retry/timeout/backoff | construtor + HTTPAdapter | Cliente HTTP | `port_with_tests` |
| Listagens products/units/stores | `/api/tables/*` + scripts probe | Sync catálogo ops | `reference_only` |
| Document types helpers | `core/netbo_document_types.py` | Mapas vendor | `reference_only` |
| Encomendas / despesas / email packs | `api/netbo_*`, templates | Fora do módulo fiscal | `do_not_port` |
| Emissão `create_invoice` (PDF) | doc vendor; não autoridade AO | — | `do_not_port` |
| SAF-T export NET-BO | PDF `import_export/saft_*` | ≠ SAF-T AO | `do_not_port` |

## Segurança / observabilidade

- Nunca logar `passwd`, tokens completos, NIFs ou payloads de documentos.
- Preferir `api_token` via secret store a passwords em query quando o fornecedor o permitir.
- Scripts `check_netbo_token_logs.sh` na app são pista operacional — não copiar sem revisão.

## Relação com fontes

- Normativo AO: [`compliance/catalog/sources.yaml`](../../../compliance/catalog/sources.yaml)
- Vendor: [`compliance/catalog/vendor-integrations.yaml`](../../../compliance/catalog/vendor-integrations.yaml)
- Matriz PDF: [`../vendor-integrations/NETBO-CAPABILITY-MATRIX.md`](../vendor-integrations/NETBO-CAPABILITY-MATRIX.md)

## Fora de âmbito

Implementação de conector, alteração a `my-bwb-app`, promoção a `AO-*`, cópia de fixtures/credenciais.
