# Conflitos entre fontes

Registar aqui conflitos legais/técnicos explícitos. Não resolver por omissão.

| ID | Tema | Estado |
|---|---|---|
| [C-DOC-001](C-DOC-001-fe-gf-ocr-gap.md) | `GF` HTML/XSD vs DE 683 p.7 (ausente visual) | documentado_divergencia |
| [C-DOC-002](C-DOC-002-rg-label.md) | Rótulo `RG` inconsistente | documentado (mesmo código) |
| [C-DOC-003](C-DOC-003-fe-vs-saft-invoice-type.md) | L4 `FA`/`RC`/`RG` ≠ L2 `InvoiceType` ≠ L3 `Payments` | aberto (mitigação fail-closed; residual DEC-REG-003) |
| [C-SIGN-001](C-SIGN-001-saft-rsa-vs-fe-jws.md) | Assinatura SAF-T RSA/SHA-1 (74/19) ≠ JWS FE RS256 | aberto (mitigação fail-closed `signsep`; residual AO-CRYPTO) |
| [C-FE-001](C-FE-001-fe-endpoint-path-inconsistency.md) | Paths HML `/ws/` vs `/v1` | aberto (mitigação fail-closed `fepath`; residual AGT/GAP-006) |
