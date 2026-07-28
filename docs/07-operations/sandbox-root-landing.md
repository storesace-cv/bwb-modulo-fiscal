# Sandbox — landing na raiz (`RM-OPS-010`)

## Problema

No site **open** do sandbox, caminhos desconhecidos (incluindo `/`) devolvam **404** via Nginx, enquanto `/v1/health` e `/admin/*` respondiam normalmente. Isso confundia operadores: 404 na raiz parecia «serviço em baixo».

## Distinção obrigatória

| Sinal | Significado |
|---|---|
| `GET /v1/health` (e admin health/ready) | Disponibilidade / liveness |
| `GET /` (landing HTML) | **Só UX** — orientação; **não** é health check |
| 404 em path desconhecido | Esperado (API allowlist) |

`FISCAL_ENV=homologation` no sandbox continua a ser designação técnica BWB — **não** homologação AGT.

## Entrega

- App: `internal/landing` montado em `GET /{$}` / `HEAD /{$}` (raiz exacta).
- Nginx open: `location = /` → `proxy_pass` à API; catch-all `/` permanece 404.
- Deny-all: **sem** proxy da raiz (continua 404) — rollback fail-closed.

## Segurança

Página estática sem JS externo, sem segredos/NIF/tokens; CSP restritiva; `Cache-Control: no-store`.
