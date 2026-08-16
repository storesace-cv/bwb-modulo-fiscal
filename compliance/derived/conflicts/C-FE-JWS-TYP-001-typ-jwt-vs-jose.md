# C-FE-JWS-TYP-001 — Protected header `typ`: JWT vs JOSE

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-08-16 |
| Severidade | média (interop JWS FE) |

## Factos

1. `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` (`pending_validation`) ilustra header com `"alg":"RS256"` e `"typ":"JWT"`.
2. Exemplos compactos em `AO-FE-SNAP-HML-2026-07-25-REGISTAR` (e serviços correlatos) decodificam header `{"typ":"JOSE","alg":"RS256"}`.
3. Não há texto inequívoco que feche JWT vs JOSE como regra AGT.

## Mitigação engenharia (RM-FEFIX-003)

- Motor `internal/fejws` **não** define `typ` por defeito.
- Perfis `eligible` assinam só com `{"alg":"RS256"}`.
- Testes podem passar `typ` explicitamente sem o declarar regra AGT.

## Não fazer

- Não escolher JWT ou JOSE silenciosamente.
- Não promover snapshot a `confirmed_normative` por causa deste conflito.
