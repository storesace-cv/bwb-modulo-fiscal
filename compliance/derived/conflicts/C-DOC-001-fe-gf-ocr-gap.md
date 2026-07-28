# C-DOC-001 — `GF` (Factura Genérica): HTML FE / XSD vs DE 683/25

| Campo | Valor |
|---|---|
| Estado | documentado_divergencia (confronto visual 2026-07-28; **não** AO-* confirmado) |
| Data | 2026-07-28 |
| Severidade | média (enum FE vs diploma) |

## Factos

1. Snapshot HML `AO-FE-SNAP-HML-2026-07-25-REGISTAR` (`eb430954…`) lista **`GF` — Factura Genérica** em `documentType`.
2. OCR `reviewed` DE 683/25 v2, PDF p.7 / gazeta **19169**: sequência `FG` → `AC` **sem** `GF` (`text.md`).
3. Texto do `searchable.pdf` privado (commit `c8a4e6e8…`) nas p.**7 / 22 / 24 / 26** também omite `GF`.
4. **Confronto visual 2026-07-28:** render PNG da p.7 a partir do `searchable.pdf` (densidade 200; sha256 render `b82b3a0c…`; **não** versionado no Git público). Na coluna de valores de `documentType` observa-se, após `FG - Factura Global`, directamente `AC - Aviso de Cobrança` — **sem** linha `GF`. Lista visual observada (p.7): `FA`, `FT`, `FR`, `FG`, `AC`, `AR`, `TV`, `RC`, `RG`, `RE`, `ND`, `NC`, `AF`, `RP`, `RA`, `CS`, `LD`.
5. Original PDF sha256 `b01e4581…` · path privado `originals/legislation/AO-LEG-DE-683-25-2025/v2/original/Diario_Republica_I_159_2025_DE_683-25.pdf`.
6. DP 71/25 Art.3 i) define Factura Genérica (gazeta **11903**) — rótulo L1, **não** código L4.
7. SAF-T XSD `InvoiceType` inclui `GF` (`e9a938e1…`, L2023–2065).

## Conclusão documental (fail-closed)

| Fonte | `GF` |
|---|---|
| DE 683/25 p.7 (OCR + searchable + **visual**) | **ausente** |
| HTML FE HML REGISTAR | presente |
| XSD ASSOFT `InvoiceType` | presente |
| DP 71/25 L1 «Factura Genérica» | rótulo (sem código `GF`) |

**Divergência confirmada** entre o diploma DE 683/25 (páginas do enum) e o snapshot HTML/XSD. Hipótese de «ruído OCR que escondeu `GF`» fica **rejeitada** para a p.7 confrontada visualmente.

## Política de produto até compliance

- Manter `bwb.ao.vendas.gf` no catálogo com `estado_normativo=conflito` e `activo=off`.
- **Não** activar emissão `GF` nem mapear como confirmado a partir do DE 683.
- **Não** apagar `GF` do inventário FE/XSD — a divergência deve permanecer visível.
- Fecho normativo exige revisão compliance / eventual orientação AGT (ou outra página oficial que enumere `GF`).

## Não fazer

- Não inventar `GF` no DE 683 «porque o HTML o tem».
- Não omitir `GF` do inventário FE/XSD «porque o DE 683 o omite».
- Não promover `AO-DOC-001` / `GF` a confirmado com esta nota.
