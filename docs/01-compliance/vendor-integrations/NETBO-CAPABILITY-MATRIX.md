# Matriz de capacidades — NET-BO API V2 (vendor)

| Campo | Valor |
|---|---|
| source_id | `VENDOR-NETBO-API-V2` |
| SHA-256 | `09bb09f8064ad49d98bf9cffc0a80eb7a2dbded5f4221c54e5ca19da82cb7c86` |
| Páginas | 52 |
| Privado | `originals/vendor-integrations/VENDOR-NETBO-API-V2/original/NETBO_API_V2.pdf` @ `a889ef6…` |
| Papel | técnico de fornecedor — **≠** AGT / DR / FE oficial |

## Resumo sanitizado

1. **Descoberta:** `GET https://companies.api.net-bo.com/companies/detailed?company=<dbname>` → hostname/URL do servidor API.
2. **Auth:** `GET https://<server>.api.net-bo.com/api/auth?db=…&login=…&passwd=…` → `token` + perfil utilizador. Token em header `token:` nas chamadas seguintes.
3. **Tabelas / catálogo:** `/api/tables/products`, `store_sale_products`, `stores`, `units`, `users`, `clients`, `create_client`.
4. **Config:** `/api/configurations/company`, `fiscal_regions_and_taxes`, `payment_methods`, teclados POS.
5. **Documentos:** `/api/data/documents`, `/api/documents/create_invoice`, `create_nc_from_invoice`, `create_receipt`; algumas rotas marcadas *Not implemented* no PDF (`create_credit_note`, `create_transport_document`).
6. **Movimentos / sales / reports / planning / import_export** (incl. `saft_export` / `saft_import` — **SAFT PT/contexto NET-BO**, não SAF-T AO AGT).
7. Host observado nos exemplos do PDF: `netbov5.api.net-bo.com` (HTTPS).

**Não inventar** endpoints, campos ou semântica fiscal além do PDF + cliente maduro auditado.

## Matriz (produto futuro)

| Capacidade | No PDF V2 | Reuso futuro módulo | Risco |
|---|---|---|---|
| Server discovery | sim | `port_with_tests` (conceito) | baixo |
| Auth + token header | sim | `port_with_tests` | **passwd em query** (doc); redacção logs |
| Sync products/units/stores/suppliers | sim | `reference_only` → adapter | mapeamento ≠ AO-DOC |
| Relatórios / stocks | sim | `reference_only` | fora do núcleo fiscal AO |
| `create_invoice` / NC / receipt | sim (parcial) | **`do_not_port` como autoridade AO** | **dupla emissão** |
| `saft_export` / import | sim | `do_not_port` para SAF-T AO | ambíto/país errado |
| Conector live | — | **não neste incremento** | — |

## Segurança (extracto)

- Credenciais na query string do `/api/auth` (doc vendor) — preferir canal seguro e nunca logar `passwd`/`token`.
- Cliente maduro (`my-bwb-app` @ `629edde`): doc pede header `token:`; implementação usa sobretudo Bearer / query para `api_token` — ver [AUD-VENDOR-NETBO](../audits/AUD-VENDOR-NETBO-REUSE-MY-BWB-APP.md).
- Exemplos do PDF podem conter dados ilustrativos — **não** versionar credenciais reais.
- Redistribuição do PDF: `uncertain` → só privado.
- Money `float` na app de origem — **não** portar para o módulo fiscal.

## Relação com `AO-*`

Nenhuma. Experiência NET-BO informa desenho de integração; requisitos fiscais Angola continuam em `sources.yaml` + matriz provisória.
