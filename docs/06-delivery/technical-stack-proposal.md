# Proposta de stack técnica — Fase 0

**Data:** 2026-07-20  
**Estado:** proposta (não implementada)  
**Restrições:** monólito modular ([ADR-0002](../02-architecture/adrs/ADR-0002-modular-monolith.md)); sem microserviços; pacotes por país ([ADR-0003](../02-architecture/adrs/ADR-0003-country-packages.md)); paridade de **pacote fiscal** cloud/Edge; precisão decimal; sem segredos no repositório.

Decisão associada: **DEC-STACK-001** em [open-decisions.md](open-decisions.md).

## Critérios de avaliação

| Critério | Porque importa |
|---|---|
| Transações fortes | Idempotência, série/número e livro fiscal na mesma unidade de trabalho (`AO-IDEM-001`, `AO-SEQ-001`) |
| Precisão decimal | `AO-TAX-001`; proibição de `float`/`double` para dinheiro (DEC-API-001) |
| Assinatura RSA / JWS | `AO-CRYPTO-001`, conector AGT (RS256); implementação real atrás de adaptador |
| XML / XSD | SAF-T (AO) futuro (`AO-SAF-001`) — fora do primeiro slice |
| Operação offline Edge | Persistência local com baixo custo operacional (`AO-OFF-*` quando autorizado) |
| Facilidade de auditoria | Trilho append-only (`AO-AUD-001`) |
| Custo operacional Edge | Instalação, backup, suporte e footprint |
| Observabilidade segura | Metadados e correlação nos logs; payloads fiscais fora dos logs |

## Alternativa A — Go + PostgreSQL (cloud) + SQLite WAL (Edge) — recomendada

### Visão

Monólito modular em **Go**. Na **cloud**: API HTTP + **PostgreSQL** + outbox na mesma base. No **Edge**: o mesmo núcleo/pacote fiscal com **SQLite em modo WAL**, **um único processo** fiscal proprietário da escrita; vários POS apenas via API local. Sem portal na primeira implementação. Abstração de persistência **limitada** (repositórios/SQL explícito), sem ORM genérico excessivo.

| Área | Escolha proposta | Notas |
|---|---|---|
| Backend | Go | Binário Edge leve; deploy simples |
| Decimal | Decimal auditado ou inteiros na menor unidade no domínio | Nunca `float64` para dinheiro |
| Cloud DB | PostgreSQL 16+ | `NUMERIC`, constraints; **numeração fiscal controlada pela aplicação** (não depender de `SERIAL`/sequences PG sem analisar rollback, cache e falhas) |
| Edge DB | SQLite WAL | Escritor único; suficiente para MVP salvo prova em contrário |
| Filas | Outbox co-transacional + worker; entrega **at-least-once** | Idempotência + deduplicação por id estável de submissão; sem exactly-once |
| Portal | **Fora da 1.ª implementação** | Slice posterior |
| Edge Linux | Binário + `systemd`; POS → API local | Sem PostgreSQL local no MVP |
| Criptografia | JWS RS256 real; chaves de teste atrás de interface/adaptador | Não certificado; sem stub descartável; sem regras 74/19 inventadas |
| XML/XSD | Mais tarde, com XSD oficial | Fora do primeiro slice |
| Observabilidade | OpenTelemetry + logs estruturados | Só metadados/IDs; outbox ≠ log |
| Testes | `go test`, contract tests, vetores `AO-*` comuns cloud/Edge | Conformidade partilhada |
| Deployment cloud | Contentores + IaC + migrações | [deployment.md](../07-operations/deployment.md) |
| Deployment Edge | Pacote/binário + atualizador assinado | [edge-architecture.md](../02-architecture/edge-architecture.md) |

### Vantagens

- Custo operacional Edge baixo (sem Postgres por instalação).
- Mesmo pacote fiscal e testes de conformidade na cloud e no Edge.
- Controlo explícito de concorrência de escrita no Edge.
- Adequado a at-least-once + reconciliação.

### Riscos

- SQLite pode revelar limites sob carga/requisito oficial — mitigar com benchmarks antes de adotar Postgres local.
- Abstração dual-store exige disciplina (interfaces estreitas).
- XML/XSD mais trabalhoso em Go quando chegar SAF-T.

### Quando considerar PostgreSQL no Edge

Apenas se benchmarks ou requisitos oficiais demonstrarem que SQLite é insuficiente. Não é o default do MVP.

## Alternativa B — Java 21 + Spring Boot + PostgreSQL (cloud) + SQLite WAL (Edge)

### Visão

Mesmo modelo de armazenamento e outbox que A; backend Java 21 / Spring Boot; Edge com JRE/`jlink` ou contentor e SQLite WAL com escritor único.

| Área | Escolha proposta | Notas |
|---|---|---|
| Backend | Java 21 + Spring Boot | Ecossistema enterprise |
| Decimal | `BigDecimal` com escala explícita | Proibir `double`/`float` |
| Cloud DB | PostgreSQL 16+ | Igual rigor na numeração fiscal |
| Edge DB | SQLite WAL | Igual à A |
| Filas | Outbox JDBC + worker at-least-once | Mesmas regras de deduplicação |
| Portal | Fora da 1.ª implementação | Igual à A |
| Criptografia | JCA + JOSE; adaptador; chaves de teste | Sem stub |
| Testes | JUnit, Testcontainers, ArchUnit | Pacote fiscal comum |
| Deployment Edge | Mais pesado que Go | Patch JRE |

### Vantagens

- XML/XSD e tipagem maduros quando SAF-T chegar.
- Mercado de programadores Java amplo.

### Riscos

- Footprint Edge e complexidade de framework superiores.
- Custo operacional Edge ainda maior se alguém cair na tentação de Postgres local.

## Comparação direta

| Dimensão | Alternativa A (Go) | Alternativa B (Java) |
|---|---|---|
| Cloud DB | PostgreSQL | PostgreSQL |
| Edge DB (MVP) | SQLite WAL | SQLite WAL |
| Decimal / JWS | Adequado | Adequado |
| Custo Edge | Mais baixo | Mais alto |
| Tempo até slice | Favorece equipa Go | Favorece equipa Java |
| ORM | Evitar ORM pesado | Preferir JDBC/repositórios claros |

## Numeração fiscal (ambos)

Não prometer «zero buracos» genericamente. Exigir:

- exclusão mútua / transação por série;
- números nunca duplicados;
- números emitidos nunca reutilizados;
- rastreabilidade de números reservados, falhados ou rejeitados;
- política de buracos/sequência contínua **dependente da regra oficial** (DEC-REG-002 / 74/19).

Não usar sequences PostgreSQL comuns como garantia fiscal sem análise de rollback, cache e falhas.

## Comunicação com a autoridade (ambos)

- Entrega **at-least-once**.
- Idempotência de submissão.
- Deduplicação por identificador estável.
- Persistência da tentativa e da resposta (outbox com controlo de acesso; payload pode ir cifrado — **não** é log operacional).
- Reconciliação quando o resultado for desconhecido.

## Recomendação

**Adotar a Alternativa A**, com PostgreSQL só na cloud e SQLite WAL no Edge.

Escolher B se a equipa núcleo for predominantemente Java.

Em ambos os casos: sem microserviços; sem portal no primeiro slice; DEC-STACK-001 antes do scaffold.

## Explicitamente fora desta proposta

- PostgreSQL local por instalação Edge como default.
- Bus de eventos empresarial para o caminho fiscal.
- Bases NoSQL para o livro fiscal.
- Exactly-once ponta a ponta com a AGT.
- Portal frontend na Fase 1 inicial.

## Próximo passo

Fechar DEC-STACK-001 em [open-decisions.md](open-decisions.md) e só então iniciar o monorepo na Fase 1.
