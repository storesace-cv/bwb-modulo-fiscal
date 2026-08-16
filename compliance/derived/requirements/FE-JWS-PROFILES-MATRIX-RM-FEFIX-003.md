# Matriz FE JWS — perfis operacionais (RM-FEFIX-003)

**Estado:** rascunho auditável — **não** confirma `AO-CRYPTO-001` / FE-RNG oficial / aceitação AGT.

**Data (UTC):** 2026-08-16

**Regra:** só `eligible` se campos, nomes, representação e tipo de assinatura forem inequívocos nas fontes. **Proibido** “intersecção mínima”. Fontes `pending_validation`.

## Fontes citadas

| source_id | Papel | Estado |
|---|---|---|
| `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | JWS compact / RS256 / exemplos de payload | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-GESTAO` | Distinção software vs contribuinte | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-REGISTAR` | registarFactura + exemplos JWS | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-SOLICITAR` | solicitarSerie | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-LISTAR` | listarSeries | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-CONSULTAR` | obterEstado | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA` | consultarFactura | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-VALIDAR` | validarDocumento | `pending_validation` |
| `AO-FE-SNAP-HML-2026-07-25-LISTAR-FATURAS` | listarFacturas | `pending_validation` |

Separação SAF-T ≠ FE: [C-SIGN-001](../conflicts/C-SIGN-001-saft-rsa-vs-fe-jws.md).

## Motor genérico

Pacote [`internal/fejws`](../../../internal/fejws/compact.go): compact JWS, `alg=RS256` apenas, payload = bytes exactos, sem `typ` por defeito, verificação com Base64URL canónico e rejeição de chaves duplicadas / `crit` desconhecido.

## Matriz

| ProfileID | Operação | Tipo assinatura | Campos exactos (ordem JSON struct) | Protected header | Estado | Conflito / notas |
|---|---|---|---|---|---|---|
| `software_info` | transversal (softwareInfo) | `jwsSoftwareSignature` | `productId`, `productVersion`, `softwareValidationNumber` | `{"alg":"RS256"}` (**sem** `typ`) | **eligible** | `typ` JWT vs JOSE → [C-FE-JWS-TYP-001](../conflicts/C-FE-JWS-TYP-001-typ-jwt-vs-jose.md); placeholder `softwareValidationNumber` ≠ certificado |
| `obter_estado_request` | `obterEstado` | `jwsSignature` | `taxRegistrationNumber`, `requestID` | `{"alg":"RS256"}` | **eligible** | Tabela = bloco Payload assinatura |
| `consultar_factura_request` | `consultarFactura` | `jwsSignature` | `taxRegistrationNumber`, `documentNo` | `{"alg":"RS256"}` | **eligible** | Tabela = bloco Payload assinatura |
| `registar_document` | `registarFactura` | `jwsDocumentSignature` | — | — | **blocked_conflict** | [C-FE-JWS-DOC-001](../conflicts/C-FE-JWS-DOC-001-document-totals-sample.md) |
| `registar_request_jwsSignature` | `registarFactura` | `jwsSignature` | — | — | **blocked_conflict** | FE-RNG-031 menciona campo; schema/exemplo de entrada não fecham payload exacto + typ |
| `solicitar_serie_request` | `solicitarSerie` | `jwsSignature` | — | — | **blocked_conflict** | [C-FE-JWS-REQ-001](../conflicts/C-FE-JWS-REQ-001-solicitar-serie-fields.md) |
| `listar_series_request` | `listarSeries` | `jwsSignature` | — | — | **blocked_conflict** | [C-FE-JWS-REQ-002](../conflicts/C-FE-JWS-REQ-002-listar-series-fields.md) |
| `validar_documento_request` | `validarDocumento` | `jwsSignature` | — | — | **blocked_conflict** | [C-FE-JWS-REQ-003](../conflicts/C-FE-JWS-REQ-003-validar-documento-fields.md) |
| `listar_facturas_request` | `listarFacturas` | `jwsSignature` | — | — | **blocked_conflict** | [C-FE-JWS-REQ-004](../conflicts/C-FE-JWS-REQ-004-listar-facturas-payload-block.md) |

## Binding

- Contribuinte: `ValidateTaxpayerBinding(ref, nif)` — mismatch sem revelar NIFs.
- Produtor efémero ≠ contribuinte; chave contribuinte ≠ software.
- `sourceLabel` não entra no JWS.

## Implementação

- Eligible: [`internal/feprofile`](../../../internal/feprofile/profiles.go)
- Blocked: funções `*Blocked` devolvem `ErrProfileBlocked`
- Custódia: [`internal/agttestkit`](../../../internal/agttestkit/provider.go)
