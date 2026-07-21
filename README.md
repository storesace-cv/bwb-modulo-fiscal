# BWB Módulo Fiscal

Documentação inicial para construção de uma plataforma fiscal certificável em Angola, integrável por API com sistemas POS e executável na cloud ou localmente em Linux. Cabo Verde constitui a segunda fase.

## Premissa de projeto

Para efeitos desta fase assume-se que a certificação do módulo fiscal externo pela AGT dispensa a validação individual de cada POS, desde que o POS não produza autonomamente o resultado fiscal e use o módulo como autoridade exclusiva de emissão/certificação.

Esta é uma premissa de produto (`ASM-REG-001`), não uma conclusão jurídica. A arquitetura deve permitir rever esta decisão sem reescrever o núcleo.

## Ordem de leitura

1. [docs/00-product/vision.md](docs/00-product/vision.md)
2. [docs/00-product/scope.md](docs/00-product/scope.md)
3. [docs/01-compliance/angola-compliance.md](docs/01-compliance/angola-compliance.md)
4. [docs/01-compliance/sources.md](docs/01-compliance/sources.md)
5. [docs/01-compliance/official-access-plan.md](docs/01-compliance/official-access-plan.md)
6. [docs/02-architecture/system-architecture.md](docs/02-architecture/system-architecture.md)
7. [docs/03-api/api-guidelines.md](docs/03-api/api-guidelines.md)
8. [docs/04-domain/domain-model.md](docs/04-domain/domain-model.md)
9. [docs/05-security/security-baseline.md](docs/05-security/security-baseline.md)
10. [docs/06-delivery/implementation-roadmap.md](docs/06-delivery/implementation-roadmap.md)
11. [docs/07-operations/operations.md](docs/07-operations/operations.md)

## Regras para desenvolvimento assistido

O ficheiro [AGENTS.md](AGENTS.md) é a fonte principal de instruções para Cursor/Codex. Antes de alterar código fiscal, consultar também o catálogo de requisitos e os ADRs.

Antes de qualquer ação, Cursor/agentes devem ainda ler [ENGINEERING_PRINCIPLES.md](ENGINEERING_PRINCIPLES.md), que define a postura sénior, ceticismo, segurança e padrão de qualidade obrigatório.

## Estado

- Etapa: fundação de persistência (PR A) + API health.
- País ativo: Angola.
- País futuro: Cabo Verde.
- Código: Go com health + schema/migrations; sem emissão fiscal HTTP.
- Contrato OpenAPI: `specs/openapi/openapi.yaml` (`0.1.1-draft`).
- Desenvolvimento local: [docs/06-delivery/local-dev.md](docs/06-delivery/local-dev.md).
