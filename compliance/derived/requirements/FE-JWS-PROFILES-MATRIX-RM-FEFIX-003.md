# Matriz FE JWS — perfis operacionais (RM-FEFIX-003)

**Estado:** rascunho auditável — **não** confirma `AO-CRYPTO-001` / FE-RNG oficial / aceitação AGT.

**Data (UTC):** 2026-08-16

**Regra:** só perfil **wire AGT** `eligible` se campos, nomes, representação, tipo de assinatura **e** protected header (`typ` incluído) forem inequívocos. **Proibido** “intersecção mínima” ou omitir `typ` para contornar [C-FE-JWS-TYP-001](../conflicts/C-FE-JWS-TYP-001-typ-jwt-vs-jose.md). Fontes `pending_validation`.

**Separação de estados:**

| Camada | Significado |
|---|---|
| `payload_confirmed_from_snapshot` | Claims/campos coerentes no snapshot — builder tipado permitido |
| `blocked_conflict` (wire) | Assinatura compacta operacional AGT **não** pode ser emitida até fecho do conflito indicado |
| Motor `internal/fejws` | Primitiva técnica; sem `typ` por defeito; **≠** perfil wire AGT |

**Contagem wire:** **zero** perfis wire AGT completos activos enquanto `C-FE-JWS-TYP-001` estiver aberto.

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
Questões AGT: [`agt-clarifications-register.md`](../../../docs/01-compliance/agt-clarifications-register.md) (`AGT-Q-005`+).

## Motor genérico

Pacote [`internal/fejws`](../../../internal/fejws/compact.go): compact JWS, `alg=RS256` apenas, payload = bytes exactos, **sem** `typ` por defeito (caller pode passar `typ` só em testes genéricos), verificação com Base64URL canónico e rejeição de chaves duplicadas / `crit` desconhecido. **Não** afirma compatibilidade operacional AGT.

## Matriz

| ProfileID | Operação | Tipo assinatura | Payload / claims | Protected header wire | Payload status | Wire status | Conflito / notas |
|---|---|---|---|---|---|---|---|
| `software_info` | transversal (softwareInfo) | `jwsSoftwareSignature` | `productId`, `productVersion`, `softwareValidationNumber` | `typ` JWT vs JOSE **aberto** | `payload_confirmed_from_snapshot` | **blocked_conflict** | [C-FE-JWS-TYP-001](../conflicts/C-FE-JWS-TYP-001-typ-jwt-vs-jose.md); omitir `typ` **não** é terceira variante aprovada; placeholder ≠ certificado |
| `obter_estado_request` | `obterEstado` | `jwsSignature` | `taxRegistrationNumber`, `requestID` | `typ` JWT vs JOSE **aberto** | `payload_confirmed_from_snapshot` | **blocked_conflict** | Claims coerentes; wire bloqueado por typ |
| `consultar_factura_request` | `consultarFactura` | `jwsSignature` | `taxRegistrationNumber`, `documentNo` | `typ` JWT vs JOSE **aberto** | `payload_confirmed_from_snapshot` | **blocked_conflict** | Claims coerentes; wire bloqueado por typ |
| `registar_document` | `registarFactura` | `jwsDocumentSignature` | — | — | — | **blocked_conflict** | [C-FE-JWS-DOC-001](../conflicts/C-FE-JWS-DOC-001-document-totals-sample.md) |
| `registar_request_jwsSignature` | `registarFactura` | `jwsSignature` | — | — | — | **blocked_conflict** | FE-RNG-031 menciona campo; schema/exemplo não fecham payload + typ |
| `solicitar_serie_request` | `solicitarSerie` | `jwsSignature` | — | — | — | **blocked_conflict** | [C-FE-JWS-REQ-001](../conflicts/C-FE-JWS-REQ-001-solicitar-serie-fields.md) |
| `listar_series_request` | `listarSeries` | `jwsSignature` | — | — | — | **blocked_conflict** | [C-FE-JWS-REQ-002](../conflicts/C-FE-JWS-REQ-002-listar-series-fields.md) |
| `validar_documento_request` | `validarDocumento` | `jwsSignature` | — | — | — | **blocked_conflict** | [C-FE-JWS-REQ-003](../conflicts/C-FE-JWS-REQ-003-validar-documento-fields.md) |
| `listar_facturas_request` | `listarFacturas` | `jwsSignature` | — | — | — | **blocked_conflict** | [C-FE-JWS-REQ-004](../conflicts/C-FE-JWS-REQ-004-listar-facturas-payload-block.md) |

## Binding

- Contribuinte: `ValidateTaxpayerBinding(ref, nif)` — mismatch sem revelar NIFs.
- Produtor efémero ≠ contribuinte; chave contribuinte ≠ software.
- `sourceLabel` não entra no JWS.

## Implementação

- Payload builders: `Marshal*Payload` em [`internal/feprofile`](../../../internal/feprofile/profiles.go)
- Wire sign (`SignSoftwareInfo`, `SignObterEstadoRequest`, `SignConsultarFacturaRequest` e `*Blocked`): `ErrProfileBlocked`
- Motor genérico: [`internal/fejws`](../../../internal/fejws/compact.go) — só testes/primitiva
- Custódia: [`internal/agttestkit`](../../../internal/agttestkit/provider.go)
