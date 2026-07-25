# Legislação AO — derivados OCR (privado)

Originais e derivados OCR vivem apenas no repositório privado
[`storesace-cv/bwb-fiscal-sources-ao`](https://github.com/storesace-cv/bwb-fiscal-sources-ao).

## Utilizável (KB auxiliar)

- `AO-LEG-DE-74-19-2019` — OCR `reviewed` em `derivatives/legislation/AO-LEG-DE-74-19-2019/v1/`
- `AO-LEG-DE-683-25-2025` — original correcto v2 + OCR `reviewed` v2 (66p; PDF p.2–65 = DE 683/25 @19164–19227; **não** citar p.66 = Aviso BNA 4/25 @19228)

## Não utilizável (diagnóstico apenas)

- `AO-LEG-RECT-10-19-2019` — original arquivado incorrecto/incompleto; OCR `rejected` sob `derivatives/.../v1/` **não** é KB.
- OCR v1 do DE 683/25 incorrecto (Aviso 4/25) — apenas `diagnostics/` no privado; **não** é KB.

Metadados públicos: [`../../catalog/sources.yaml`](../../catalog/sources.yaml).

**Não** usar `local/` em build, testes, CI ou agentes. RM-SRC-004/RM-M2-C permanecem BLOQUEADOS até a Rect. 10/19 integral oficial ter OCR `reviewed` (conjunto 74+Rect+683).
