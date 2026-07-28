# Vendor integrations (técnico; **não** normativo AGT)

**Estado:** inventário 2026-07-28 · `RM-VENDOR-001`  
**Catálogo:** [`../catalog/vendor-integrations.yaml`](../catalog/vendor-integrations.yaml)  
**Originais:** privado [`storesace-cv/bwb-fiscal-sources-ao`](https://github.com/storesace-cv/bwb-fiscal-sources-ao) · `originals/vendor-integrations/` · commit [`a889ef6…`](https://github.com/storesace-cv/bwb-fiscal-sources-ao/commit/a889ef623c367e96cb7246ab42a274e54cbb2dc3)

## Separação obrigatória

| Catálogo | Conteúdo | Gera `AO-*`? |
|---|---|---|
| `compliance/catalog/sources.yaml` | Legislação DR, FE AGT, XSD SAF-T | candidatos (com citação) |
| `compliance/catalog/vendor-integrations.yaml` | Docs técnicos de fornecedores | **nunca** |

- IDs `VENDOR-*` ≠ `AO-LEG-*` / `AO-FE-*`.
- PDFs **nunca** no Git público; só hashes + resumos sanitizados.
- `local/` permanece consulta; CI não depende de `local/`.
- **Sem** conector live neste incremento.

## Fontes catalogadas

| ID | SHA-256 (prefixo) | Páginas | Doc |
|---|---|---|---|
| `VENDOR-NETBO-API-V2` | `09bb09f8…` | 52 | [NETBO-CAPABILITY-MATRIX.md](NETBO-CAPABILITY-MATRIX.md) |
| `VENDOR-PTCERT-REST-API` | `52e34080…` | 19 | [PTCERT-POS-API-NOTES.md](PTCERT-POS-API-NOTES.md) |

Auditoria de reutilização NET-BO (read-only, `my-bwb-app`): [AUD-VENDOR-NETBO-REUSE-MY-BWB-APP.md](../audits/AUD-VENDOR-NETBO-REUSE-MY-BWB-APP.md).

## Decisão de autoridade de emissão

O módulo fiscal BWB é a **única** autoridade de numeração/emissão para o âmbito AGT Angola do produto. Integrações NET-BO / PT-CERT são **operacionais** (catálogo, sync, edge POS) e **não** podem emitir o número fiscal final paralelo. Ver notas PT-CERT e matriz NET-BO (risco de dupla emissão).
