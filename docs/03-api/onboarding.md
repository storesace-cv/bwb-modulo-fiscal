# Onboarding — software house / POS

## Dados necessários

| Parte | Sandbox | Produção futura |
|---|---|---|
| Contribuinte | Identidade fiscal **sintéticas** (não oficial) alinhada ao scope | Dados reais via processo formal (regras ainda abertas) |
| Lojas / scopes | Scope `homologation` + timezone `Africa/Luanda` | Por estabelecimento, quando existir |
| Software house | Contactos técnicos | Contrato comercial |

O contacto operacional e o canal seguro de credenciais são fornecidos pela BWB durante o onboarding.

## Credenciais (estado actual)

Até existir backoffice self-service, a BWB opera via `fiscal-admin` / helper de staging:

1. Criar scope (identidade sintética no sandbox).
2. Emitir credencial → **token mostrado uma única vez**.
3. Entregar por canal seguro; guardar só em ficheiro `0600` (não git, não logs, não argv).
4. Rodar / revogar conforme política BWB.

**Sandbox ≠ produção.** Hostname de produção apex está reservado e não está disponível neste draft.

## Responsabilidades

| Actor | Responsável por |
|---|---|
| BWB | Módulo, scopes sandbox, emissão/revogação de credenciais, OpenAPI, suporte de onboarding |
| Software house | Cliente POS, custódia do token, idempotência, retries seguros, evidências do checklist |
| Contribuinte | Dados comerciais/fiscais reais quando a produção existir; autorização de uso do módulo |

## Kit e evidências

- Kit: `scripts/integration/pos-sandbox-kit.sh`
- Caso `token_revoked_401` no checklist só fecha com `--revoked-token-file` fornecido pela BWB (evidência real).
