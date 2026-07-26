# C-SIGN-001 — Assinatura SAF-T (DE 74/19) ≠ JWS FE (DE 683/25)

| Campo | Valor |
|---|---|
| Estado | aberto (registo explícito; não fundir mecanismos) |
| Data | 2026-07-26 |
| Severidade | alta (certificação / implementação) |

## Factos

1. **DE 74/19** Anexo I n.º**34** (OCR, PDF p.**8–10**, gazeta **1582–1584**): assinatura de documentos para SAF-T com **RSA**, mensagem **SHA-1**, campos concatenados (`InvoiceDate`; `SystemEntryDate`; `InvoiceNo`; `GrossTotal`; `Hash` anterior), chave fabricante, Modelo 8.
2. **DE 683/25** + snapshot FE HML: assinatura de documento FE via **JWS/RS256** (`jwsDocumentSignature`), campos distintos (`documentNo`, `taxRegistrationNumber`, `documentType`, …).
3. `compliance/POLICY.md` e AGENTS.md já exigem não confundir os dois mecanismos.

## Não fazer

- Não reutilizar a cadeia Hash SAF-T do 74/19 como assinatura FE.
- Não afirmar que RS256 FE «substitui» o n.º 34 do 74/19 (âmbitos diferentes até decisão AGT/compliance).

## Resolução candidata

Manter implementações e requisitos `AO-CRYPTO-*` / SAF-T separados; fechar só com revisão compliance + testes por mecanismo.
