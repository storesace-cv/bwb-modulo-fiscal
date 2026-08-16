# Dependências AGT — inventário

**Data:** 2026-07-26
**Política:** credenciais e confirmação externa = **`BLOQUEADO_EXTERNO` / `ADIADO`** — **não** travam catálogo, domínio, simulador, contratos, persistência nem testes.
**Proibido:** inventar respostas da AGT; declarar conformidade / `AO-*` confirmados sem evidência.
**Relaciona:** [`regulatory-gaps.md`](regulatory-gaps.md) · [`open-decisions.md`](../06-delivery/open-decisions.md) · [`official-access-plan.md`](official-access-plan.md) · [`agt-clarifications-register.md`](agt-clarifications-register.md) · `DEC-DEL-002`

Legenda de classificação (semântica; ROADMAP usa só `BLOQUEADO` \| `ADIADO`):

| Classe | Significado |
|---|---|
| `BLOQUEADO_EXTERNO` | Fecho exige AGT/credencial/área autenticada; engenharia interna **continua** |
| `ADIADO` | Fora do caminho crítico interno actual; retomar em marco (ex. M6/M9) |

## 0. O que **não** fica travado

Enquanto as lacunas §1–§2 estiverem abertas, **é obrigatório poder avançar**:

| Trilho interno | Notas |
|---|---|
| Catálogo / matrizes / `DEC-PROD-*` | Incl. `DOCUMENT-CATALOG-RM-REQ-001` |
| Domínio / máquina de estados | Incl. `DEC-PROD-009` |
| Simulador AGT / sandbox BWB | ≠ HML oficial; marcar como não certificado |
| Contratos OpenAPI / guidelines | Sem semântica jurídica final (`DEC-API-004`) |
| Persistência / SealInTx / outbox / idempotência | Já no slice |
| Testes unitários, conformidade de harness, CI | Vetores internos; ≠ GAP-010 |

## 1. `BLOQUEADO_EXTERNO` — credenciais e confirmação externa

| ID | Tema | Evidência de fecho | **Não** bloqueia | Bloqueia só |
|---|---|---|---|---|
| **GAP-006** | Credenciais + ambiente HML/PRD oficiais | Credenciais em cofre; ambiente registado sem segredos no Git | Simulador, contratos, testes internos | Cliente FE contra AGT real |
| **GAP-003** | Modelo 8 / área produtores autenticada | Acesso + checklist versionado | Catálogo, domínio, sandbox | Dossier de submissão/certificação |
| **GAP-007** / **DEC-REG-001** | Confirmação processual `ASM-REG-001` | Resposta AGT **ou** ata de risco + plano B | Premissa de produto intacta; ADR-0001 | Narrativa de certificação «aceite AGT» |
| **GAP-013** / **DEC-REG-KEY-CUSTODY** | Custódia chave privada contribuinte | Escrito AGT / norma | Slice com chave efémera; constraints `DEC-PROD-012` | `TaxpayerKeyRef` definitivo na plataforma |
| **DEC-SEC-EDGE-KEYS** | Local da assinatura cloud vs Edge | KEY-CUSTODY + contingência oficial | Desenho; testes sem material real | Assinatura Edge com chave de contribuinte |
| **GAP-010** | Vetores / testes oficiais AGT | Relatórios oficiais | Harness/simulador interno | Declaração de conformidade |
| **GAP-004** (residual vigência) | XSD autenticado AGT | Confirmação AGT + vigência | XSD ASSOFT `pending_validation` + testes schema | Gerador SAF-T **de produção** declarado conforme |
| **DEC-API-004** | Momento jurídico emissão vs aceite | Diploma/orientação AGT | Estados produto `DEC-PROD-009`; OpenAPI `sealed_locally` | Semântica jurídica final OpenAPI v1 |
| **GAP-009** / **DEC-REG-004** | Offline certificável | Texto/orientação AGT | Outbox/reenvio `DEC-PROD-010` | `AO-OFF-*` confirmados; contingência legal |

## 2. `ADIADO` (marco) — integração/certificação real

| Tema | Marco típico | ROADMAP |
|---|---|---|
| Cliente FE em HML AGT real | M6 | `RM-FE-001` |
| Registo produtor / Modelo 8 submetível | M9 | `RM-CERT-001` |
| Produção FE autorizada | M6–M11 | `RM-FE-005` / `RM-M6` |

## 3. Parcialmente mitigadas (trabalho interno continua)

| ID | Interno permitido | Externo ainda em falta |
|---|---|---|
| **GAP-001/002/014** | Citações OCR `reviewed` | URL estável; AO-* confirmados |
| **GAP-005** | Matriz FE / FE-RNG; snapshots privados | Redistribuição; C-FE-001; GAP-006 |
| **GAP-008** | Modelo + catálogo seed | C-DOC-*; cálculo; AO-DOC/TAX confirmados |
| **C-DOC-*/C-SIGN-001/C-FE-001** | Conflitos registados; fail-closed | Fecho oficial sem inventar |

## 4. Produto decidido — **não** inventa AGT

| Decisão | Permitido | Proibido |
|---|---|---|
| **DEC-PROD-009** | Ciclo `sealed_locally`…`accepted`/`rejected` | Aceitação antes de `accepted`; fecho jurídico `DEC-API-004` |
| **DEC-PROD-010** | Outbox / reenvio / idempotência | «Emissão offline certificada» |
| **DEC-PROD-012** | Constraints de chaves | Custódia definitiva sem AGT |
| **DEC-PROD-014/015** | Catálogo completo + esquema | Confirmar `AO-DOC-*`; inventar códigos |

## 5. Pedidos formais (quando houver canal) — sem inventar resposta

1. Custódia chave contribuinte (GAP-013).
2. `ASM-REG-001` (GAP-007).
3. Contingência offline (GAP-009).
4. Vigência XSD SAF-T (GAP-004).
5. Credenciais HML + Modelo 8 (GAP-006, GAP-003).
6. Momento jurídico emissão/aceitação (DEC-API-004).
7. Mapeamento FA/RC/RG vs SAF-T (C-DOC-003) **se** a AGT publicar.

## 6. Gate de conformidade

- Sem evidência verificável: **não** `AO-*` confirmados; **não** «certificado AGT»; **não** inventar respostas.
- Simulador / sandbox BWB **≠** homologação AGT.
- `BLOQUEADO_EXTERNO` no inventário **≠** parar `RM-REQ-001`, domínio, contratos, persistência ou CI.
