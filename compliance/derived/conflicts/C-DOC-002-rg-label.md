# C-DOC-002 — rótulo `RG` (Recibo vs Outros Recibos Emitidos)

| Campo | Valor |
|---|---|
| Estado | documentado (mesmo código; rótulos divergentes; sem terceiro código) |
| Data | 2026-07-28 |
| Severidade | baixa (rótulo) |

## Factos

1. OCR / searchable DE 683/25 PDF p.7 / **19169**: `RG` descrito como «Recibo».
2. Mesmo diploma PDF p.8 / **19170** e p.22 / **19184**: `RG` como «Outros Recibos Emitidos».
3. HTML HML `servico_registar.html`: `RG` — «Recibo».
4. Regras FE: `lines` vazio e `paymentReceipt` obrigatório para AR/RC/RG (OCR p.7–8).
5. Catálogo seed (`bwb.ao.pagamentos.rg`): designação «Outros Recibos Emitidos»; SAF-T `PaymentType=RG` (C-DOC-003).

## Decisão de documentação (produto; ≠ AO-* confirmado)

Tratar como o **mesmo** código L4/`PaymentType` `RG` com rótulos inconsistentes entre páginas/fontes. Preferência de designação longa no catálogo: «Outros Recibos Emitidos» (p.8/p.22). Canal FE continua a usar o código `RG` sem inventar um terceiro código.

## Não fazer

- Não inventar um terceiro código.
- Não mapear `RG` para `InvoiceType`.
- Não fechar `AO-DOC-001` só com harmonização de rótulo.
