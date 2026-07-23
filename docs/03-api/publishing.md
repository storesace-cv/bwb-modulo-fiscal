# Publicação do contrato e breaking changes

## Artefactos públicos

| Artefacto | Local |
|---|---|
| OpenAPI | `specs/openapi/openapi.yaml` (`0.1.6-draft`) |
| Guias | `docs/03-api/` |
| Exemplos | `docs/03-api/examples/` (sem credenciais) |
| Kit | `scripts/integration/` |

Descarregar / consultar o YAML no repositório. Validar com ferramentas OpenAPI 3.1 (ex. Redocly).

## Versionamento

- OpenAPI `0.1.x-draft`: muda só com alteração de contrato.
- CHANGELOG da aplicação `0.2.x-draft`: incrementos de produto/docs/tooling.

## Breaking changes

Alterações incompatíveis no contrato draft devem:

1. Bumpar a versão OpenAPI draft;
2. Registar entrada no CHANGELOG;
3. Notificar integradores pelo canal definido no onboarding BWB.

O contacto operacional e o canal seguro de credenciais são fornecidos pela BWB durante o onboarding.

## Autenticação (texto público)

Descrever apenas: **Bearer credential issued by BWB for the sandbox**.  
Detalhes de implementação interna do validador permanecem na documentação operacional.
