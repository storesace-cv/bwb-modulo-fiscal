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
| Estado | bloqueada-por-lacuna |
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

**Recomendação:** opção 1. A opção 3 é **rejeitada** como fonte normativa. Consulta em `local/` apenas para familiarização, sem cópia automática para o repositório.

**Evidência:** SHA-256, URL/origem, data de obtenção, versão/rectificação.

---

## DEC-REG-003 — Tipos documentais do MVP Angola

| Campo | Valor |
|---|---|
| Estado | aberta |
| Tipo | Regulatória + produto |
| Prazo máximo | Dentro das 2–4 semanas internas |
| Responsável | Product Owner + Compliance |

**Opções:**

1. MVP mínimo: fatura + nota de crédito (alinhado ao esqueleto OpenAPI).
2. MVP alargado: incluir documentos de transporte / conferência se a fonte oficial exigir assinatura.
3. Adiar tipos retificativos para após a primeira fatura ponta a ponta.

| Opção | Vantagens | Riscos |
|---|---|---|
| 1 | Encaixa no vertical slice | Pode ser incompleto face ao diploma oficial |
| 2 | Cobertura regulatória maior | Atrasa Fase 1 |
| 3 | Entrega mais cedo | Dívida no OpenAPI e pacote AO |

**Recomendação:** opção 1 para o vertical slice; reavaliar opção 2 após G0-R1 (74/19 oficial).

Relaciona: `AO-DOC-001`, `AO-DOC-002`.

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

**Recomendação:** opção 2 para arquitetura futura; opção 1 para fecho de `AO-OFF-001` / `AO-OFF-002`. Contingência Edge completa **excluída** do primeiro vertical slice.

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
| Estado | aberta |
| Tipo | Operacional / fiscal |
| Prazo máximo | Antes da distribuição Edge (após o primeiro slice) |
| Responsável | Arquitetura + Operações |

Relaciona: [edge-architecture.md](../02-architecture/edge-architecture.md).

**Opções:**

1. Uma instalação Edge = um processo fiscal proprietário da escrita; séries atribuídas em exclusividade.
2. Protocolo formal de partição/lease de séries via cloud.
3. Multi-Edge na mesma série sem coordenação — **rejeitada**.

**Recomendação:** opção 1 no MVP Edge; POS múltiplos apenas via API local. Alinhado a SQLite WAL com escritor único ([technical-stack-proposal.md](technical-stack-proposal.md)).

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

## Prioridade de decisão (abertas)

1. **DEC-REG-KEY-CUSTODY** — custódia externa da chave privada do contribuinte (**bloqueante**).
2. **DEC-REG-002** — Decreto 74/19 e rectificação oficiais.
3. **DEC-REG-001** — confirmação processual de `ASM-REG-001`.
4. **DEC-SEC-EDGE-KEYS** — local da assinatura cloud/Edge (**bloqueante**; depende de DEC-REG-KEY-CUSTODY e contingência).
5. **DEC-REG-003** — tipos documentais do MVP.
6. **DEC-API-004** — momento jurídico da emissão/aceitação.

**Já decididas (fora da lista prioritária):** DEC-STACK-001, DEC-DEL-001, DEC-API-001, DEC-API-003.

---

## Decisões explicitamente fora de âmbito agora

- Implementação de Cabo Verde / SAF-T (CV).
- Escolha de fornecedor cloud específico.
- Microserviços (rejeitado por ADR-0002 até necessidade comprovada).
- Alteração de `ASM-REG-001`.
- Portal frontend e webhooks no primeiro vertical slice.
- Promessas de exactly-once na comunicação com a autoridade.
