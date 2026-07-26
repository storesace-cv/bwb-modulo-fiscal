# Decisões em aberto — Fase 0

**Data:** 2026-07-20
**Regra:** nenhuma decisão abaixo autoriza inventar regras fiscais nem alterar `ASM-REG-001`.
**Plano associado:** [phase-0-execution-plan.md](phase-0-execution-plan.md)

## Como usar este documento

Para cada decisão: opções, vantagens, riscos, recomendação ou decisão, responsável e prazo máximo.
Estados: `aberta` | `recomendada` | `decidida` | `bloqueada-por-lacuna`.

---

## DEC-REG-001 — Validação jurídica de `ASM-REG-001`

| Campo | Valor |
|---|---|
| Estado | aberta |
| Tipo | Regulatória / produto |
| Prazo máximo | Durante as 2–4 semanas internas (pedido enviado; resposta pode atrasar) |
| Responsável | Compliance + Jurídico (decisão final: Direção) |

**Contexto:** `ASM-REG-001` é premissa de produto (README, ADR-0001): certificação do módulo externo dispensa validação individual de cada POS. Não é conclusão jurídica.

**Opções:**

1. Manter premissa e obter confirmação escrita/processual da AGT.
2. Manter premissa e planear processo de registo por integrador sem mudar o domínio (previsto no ADR-0001).
3. Rever o modelo comercial (fora do âmbito técnico imediato).

| Opção | Vantagens | Riscos |
|---|---|---|
| 1 | Alinha produto e certificação | AGT pode não confirmar; atraso externo |
| 2 | Arquitetura já preparada | Complexidade operacional adicional |
| 3 | Clareza comercial | Impacto em roadmap e integrações |

**Recomendação:** opção 1 em paralelo com desenho compatível com opção 2; **não alterar** a premissa nesta fase.

**Evidência para fechar:** resposta AGT ou ata interna com risco aceite e plano B.

---

## DEC-REG-002 — Fonte normativa do Decreto Executivo n.º 74/19

| Campo | Valor |
|---|---|
| Estado | parcialmente mitigada (arquivo + OCR `reviewed`; fecho AO-* / URL estável ainda abertos) |
| Tipo | Regulatória |
| Prazo máximo | Paralelo externo; waiver se indisponível no gate interno |
| Responsável | Compliance |

**Opções:**

1. Arquivar PDF oficial do Diário da República / MINFIN / AGT + rectificação, com hash.
2. Continuar só com documentação técnica pública FE (insuficiente para requisitos de validação de software).
3. Usar a proposta em `local/docs/minfin055809.pdf` como norma.

| Opção | Vantagens | Riscos |
|---|---|---|
| 1 | Conformidade com [sources.md](../01-compliance/sources.md) | Dependência de acesso/arquivo |
| 2 | Desbloqueia conector/simulador | Lacunas em assinatura, menções, séries |
| 3 | Texto disponível localmente | **Inválida:** o ficheiro é «Proposta de Decreto Executivo» (2018), não o 74/19 publicado |

**Recomendação:** opção 1. A opção 3 é **rejeitada** como fonte normativa.

**Estado 2026-07-26:** opção 1 **cumprida no arquivo** — DE 74/19 `5b63c80e…` + Rect. 10/19 v2 `b3db14e2…` no privado (`c8a4e6e…`) com OCR `reviewed`; citações provisórias em Citação F / matriz de tipos. **Ainda aberto:** URL oficial estável Imprensa Nacional; promoção a AO-* confirmados; confronto PDF página a página residual.

**Evidência:** SHA-256, paths privados, Rect. v2 (histórico `77b77f01…` só diagnostics).

---

## DEC-PROD-001 — Organização do catálogo documental (5 grupos SAF-T)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / catálogo |
| Prazo máximo | — |
| Responsável | Product Owner |
| Decisão | 2026-07-26 |

**Decisão:** o catálogo documental do produto organiza-se **exactamente** em **5 grupos funcionais** alinhados a `SourceDocuments` do SAF-T (AO) — camada **L3** (estrutura), **não** L1 legal nem L4 `documentType` FE:

| # | Grupo funcional (produto) | Estrutura L3 XSD | Tipo L2 associado |
|---|---|---|---|
| 1 | Vendas | `SalesInvoices` | `InvoiceType` |
| 2 | Movimentação de mercadorias | `MovementOfGoods` | `MovementType` |
| 3 | Conferência / trabalho | `WorkingDocuments` | `WorkType` |
| 4 | Pagamentos / recibos | `Payments` | `SAFTAOPaymentType` / `PaymentType` |
| 5 | Compras | `PurchaseInvoices` | `PurchaseType` |

**Não autoriza:** bijecção com L4 FE; alargamento OpenAPI sem revisão; truncar o modelo ao slice (`DEC-PROD-014`); promoção de `AO-DOC-*`.

**Evidência:** inventário em [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md) secção C; XSD `SAFTAO1.01_01.xsd` (`e9a938e1…`, `pending_validation`).

---

## DEC-PROD-002 — Critério de inclusão no catálogo (canais SAF-T / FE)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / catálogo |
| Prazo máximo | — |
| Responsável | Product Owner |
| Decisão | 2026-07-26 |

**Decisão:** o catálogo inclui **apenas** tipos utilizáveis em pelo menos um destes canais:

1. **SAF-T (AO)** — código presente num enum L2 do XSD (`InvoiceType`, `MovementType`, `WorkType`, `SAFTAOPaymentType`, `PurchaseType`) sob um dos 5 grupos L3 (`DEC-PROD-001`);
2. **FE AGT** — código presente em `documentType` L4 (DE 683/25 + HTML HML oficial catalogado);
3. **Ambos** — (1) e (2).

**Excluir** do catálogo de tipos: qualquer figura/código que **não** sirva nem SAF-T nem FE (ex.: rótulos L1 sem código de canal; modalidades; inventários comerciais sem enum).

**Não autoriza:** inventar códigos para «completar» um canal; bijecção L4↔L2; fecho MVP (`DEC-REG-003`); promoção `AO-DOC-*`.

**Nota:** `FA` (FE sem `InvoiceType`) **inclui-se** (canal FE). Tipos só em `MovementType`/`WorkType`/`PurchaseType` **incluem-se** (canal SAF-T).

**Evidência:** [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md) secção C.7.

---

## DEC-PROD-003 — Activação de grupos e tipos na configuração POS

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / configuração |
| Prazo máximo | — |
| Responsável | Product Owner |
| Decisão | 2026-07-26 |

**Decisão:** a configuração por contexto POS (integrador / estabelecimento / terminal — âmbito exacto na modelação) permite:

1. **Activar ou desactivar um grupo inteiro** dos 5 grupos `DEC-PROD-001`;
2. **Activar ou desactivar tipos individuais** dentro de um grupo activo (tipos do catálogo `DEC-PROD-002` apenas).

**Regras fail-closed**

| Regra | Implicação |
|---|---|
| Grupo desactivo | Todos os tipos desse grupo ficam indisponíveis para emissão via esse POS |
| Tipo activo | Exige grupo pai activo |
| Fora do catálogo | Impossível activar (não inventar tipos) |
| Pedido POS de tipo inactivo / grupo inactivo | Rejeição pelo módulo fiscal (autoridade) — o POS não emite |
| Activar ≠ obrigação legal | Configuração de produto ≠ norma; não confirma `AO-DOC-*` |
| Activar ≠ OpenAPI slice | Exposição contratual faseada (`DEC-REG-003`); modelo completo = `DEC-PROD-014` |

**Não autoriza:** o POS a numerar/selar; bypass de canais SAF-T/FE; inventar mapeamento L4↔L2.

**Dependências de implementação:** modelo de scope (empresa/estabelecimento/terminal/integrador); API/admin de configuração; enforcement na emissão. Fora do vertical slice até existir tarefa de configuração. Adesão FE do contribuinte: `DEC-PROD-004`.

**Evidência de decisão:** este registo; inventário de grupos/tipos em [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md).

---

## DEC-PROD-004 — Adesão FE (contribuinte) vs séries/config (estabelecimento)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / domínio |
| Prazo máximo | — |
| Responsável | Product Owner + Domínio |
| Decisão | 2026-07-26 |

**Decisão:**

1. **Adesão à FE AGT** pertence ao **cliente/contribuinte** vinculado ao **NIF** (`Taxpayer`) — não ao estabelecimento, não ao POS, não ao integrador.
2. **Estabelecimento** mantém **séries** e **configuração própria** (inclui activação de grupos/tipos `DEC-PROD-003` no seu âmbito).
3. Estado de adesão FE é **máquina de estados**, **não** booleano:

| Estado | Significado (produto) |
|---|---|
| `not_enrolled` | Sem adesão FE registada para o contribuinte (nesse ambiente) |
| `pending` | Adesão iniciada / em curso junto da AGT ou processo interno — **não** emitir FE como `active` |
| `active` | Adesão utilizável para operações FE permitidas pelo ambiente |
| `suspended` | Adesão existiu mas está suspensa — emissão FE bloqueada até reactivação |

**Invariantes**

| Regra | Implicação |
|---|---|
| 1 NIF → 1 `Taxpayer` | Adesão FE não se duplica por estabelecimento |
| Estabelecimento sem NIF próprio | Herda elegibilidade FE do `Taxpayer` pai; séries/config são locais ao estabelecimento |
| `active` no contribuinte | Condição necessária (não suficiente) para emissão FE; ainda exige séries/config activas no estabelecimento + regras oficiais |
| `not_enrolled` / `pending` / `suspended` | Pedidos FE rejeitados (fail-closed) |
| Ambiente | Estado modelado por `Environment` (`homologation` \| `production`) — sem partilha entre ambientes |
| ≠ booleano | Proibido `fe_enabled: true/false` como modelo canónico |

**Não autoriza:** inventar procedimento AGT de adesão; confirmar `AO-AGT-*` / `AO-ID-*`; colocar Basic Auth produtor no contribuinte (`ProducerCredential` permanece plataforma).

**Evidência:** este registo; [`domain-model.md`](../04-domain/domain-model.md). Disponibilidade efectiva: `DEC-PROD-005`.

---

## DEC-PROD-005 — Disponibilidade efectiva para emissão

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / domínio |
| Prazo máximo | — |
| Responsável | Product Owner + Domínio |
| Decisão | 2026-07-26 |

**Decisão:** um tipo só está **efectivamente disponível** para emissão quando **todas** as condições seguintes são verdadeiras (conjunção fail-closed — **AND**):

| # | Gate | Fonte / decisão | Falha ⇒ |
|---|---|---|---|
| 1 | **Tipo canónico** no catálogo | `DEC-PROD-001` + `DEC-PROD-002` (grupo L3 + canal SAF-T e/ou FE) | rejeitar (tipo inexistente / excluído) |
| 2 | **Grupo e tipo activos** no contexto POS/estabelecimento | `DEC-PROD-003` | rejeitar (inactivo) |
| 3 | **Regime / NIF** do contribuinte elegível para o tipo | `Taxpayer` + regime fiscal aplicável | rejeitar (regime/NIF) |
| 4 | **Adesão FE** adequada quando o canal FE é exigido | `DEC-PROD-004` — tipicamente `active` | rejeitar (`not_enrolled`/`pending`/`suspended`) |
| 5 | **Série AGT válida** quando o canal/regra a exige | série do estabelecimento (`solicitarSerie` / regras oficiais) | rejeitar (série ausente/inválida) |
| 6 | **Ambiente** correcto | `Environment` homologation ≠ production | rejeitar (ambiente) |
| 7 | **Restrição sectorial** | ex. tipos segurador só se sector/regime o permitir | rejeitar (sector) |

**Fórmula (produto):**

`available = canonical ∧ group_type_active ∧ regime_nif ∧ fe_enrollment_ok ∧ series_ok_when_required ∧ environment_ok ∧ sector_ok`

**Regras**

- Qualquer gate falso ⇒ **indisponível**; sem emissão parcial.
- «Série quando exigida»: tipos/canais que a norma ou FE obrigam a série AGT; se não exigida, o gate 5 é N/A (não inventar exigência).
- Canal só-SAF-T sem FE: gate 4 pode ser N/A para esse pedido; canal FE: gate 4 obrigatório.
- Activação POS (gate 2) **não** substitui gates 3–7.
- Esta decisão **não** confirma requisitos `AO-*` nem fecha `DEC-REG-003` (quais tipos no MVP).

**Evidência:** este registo; [`domain-model.md`](../04-domain/domain-model.md). Routing por canal: `DEC-PROD-006`.

---

## DEC-PROD-006 — Routing por canal (SAF-T vs FE)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / domínio |
| Prazo máximo | — |
| Responsável | Product Owner + Domínio |
| Decisão | 2026-07-26 |

**Decisão:** o canal de saída depende da adesão FE (`DEC-PROD-004`) e do canal do tipo (`DEC-PROD-002`):

| Situação | Comportamento |
|---|---|
| **Sem FE activa** (`not_enrolled` \| `pending` \| `suspended`) | Apenas fluxos **SAF-T** aplicáveis aos tipos com canal `SAF-T` ou `ambos` (sujeito a `DEC-PROD-005`). **Proibido** chamar endpoint FE AGT. |
| **Com FE activa** (`active`) | Apenas documentos **elegíveis no endpoint AGT** **e** autorizados para o contribuinte/série (gates `DEC-PROD-005`). |
| Tipo **SAF-T-only** | **Nunca** vai ao endpoint FE — mesmo com adesão `active`. |
| Tipo **FE-only** (ex. `FA`) | Continua possível **só** no canal FE (quando FE `active` + restantes gates); **não** inventar estrutura SAF-T. |
| Tipo **ambos** | Pode usar FE quando `active` e elegível; exportação/arquivo SAF-T conforme regras L2/L3 do grupo — **sem** bijecção L4↔L2. |

**Fail-closed**

- Pedido FE para tipo SAF-T-only ⇒ rejeitar.
- Pedido FE sem adesão `active` ⇒ rejeitar.
- Pedido FE para tipo não elegível no endpoint / série não autorizada ⇒ rejeitar.
- Sem FE activa: não simular transmissão FE «offline» como substituto (contingência = `DEC-REG-004` / `AO-OFF-*`, não esta decisão).

**Não autoriza:** inventar mapeamento FE→`InvoiceType` para FE-only; fechar C-DOC-003; confirmar `AO-AGT-*`.

**Evidência:** este registo; [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md) C.7; [`domain-model.md`](../04-domain/domain-model.md). Conceito canónico + adaptadores: `DEC-PROD-007`.

---

## DEC-PROD-007 — Conceito canónico e adaptadores por canal

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / API / domínio |
| Prazo máximo | — |
| Responsável | Product Owner + API Owner + Domínio |
| Decisão | 2026-07-26 |

**Decisão:**

1. **Um conceito canónico** interno por tipo documental do catálogo (identidade de produto estável — não é L4 FE nem L2 SAF-T crus).
2. **Adaptadores por canal** traduzem o canónico → representação do canal (`documentType` FE L4, enum/estrutura SAF-T L2/L3, futuros canais) sob `DEC-PROD-006` e disponibilidade `DEC-PROD-005`.
3. O **POS envia código próprio** (alias/integrador) que o módulo resolve via **mapping configurado** para o conceito canónico.
4. **Proibido:** POS enviar código fiscal cru (FE `FT`/`NC`/… ou SAF-T `InvoiceType`/…) **sem** mapping explícito para o canónico.

**Fail-closed**

| Caso | Comportamento |
|---|---|
| Código POS sem mapping | rejeitar |
| Código fiscal cru no pedido POS | rejeitar (não «adivinhar» canónico) |
| Canónico sem adaptador para o canal exigido | indisponível nesse canal |
| Mapping ambíguo / multi-alvo sem desambiguação | rejeitar |

**Não autoriza:** bijecção canónico↔L4↔L2; o POS a escolher L3 (`SalesInvoices` vs `Payments`); alargar OpenAPI MVP sem `DEC-REG-003`; inventar códigos fiscais.

**Nota:** aliases OpenAPI actuais (`invoice` / `credit_note`) são candidatos a códigos de produto/mapping — **não** códigos AGT crus; formalização no contrato na revisão autorizada.

**Evidência:** este registo; [`domain-model.md`](../04-domain/domain-model.md); camadas L1–L4 em [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md). Modelo multi-POS: `DEC-PROD-008`.

---

## DEC-PROD-008 — Modelo POS: módulo autoridade; multi-POS via API

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / arquitectura |
| Prazo máximo | — |
| Responsável | Product Owner + Arquitectura |
| Decisão | 2026-07-26 |

**Decisão:**

1. O **módulo fiscal** é a **única autoridade** de emissão e numeração fiscal (série/número, selagem, artefactos).
2. **Vários POS** (terminais / integradores) acedem **apenas via API** do módulo — sem autoridade fiscal no POS.
3. Alinha e **não altera** [`ADR-0001`](../02-architecture/adrs/ADR-0001-external-fiscal-module.md) nem a premissa `ASM-REG-001` (validação jurídica AGT permanece `DEC-REG-001`).

**Fail-closed**

| Proibido no POS | Obrigatório no módulo |
|---|---|
| Reservar / atribuir número fiscal | Atribuir série e número |
| Emitir / «selar» documento fiscal localmente como autoridade | Validar intenção, persistir, assinar conforme regras |
| Partilhar série entre escritores sem coordenação do módulo | Enforcement de exclusividade de escrita por série |
| Tratar HTTP 2xx como aceite AGT | Devolver estado fiscal explícito |

**Consequências de produto**

- N POS → 1 módulo (cloud e/ou Edge); identidade por `Integrator` / `Terminal` / credencial API.
- Configuração (`DEC-PROD-003`–`005`) e mapping (`DEC-PROD-007`) aplicam-se por contexto; numeração continua centralizada.
- Edge: um escritor (`DEC-PROD-011` / `DEC-OPS-001`); POS só API local.

**Evidência:** este registo; ADR-0001; [`scope.md`](../00-product/scope.md); [`domain-model.md`](../04-domain/domain-model.md). Estados documentais: `DEC-PROD-009`.

---

## DEC-PROD-009 — Estados do documento (ciclo de vida vs AGT)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / domínio / API |
| Prazo máximo | — |
| Responsável | Product Owner + API Owner |
| Decisão | 2026-07-26 |

**Decisão:** estados de produto do documento fiscal (identificadores canónicos API em inglês; rótulos PT):

| Estado | Rótulo | Significado |
|---|---|---|
| `sealed_locally` | selado localmente | Persistido/selado pelo módulo; **não** é aceitação AGT |
| `submitted` | enviado | Submetido / em trânsito para a autoridade |
| `received` | recebido | Autoridade acusou recepção / em processamento conhecido |
| `accepted` | aceite | Aceite pela AGT (único estado que afirma aceitação fiscal pela autoridade) |
| `rejected` | rejeitado | Rejeitado pela AGT (ou rejeição de validação pré-selagem, se modelada no mesmo enum — distinguir em erro/código) |

**Regra absoluta:** **não** afirmar aceitação fiscal perante a AGT antes do estado `accepted`. Em particular:

- `sealed_locally` ≠ aceite AGT ≠ «fiscalmente emitido» jurídico (`DEC-API-004` permanece aberta para o momento legal de emissão);
- HTTP 2xx / `sealed_locally` **não** autoriza o POS a apresentar o documento como aceite pela AGT;
- proibido reintroduzir `fiscally_issued` como sinónimo de selagem local.

**Notas de modelação**

- Subestados técnicos (`queued_for_authority`, `authority_outcome_unknown`, `contingency_pending`) podem existir na implementação/reconciliação; **não** substituem nem antecipam `accepted`.
- Anulação/`cancelled`: continua `DEC-API-002` (fora deste enum até decisão).
- Aplicação completa no OpenAPI/máquina pública = revisão contratual formal (hoje `createDocument` expõe `sealed_locally`).

**Evidência:** este registo; [`document-state-machine.md`](../04-domain/document-state-machine.md). Offline técnico: `DEC-PROD-010`.

---

## DEC-PROD-010 — Offline técnico sem emissão certificada

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / arquitectura |
| Prazo máximo | — |
| Responsável | Product Owner + Arquitectura |
| Decisão | 2026-07-26 |

**Decisão:**

1. **Implementar** capacidades técnicas de resiliência: **outbox**, **reenvio** seguro e **idempotência** (Edge/cloud), alinhadas ao slice e a `AO-IDEM-001`.
2. **Não declarar** emissão offline / contingência como **certificada** ou conforme `AO-OFF-*` até existir **regra oficial** + fecho de `DEC-REG-004` / requisitos confirmados.
3. Estado `contingency_pending` permanece **reservado**; não inventar fluxo legal de contingência.

**Fail-closed**

| Permitido (técnico) | Proibido (produto/compliance) |
|---|---|
| Outbox durável + worker de reenvio | Afirmar «emissão offline certificada AGT» |
| Idempotência em retries / timeouts | Usar `sealed_locally` offline como `accepted` |
| Reconciliação após restabelecer ligação | Inventar séries/números de contingência sem fonte |
| Marcar documentos/submissões como pendentes de autoridade | Promover `AO-OFF-001`/`002` a confirmados só com engenharia |

**Relação com `DEC-REG-004`:** adopta a opção técnica 2 (desenhar outbox/Edge **sem** declarar conformidade); o fecho regulatório continua opção 1 (texto oficial + AGT).

**Evidência:** este registo; SealInTx/outbox no slice; [`document-state-machine.md`](../04-domain/document-state-machine.md). Edge escritor único: `DEC-PROD-011`.

---

## DEC-PROD-011 — Edge: um único processo fiscal escritor

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / Edge / operações |
| Prazo máximo | — |
| Responsável | Product Owner + Arquitectura |
| Decisão | 2026-07-26 |

**Decisão:**

1. Em cada instalação **Fiscal Edge**: **um único processo fiscal escritor** (proprietário da escrita fiscal / SealInTx / numeração local).
2. **Proibido** multi-instância Edge (ou multi-processo escritor) na **mesma série** **sem** coordenação formal.
3. Vários POS no Edge → **só via API local** do único escritor (`DEC-PROD-008`).
4. Fecha `DEC-OPS-001` na opção 1 (MVP). Opção 2 (lease/partição via cloud) fica como evolução futura **explícita**, não implícita.

**Fail-closed**

| Proibido | Obrigatório |
|---|---|
| Dois escritores na mesma série sem lease/coordenação | Um escritor por instalação Edge |
| «Active-active» Edge na mesma série | Séries exclusivas do processo escritor |
| POS a escrever na BD/série directamente | POS → API local do módulo |

**Alinha:** DEC-STACK-001 (SQLite WAL, escritor único); [`edge-architecture.md`](../02-architecture/edge-architecture.md).

**Evidência:** este registo; `DEC-OPS-001` decidida por referência. Chaves: `DEC-PROD-012`.

---

## DEC-PROD-012 — Política de produto para chaves fiscais

| Campo | Valor |
|---|---|
| Estado | **decidida** (constraints de produto; **fecho definitivo bloqueado por AGT**) |
| Tipo | Produto / segurança |
| Prazo máximo | — (constraints); definitivo após `DEC-REG-KEY-CUSTODY` |
| Responsável | Product Owner + Segurança |
| Decisão | 2026-07-26 |

**Decisão (constraints obrigatórios de produto):**

| Regra | Implicação |
|---|---|
| **Segregadas por contribuinte** | Material/`TaxpayerKeyRef` por NIF/`Taxpayer` (+ ambiente); sem partilha entre contribuintes |
| **Não exportáveis quando possível** | Preferir KMS/HSM/keystore com privada non-exportable; sem exportação de privada para ficheiros/UI/logs |
| **Nunca nos POS** | POS não recebe, armazena nem assina com chave fiscal do contribuinte (`DEC-PROD-008`) |
| **Rotação auditada** | Toda rotação/revogação gera evento de auditoria imutável; sem rotação silenciosa |
| **Definitivo aguarda AGT** | Local/custódia exacta (cloud vs Edge, HSM, permissão de custódia externa) **não** fechados — `DEC-REG-KEY-CUSTODY`, `DEC-SEC-001`, `DEC-SEC-EDGE-KEYS`, GAP-013 |

**Fail-closed**

- Privada em Git, imagem, POS, logs ou telemetria → **proibido**.
- Provisionar `TaxpayerKeyRef` na plataforma como custódia definitiva **antes** de resposta AGT → **proibido**.
- Slice: chave efémera de teste atrás de adaptador; **não** material de contribuinte real; **não** certificado.

**Não altera** `ASM-REG-001`. Não confirma `AO-KEY-001` / `AO-CRYPTO-001`.

**Evidência:** este registo; [`backoffice-architecture.md`](../02-architecture/backoffice-architecture.md). Auditoria: `DEC-PROD-013`.

---

## DEC-PROD-013 — Auditoria append-only; retenção normativa

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / domínio / operações |
| Prazo máximo | — (append-only); retenção final após norma consolidada |
| Responsável | Product Owner + Operações + Compliance |
| Decisão | 2026-07-26 |

**Decisão:**

1. O trilho de **auditoria** (e o livro fiscal de transições relevantes) é **append-only**: sem UPDATE/DELETE destrutivo de eventos; correcções = novos eventos.
2. A **retenção final** (prazo legal de arquivo/purga) **depende da norma consolidada** aplicável — **não** inventar anos/prazos até citação oficial + revisão compliance.
3. Até lá: preservar evidências; política de retenção técnica mínima operacional **sem** declarar conformidade com prazo fiscal.

**Fail-closed**

| Proibido | Obrigatório |
|---|---|
| Apagar/alterar eventos de auditoria ou documentos emitidos | Append de novos eventos / documentos retificativos |
| Fixar prazo de retenção fiscal «por default» sem fonte | Marcar retenção como `pending_norm` até norma consolidada |
| Promover `AO-AUD-001` a confirmado só com esta decisão | Ligar implementação a `AO-AUD-001` (candidato; matriz provisória) |

**Evidência:** este registo; [`domain-model.md`](../04-domain/domain-model.md); `AO-AUD-001` no catálogo (não confirmado). Âmbito do modelo de tipos: `DEC-PROD-014`.

---

## DEC-PROD-014 — Âmbito do modelo de tipos (completo; implementação faseada)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / domínio |
| Prazo máximo | — |
| Responsável | Product Owner |
| Decisão | 2026-07-26 |

**Decisão:**

1. O **modelo de produto** inclui **todos** os tipos **legalmente aplicáveis** que pertençam a **SAF-T (AO)**, **FE AGT**, ou **ambos** (`DEC-PROD-002` + inventário L1/L2/L3/L4; 5 grupos `DEC-PROD-001`).
2. A **implementação pode ser faseada** (slice/OpenAPI/activação defaults) **sem limitar o modelo** — o domínio/catálogo canónico não é truncado ao MVP.
3. Tipos sem canal SAF-T nem FE continuam **excluídos** (`DEC-PROD-002`).
4. «Legalmente aplicável» ≠ inventar códigos; lacunas C-DOC-* / ausência de enum = fail-closed até fonte/decisão.

**Fail-closed**

| Proibido | Obrigatório |
|---|---|
| Modelar só `invoice`/`credit_note` como catálogo completo | Catálogo = união canal SAF-T ∪ FE (legalmente aplicável) |
| Remover tipos do modelo porque o slice não os emite | Fasear via activação (`DEC-PROD-003`) / disponibilidade (`DEC-PROD-005`) / OpenAPI |
| Confirmar `AO-DOC-001` só com esta decisão | Manter inventário citado; C-DOC-* abertos |

**Relação com `DEC-REG-003`:** passa a decidir **ordem de implementação / defaults do slice**, **não** o perímetro do modelo.

**Evidência:** este registo; [`DOCUMENT-TYPES-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md). Esquema mínimo do catálogo: `DEC-PROD-015`.

---

## DEC-PROD-015 — Esquema mínimo do catálogo / matriz de tipos

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / domínio / compliance |
| Prazo máximo | — |
| Responsável | Product Owner + Domínio |
| Decisão | 2026-07-26 |

**Decisão:** cada entrada do catálogo/matriz de tipos **deve** conter **pelo menos**:

| # | Campo | Conteúdo |
|---|---|---|
| 1 | `grupo` | Um dos 5 grupos L3 (`DEC-PROD-001`) |
| 2 | `codigo_canonico` | Identidade canónica de produto (`DEC-PROD-007`) — **não** código AGT cru |
| 3 | `designacao` | Designação humana |
| 4 | `codigos_canal` | Códigos por canal (FE L4, SAF-T L2) quando existirem |
| 5 | `estrutura_saft` | Estrutura SAF-T L3 (ex. `SalesInvoices`, `Payments`) |
| 6 | `elegibilidade` | `SAF-T` \| `FE` \| `ambos` (`DEC-PROD-002`) |
| 7 | `natureza_juridica` | Figura L1 / referência normativa (ou `n/a`) |
| 8 | `restricao_sectorial` | Sector/regime ou `nenhuma` / `pending` |
| 9 | `serie_necessaria` | Se série AGT/estabelecimento é exigida (`sim`/`não`/`pending`/`condicional`) |
| 10 | `requisitos` | Requisitos de emissão/preenchimento (citas ou `pending`) |
| 11 | `regras_rectificacao_anulacao` | Rectificação/anulação (citas ou `pending`) |
| 12 | `estado_normativo` | Ex.: `hipótese` \| `conflito` \| `pending_validation` \| auxiliar `reviewed` |
| 13 | `activo` | Estado de activação de produto/slice (`on`/`off`/`pending_dec_reg_003`) |

**Fail-closed:** campo desconhecido → `pending` / `hipótese`; **não** inventar. Catálogo vivo em [`DOCUMENT-CATALOG-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-CATALOG-RM-REQ-001.md).

**Evidência:** este registo + ficheiro do catálogo.

---

## DEC-REG-003 — Ordem de implementação / defaults do slice (tipos)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / entrega (já **não** limita o modelo — ver `DEC-PROD-014`) |
| Prazo máximo | — |
| Responsável | Product Owner + Compliance |
| Decisão | 2026-07-26 |

**Contexto:** o modelo de tipos está decidido em **`DEC-PROD-014`** (completo por canal). Esta decisão escolhe **o que implementar primeiro** e defaults de activação/OpenAPI.

**Decisão:** opção **1** — slice mínimo activo = **Factura** + **Nota de Crédito**.

| OpenAPI `document_type` | `codigo_canonico` | Catálogo `activo` |
|---|---|---|
| `invoice` | `bwb.ao.vendas.ft` | `on` |
| `credit_note` | `bwb.ao.vendas.nc` | `on` |

Restantes entradas do seed: modeladas com `activo=off` (não expostas neste slice). Catálogo completo permanece no modelo (`DEC-PROD-014`).

**Não autoriza:** truncar o modelo; promover `AO-DOC-*`; inventar regras NC de referência; credenciais/deploy AGT.

**Evidência:** este registo; [`DOCUMENT-CATALOG-RM-REQ-001.md`](../../compliance/derived/requirements/DOCUMENT-CATALOG-RM-REQ-001.md); pacote `internal/doctype`.

**Nota:** modelo = **DEC-PROD-014**; esquema = **DEC-PROD-015**; `DEC-REG-003` = faseamento apenas.

Relaciona: `AO-DOC-001`, `AO-DOC-002`, `DEC-PROD-001`–`015`.

---

## DEC-REG-004 — Contornos legais da contingência offline

| Campo | Valor |
|---|---|
| Estado | bloqueada-por-lacuna |
| Tipo | Regulatória |
| Prazo máximo | Fora do primeiro vertical slice; waiver até fonte oficial |
| Responsável | Compliance |

**Opções:**

1. Definir contingência apenas após texto oficial + orientação AGT.
2. Definir comportamento técnico de outbox/Edge e marcar emissão em contingência como «não certificável até validação».
3. Assumir regras de mercados vizinhos / fontes comunitárias.

| Opção | Vantagens | Riscos |
|---|---|---|
| 1 | Correto juridicamente | Pode atrasar Edge completo |
| 2 | Permite desenho e testes | Não declarar conformidade de `AO-OFF-*` |
| 3 | Rápido | **Rejeitada** — viola regras do projeto |

**Recomendação:** opção 2 para arquitetura; opção 1 para fecho de `AO-OFF-001` / `AO-OFF-002`.

**Nota 2026-07-26:** produto **`DEC-PROD-010`** — implementar outbox/reenvio/idempotência **sem** declarar emissão offline certificada até regra oficial (alinha opção 2 técnica; fecho `AO-OFF-*` continua bloqueado).

---

## DEC-STACK-001 — Stack tecnológica

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Técnica |
| Prazo máximo | — |
| Responsável | Arquitetura (aprovação: Tech Lead + PO) |
| Decisão | 2026-07-21 |

Ver análise completa em [technical-stack-proposal.md](technical-stack-proposal.md).

**Decisão inequívoca:** Go no backend; PostgreSQL na cloud; SQLite em WAL no Edge (um processo fiscal escritor; POS só via API local); pacote fiscal e testes comuns; abstração de persistência limitada (sem ORM genérico excessivo). PostgreSQL local no Edge só com benchmark/requisito oficial que prove SQLite insuficiente. Sem portal na primeira implementação. Sem scaffold nesta fase.

**Condições preservadas:**

- Validação XSD/SAF-T oficial com ferramenta comprovada baseada em libxml2/xmllint (ou componente isolado equivalente) e testes contra o XSD oficial da AGT; não implementar até obter esse XSD.
- Separar assinatura interna da API (se aplicável) da assinatura fiscal AGT (algoritmo, canonicalização e campos dependem do Decreto 74/19 e documentação oficial pendente); não tratar JWS RS256 como regra fiscal confirmada.
- Estratégia transacional de numeração definida após confirmação das regras oficiais; sem conclusões prematuras sobre duplicados ou «buracos».
- Nenhuma dependência fiscal, biblioteca XML/XSD ou algoritmo criptográfico entra em produção sem evidência documental oficial e testes de conformidade.

A alternativa Java 21 permanece apenas como comparação histórica em [technical-stack-proposal.md](technical-stack-proposal.md); não é opção residual.

---

## DEC-API-001 — Representação externa de dinheiro

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Técnica / contrato |
| Prazo máximo | — |
| Responsável | API Owner + Domínio |
| Decisão | 2026-07-20 |

**Contradição:** CTX-002 — fechada ao nível de decisão; texto do domínio/OpenAPI formal na primeira revisão contratual autorizada.

**Decisão:**

1. Valores monetários no JSON como **strings decimais**.
2. Formato canónico, escala e limites **explícitos** no OpenAPI (aplicados na tarefa zero: pattern sem sinal, escala 2, máx. 16 dígitos inteiros técnicos).
3. Representação interna com **decimal exato**.
4. **Proibição** de `float` / `double` para dinheiro.

**Aplicação:** `specs/openapi/openapi.yaml` `0.1.1-draft` (tarefa zero Fase 1).

Relaciona: `AO-TAX-001`.

---

## DEC-API-002 — Harmonizar estados documentais (`cancelled` e outros)

| Campo | Valor |
|---|---|
| Estado | aberta |
| Tipo | Técnica / domínio |
| Prazo máximo | Após DEC-API-004 e fontes de anulação; aplicação no OpenAPI na 1.ª revisão |
| Responsável | API Owner |

**Contradição:** CTX-001.

**Opções:**

1. Remover `cancelled` das diretrizes até existir comando legal de anulação modelado.
2. Adicionar `cancelled` ao OpenAPI e à máquina de estados com regras explícitas.
3. Introduzir estado distinto (ex.: anulação apenas via documento retificativo).

| Opção | Vantagens | Riscos |
|---|---|---|
| 1 | Evita inventar semântica | Diretrizes temporariamente reduzidas |
| 2 | Completude do contrato | Pode conflitar com `AO-DOC-002` |
| 3 | Mais preciso juridicamente | Mais modelação |

**Recomendação:** opção 1 até validar anulações/retificações nas fontes oficiais. Não alterar `openapi.yaml` agora.

---

## DEC-API-003 — Schema de `quantity` no OpenAPI

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Técnica |
| Prazo máximo | — (aplicado na tarefa zero) |
| Responsável | API Owner |
| Decisão | 2026-07-20 |

**Contradição:** CTX-003 — fechada ao nível de decisão; YAML atualizado na tarefa zero.

**Decisão:** criar schema `DecimalQuantity` separado de `Money` (quantidade estritamente positiva e canónica; limites 12/4 técnicos).

**Aplicação:** `DocumentLine.quantity` → `DecimalQuantity` em `openapi.yaml` `0.1.1-draft`.

---

## DEC-API-004 — Momento jurídico da emissão vs aceitação AGT

| Campo | Valor |
|---|---|
| Estado | aberta (pode ficar `bloqueada-por-lacuna`) |
| Tipo | Regulatória + contrato |
| Prazo máximo | Antes de declarar semântica final no OpenAPI v1; slice usa termos neutros até lá |
| Responsável | Compliance + API Owner |

**Contradição:** CTX-006.

**Contexto:** não assumir que `fiscally_issued` ocorre antes da aceitação da AGT. A semântica depende da legislação e do comportamento oficial da faturação eletrónica.

**Pontos a decidir:**

1. Quando o documento passa a considerar-se **fiscalmente emitido**.
2. Diferença entre: selado/persistido localmente; submetido; recebido pela AGT; aceite pela AGT.
3. Comportamento em **contingência** (quando autorizado).

**Opções (rascunho, não normativas):**

1. Emissão fiscal = selagem local + número + assinatura, independentemente da aceitação AGT.
2. Emissão fiscal só após aceitação AGT; até lá o documento é preparado/selado localmente.
3. Modelo híbrido conforme regras oficiais de contingência e FE.

| Opção | Vantagens | Riscos |
|---|---|---|
| 1 | Alinha a operação offline potencial | Pode divergir da FE oficial |
| 2 | Alinha a aceitação autoridade | Impacto em UX/POS e contingência |
| 3 | Flexível | Complexidade; exige fonte oficial |

**Recomendação:** não escolher 1–3 sem fonte oficial. O contrato usa o estado técnico `sealed_locally` (tarefa zero OpenAPI aplicada); DEC-API-004 permanece **aberta** para a semântica jurídica final.

**Nota 2026-07-26:** ciclo de vida de produto (`sealed_locally` → `submitted` → `received` → `accepted` \| `rejected`) e proibição de afirmar aceitação antes da AGT = **`DEC-PROD-009`**. Isto **não** fecha sozinho qual opção 1–3 é a emissão jurídica.

**Evidência para fechar:** diploma/orientação AGT + snapshot FE + ata de compliance.

---

## DEC-SEC-001 — Custódia de chaves fiscais (cloud vs Edge)

| Campo | Valor |
|---|---|
| Estado | aberta |
| Tipo | Segurança |
| Prazo máximo | Antes do scaffold de criptografia (início Fase 1) |
| Responsável | Segurança + Arquitetura |

Relaciona: `AO-CRYPTO-001`, `AO-KEY-001`.

**Opções:**

1. Cloud: KMS/HSM; Edge: keystore OS com cifra em repouso; slice: par RSA **efémero** gerado nos testes/arranque do simulador, atrás de adaptador.
2. HSM dedicado também no Edge (quando volume justificar).
3. Chaves em ficheiro no repositório / imagem — **rejeitada**.
4. Cofre de CI para chaves falsas do vertical slice — **rejeitada** (custo sem benefício).

**Recomendação:** opção 1 para MVP de infraestrutura genérica, sujeita a DEC-REG-KEY-CUSTODY para material do contribuinte. Segredos nunca no Git. No slice: JWS RS256 real; chave privada efémera **nunca** persistida nem commitada; fixtures públicas só com chave pública ou vetores estáticos não secretos, se necessário; marcado como não certificado — **sem** stub descartável e **sem** regras legais do 74/19 ainda desconhecidas.

**Nota 2026-07-26:** constraints de produto **`DEC-PROD-012`** (segregação por contribuinte; non-exportable quando possível; nunca nos POS; rotação auditada). Fecho de opções 1–2 e local Edge/cloud continua a aguardar AGT.

---

## DEC-REG-KEY-CUSTODY — Custódia externa da chave privada do contribuinte

| Campo | Valor |
|---|---|
| Estado | aberta |
| Tipo | Regulatória |
| Criticidade | **Bloqueante** |
| Prazo máximo | Antes de provisionar `TaxpayerKeyRef` no `SecretStore` da plataforma |
| Responsável | Compliance + Jurídico (confirmação junto da AGT) |

**Contexto:** a autorização contratual do contribuinte é necessária, mas pode não ser suficiente. Um contrato privado não prova que a AGT permite entregar a chave privada do contribuinte a um fornecedor externo (módulo fiscal).

**Pergunta oficial:** a AGT permite que um módulo fiscal externo detenha e utilize a chave privada do contribuinte?

**Opções (após resposta oficial):**

1. Custódia/uso no `SecretStore` da plataforma permitido sob condições oficiais.
2. Custódia externa proibida — chave só em ambiente controlado pelo contribuinte/Edge, ou mecanismo oficial de delegação/assinatura remota.
3. Modelo híbrido definido pela AGT.

**Evidência para fechar:** orientação/escrito oficial AGT ou regra em diploma/manual versionado. Ver GAP-013 em [regulatory-gaps.md](../01-compliance/regulatory-gaps.md) e [backoffice-architecture.md](../02-architecture/backoffice-architecture.md).

**Dependentes:** DEC-SEC-EDGE-KEYS; provisionamento de `TaxpayerKeyRef` na plataforma.

**Nota 2026-07-26:** constraints de produto já em **`DEC-PROD-012`**; esta decisão permanece **aberta** até AGT (definitivo).

---

## DEC-SEC-EDGE-KEYS — Local da assinatura fiscal cloud vs Edge

| Campo | Valor |
|---|---|
| Estado | aberta |
| Tipo | Segurança / arquitetura |
| Criticidade | **Bloqueante** |
| Prazo máximo | Antes de implementar assinatura fiscal em Edge ou sync de privadas |
| Responsável | Segurança + Arquitetura + Compliance |

**Dependências:** regras oficiais de contingência (`AO-OFF-*`) **e** DEC-REG-KEY-CUSTODY.

**Contexto:** Edge offline não assina com chave que exista apenas num `SecretStore` cloud. Cópia automática cloud↔Edge de privadas é proibida.

**Opções (nenhuma escolhida):**

| ID | Descrição | Offline fiscal |
|---|---|---|
| E1 | Assinatura fiscal exclusivamente cloud | Não |
| E2 | Chave do contribuinte provisionada diretamente no keystore Edge | Sim, nos limites legais |
| E3 | Assinatura remota via `SecretStore` (Edge online) | Não |

Se DEC-REG-KEY-CUSTODY proibir custódia externa, E1/E3 com privada na cloud BWB ficam inviáveis para a chave do contribuinte.

**Evidência para fechar:** DEC-REG-KEY-CUSTODY + texto oficial de contingência. Ver [backoffice-architecture.md](../02-architecture/backoffice-architecture.md).

---

## DEC-OPS-001 — Propriedade de séries em Edge multi-instância

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Operacional / fiscal |
| Prazo máximo | — |
| Responsável | Arquitetura + Operações + Product Owner |
| Decisão | 2026-07-26 (via `DEC-PROD-011`) |

Relaciona: [edge-architecture.md](../02-architecture/edge-architecture.md); `DEC-PROD-011`.

**Decisão:** opção **1** — uma instalação Edge = um processo fiscal escritor; séries em exclusividade; POS só via API local. Opção **3** rejeitada. Opção **2** (lease/partição) = evolução futura explícita, não MVP.

Alinhado a SQLite WAL com escritor único ([technical-stack-proposal.md](technical-stack-proposal.md); DEC-STACK-001).

---

## DEC-DEL-001 — Critério do gate «contrato API rascunhado» na Fase 0

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Entrega |
| Prazo máximo | — |
| Responsável | Product Owner |
| Decisão | 2026-07-20 |

**Contradição:** CTX-004 — fechada.

**Decisão:** o OpenAPI `0.1.0-draft` cumpriu o gate documental da Fase 0. A **tarefa zero da Fase 1** foi aplicada em `0.1.1-draft` (DEC-API-001/003, `sealed_locally`, `authority_outcome_unknown`; `contingency_pending` reservado; sem `cancelled`). Endpoints de produção ainda não implementados.

---

## DEC-DEL-002 — Credenciais/confirmação AGT não travam trilho interno

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Entrega / governação |
| Prazo máximo | — |
| Responsável | Product Owner + Compliance |
| Decisão | 2026-07-26 |

**Decisão:** itens de **credenciais** e **confirmação externa** AGT classificam-se como **`BLOQUEADO_EXTERNO` / `ADIADO`** (ver [`agt-dependencies.md`](../01-compliance/agt-dependencies.md)).

**Não travam:** catálogo, domínio, simulador, contratos (OpenAPI), persistência (SealInTx/outbox) nem testes/CI.

**Não autorizam:** inventar respostas da AGT; declarar conformidade ou `AO-*` confirmados; tratar simulador como HML oficial.

**ROADMAP:** estados canónicos continuam `BLOQUEADO` \| `ADIADO`; a etiqueta `BLOQUEADO_EXTERNO` vive no inventário AGT e na coluna Done/gate.

---

## DEC-TIME-001 — Tempo fiscal (`issued_at`) vs tempo técnico (`created_at`)

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Domínio / API / persistência |
| Prazo máximo | — |
| Responsável | Arquitectura + engenharia |
| Decisão | 2026-07-21 |

**Decisão:**

| Campo | Natureza | Regra |
|---|---|---|
| `created_at` | Técnico | Gerado pelo módulo, UTC, precisão microsegundo; independente do TZ do processo/host |
| `issued_at` | Fiscal | Civil no estabelecimento; offset obrigatório compatível com IANA do scope; persistir UTC micro + `issued_timezone` + `issued_offset_minutes` |
| Hash | `canonical_v2` | Inclui timezone e offset; `canonical_v1` congelado (golden) |
| Desenvolvimento AO | Scope | `FISCAL_SCOPE_TIMEZONE=Africa/Luanda` (fail-closed); Cabo Verde não wired |
| Migration `0002` | Forward-only | Aborta se existirem `documents` ou `idempotency_records`; sem recalculo de hashes |

**Evidência:** OpenAPI `0.1.3-draft`, migrations `0002`, testes golden + HTTP + migrate guards.

---

## DEC-BO-001 — Backoffice funcional vs zona de administração de integração/segredos

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / arquitectura / segurança |
| Prazo máximo | — |
| Responsável | Product Owner (Jorge) + Arquitectura + Segurança |
| Decisão | 2026-07-26 |

**Decisão:** o portal operacional separa **dois planos** com superfícies, papéis e APIs distintas.

### Plano A — Backoffice funcional (ops comuns)

Gere e observa, **sem** material secreto:

- contribuintes (`Taxpayer`), estabelecimentos (`Establishment`);
- scopes / bindings POS (metadados);
- séries e configuração **não secreta** (timezone, códigos de série, activação de tipos/grupos);
- estados operacionais (activo/inactivo, adesão FE como enum de estado);
- auditoria e visibilidade de submissões/erros **sem** payloads secretos.

**Proibido** no backoffice comum (UI e API admin funcional):

- credenciais AGT (Basic Auth / tokens de produtor);
- chaves privadas (produtor ou contribuinte);
- passwords, tokens em claro, DSN/URLs privadas de cofre ou endpoints privados de override;
- qualquer leitura ou exportação de material write-only.

### Plano B — Zona dedicada de administração de integração e segredos

- Acesso **exclusivo do owner** (Jorge) — não partilhado com operadores do backoffice comum;
- valores **write-only** (provisionamento; sem GET do segredo; sem echo na resposta);
- cofre / storage cifrado (`SecretStore`);
- auditoria append-only de provisionamento, rotação e revogação;
- separação estrita **HML / PRD** (sem cópia automática de segredos entre ambientes);
- endpoints públicos documentados podem ser configuração técnica versionada; **overrides privados** (URLs, credenciais, headers secretos) ficam **só** no cofre operacional.

### O que o backoffice comum pode mostrar (metadados sanitizados)

| Campo permitido | Exemplos |
|---|---|
| Ambiente | `homologation` \| `production` |
| Estado da ref | `absent` \| `present` \| `rotating` \| `revoked` |
| Fingerprint | derivado de chave **pública** ou metadado seguro do provisionamento |
| Validade | `expires_at` / janela conhecida sem o segredo |
| Última verificação | instante da última probe bem-sucedida (sem corpo secreto) |

### Fail-closed

| Proibido | Obrigatório |
|---|---|
| Segredo na UI, logs, telemetria ou resposta JSON do plano A | Plano A só metadados sanitizados |
| Operador comum com ACL de leitura de segredos | Plano B owner-only + write-only |
| Misturar overrides privados na config pública versionada | Overrides no cofre; públicos documentados à parte |
| Credenciais reais AGT neste slice | Simulator / fail-closed até acesso AGT |

**Não fecha** `DEC-REG-KEY-CUSTODY` (custódia legal da chave do contribuinte). Não autoriza deploy/SSH.

**Evidência:** este registo; [`backoffice-architecture.md`](../02-architecture/backoffice-architecture.md); [`security-baseline.md`](../05-security/security-baseline.md); ROADMAP `RM-ARCH-006`, `RM-BO-*`, `RM-SECADM-*`.

---

## DEC-BO-002 — Superfície Admin API `/admin/v1` e auth OIDC/JWT RBAC

| Campo | Valor |
|---|---|
| Estado | **decidida** |
| Tipo | Produto / API / segurança |
| Prazo máximo | — |
| Responsável | Product Owner + Arquitectura |
| Decisão | 2026-07-26 |

**Decisão:**

1. Backoffice/admin usa superfície **separada** da API POS: prefixo **`/admin/v1`**.
2. OpenAPI administrativo próprio: [`specs/admin/openapi.yaml`](../../specs/admin/openapi.yaml) começando em **`0.1.0-draft`** (evolução actual `0.1.3-draft`) — **não** misturar com `specs/openapi/openapi.yaml` POS `0.1.x`. Path canónico é `specs/admin/` (não `specs/openapi/`).
3. Autenticação de operadores por contrato **OIDC/JWT + RBAC**, distinta do POS Bearer `credential_store`.
4. Papéis iniciais: `owner` | `admin` | `operator` | `auditor`.
5. **Apenas `owner`** acede à zona de administração de integração/segredos (SecAdm / `RM-SECADM-*`).
6. Enquanto não houver IdP real: `Authenticator` **injectável** e **fail-closed** por omissão; `FISCAL_ADMIN_AUTH_MODE=injected` só em `FISCAL_ENV=development` (testes/dev explícito). **Proibido** login local improvisado e credenciais reais AGT.
7. Mutações admin geram auditoria **append-only** (`admin_audit_events`); segredos nunca na resposta.
8. Matriz RBAC tipada (`adminauth.Allows` / RM-BO-004): só `owner` tem `secadm.write`; **não existe** permissão de revelar segredo. MFA de operador **adiado** até IdP real.

**Evidência:** este registo; OpenAPI admin; `internal/adminauth`, `internal/adminapi`, `internal/adminaudit`.

---

## Prioridade de decisão (abertas)

Inventário AGT: [`../01-compliance/agt-dependencies.md`](../01-compliance/agt-dependencies.md).

1. **DEC-REG-KEY-CUSTODY** — custódia externa da chave privada do contribuinte (**bloqueante** / AGT).
2. **DEC-REG-002** — Decreto 74/19 + Rect. arquivados/`reviewed`; falta fecho AO-* + URL estável.
3. **DEC-REG-001** — confirmação processual de `ASM-REG-001` (**AGT**).
4. **DEC-SEC-EDGE-KEYS** — local da assinatura cloud/Edge (**bloqueante**; depende de DEC-REG-KEY-CUSTODY e contingência AGT).
5. **DEC-API-004** — momento jurídico da emissão/aceitação (**AGT** / norma).
6. **DEC-REG-004** — contingência offline certificável (**AGT**; produto técnico = `DEC-PROD-010`).

**Já decididas (fora da lista prioritária):** DEC-STACK-001, **DEC-DEL-001**, **DEC-DEL-002**, DEC-API-001, DEC-API-003, **DEC-TIME-001**, **DEC-OPS-001**, **DEC-PROD-001**–**015**, **DEC-REG-003**, **DEC-BO-001**, **DEC-BO-002**.

---

## Decisões explicitamente fora de âmbito agora

- Implementação de Cabo Verde / SAF-T (CV).
- Escolha de fornecedor cloud específico.
- Microserviços (rejeitado por ADR-0002 até necessidade comprovada).
- Alteração de `ASM-REG-001`.
- Portal frontend e webhooks no primeiro vertical slice.
- Promessas de exactly-once na comunicação com a autoridade.
