# C-DOC-003 — L4 `documentType` FE ≠ L2 `InvoiceType` ≠ L3 estrutura (`FA`, `RC`, `RG`)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação produto fail-closed documentada; residual DEC-REG-003) |
| Data | 2026-07-28 |
| Severidade | alta (exportação / dual-stack) |

## Factos (quatro camadas)

1. **L4** FE `documentType` (DE 683/25 + HTML HML) inclui **`FA`**, **`RC`**, **`RG`**.
2. **L2** SAF-T `InvoiceType` (`SalesInvoices`, L2023–2065, `e9a938e1…`, `pending_validation`) enumera `FT|FR|GF|FG|AC|AR|ND|NC|AF|TV|RP|RE|CS|LD|RA` — **sem** `FA`, `RC`, `RG`.
3. **L3** SAF-T: recibos tipicamente em `Payments/Payment`, **não** em `SalesInvoices/Invoice`.
4. **L2** `SAFTAOPaymentType` (**L2740–2754**) sob L3 `Payments` enumera **`RC`**, **`RG`**, **`AR`** — **outro** enum e **outra** estrutura que `InvoiceType`.
5. **L1** DP 71/25 reconhece Factura Adiantamento e Recibo (Art.3 g)/o)) — rótulos legais, **não** códigos.
6. `FA` ausente de `InvoiceType` e de `SAFTAOPaymentType`.
7. `AR` aparece em **ambos** os enums L2 (`InvoiceType` e `PaymentType`) sob L3 distintos — **não** prova bijecção L4↔L2.

## Matriz de routing fail-closed (produto; ≠ AO-* confirmado)

| Código L4 FE | Canal FE | Adaptador SAF-T seed | L3 | Política (`DEC-PROD-006`) |
|---|---|---|---|---|
| `FA` | permitido se FE `active` + gates | **∅** (FE-only) | — | **Não** inventar `InvoiceType`/`PaymentType`/`SalesInvoices` |
| `RC` | permitido se elegível | `PaymentType=RC` | `Payments` | **Proibido** `InvoiceType=RC` |
| `RG` | permitido se elegível | `PaymentType=RG` | `Payments` | **Proibido** `InvoiceType=RG`; rótulo C-DOC-002 |
| `FT`/`NC`/… com `InvoiceType=*` | conforme seed | `InvoiceType=*` | `SalesInvoices` | Sem bijecção automática L4↔L2 para recibos |

## Mitigação de engenharia (2026-07-28)

- Catálogo seed + `doctype.CheckCDOC003Invariants` / testes: `FA` eligibility=`FE` e SAF-T vazio; `RC`/`RG` só `PaymentType`.
- `saftao.ValidInvoiceType` rejeita `FA`/`RC`/`RG`; `ValidPaymentType` aceita `RC`/`RG`/`AR`.
- `DEC-PROD-006` já proíbe inventar mapeamento FE→`InvoiceType` para FE-only.
- **Não** fecha o conflito jurídico/normativo: falta `DEC-REG-003` + revisão compliance; exportação dual-stack completa continua aberta.

## Não fazer

- Não tratar L4=`RC` como L2=`InvoiceType` nem como prova de mapeamento.
- Não tratar L2=`PaymentType=RC` como fecho de L4→SAF-T (são camadas distintas).
- Não inventar estrutura L3 para `FA` sem fonte oficial / `DEC-REG-003`.
- Não confundir L1 «Recibo» com L4 `RG`/`RC` nem com L2 `PaymentType`.
- Não enviar tipo SAF-T-only ao endpoint FE; `FA` (FE-only) só no canal FE (`DEC-PROD-006`).
- Não marcar C-DOC-003 como resolvido só porque o seed/testes estão alinhados.
