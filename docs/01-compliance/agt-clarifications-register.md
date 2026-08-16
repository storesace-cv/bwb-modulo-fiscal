# Registo cumulativo de esclarecimentos AGT

**Canónico:** este ficheiro.
**Data UTC (última revisão estrutural):** 2026-08-16
**Âmbito:** problemas e perguntas que podem exigir esclarecimento da Autoridade Geral Tributária (Angola).

## Aviso operacional (obrigatório)

- **Sem chamadas reais** aos endpoints HML/PRD da API de Facturação Electrónica AGT registadas neste documento até à data acima.
- Distinguir sempre:
  1. incidente real de rede/API observado;
  2. conflito na documentação;
  3. credencial/informação em falta;
  4. dúvida normativa/processual;
  5. incidente interno de desenvolvimento/CI (**não** entra neste registo).
- Sanitização: **proibido** NIF, nomes pessoais/empresariais reais, PEM, fingerprints, tokens, Basic Auth ou segredos.
- Detalhe técnico dos conflitos: links para `compliance/derived/conflicts/C-*` — **não** duplicar aqui.

**Categorias allowlist:** `missing_credential_or_info` | `documentation_conflict` | `normative_or_process_doubt` | `network_or_portal_incident`

**Estados allowlist:** `OPEN` | `ANSWERED` | `MITIGATED` | `NOT_APPLICABLE`

**Relaciona:** [`agt-dependencies.md`](agt-dependencies.md) · [`regulatory-gaps.md`](regulatory-gaps.md) · [`official-access-plan.md`](official-access-plan.md) · [`../../compliance/derived/conflicts/README.md`](../../compliance/derived/conflicts/README.md) · [`../../ROADMAP.md`](../../ROADMAP.md)

---

## Índice

| ID | Título curto | Categoria | Estado | Pronto email |
|---|---|---|---|---|
| [AGT-Q-001](#agt-q-001) | Credenciais Basic Auth HML/PRD | missing_credential_or_info | OPEN | yes |
| [AGT-Q-002](#agt-q-002) | Registo produtor / Modelo 8 / softwareValidationNo | missing_credential_or_info | OPEN | yes |
| [AGT-Q-003](#agt-q-003) | Workbook RSA de teste — instruções de uso | missing_credential_or_info | OPEN | yes |
| [AGT-Q-004](#agt-q-004) | Paths FE `/fe/v1` vs `/fe/ws/v1` | documentation_conflict | OPEN | yes |
| [AGT-Q-005](#agt-q-005) | Header JWS `typ` JWT vs JOSE | documentation_conflict | OPEN | yes |
| [AGT-Q-006](#agt-q-006) | Campos `jwsDocumentSignature` / documentTotals | documentation_conflict | OPEN | yes |
| [AGT-Q-007](#agt-q-007) | Payload assinatura solicitarSerie | documentation_conflict | OPEN | yes |
| [AGT-Q-008](#agt-q-008) | Payload assinatura listarSeries | documentation_conflict | OPEN | yes |
| [AGT-Q-009](#agt-q-009) | Payload assinatura validarDocumento | documentation_conflict | OPEN | yes |
| [AGT-Q-010](#agt-q-010) | Payload assinatura listarFacturas | documentation_conflict | OPEN | yes |
| [AGT-Q-011](#agt-q-011) | registarFactura jwsSignature (FE-RNG-031) | documentation_conflict | OPEN | yes |
| [AGT-Q-012](#agt-q-012) | softwareValidationNumber vs softwareValidationNo | documentation_conflict | OPEN | yes |
| [AGT-Q-013](#agt-q-013) | Custódia chave privada do contribuinte | normative_or_process_doubt | OPEN | yes |
| [AGT-Q-014](#agt-q-014) | XSD SAF-T AO oficial e vigência | missing_credential_or_info | OPEN | yes |
| [AGT-Q-015](#agt-q-015) | Vetores / resultados oficiais de homologação | missing_credential_or_info | OPEN | yes |
| [AGT-Q-016](#agt-q-016) | Contingência / faturação offline | normative_or_process_doubt | OPEN | yes |
| [AGT-Q-017](#agt-q-017) | Momento jurídico emissão vs aceitação | normative_or_process_doubt | OPEN | yes |
| [AGT-Q-018](#agt-q-018) | Instabilidade portais (≠ API FE) | network_or_portal_incident | OPEN | yes |

---

## AGT-Q-001

| Campo | Valor |
|---|---|
| ID | AGT-Q-001 |
| Data primeira observação (UTC) | 2026-07-20 |
| Categoria | missing_credential_or_info |
| Severidade | high |
| Ambiente/serviço | FE HML/PRD (Basic Auth) |
| Estado | OPEN |
| Fonte / secção | GAP-006; [`official-access-plan.md`](official-access-plan.md) |
| Pronto para email | yes |

**Facto observado:** Credenciais Basic Auth oficiais para HML/PRD ainda não foram fornecidas ao projecto.

**Impacto no projecto:** Impede cliente FE contra AGT real (`RM-FE-001`); não trava simulador/contratos/testes internos.

**Mitigação actual:** Classificado `BLOQUEADO_EXTERNO` / `ADIADO` em [`agt-dependencies.md`](agt-dependencies.md); sem inventar credenciais.

**Pergunta exacta a enviar à AGT:** Qual o canal e os requisitos para emissão de credenciais Basic Auth de homologação e produção da API de Facturação Electrónica, e qual o formato exacto do cabeçalho Authorization esperado?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** GAP-006 · RM-FE-001 · DEC-DEL-002

---

## AGT-Q-002

| Campo | Valor |
|---|---|
| ID | AGT-Q-002 |
| Data primeira observação (UTC) | 2026-07-20 |
| Categoria | missing_credential_or_info |
| Severidade | high |
| Ambiente/serviço | Portal produtores / Modelo 8 |
| Estado | OPEN |
| Fonte / secção | GAP-003; área autenticada produtores |
| Pronto para email | yes |

**Facto observado:** Registo do produtor, Modelo 8 e valor oficial de `softwareValidationNo` ainda não fornecidos/confirmados ao projecto.

**Impacto no projecto:** Bloqueia dossier de certificação/`RM-CERT-*`; não trava catálogo/domínio/CI.

**Mitigação actual:** Placeholders sintéticos apenas em testes; nunca apresentados como certificado.

**Pergunta exacta a enviar à AGT:** Quais os passos oficiais para registo de produtor de software, obtenção/preenchimento do Modelo 8, e emissão do identificador `softwareValidationNo` a usar nos pedidos FE?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** GAP-003 · RM-CERT-001 · AGT-Q-012

---

## AGT-Q-003

| Campo | Valor |
|---|---|
| ID | AGT-Q-003 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | missing_credential_or_info |
| Severidade | medium |
| Ambiente/serviço | Identidades RSA de teste (workbook) |
| Estado | OPEN |
| Fonte / secção | Proveniência workbook teste; RM-FEFIX-001/002 |
| Pronto para email | yes |

**Facto observado:** Foi recebido um workbook RSA de teste sem instruções oficiais sobre finalidade das identidades, perfil a usar, se chaves públicas já estão registadas, operações autorizadas, nem validade/rotação/revogação. (Sem NIF/nomes/fingerprints neste registo.)

**Impacto no projecto:** Custódia e assinatura de teste locais possíveis; uso operacional contra AGT indeterminado.

**Mitigação actual:** Inventário sanitizado + custódia opaca em memória; CI só com workbook sintético; ≠ certificados/Basic Auth.

**Pergunta exacta a enviar à AGT:** Para o material de identidades RSA de teste já partilhado: (1) qual a finalidade exacta de cada perfil; (2) que identidade usar em cada operação FE; (3) as chaves públicas já estão registadas nos sistemas AGT; (4) que operações estão autorizadas; (5) qual a política de validade, rotação e revogação?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** RM-FEFIX-001 · RM-FEFIX-002 · GAP-006

---

## AGT-Q-004

| Campo | Valor |
|---|---|
| ID | AGT-Q-004 |
| Data primeira observação (UTC) | 2026-07-28 |
| Categoria | documentation_conflict |
| Severidade | high |
| Ambiente/serviço | FE HML paths |
| Estado | OPEN |
| Fonte / secção | [C-FE-001](../../compliance/derived/conflicts/C-FE-001-fe-endpoint-path-inconsistency.md) |
| Pronto para email | yes |

**Facto observado:** Snapshots FE mostram inconsistência de paths `/sigt/fe/v1/…` vs `/sigt/fe/ws/v1/…` entre serviços/ambientes.

**Impacto no projecto:** Cliente FE real bloqueado para serviços conflituosos; mitigação fail-closed `internal/fepath`.

**Mitigação actual:** Ver C-FE-001; não inventar path «correcto».

**Pergunta exacta a enviar à AGT:** Qual é o prefixo de path canónico por ambiente (HML e PRD) e por operação FE — `/sigt/fe/v1` ou `/sigt/fe/ws/v1` — e como devem ser interpretadas as divergências publicadas no snapshot HML?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** C-FE-001 · GAP-006 · RM-FE-001

---

## AGT-Q-005

| Campo | Valor |
|---|---|
| ID | AGT-Q-005 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | medium |
| Ambiente/serviço | JWS FE protected header |
| Estado | OPEN |
| Fonte / secção | [C-FE-JWS-TYP-001](../../compliance/derived/conflicts/C-FE-JWS-TYP-001-typ-jwt-vs-jose.md) |
| Pronto para email | yes |

**Facto observado:** `estrutura.html` ilustra `typ=JWT`; exemplos compactos de serviços decodificam `typ=JOSE`. A documentação não autoriza omitir `typ`.

**Impacto no projecto:** **Zero** perfis wire JWS AGT activos (`RM-FEFIX-003`); motor genérico sem `typ` por defeito apenas como primitiva.

**Mitigação actual:** Fail-closed em `feprofile` Sign*; builders de payload separados; ver matriz FE-JWS.

**Pergunta exacta a enviar à AGT:** No JWS Compact Serialization RS256 da FE, qual o valor obrigatório do parâmetro `typ` no protected header (`JWT`, `JOSE`, outro, ou omissão explícita permitida), e aplica-se a todas as assinaturas (`jwsSoftwareSignature`, `jwsDocumentSignature`, `jwsSignature`)?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** C-FE-JWS-TYP-001 · RM-FEFIX-003 · RM-FE-002

---

## AGT-Q-006

| Campo | Valor |
|---|---|
| ID | AGT-Q-006 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | high |
| Ambiente/serviço | registarFactura / jwsDocumentSignature |
| Estado | OPEN |
| Fonte / secção | [C-FE-JWS-DOC-001](../../compliance/derived/conflicts/C-FE-JWS-DOC-001-document-totals-sample.md) |
| Pronto para email | yes |

**Facto observado:** Divergência entre campos documentados e exemplo de assinatura de documento / `documentTotals`.

**Impacto no projecto:** Perfil `registar_document` bloqueado.

**Mitigação actual:** Ver C-FE-JWS-DOC-001; sem intersecção mínima.

**Pergunta exacta a enviar à AGT:** Quais são exactamente os campos, nomes JSON, ordem/serialização e significado de `documentTotals` no payload de `jwsDocumentSignature` para `registarFactura`?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** C-FE-JWS-DOC-001 · RM-FEFIX-003 · RM-FE-002

---

## AGT-Q-007

| Campo | Valor |
|---|---|
| ID | AGT-Q-007 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | medium |
| Ambiente/serviço | solicitarSerie |
| Estado | OPEN |
| Fonte / secção | [C-FE-JWS-REQ-001](../../compliance/derived/conflicts/C-FE-JWS-REQ-001-solicitar-serie-fields.md) |
| Pronto para email | yes |

**Facto observado:** Tabela de campos ≠ bloco «Payload assinatura» em `solicitarSerie`.

**Impacto no projecto:** Perfil request bloqueado.

**Mitigação actual:** Ver C-FE-JWS-REQ-001.

**Pergunta exacta a enviar à AGT:** Qual é o conjunto exacto de claims do JWS de `solicitarSerie` (nomes e obrigatoriedade), reconciliando a tabela de campos com o bloco «Payload assinatura»?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** C-FE-JWS-REQ-001 · RM-FEFIX-003

---

## AGT-Q-008

| Campo | Valor |
|---|---|
| ID | AGT-Q-008 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | medium |
| Ambiente/serviço | listarSeries |
| Estado | OPEN |
| Fonte / secção | [C-FE-JWS-REQ-002](../../compliance/derived/conflicts/C-FE-JWS-REQ-002-listar-series-fields.md) |
| Pronto para email | yes |

**Facto observado:** Tabela de campos ≠ bloco «Payload assinatura» em `listarSeries`.

**Impacto no projecto:** Perfil request bloqueado.

**Mitigação actual:** Ver C-FE-JWS-REQ-002.

**Pergunta exacta a enviar à AGT:** Qual é o conjunto exacto de claims do JWS de `listarSeries`, reconciliando a tabela de campos com o bloco «Payload assinatura»?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** C-FE-JWS-REQ-002 · RM-FEFIX-003

---

## AGT-Q-009

| Campo | Valor |
|---|---|
| ID | AGT-Q-009 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | medium |
| Ambiente/serviço | validarDocumento |
| Estado | OPEN |
| Fonte / secção | [C-FE-JWS-REQ-003](../../compliance/derived/conflicts/C-FE-JWS-REQ-003-validar-documento-fields.md) |
| Pronto para email | yes |

**Facto observado:** Tabela de campos ≠ bloco «Payload assinatura» em `validarDocumento`.

**Impacto no projecto:** Perfil request bloqueado.

**Mitigação actual:** Ver C-FE-JWS-REQ-003.

**Pergunta exacta a enviar à AGT:** Qual é o conjunto exacto de claims do JWS de `validarDocumento`, reconciliando a tabela de campos com o bloco «Payload assinatura»?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** C-FE-JWS-REQ-003 · RM-FEFIX-003

---

## AGT-Q-010

| Campo | Valor |
|---|---|
| ID | AGT-Q-010 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | medium |
| Ambiente/serviço | listarFacturas |
| Estado | OPEN |
| Fonte / secção | [C-FE-JWS-REQ-004](../../compliance/derived/conflicts/C-FE-JWS-REQ-004-listar-facturas-payload-block.md) |
| Pronto para email | yes |

**Facto observado:** Página `listarFacturas` sem bloco «Payload assinatura» cruzável de forma inequívoca.

**Impacto no projecto:** Perfil request bloqueado.

**Mitigação actual:** Ver C-FE-JWS-REQ-004.

**Pergunta exacta a enviar à AGT:** Quais são exactamente os claims do JWS exigido em `listarFacturas`, e onde está a especificação autoritativa do payload de assinatura?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** C-FE-JWS-REQ-004 · RM-FEFIX-003 · C-FE-001

---

## AGT-Q-011

| Campo | Valor |
|---|---|
| ID | AGT-Q-011 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | medium |
| Ambiente/serviço | registarFactura |
| Estado | OPEN |
| Fonte / secção | FE-RNG-031; snapshot REGISTAR; matriz FE-JWS |
| Pronto para email | yes |

**Facto observado:** FE-RNG-031 menciona `jwsSignature`, mas schema/exemplo de entrada de `registarFactura` não fecham campo/payload exactos para esse JWS de pedido.

**Impacto no projecto:** Perfil `registar_request_jwsSignature` bloqueado.

**Mitigação actual:** Não emitir `jwsSignature` de pedido só porque o código FE-RNG o nomeia.

**Pergunta exacta a enviar à AGT:** Em `registarFactura`, o campo `jwsSignature` (além de `jwsSoftwareSignature` / `jwsDocumentSignature`) é obrigatório? Se sim, qual o payload exacto e o protected header?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** FE-RNG-031 · RM-FEFIX-003 · AGT-Q-005 · AGT-Q-006

---

## AGT-Q-012

| Campo | Valor |
|---|---|
| ID | AGT-Q-012 |
| Data primeira observação (UTC) | 2026-08-16 |
| Categoria | documentation_conflict |
| Severidade | medium |
| Ambiente/serviço | softwareInfo / processo produtores |
| Estado | OPEN |
| Fonte / secção | Snapshots FE; GAP-003 |
| Pronto para email | yes |

**Facto observado:** Distinção entre claim `softwareValidationNumber` (assinatura software) e campo/processo `softwareValidationNo` (body/registo) sem mapeamento oficial fechado.

**Impacto no projecto:** Conservar nomes distintos; sem conversão automática; sem tratar placeholder como certificado.

**Mitigação actual:** Builders usam só o nome de claim documentado no snapshot; valor sintético em testes.

**Pergunta exacta a enviar à AGT:** `softwareValidationNumber` (claim JWS) e `softwareValidationNo` (campo/processo) são o mesmo identificador? Se não, qual a relação oficial e qual o valor a usar em cada contexto?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** AGT-Q-002 · GAP-003 · RM-FEFIX-003

---

## AGT-Q-013

| Campo | Valor |
|---|---|
| ID | AGT-Q-013 |
| Data primeira observação (UTC) | 2026-07-26 |
| Categoria | normative_or_process_doubt |
| Severidade | high |
| Ambiente/serviço | Custódia criptográfica contribuinte |
| Estado | OPEN |
| Fonte / secção | GAP-013; snapshot GESTAO |
| Pronto para email | yes |

**Facto observado:** Sem confirmação oficial de que um módulo fiscal externo possa custodiar/usar a chave privada do contribuinte.

**Impacto no projecto:** `TaxpayerKeyRef` definitivo e Edge com chave real bloqueados; testes com chave efémera permitidos.

**Mitigação actual:** Constraints `DEC-PROD-012`; ver [`agt-dependencies.md`](agt-dependencies.md).

**Pergunta exacta a enviar à AGT:** Um módulo fiscal externo (cloud e/ou Linux local), distinto do POS, pode custodiar e usar a chave privada do contribuinte para assinar pedidos/documentos FE? Em que condições e com que controlos?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** GAP-013 · DEC-REG-KEY-CUSTODY · DEC-SEC-EDGE-KEYS

---

## AGT-Q-014

| Campo | Valor |
|---|---|
| ID | AGT-Q-014 |
| Data primeira observação (UTC) | 2026-07-20 |
| Categoria | missing_credential_or_info |
| Severidade | high |
| Ambiente/serviço | SAF-T (AO) |
| Estado | OPEN |
| Fonte / secção | GAP-004; XSD ASSOFT `pending_validation` |
| Pronto para email | yes |

**Facto observado:** XSD SAF-T AO autenticado pela AGT e respectiva versão/vigência ainda não confirmados (cópia ASSOFT no Git é `pending_validation`).

**Impacto no projecto:** Gerador SAF-T de produção não pode ser declarado conforme.

**Mitigação actual:** Testes schema com XSD versionado; sem promoção a `confirmed_normative`.

**Pergunta exacta a enviar à AGT:** Qual o XSD SAF-T (AO) oficial autenticado pela AGT, o número de versão e o período de vigência, e onde obter o pacote autoritativo?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** GAP-004 · AO-SAF-001 · AO-SAF-002

---

## AGT-Q-015

| Campo | Valor |
|---|---|
| ID | AGT-Q-015 |
| Data primeira observação (UTC) | 2026-07-20 |
| Categoria | missing_credential_or_info |
| Severidade | high |
| Ambiente/serviço | Homologação / certificação |
| Estado | OPEN |
| Fonte / secção | GAP-010 |
| Pronto para email | yes |

**Facto observado:** Vetores e resultados oficiais de testes de homologação AGT não estão disponíveis ao projecto.

**Impacto no projecto:** Impede declaração de conformidade alinhada a harness oficial.

**Mitigação actual:** Harness/simulador interno ≠ GAP-010 fechado.

**Pergunta exacta a enviar à AGT:** Quais os vetores de teste oficiais e o formato dos resultados de homologação que um produtor deve reproduzir, e como são disponibilizados?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** GAP-010 · RM-CERT-* · RM-FE-005

---

## AGT-Q-016

| Campo | Valor |
|---|---|
| ID | AGT-Q-016 |
| Data primeira observação (UTC) | 2026-07-20 |
| Categoria | normative_or_process_doubt |
| Severidade | high |
| Ambiente/serviço | Contingência / Edge offline |
| Estado | OPEN |
| Fonte / secção | GAP-009; DEC-REG-004 |
| Pronto para email | yes |

**Facto observado:** Regras oficiais de contingência/faturação offline certificável não estão fechadas.

**Impacto no projecto:** Outbox/reenvio interno permitido sem rotular «emissão offline certificada».

**Mitigação actual:** `DEC-PROD-010`; sem `AO-OFF-*` confirmados por inferência indevida.

**Pergunta exacta a enviar à AGT:** Quais são as regras oficiais de contingência e faturação offline certificável (incluindo prazos, marcação documental e obrigações de reenvio)?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** GAP-009 · DEC-REG-004 · AO-OFF-001

---

## AGT-Q-017

| Campo | Valor |
|---|---|
| ID | AGT-Q-017 |
| Data primeira observação (UTC) | 2026-07-26 |
| Categoria | normative_or_process_doubt |
| Severidade | high |
| Ambiente/serviço | Semântica jurídica documento |
| Estado | OPEN |
| Fonte / secção | DEC-API-004; DEC-PROD-009 |
| Pronto para email | yes |

**Facto observado:** Momento jurídico de emissão vs aceitação AGT não está fechado por orientação escrita.

**Impacto no projecto:** Estados produto internos (`sealed_locally`…`accepted`) ≠ semântica jurídica final OpenAPI.

**Mitigação actual:** Não declarar emissão fiscal «aceite AGT» só com selo local.

**Pergunta exacta a enviar à AGT:** Em que momento um documento se considera juridicamente emitido perante a AGT — no selo/assinatura local do software, na recepção do pedido, ou apenas após aceitação explícita pela AGT — e como deve o software comunicar cada estado ao contribuinte?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** DEC-API-004 · DEC-PROD-009 · RM-FEFIX-005

---

## AGT-Q-018

| Campo | Valor |
|---|---|
| ID | AGT-Q-018 |
| Data primeira observação (UTC) | 2026-07-20 |
| Categoria | network_or_portal_incident |
| Severidade | low |
| Ambiente/serviço | Portais AGT/MINFIN (contribuinte/parceiro) — **≠** API FE |
| Estado | OPEN |
| Fonte / secção | GAP-011; [`regulatory-gaps.md`](regulatory-gaps.md) |
| Pronto para email | yes |

**Facto observado:** Instabilidade/manutenção/timeout de portais web oficiais observada e documentada em 2026-07-20. **Não** confundir com erro da API FE (sem chamadas reais à API FE neste registo).

**Impacto no projecto:** Orientação operacional intermitente; não prova falha do conector FE.

**Mitigação actual:** Reconsulta + arquivo permitido quando acessível; separar de incidentes API.

**Pergunta exacta a enviar à AGT:** Existe canal oficial de estado/manutenção dos portais e da API FE, e como devem os produtores distinguir indisponibilidade de portal vs indisponibilidade da API?

**Resposta/evidência AGT:** _(vazio)_

**Relacionados:** GAP-011

---

## Perguntas abertas prontas para comunicação à AGT

Compostas apenas a partir de itens `OPEN` com `Pronto para email = yes`, sanitizadas. **Não enviadas** automaticamente.

1. **AGT-Q-001:** Qual o canal e os requisitos para emissão de credenciais Basic Auth de homologação e produção da API de Facturação Electrónica, e qual o formato exacto do cabeçalho Authorization esperado?
2. **AGT-Q-002:** Quais os passos oficiais para registo de produtor de software, obtenção/preenchimento do Modelo 8, e emissão do identificador `softwareValidationNo` a usar nos pedidos FE?
3. **AGT-Q-003:** Para o material de identidades RSA de teste já partilhado: (1) finalidade exacta de cada perfil; (2) identidade por operação FE; (3) chaves públicas já registadas; (4) operações autorizadas; (5) validade/rotação/revogação?
4. **AGT-Q-004:** Qual o prefixo de path canónico por ambiente e operação — `/sigt/fe/v1` ou `/sigt/fe/ws/v1`?
5. **AGT-Q-005:** Qual o `typ` obrigatório no protected header JWS FE (`JWT`, `JOSE`, outro, ou omissão permitida)?
6. **AGT-Q-006:** Campos exactos de `jwsDocumentSignature` / `documentTotals` em `registarFactura`?
7. **AGT-Q-007:** Claims exactos do JWS de `solicitarSerie`?
8. **AGT-Q-008:** Claims exactos do JWS de `listarSeries`?
9. **AGT-Q-009:** Claims exactos do JWS de `validarDocumento`?
10. **AGT-Q-010:** Claims exactos do JWS de `listarFacturas` e especificação autoritativa?
11. **AGT-Q-011:** `jwsSignature` em `registarFactura` é obrigatório e com que payload/header?
12. **AGT-Q-012:** Relação oficial entre `softwareValidationNumber` e `softwareValidationNo`?
13. **AGT-Q-013:** Módulo fiscal externo pode custodiar/usar a chave privada do contribuinte?
14. **AGT-Q-014:** XSD SAF-T (AO) oficial autenticado, versão e vigência?
15. **AGT-Q-015:** Vetores e formato de resultados oficiais de homologação?
16. **AGT-Q-016:** Regras oficiais de contingência/faturação offline certificável?
17. **AGT-Q-017:** Momento jurídico de emissão vs aceitação AGT?
18. **AGT-Q-018:** Canal de estado/manutenção de portais vs API FE?

---

## Fora deste registo (relatório interno)

Não listar aqui: trailing whitespace, ShellCheck, falhas de mocks, GitHub Actions, ou bugs internos sem dependência AGT.
