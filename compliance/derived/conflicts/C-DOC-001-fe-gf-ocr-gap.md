# C-DOC-001 — `GF` (Factura Genérica) no HTML FE vs OCR DE 683/25

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-07-26 |
| Severidade | média (enum FE incompleto se só OCR) |

## Factos

1. Snapshot HML `AO-FE-SNAP-HML-2026-07-25-REGISTAR` (`servico_registar.html`, campo `documentType`) lista **`GF` — Factura Genérica**.
2. OCR `reviewed` de `AO-LEG-DE-683-25-2025` v2, PDF p.7 / gazeta **19169**, na lista de `documentType` **não** mostra `GF` entre `FG` e `AC` (lacuna/ruído OCR possível).
3. DP 71/25 Art.3 i) define Factura Genérica (PDF p.3 / gazeta **11903**).
4. SAF-T XSD `InvoiceType` inclui `GF`.

## Não fazer

- Não omitir `GF` do inventário FE só porque o OCR p.7 o omite.
- Não confirmar enum FE completo sem confronto visual do PDF original p.7 (e páginas repetidas do enum).

## Resolução candidata

Confrontar PDF original DE 683 p.7 (e p.22/24/26) com HTML; actualizar OCR/`low_confidence` se necessário; só então fechar.
