# Conflitos entre fontes

Registar aqui conflitos legais/técnicos explícitos. Não resolver por omissão.

| ID | Tema | Estado |
|---|---|---|
| [C-DOC-001](C-DOC-001-fe-gf-ocr-gap.md) | `GF` HTML/XSD vs DE 683 p.7 (ausente visual) | documentado_divergencia |
| [C-DOC-002](C-DOC-002-rg-label.md) | Rótulo `RG` inconsistente | documentado (mesmo código) |
| [C-DOC-003](C-DOC-003-fe-vs-saft-invoice-type.md) | L4 `FA`/`RC`/`RG` ≠ L2 `InvoiceType` ≠ L3 `Payments` | aberto (mitigação fail-closed; residual DEC-REG-003) |
| [C-DOC-004](C-DOC-004-ar-dual-l3.md) | Homónimo L4 `AR` em `InvoiceType` **e** `PaymentType` (L3 distintos) | aberto (mitigação fail-closed; residual «grupo único») |
| [C-DOC-005](C-DOC-005-insurer-invoice-vs-worktype.md) | Homónimos segurador `RP`/`RE`/`CS`/`LD`/`RA` em `InvoiceType` **e** `WorkType` | aberto (mitigação fail-closed; residual DEC-REG-003) |
| [C-DOC-006](C-DOC-006-rc-payment-vs-purchase.md) | Homónimo L2 `RC` em `PaymentType` **e** `PurchaseType` (L3 distintos) | aberto (mitigação fail-closed; residual DEC-REG-003) |
| [C-DOC-007](C-DOC-007-gr-movement-vs-worktype.md) | Homónimo L2 `GR` em `MovementType` **e** `WorkType` (L3 distintos) | aberto (mitigação fail-closed; residual DEC-REG-003) |
| [C-DOC-008](C-DOC-008-ft-nc-invoice-vs-purchase.md) | Homónimos L2 `FT`/`NC` em `InvoiceType` **e** `PurchaseType` (L3 distintos) | aberto (mitigação fail-closed; residual DEC-REG-003 compras) |
| [C-DOC-009](C-DOC-009-ar-purchase-third-l3.md) | Homónimo L2 `AR` também em `PurchaseType` (3.º L3; extensão C-DOC-004) | aberto (mitigação fail-closed; residual «grupo único») |
| [C-DOC-010](C-DOC-010-invoice-purchase-remaining.md) | Homónimos L2 restantes `FR`/`GF`/`FG`/`AC`/`AF`/`TV` em `InvoiceType` **e** `PurchaseType` | aberto (mitigação fail-closed; residual DEC-REG-003 compras) |
| [C-SIGN-001](C-SIGN-001-saft-rsa-vs-fe-jws.md) | Assinatura SAF-T RSA/SHA-1 (74/19) ≠ JWS FE RS256 | aberto (mitigação fail-closed `signsep`; residual AO-CRYPTO) |
| [C-FE-001](C-FE-001-fe-endpoint-path-inconsistency.md) | Paths HML `/ws/` vs `/v1` | aberto (mitigação fail-closed `fepath`; residual AGT/GAP-006) |
| [C-FE-QR-001](C-FE-QR-001-qr-url-de683-vs-fe-hml.md) | URL QR impresso DE 683 vs FE HML | aberto (mitigação fail-closed `feqr`) |
