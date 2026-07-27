# Observabilidade do backoffice (RM-BO-007)

**Âmbito:** `/admin/v1` + `/admin/ui` · **≠** POS `/v1` · **≠** SecAdm plaintext
**Data:** 2026-07-27

## Endpoints

| Rota | Auth | Uso |
|---|---|---|
| `GET /admin/v1/health` | não | Liveness do componente admin |
| `GET /admin/v1/ready` | não | Readiness: ping BD (timeout 2s), `admin_auth_mode`, presença do gate SecAdm (`configured`/`absent`) |
| `GET /admin/v1/ops/metrics` | `ops.read` | Contadores de baixa cardinalidade + `auth_ok`/`auth_fail` |
| `GET /admin/v1/ops/submissions` | `ops.read` | Fila sanitizada (RM-BO-015) |
| `POST /admin/v1/ops/submissions/{id}/actions` | `ops.write` | retry/cancel/manual_review (RM-BO-016); métricas sem payload |

Todas as respostas admin observáveis incluem / propagam `X-Request-Id` (`areq_…`).

## Logs estruturados

Evento `admin_request` (JSON via `slog`):

- **Inclui:** `request_id`, `route_class`, `method`, `status`, `duration_ms`, `roles` (allowlist `owner|admin|operator|auditor`), `auth_mode`
- **Nunca inclui:** `Authorization`, cookies, JWT, DSN, JWKS, chaves, NIF, subject, corpos, path com IDs

`route_class` colapsa IDs (`/admin/v1/taxpayers/{id}` → `taxpayers`). SecAdm permanece classe separada `secadm`.

## Métricas

- Labels: `route_class`, `method`, `outcome` (`ok|unauthorized|forbidden|validation|error`)
- Limite: 256 séries; excesso incrementa `series_dropped` (fail-safe, sem panic)
- Contadores de autenticação agregados (`auth_ok` / `auth_fail`) — sem subject

## Diagnóstico rápido

1. `curl -sS http://127.0.0.1:8080/admin/v1/health`
2. `curl -sS http://127.0.0.1:8080/admin/v1/ready` — `checks.database`, `admin_auth_mode`, `secadm_gate`
3. Com Bearer operador: `GET /admin/v1/ops/metrics` — ver `auth_fail` vs `auth_ok` e séries `unauthorized`/`forbidden`
4. Correlacionar com logs pelo `request_id` / header `X-Request-Id`

## Separação SecAdm

- Readiness só reporta se o **gate** está configurado — nunca estado de refs/segredos.
- Métricas `secadm` contam pedidos HTTP write-only; auditoria append-only continua em `admin_audit_events`.
- Plano A (cadastros/ops/UI) e plano B (SecAdm) partilham correlação, não material secreto.
