# Changelog

## 0.1.3-draft — 2026-07-21

- DEC-STACK-001 decidida: Go + PostgreSQL na cloud + SQLite WAL no Edge (condições XSD oficial, assinatura fiscal AGT e numeração preservadas).

## 0.1.2-draft — 2026-07-20

- Harmonização final do plano Fase 0: DEC-STACK-001 recomendada, `sealed_locally` único, OpenAPI tarefa zero, RSA efémero.
- Correção do plano técnico da Fase 0: at-least-once (sem exactly-once), JWS RS256 real com chaves de teste, estados neutros até DEC-API-004.
- Edge MVP com SQLite WAL (escritor único); PostgreSQL apenas na cloud.
- DEC-API-001, DEC-API-003 e DEC-DEL-001 decididas; DEC-API-004 aberta; prioridades de decisão reordenadas.
- Fase 0 interna reduzida a 2–4 semanas; vertical slice sem portal, webhooks nem frontend POS.
- Outbox distinta de logs operacionais; numeração sem promessa genérica de «zero buracos».

## 0.1.1-draft — 2026-07-20

- Plano executável da Fase 0 em `docs/06-delivery/phase-0-execution-plan.md`.
- Decisões técnicas e regulatórias em aberto em `docs/06-delivery/open-decisions.md`.
- Inventário de lacunas regulatórias em `docs/01-compliance/regulatory-gaps.md`.
- Proposta de stack (duas alternativas, sem implementação) em `docs/06-delivery/technical-stack-proposal.md`.
- Especificação do primeiro vertical slice (demo ponta a ponta) em `docs/06-delivery/first-vertical-slice.md`.
- Premissa `ASM-REG-001` mantida; OpenAPI e código de produção não alterados.
- Contradições documentais inventariadas (estados API, Money/quantity, proposta vs Decreto 74/19).

## 0.1.0-draft — 2026-07-20

- Documentação inicial do produto Angola-first.
- Registo da premissa `ASM-REG-001`.
- Arquitetura cloud/Edge e pacotes por país.
- Catálogo inicial de conformidade.
- Esqueleto OpenAPI.
- Baseline de segurança, testes, operações e roadmap.
- Portal do Contribuinte de Angola registado como fonte oficial prioritária.
- Guia Rápido de Emissão de Facturas e Portal institucional da AGT adicionados ao registo de fontes.
- Documentação técnica FE, Portal do Parceiro, Decreto 74/19 e área restrita de produtores registados no inventário de fontes.
- Criado plano de acesso, preservação e versionamento de artefactos oficiais.
- Definida `local/` como pasta exclusiva de consulta, integralmente excluída do GitHub.
