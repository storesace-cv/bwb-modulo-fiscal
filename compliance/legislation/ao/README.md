# Legislação AO — derivados OCR (privado)

Originais e derivados OCR (`searchable_pdf`, `markdown_text`, `text.txt`) vivem apenas no repositório privado
[`storesace-cv/bwb-fiscal-sources-ao`](https://github.com/storesace-cv/bwb-fiscal-sources-ao)
(commit de referência no catálogo: `private_commit`).

Paths canónicos:

```text
originals/legislation/{SOURCE_ID}/original/*.pdf
derivatives/legislation/{SOURCE_ID}/v1/searchable.pdf
derivatives/legislation/{SOURCE_ID}/v1/text.md
derivatives/legislation/{SOURCE_ID}/v1/text.txt
derivatives/legislation/{SOURCE_ID}/v1/review.json
```

Metadados públicos: [`../../catalog/sources.yaml`](../../catalog/sources.yaml).

**Não** usar `local/` em build, testes, CI ou agentes. OCR `rejected` ou não-`reviewed` **não** confirma requisitos `AO-*`.
