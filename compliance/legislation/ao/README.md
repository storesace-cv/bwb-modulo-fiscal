# Legislação AO — derivados OCR (privado)

Originais e derivados OCR vivem apenas no repositório privado
[`storesace-cv/bwb-fiscal-sources-ao`](https://github.com/storesace-cv/bwb-fiscal-sources-ao).

## Utilizável (KB auxiliar)

- `AO-LEG-DE-74-19-2019` — OCR `reviewed` em `derivatives/legislation/AO-LEG-DE-74-19-2019/v1/`

## Não utilizável (diagnóstico apenas)

- `AO-LEG-RECT-10-19-2019` — original arquivado incorrecto/incompleto; OCR `rejected` sob `derivatives/.../v1/` **não** é KB.
- `AO-LEG-DE-683-25-2025` — original correcto v2 no privado (`…/v2/original/…`, 66p); OCR v2 pendente de revisão; OCR v1 do original incorrecto está em `diagnostics/` no privado e **não** é KB.

Metadados públicos: [`../../catalog/sources.yaml`](../../catalog/sources.yaml).

**Não** usar `local/` em build, testes, CI ou agentes. RM-SRC-004/RM-M2-C permanecem BLOQUEADOS até existirem 3 diplomas correctos com OCR `reviewed`.
