# C-SIGN-001 — Assinatura SAF-T (DE 74/19) ≠ JWS FE (DE 683/25)

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação engenharia fail-closed 2026-07-28; residual AO-CRYPTO / compliance) |
| Data | 2026-07-28 |
| Severidade | alta (certificação / implementação) |

## Factos

1. **DE 74/19** Anexo I n.º**34** (OCR, PDF p.**8–10**, gazeta **1582–1584**): assinatura de documentos para SAF-T com **RSA**, mensagem **SHA-1**, campos concatenados (`InvoiceDate`; `SystemEntryDate`; `InvoiceNo`; `GrossTotal`; `Hash` anterior), chave fabricante, Modelo 8.
2. **DE 683/25** + snapshot FE HML: assinatura de documento FE via **JWS/RS256** (`jwsDocumentSignature`), campos distintos (`documentNo`, `taxRegistrationNumber`, `documentType`, …).
3. `compliance/POLICY.md` e AGENTS.md já exigem não confundir os dois mecanismos.
4. XSD ASSOFT tem `Hash`/`HashControl` **sem** algoritmo (`e9a938e1…`, L1361–1374) — `PendingHashAlgorithm`.

## Matriz de separação fail-closed (produto; ≠ AO-* confirmado)

| Mecanismo | Âmbito | Algoritmo no módulo | Pacote |
|---|---|---|---|
| `saft_hash_chain` | Encadeamento documental SAF-T | **não** implementado como confirmado; export marca `hash_algorithm_pending_ao` | `internal/saftao` |
| `fe_jws_rs256` | Envelope técnico / preparação FE | `RS256` efémero (`fiscaljws`); ≠ FE-RNG certificado | `internal/fiscaljws` |
| Integridade artefacto XML | SHA-256 dos bytes exportados | **≠** `Invoice.Hash` | `ExportResult.SHA256` |

## Mitigação de engenharia (2026-07-28)

- Pacote `internal/signsep`: invariantes + `RejectConflatedAlgorithm` (rejeita SHA-1 como FE JWS e RS256 como Hash SAF-T).
- Testes: export SAF-T inclui `PendingHashAlgorithm`; `saftao.Meta().Certified=false`.
- **Não** fecha `AO-CRYPTO-001` nem implementa n.º34 / JWS FE oficial.

## Não fazer

- Não reutilizar a cadeia Hash SAF-T do 74/19 como assinatura FE.
- Não afirmar que RS256 FE «substitui» o n.º 34 do 74/19 (âmbitos diferentes até decisão AGT/compliance).
- Não tratar SHA-256 do artefacto XML como algoritmo de `Invoice.Hash`.
- Não marcar C-SIGN-001 como resolvido só porque os guards existem.

## Resolução candidata

Manter implementações e requisitos `AO-CRYPTO-*` / SAF-T separados; fechar só com revisão compliance + testes por mecanismo + (se aplicável) aceitação AGT.
