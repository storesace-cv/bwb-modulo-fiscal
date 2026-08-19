# BWB Módulo Fiscal

Documentação inicial para construção de uma plataforma fiscal certificável em Angola, integrável por API com sistemas POS e executável na cloud ou localmente em Linux. Cabo Verde constitui a segunda fase.

## Premissa de projeto

Para efeitos desta fase assume-se que a certificação do módulo fiscal externo pela AGT dispensa a validação individual de cada POS, desde que o POS não produza autonomamente o resultado fiscal e use o módulo como autoridade exclusiva de emissão/certificação.

Esta é uma premissa de produto (`ASM-REG-001`), não uma conclusão jurídica. A arquitetura deve permitir rever esta decisão sem reescrever o núcleo.

## Ordem de leitura

1. [ROADMAP.md](ROADMAP.md) — estado e progresso canónicos
2. [docs/00-product/vision.md](docs/00-product/vision.md)
3. [docs/00-product/scope.md](docs/00-product/scope.md)
4. [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md)
5. [docs/01-compliance/sources.md](docs/01-compliance/sources.md)
6. [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md)
7. [docs/02-architecture/system-architecture.md](docs/02-architecture/system-architecture.md)
8. [docs/03-api/api-guidelines.md](docs/03-api/api-guidelines.md) · [quickstart S4](docs/03-api/quickstart.md)
9. [docs/04-domain/domain-model.md](docs/04-domain/domain-model.md)
10. [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md)
11. [docs/07-operations/operations.md](docs/07-operations/operations.md)

Apontador legado (compatibilidade): [docs/06-delivery/implementation-roadmap.md](docs/06-delivery/implementation-roadmap.md).

## Regras para desenvolvimento assistido

O ficheiro [AGENTS.md](AGENTS.md) é a fonte principal de instruções para Cursor/Codex. Antes de alterar código fiscal, consultar também o catálogo de requisitos e os ADRs.

Antes de qualquer ação, Cursor/agentes devem ainda ler [ENGINEERING_PRINCIPLES.md](ENGINEERING_PRINCIPLES.md), que define a postura sénior, ceticismo, segurança e padrão de qualidade obrigatório.

## Estado

- Progresso canónico: [ROADMAP.md](ROADMAP.md).
- Sandbox autenticado e **confirmado** (`S3C2 CONFIRMED`) em `https://sandbox.fiscalmod.bwb.pt` — `credential_store` + `FISCAL_ENV=homologation` (ambiente técnico BWB; **não** homologação AGT).
- Kit POS validado **9/9** no sandbox.
- Gate pré-deploy PostgreSQL (`pg_dump`) validado no Ubuntu; INC-S4-003 resolvido; INC-B2-001 aberto.
- Fontes fiscais **SRC-A** / **B0** / **B1** / **B2** concluídos no âmbito documentado; **OCR RM-SRC-004/RM-M2-C BLOQUEADOS** (falta Rect. 10/19 integral; 74/19+683/25 v2 com OCR reviewed); requisitos `AO-*` pendentes.
- Fundação transacional (SealInTx / `sealed_locally`) concluída para o slice inicial; **motor regulamentar Angola / integração oficial AGT ainda não implementados**.
- País activo: Angola. País futuro: Cabo Verde.
- Contrato OpenAPI POS: `specs/openapi/openapi.yaml` (`0.1.6-draft`).
- Admin API: [`specs/admin/openapi.yaml`](specs/admin/openapi.yaml) (`0.1.5-draft`, `/admin/v1`).
- Backoffice UI (M7): `/admin/ui/` — SSR Go (`internal/adminui`); owner: AuthorityProfile + SecAdm metadados.
- Schema: `ExpectedVersion=14`.
- Integração POS: [docs/03-api/quickstart.md](docs/03-api/quickstart.md) · kit [scripts/integration/](scripts/integration/).
- Desenvolvimento local: [docs/06-delivery/local-dev.md](docs/06-delivery/local-dev.md).
- Staging: [docs/07-operations/staging-runbook.md](docs/07-operations/staging-runbook.md).
