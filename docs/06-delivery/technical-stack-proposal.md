# Proposta de stack técnica — Fase 0

**Data:** 2026-07-20  
**Estado:** proposta (não implementada)  
**Restrições:** monólito modular ([ADR-0002](../02-architecture/adrs/ADR-0002-modular-monolith.md)); sem microserviços; pacotes por país ([ADR-0003](../02-architecture/adrs/ADR-0003-country-packages.md)); paridade cloud/Edge; precisão decimal; sem segredos no repositório.

Decisão associada: **DEC-STACK-001** em [open-decisions.md](open-decisions.md).

## Critérios de avaliação

| Critério | Porque importa |
|---|---|
| Transações fortes | Idempotência, série/número e livro fiscal na mesma unidade de trabalho (`AO-IDEM-001`, `AO-SEQ-001`) |
| Precisão decimal | `AO-TAX-001`; proibição de `float`/`double` para dinheiro |
| Assinatura RSA / JWS | `AO-CRYPTO-001`, conector AGT (RS256 observado na doc pública) |
| XML / XSD | SAF-T (AO) (`AO-SAF-001`) |
| Operação offline Edge | Contingência e persistência local (`AO-OFF-*`) |
| Facilidade de auditoria | Trilho append-only, evidências reproduzíveis (`AO-AUD-001`) |
| Distribuição Linux | Binário ou pacote com atualização assinada (`AO-UPD-001`) |
| Observabilidade segura | Métricas/estados sem payload fiscal completo |

## Alternativa A — Go + PostgreSQL (recomendada)

### Visão

Monólito modular em **Go**, API HTTP alinhada ao OpenAPI, **PostgreSQL** como sistema de registo, outbox transacional na mesma base, workers no mesmo deploy (processos/goroutines ou binários auxiliares partilhando o núcleo), portal web em **TypeScript + React**, Edge como **binário único** gerido por `systemd`.

| Área | Escolha proposta | Notas |
|---|---|---|
| Backend | Go (versão LTS/estável da equipa) | Bom para binário Edge; tipagem e deploy simples |
| Decimal | `shopspring/decimal` ou equivalente auditado; ou inteiros na menor unidade no domínio interno | Nunca `float64` para dinheiro |
| Base de dados | PostgreSQL 16+ | `NUMERIC`, constraints únicos, `SERIAL`/sequências controladas pela aplicação para séries fiscais |
| Filas | Outbox na PostgreSQL + worker poller; opcional NATS/Redis só para webhooks não fiscais | Evita dual-write; privilégia consistência fiscal |
| Portal | TypeScript + React (Vite ou framework leve) | Consome a mesma API; sem regras fiscais no browser |
| Edge Linux | Mesmo binário/núcleo; store PostgreSQL embutido **ou** SQLite só se a paridade transacional for demonstrada — **preferir PostgreSQL embutido/local** no MVP Edge se a operação o permitir; caso contrário, decisão explícita DEC futura | Paridade de pacote fiscal |
| Criptografia | Bibliotecas RSA/SHA-256 maduras; JWS (ex.: biblioteca JOSE auditada) | Chaves via KMS/keystore — não no código |
| XML/XSD | Geração determinística + validação XSD (libxml ou equivalente) | Só com XSD oficial quando disponível |
| Observabilidade | OpenTelemetry + logs estruturados + Prometheus | Campos redigidos |
| Testes | `go test`, contract tests OpenAPI, property tests de dinheiro/séries, contentores Testcontainers | Vetores `AO-*` |
| Deployment cloud | Contentores + IaC; migrações versionadas | Ambientes em [deployment.md](../07-operations/deployment.md) |
| Deployment Edge | Pacote `.deb`/binário + `systemd` + atualizador assinado | [edge-architecture.md](../02-architecture/edge-architecture.md) |

### Vantagens

- Distribuição Edge enxuta e auditoria de binário mais simples.
- Excelente controlo de concorrência e timeouts para idempotência.
- Um único repositório de núcleo fiscal facilita paridade cloud/Edge.
- Ecossistema adequado a serviços long-running e workers.

### Riscos

- Ecossistema XML/XSD menos «enterprise default» que Java — mitigar com libs maduras e testes de schema.
- Disciplina de módulos em Go exige convenções claras (packages internos, sem acesso cruzado a tables).
- Contratação: mercado Go vs Java pode variar conforme a equipa.

## Alternativa B — Java 21 + Spring Boot + PostgreSQL

### Visão

Monólito modular em **Java 21** com **Spring Boot**, mesma PostgreSQL e mesmo modelo de outbox, portal idêntico em TypeScript/React, Edge como JAR ou imagem contentorizada sob `systemd`.

| Área | Escolha proposta | Notas |
|---|---|---|
| Backend | Java 21 + Spring Boot | Ecossistema fiscal/enterprise maduro |
| Decimal | `BigDecimal` com política de escala explícita | Proibir `double`/`float` |
| Base de dados | PostgreSQL 16+ + Flyway/Liquibase | Transações JPA/JDBC com isolation adequada |
| Filas | Outbox JDBC + `@Scheduled`/worker; opcional broker só periférico | Igual ênfase em consistência |
| Portal | TypeScript + React | Igual à A |
| Edge Linux | JRE/`jlink` + `systemd` ou contentor | Imagem maior que Go |
| Criptografia | JCA/Bouncy Castle + JOSE | HSM/KMS via providers |
| XML/XSD | JAXB/`javax.xml` / bibliotecas de validação XSD maduras | Ponto forte vs Go |
| Observabilidade | Micrometer + OTel | Idem redacção |
| Testes | JUnit, Testcontainers, ArchUnit para limites de módulos | Conformidade `AO-*` |
| Deployment | Contentores + IaC | Similar à A |

### Vantagens

- XML/XSD, tipagem e tooling de auditoria muito maduros.
- Ampla disponibilidade de programadores e padrões de monólito modular.
- Integração KMS/HSM frequentemente bem documentada.

### Riscos

- Footprint Edge e tempos de arranque superiores.
- Complexidade de framework se os limites de módulo não forem policed (ArchUnit obrigatório).
- Mais superfície de configuração para o mesmo resultado fiscal.

## Comparação direta

| Dimensão | Alternativa A (Go) | Alternativa B (Java) |
|---|---|---|
| Transações fortes | PostgreSQL + outbox | PostgreSQL + outbox |
| Decimal | Libs dedicadas / inteiros | `BigDecimal` (nativo e familiar) |
| RSA / JWS | Maduro, escolha de lib crítica | Maduro, JCA |
| XML / XSD | Adequado com esforço | Mais forte por defeito |
| Offline Edge | Binário leve, excelente | Viável, mais pesado |
| Auditoria | Clara com packages + SQL | Clara com módulos + ArchUnit |
| Tempo até vertical slice | Potencialmente mais curto se a equipa conhece Go | Potencialmente mais curto se a equipa conhece Java |
| Risco operacional Edge | Menor footprint | Maior footprint / patch JRE |

## Recomendação

**Adotar a Alternativa A (Go + PostgreSQL)**, desde que a equipa aceite o investimento em validação XML/XSD e convenções de monólito modular.

Escolher a **Alternativa B** se:

- a equipa núcleo for predominantemente Java; ou
- a validação XSD/SAF-T for o caminho crítico imediato pós-obtenção do schema oficial.

Em ambos os casos:

- PostgreSQL é a base de dados recomendada;
- outbox transacional > fila externa para o caminho fiscal;
- portal separado sem lógica de numeração/assinatura;
- sem microserviços na Fase 0/1;
- decisão formal via DEC-STACK-001 **antes** do scaffold.

## Explicitamente fora desta proposta

- Escolha de cloud vendor (AWS/Azure/GCP) — pode ser ADR posterior.
- Bus de eventos empresarial.
- Bases NoSQL para o livro fiscal.
- Linguagens que empurram número IEEE-754 como tipo monetário por defeito na API interna.

## Próximo passo

Registar a decisão em [open-decisions.md](open-decisions.md) (DEC-STACK-001) e só então iniciar o monorepo na Fase 1.
