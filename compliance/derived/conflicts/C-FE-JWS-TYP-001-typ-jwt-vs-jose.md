# C-FE-JWS-TYP-001 — Protected header `typ`: JWT vs JOSE

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-08-16 |
| Severidade | média (interop JWS FE) |
| Registo AGT | [AGT-Q-005](../../../docs/01-compliance/agt-clarifications-register.md#agt-q-005) |

## Factos

1. `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` (`pending_validation`) ilustra header com `"alg":"RS256"` e `"typ":"JWT"`.
2. Exemplos compactos em `AO-FE-SNAP-HML-2026-07-25-REGISTAR` (e serviços correlatos) decodificam header `{"typ":"JOSE","alg":"RS256"}`.
3. Não há texto inequívoco que feche JWT vs JOSE como regra AGT.
4. A documentação consultada **não** afirma que `typ` possa ser omitido.

## Mitigação engenharia (RM-FEFIX-003)

- Motor `internal/fejws` **não** define `typ` por defeito (primitiva técnica; testes podem passar `typ` explicitamente **sem** o declarar regra AGT).
- Perfis com payload confirmado no snapshot (`software_info`, `obter_estado_request`, `consultar_factura_request`) expõem **apenas** builders de claims.
- Métodos operacionais de assinatura devolvem `ErrProfileBlocked` com este conflito — **zero** perfis wire AGT activos.
- **Proibido** omitir `typ` como terceira variante “aprovada” ou escolher JWT/JOSE silenciosamente.

## Não fazer

- Não escolher JWT ou JOSE silenciosamente.
- Não omitir `typ` para contornar este conflito (“intersecção mínima”).
- Não promover snapshot a `confirmed_normative` por causa deste conflito.
- Não apresentar JWS assinado localmente como aceite AGT.
