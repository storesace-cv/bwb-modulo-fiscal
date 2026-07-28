# Matriz FE HML — serviços, endpoints e FE-RNG (provisória)

**Estado:** rascunho rastreável — **não** confirma `AO-AGT-001` / `AO-AGT-002` / `AO-SEQ-002`.
**Data:** 2026-07-28
**Regra:** snapshots HTML HML são auxiliares (`pending_validation`); **não** inventar `FE-RNG-*`, endpoints nem campos. Bytes HTML permanecem no privado (`license_redistribution: uncertain`); este ficheiro só cita `source_id` + sha256 + extractos.

`private_commit` FE alinhado: `c93db4f89352eab884a4862943faa2ed874217fc` (originais); inventário expandido 2026-07-28 sem cópia de bytes HTML para o Git público.

## Fontes (catálogo)

| Serviço / tema | source_id | Artefacto privado | SHA256 (prefixo) | Estado |
|---|---|---|---|---|
| Índice / arquitectura | `AO-FE-SNAP-HML-2026-07-25-INDEX` | `…/index.html` | `67fc40a6…` | `pending_validation` |
| Auth / exemplos | `AO-FE-SNAP-HML-2026-07-25-API` | `…/api.html` | `06a9dbdf…` | `pending_validation` |
| JWS RS256 | `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | `…/estrutura.html` | `50ba18e0…` | `pending_validation` |
| Registar factura + FE-RNG | `AO-FE-SNAP-HML-2026-07-25-REGISTAR` | `…/servico_registar.html` | `eb430954…` | `pending_validation` |
| Solicitar série + FE-RNG séries | `AO-FE-SNAP-HML-2026-07-25-SOLICITAR` | `…/servico_solicitar.html` | `f8fb22e7…` | `pending_validation` |
| Listar séries | `AO-FE-SNAP-HML-2026-07-25-LISTAR` | `…/servico_listar.html` | `5729f02c…` | `pending_validation` |
| Listar facturas | `AO-FE-SNAP-HML-2026-07-25-LISTAR-FATURAS` | `…/servico_listar_faturas.html` | `c748caca…` | `pending_validation` |
| Obter estado | `AO-FE-SNAP-HML-2026-07-25-CONSULTAR` | `…/servico_consultar.html` | `1cac2844…` | `pending_validation` |
| Consultar factura | `AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA` | `…/servico_consultar_fatura.html` | `6d5cc1a0…` | `pending_validation` |
| Validar documento | `AO-FE-SNAP-HML-2026-07-25-VALIDAR` | `…/servico_validar.html` | `7ab70629…` | `pending_validation` |
| Modelo assíncrono | `AO-FE-SNAP-HML-2026-07-25-MODELO` | `…/modelo.html` | `f851f512…` | `pending_validation` |
| Gestão certificados/chaves | `AO-FE-SNAP-HML-2026-07-25-GESTAO` | `…/gestao.html` | `de423e66…` | `pending_validation` |
| QR Code impresso | `AO-FE-SNAP-HML-2026-07-25-QRCODE` | `…/qrcode.html` | `ccade20b…` | `pending_validation` |

Cruzamento diploma: DE 683 Anexo I–III (Citação G) · [`PROVISIONAL-MATRIX-RM-REQ-001.md`](PROVISIONAL-MATRIX-RM-REQ-001.md).

## A. Endpoints citados no snapshot (HML vs PRD)

| Serviço | Método | HML (snapshot) | PRD (snapshot) | Fonte |
|---|---|---|---|---|
| `registarFactura` | POST | `https://sifphml.minfin.gov.ao/sigt/fe/v1/registarFactura` | `https://sifp.minfin.gov.ao/sigt/fe/v1/registarFactura` | REGISTAR; API |
| `solicitarSerie` | POST | **inconsistente** — HTML mostra `…/sigt/fe/ws/v1/registarFactura` | `https://sifp.minfin.gov.ao/sigt/fe/v1/solicitarSerie` | SOLICITAR — ver [C-FE-001](../conflicts/C-FE-001-fe-endpoint-path-inconsistency.md) |
| `listarSeries` | POST | `https://sifphml.minfin.gov.ao/sigt/fe/v1/listarSeries` | `https://sifp.minfin.gov.ao/sigt/fe/v1/listarSeries` | LISTAR |
| `listarFacturas` | POST | `https://sifphml.minfin.gov.ao/sigt/fe/ws/v1/listarFacturas` (`/ws/`) | `https://sifp.minfin.gov.ao/sigt/fe/v1/listarFacturas` | LISTAR-FATURAS — C-FE-001 |
| `obterEstado` | POST | `https://sifphml.minfin.gov.ao/sigt/fe/v1/obterEstado` | `https://sifp.minfin.gov.ao/sigt/fe/v1/obterEstado` | CONSULTAR |
| `consultarFactura` | POST | `https://sifphml.minfin.gov.ao/sigt/fe/v1/consultarFactura` | `https://sifp.minfin.gov.ao/sigt/fe/v1/consultarFactura` | CONSULTAR-FATURA (HML/PRD `/v1/` alinhados) |
| `validarDocumento` | POST | `https://sifphml.minfin.gov.ao/sigt/fe/v1/validarDocumento` | `https://sifp.minfin.gov.ao/sigt/fe/v1/validarDocumento` | VALIDAR (HML/PRD `/v1/` alinhados) |

Auth (API): Basic Auth (`Authorization: Basic …`); credenciais via pedido formal AGT — **GAP-006**; **nunca** versionar credenciais reais (exemplos do HTML são ilustrativos).

**Nota C-FE-001:** `consultarFactura` e `validarDocumento` **não** resolvem o conflito `/ws/` vs `/v1/` em `solicitarSerie` / `listarFacturas`; apenas documentam serviços adicionais com paths alinhados no snapshot.

## B. Modelo assíncrono (AO-AGT-002 candidato)

Fonte: `AO-FE-SNAP-HML-2026-07-25-MODELO` · sha256 `f851f512…` · `pending_validation`.

Extracto (auxiliar): submissão via `registarFactura` → validação estrutural JSON → fila → resposta imediata com **`requestID`** → processamento (validações fiscais / série / integridade) → consulta posterior via **Consultar Estado** (`obterEstado`).

Limite: **não** fecha máquina de estados produto (DEC-API-004); snapshot ≠ aceite AGT.

## C. FE-RNG — inventário extractado (só códigos presentes)

### C.1 `servico_registar.html` (catálogo principal)

Fonte: `AO-FE-SNAP-HML-2026-07-25-REGISTAR` · `eb430954…`.

| Código | Código erro (HTML) | Tema (resumo extractado) |
|---|---|---|
| FE-RNG-001 | *(célula HTML partilhada com 002 — descrição própria **não** isolada)* | Presente no snapshot; **não** inventar texto |
| FE-RNG-002 | E01 | Falta de parâmetro |
| FE-RNG-003 | E02 | Formato inválido de parâmetro |
| FE-RNG-004 | E03 | Valor não esperado de parâmetro |
| FE-RNG-005 | E04 | `numberOfEntries` ≠ ocorrências de `documents` |
| FE-RNG-006 | E05 | NIF emissor sem actividade registada |
| FE-RNG-007 | E28 | NIF emissor sem adesão FE |
| FE-RNG-008 | E06 | `creationDate` fora do período |
| FE-RNG-009 | E07 | `softwareValidationNo` não certificado |
| FE-RNG-010 | E08 | `jwsSoftwareSignature` inválida |
| FE-RNG-011 | E39 | dados de `jwsSoftwareSignature` ≠ certificação |
| FE-RNG-012 | E09 | Factura já no repositório |
| FE-RNG-013 | E10 | `customerTaxID` / `taxIDNumber` / `taxIDCountry` incorrectos |
| FE-RNG-014 | E11 | NIF angolano desconhecido |
| FE-RNG-015 | E12 | `lineNo` com repetição/quebra |
| FE-RNG-016 | E13 | Falta `referenceInfo` |
| FE-RNG-017 | E14 | `reference` desconhecido |
| FE-RNG-018 | E15 | Parâmetros simultâneos incompatíveis |
| FE-RNG-019 | E16 | Créditos vs débitos (nota de crédito) |
| FE-RNG-020 | E17 | Créditos vs débitos (não-NC) |
| FE-RNG-021 | E18 | Combinação de campos não permitida |
| FE-RNG-022 | E19 | `taxBase` vs `debitAmount`/`creditAmount` |
| FE-RNG-023 | E20 | Combinação de campos não permitida |
| FE-RNG-024 | E21 | Fórmula `quantity * unitPriceBase` |
| FE-RNG-025 | E22 | `taxPayable` ≠ soma impostos linhas |
| FE-RNG-026 | E23 | `netPayable` ≠ soma linhas |
| FE-RNG-027 | E24 | `grossPayable` ≠ soma linhas |
| FE-RNG-028 | E25 | `grossTotal` vs câmbio/pagamento |
| FE-RNG-029 | E26 | Uso incorrecto de `lines` por tipo |
| FE-RNG-030 | E29 | `documentDate` anterior à adesão FE |
| FE-RNG-031 | E40 | `jwsSignature` da chamada inválida |
| FE-RNG-061 | E41 | Regularização excede remanescente (`OriginatingON`) |
| FE-RNG-068 | E42 | Anulação/devolução excede montante (`reference`) |
| FE-RNG-069 | E43 | `quantity` zero com `taxBase` em correcções |
| FE-RNG-073 | E46 | Reutilização de `documentNo` de doc rejeitado |

### C.2 Séries — `servico_solicitar.html` (+ partilhados em listar)

Fonte: `AO-FE-SNAP-HML-2026-07-25-SOLICITAR` · `f8fb22e7…` (também `FE-RNG-010`/`011`/`032` em LISTAR).

| Código | Código erro | Tema (resumo extractado) |
|---|---|---|
| FE-RNG-010 | E08 | `jwsSoftwareSignature` |
| FE-RNG-011 | E39 | dados assinatura software ≠ certificação |
| FE-RNG-032 | E40 | `jwsSignature` da chamada |
| FE-RNG-049 | E30 | Contribuinte sem actividade |
| FE-RNG-050 | E06 | Contribuinte sem adesão FE |
| FE-RNG-051 | E31 | Código de série já em utilização |
| FE-RNG-053 | E32 | Série mal construída (ano 2/4 dígitos) |
| FE-RNG-055 | E33 | Ano da série vs data sistema (regra 15/Dez) |
| FE-RNG-056 | E34 | Série inexistente para contribuinte |
| FE-RNG-057 | E35 | Série de facturação **não** electrónica |
| FE-RNG-058 | E36 | Software emissor ≠ software da série |
| FE-RNG-059 | E37 | `documentType` ≠ tipo da série |
| FE-RNG-060 | E38 | Ano documento ≠ ano da série |
| FE-RNG-080 | E48 | `establishmentNumber` desconhecido / não registado na adesão |
| FE-RNG-081 | E49 | Ampliação de série com estabelecimento incompatível |

**Referências de campo (não inventar E-code):** o HTML de resposta de `solicitarSerie` menciona orientações **FE-RNG-082** e **FE-RNG-083** junto de `authorizedQuantity` — **sem** linha de tabela de erro isolada extractável neste inventário.

Campos série relevantes (SOLICITAR/LISTAR, alinháveis a DE 683): `seriesCode`, `seriesYear`, `documentType`, `establishmentNumber`, `seriesContingencyIndicator`, `seriesStatus` (A/U/F), `invoicingMethod` (FEPC/FESF/SF), `firstDocumentNo`, `authorizedQuantity`.

### C.3 Validar documento — `servico_validar.html`

Fonte: `AO-FE-SNAP-HML-2026-07-25-VALIDAR` · `7ab70629…`.

| Código | Código erro | Tema (resumo extractado) |
|---|---|---|
| FE-RNG-033 | E44 | Estado do documento **não** permite confirmação pelo adquirente |
| FE-RNG-034 | E45 | Estado do documento **não** permite rejeição pelo adquirente |
| FE-RNG-077 | E47 | `deductibleVATPercentage` **não** pode coexistir com `nonDeductibleAmount` |

Limite: snapshot auxiliar; **não** inventar outros `FE-RNG-*` deste serviço; **não** fechar máquina de estados jurídica (DEC-API-004).

### C.4 Consultar factura — campos observados

Fonte: `AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA` · `6d5cc1a0…`.

Campos citados no HTML (entrada/saída, sem inventar semântica): `taxRegistrationNumber`, `documentNo`, `jwsSignature`, `documents`, `statusFEResult` / `statusResult`, `validationStatus`. Também referencia `FE-RNG-010` / `011` / `032` (assinaturas software/chamada).

### C.5 Gestão de certificados e chaves (≠ fecho GAP-013)

Fonte: `AO-FE-SNAP-HML-2026-07-25-GESTAO` · `de423e66…` · `pending_validation`.

Extractos auxiliares (HTML; **não** norma publicada autenticada):

1. Software (`jwsSoftwareSignature`): produtor gera RSA localmente; **chave privada não sai** do ambiente do produtor; pública submetida à AGT no registo; RSA **mínimo 2048 bits**.
2. Contribuinte (`jwsDocumentSignature` / `jwsSignature`): HTML afirma emissão pela AGT e disponibilidade na conta do portal do contribuinte.
3. Comprometimento: revogar chave do produtor; documentos já aceites pela AGT permanecem válidos segundo o texto do snapshot; novas submissões com chave revogada deixam de ser aceites.

**Não fecha** GAP-013 / `DEC-REG-KEY-CUSTODY`: o snapshot **não** autoriza custódia da chave privada do contribuinte por módulo fiscal externo. Manter `AO-KEY-001` `blocked`.

### C.6 QR Code impresso

Fonte: `AO-FE-SNAP-HML-2026-07-25-QRCODE` · `ccade20b…` · `pending_validation`.

Parâmetros extractados: Model 2; versão 4 (33×33); correção M (15%); modo Byte; UTF-8; PNG 350×350; URL `https://quiosqueagt.minfin.gov.ao/facturacao-eletronica/consultar-fe?emissor=nifEmissor&document=documentNo` (espaços em `documentNo` → `%20`); logotipo AGT &lt;20% da imagem se incluído.

**Conflito:** DE 683 Anexo III @**19194–19195** cita host/params distintos (OCR) — ver [C-FE-QR-001](../conflicts/C-FE-QR-001-qr-url-de683-vs-fe-hml.md) + `internal/feqr`.

Limite: **não** implementa `RM-ENG-007` nem confirma FE-RNG QR; snapshot ≠ aceite AGT.

## D. Implicações fail-closed

1. **Não** implementar paths `/fe/ws/v1` vs `/fe/v1` sem fechar [C-FE-001](../conflicts/C-FE-001-fe-endpoint-path-inconsistency.md). Mitigação engenharia: `internal/fepath` (`ConflictOpen`; recusa `solicitarSerie`/`listarFacturas`; alinhados só `/v1`) — **≠** fecho do conflito.
2. **Não** inventar `FE-RNG-*` em falta (incl. descrição isolada de `FE-RNG-001`, nem E-codes para 082/083).
3. Snapshots HML **não** substituem validação AGT nem fecham GAP-006 (credenciais).
4. JWS FE (ESTRUTURA/RS256) ≠ Hash SAF-T ([C-SIGN-001](../conflicts/C-SIGN-001-saft-rsa-vs-fe-jws.md)).
5. **Não** gerar QR impresso sem fechar [C-FE-QR-001](../conflicts/C-FE-QR-001-qr-url-de683-vs-fe-hml.md) (`internal/feqr`).

## Referências

- Matriz AO-*: [`PROVISIONAL-MATRIX-RM-REQ-001.md`](PROVISIONAL-MATRIX-RM-REQ-001.md)
- Tipos: [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](DOCUMENT-TYPES-MATRIX-RM-REQ-001.md)
- Catálogo: [`../../catalog/sources.yaml`](../../catalog/sources.yaml)
- Espaço snapshots públicos: [`../../fe/snapshots/README.md`](../../fe/snapshots/README.md)
