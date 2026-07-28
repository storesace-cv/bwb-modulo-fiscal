# C-DOC-001 — `GF` (Factura Genérica) no HTML FE vs DE 683/25

| Campo | Valor |
|---|---|
| Estado | aberto (confronto texto PDF/OCR 2026-07-28; residual visual PNG) |
| Data | 2026-07-28 |
| Severidade | média (enum FE vs diploma) |

## Factos

1. Snapshot HML `AO-FE-SNAP-HML-2026-07-25-REGISTAR` (`servico_registar.html`, campo `documentType`) lista **`GF` — Factura Genérica**.
2. OCR `reviewed` de `AO-LEG-DE-683-25-2025` v2, PDF p.7 / gazeta **19169**, na lista de `documentType` passa de `FG` para `AC` **sem** `GF` (text.md + evidência).
3. Extracção de texto do `searchable.pdf` privado (`derivatives/…/v2/searchable.pdf`, commit `c8a4e6e8…`) nas páginas **7, 22, 24, 26** também **omite** `GF` entre `FG` e `AC` (mesma sequência FA→…→FG→AC).
4. DP 71/25 Art.3 i) define Factura Genérica (PDF p.3 / gazeta **11903**) — rótulo L1, **não** código L4.
5. SAF-T XSD `InvoiceType` inclui `GF` (`e9a938e1…`, L2023–2065).

## Interpretação provisória (não normativa)

Há **divergência** entre (a) HTML HML + XSD ASSOFT e (b) texto extractável do DE 683/25 nas páginas do enum. Hipóteses em aberto: omissão no diploma impresso; ruído de OCR/searchable em todas as camadas; ou `GF` só em outra fonte oficial. **Fail-closed:** manter `GF` no catálogo como `conflito` / `activo=off` até revisão compliance + confronto visual PNG do original.

## Não fazer

- Não omitir `GF` do inventário FE só porque o OCR/searchable o omite — o HTML e o XSD citam-no.
- Não confirmar enum FE completo do DE 683 sem confronto visual do PDF original p.7 (e páginas repetidas do enum).
- Não promover `AO-DOC-001` / `GF` a confirmado com esta nota.

## Resolução candidata

1. Confronto visual PNG do original p.7 / p.22 / p.24 / p.26 (page-evidence já referencia renders locais não versionados).
2. Se o PDF impresso **não** tiver `GF`: registar divergência HTML/XSD vs DE 683 e pedir orientação compliance (não inventar).
3. Se o PDF tiver `GF` e OCR falhou: actualizar OCR/`low_confidence` no privado e só então fechar.
