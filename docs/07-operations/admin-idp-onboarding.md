# Onboarding IdP provider-neutral (Admin API / UI)

**Âmbito:** autenticação de operadores do backoffice (`/admin/v1`, `/admin/ui`)  
**≠** POS `credential_store` · **≠** credenciais AGT · **≠** IdP de teste local inseguro  
**Itens:** `RM-BO-006` (fundação OIDC/JWT) · `RM-BO-018` (onboarding + readiness) · `RM-UI-005` (sessão)

## Política fail-closed

| Modo | Onde | Comportamento |
|---|---|---|
| `fail_closed` (omissão) | sandbox/produção sem IdP | Nega autenticação; processo sobe; UI/API admin exigem auth |
| `injected` | **só** `FISCAL_ENV=development` | Claims estáticos para testes locais — **proibido** no sandbox |
| `oidc_jwt` | sandbox/produção com IdP real | Valida Bearer JWT via JWKS **https**; mapa de roles; owner allowlist |

**Proibido:** inventar IdP local, passwords em claro, bypass CSRF, `FISCAL_ADMIN_AUTH_MODE=injected` em `homologation`/`production`, copiar credenciais AGT para o admin.

## Diagnóstico (sem segredos)

`GET /admin/v1/ready` (público) inclui checks sanitizados:

| Check | Valores |
|---|---|
| `admin_auth_mode` | `fail_closed` \| `injected` \| `oidc_jwt` |
| `admin_oidc` | `ok` \| `not_configured` \| `incomplete` |
| `admin_interactive_login` | `unavailable` (redirect authorize ainda não ligado) \| `ready` (futuro) |

Código: `adminauth.DiagnoseAuthReadiness` — **não** contacta o IdP; **não** devolve issuer completo com segredos, JWKS, tokens ou subjects.

No sandbox actual: espera-se `fail_closed` + `admin_oidc=not_configured` + `admin_interactive_login=unavailable`.

## Variáveis (provider-neutral)

Documentadas em [`docs/06-delivery/local-dev.md`](../06-delivery/local-dev.md):

- `FISCAL_ADMIN_AUTH_MODE=oidc_jwt`
- `FISCAL_ADMIN_OIDC_ISSUER` — `iss` exacto
- `FISCAL_ADMIN_OIDC_AUDIENCE` — `aud` exacto
- `FISCAL_ADMIN_OIDC_JWKS_URL` — **https** obrigatório
- `FISCAL_ADMIN_OIDC_ALGS` — allowlist (default `RS256`)
- `FISCAL_ADMIN_OIDC_ROLE_CLAIM` / `FISCAL_ADMIN_OIDC_ROLE_MAP`
- `FISCAL_ADMIN_OIDC_OWNER_SUBJECTS` — allowlist de `sub` para `owner`
- `FISCAL_ADMIN_OWNER_SUBJECT` — gate SecAdm

Estas chaves **ainda não** estão na allowlist de `fiscal.env` do sandbox. Activar `oidc_jwt` no host exige:

1. Decisão de segurança (fornecedor IdP + tenants).
2. Extensão explícita de `deploy/env.allowlist` + valores em `.env.deploy.local` (sem commit de segredos).
3. Deploy via updater endurecido.
4. Verificação de `/admin/v1/ready` (`admin_oidc=ok`) e mint de sessão com Bearer real.

## Fluxo de sessão (quando oidc_jwt estiver activo)

1. Operador obtém JWT no IdP (fora deste módulo — redirect authorize **ainda não** ligado).
2. `POST /admin/ui/auth/session` com `Authorization: Bearer <JWT>` → cookie opaca HttpOnly.
3. Pedidos UI usam a cookie; logout com CSRF.
4. **Não** persistir JWT no browser.

## Relatório separado (ops)

Após deploy do código:

1. **Código online** — `health.revision` = `current-sha` = `origin/main`.
2. **Admin UI alcançável** — HTTPS `/admin/ui/login` ou shell responde (HTML); rotas protegidas 401/403.
3. **Login interactivo indisponível** — até IdP real + `oidc_jwt` + (futuro) redirect authorize.

## Sandbox vs produção

- Sandbox `FISCAL_ENV=homologation` é técnico BWB ≠ homologação AGT.
- `FISCAL_AUTHORITY=simulator` permitido em homologation/development; **proibido** em `production`.
- Apex `fiscalmod.bwb.pt` fora de âmbito deste onboarding.
