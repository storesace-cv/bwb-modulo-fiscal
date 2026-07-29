# Requisitos derivados (PR C)

Requisitos `AO-*` confirmados só a partir de páginas OCR com `review_status: reviewed` e confronto visual com o PDF original.

- [`CONFIRMED-MATRIX-RM-REQ-001.md`](CONFIRMED-MATRIX-RM-REQ-001.md) — requisitos com obrigação normativa `confirmed_normative` (hoje: **AO-DOC-002**, **AO-SEQ-001**); **≠** homologação AGT
- [`PROVISIONAL-MATRIX-RM-REQ-001.md`](PROVISIONAL-MATRIX-RM-REQ-001.md) — linhas AO-* ainda não promovidas (Citação A–K); IDs `promoted` apontam para a matriz confirmada
- [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md) — L1·L2·L3·L4; `DEC-PROD-001`–`015`; **não** fecha `AO-DOC-001`
- [`DOCUMENT-CATALOG-RM-REQ-001.md`](DOCUMENT-CATALOG-RM-REQ-001.md) — catálogo canónico (esquema `DEC-PROD-015`); seed incremental; **não** confirma AO-*
- [`FE-SERVICES-MATRIX-RM-REQ-001.md`](FE-SERVICES-MATRIX-RM-REQ-001.md) — serviços FE HML, endpoints e `FE-RNG-*` (Citação H); inclui VALIDAR/CONSULTAR-FATURA/GESTAO/QRCODE; **não** fecha `AO-AGT-*`
