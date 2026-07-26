# C-DOC-002 — rótulo `RG` (Recibo vs Outros Recibos Emitidos)

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-07-26 |
| Severidade | baixa (rótulo) |

## Factos

1. OCR DE 683/25 PDF p.7 / **19169**: `RG` descrito como «Recibo».
2. Mesmo diploma OCR PDF p.22 / **19184**: `RG` como «Outros Recibos Emitidos».
3. HTML HML `servico_registar.html`: `RG` — «Recibo».
4. Regras FE: `lines` vazio e `paymentReceipt` obrigatório para AR/RC/RG (OCR p.7–8).

## Não fazer

- Não inventar um terceiro código; tratar como o mesmo `RG` com rótulos inconsistentes entre páginas/fontes até revisão PDF.
