# RM-OPS-008 — SMTP foundation + sandbox deploy

**Host:** `sandbox.fiscalmod.bwb.pt` (nunca apex produção)
**Data (UTC):** 2026-07-27
**Código:** `1c425ec1863a5245da2ef07d0191a7b45cc79303` (`origin/main`)

## Relatório separado

| # | Declaração | Estado |
|---|---|---|
| 1 | Código online (`health.revision` = `current-sha` = `main`) | **SIM** — `1c425ec…` / `0.2.74-staging` |
| 2 | Admin UI alcançável | **SIM** — `/admin/ui/login` 200; ready `fail_closed` |
| 3 | Login interactivo indisponível | **SIM** — `admin_interactive_login=unavailable`; OIDC `not_configured` |

## Deploy (updater endurecido)

| Campo | Valor |
|---|---|
| Lock | `20260727T200634Z-1c425ec…` (adquirido/libertado) |
| Gate `pg_dump` | ok |
| Env backup | ok (incl. smtp após sync helper) |
| Schema | `12` → `12` `dirty=false` |
| `smtp.env` | `root:root` `0600` instalado |
| systemd | `EnvironmentFile=-/etc/bwb-modulo-fiscal/smtp.env` |
| Documents sem token | **401** |
| PG/API externos | fechados |
| Nginx open | re-arm+confirm no tip; timer inactive |

**Nota:** primeira tentativa falhou com helper antigo (`invalid env name` em `smtp.env`); helper/libs sincronizados; redeploy OK.

## Teste SMTP autorizado (sanitizado)

Via `bwb-fiscal-deploy-helper smtp-send-test <sha>` (sem IdP; sem expor password):

```json
{"status":"failed","reason":"smtp_tls_failed","to_domain":"bwb.pt","tls_mode":"implicit","port":465,"request_id":"cli_smtp_send_test"}
```

Fundação e caminho operacional estão correctos; a entrega real falhou na verificação TLS do servidor SMTP configurado. **Decisão pendente:** corrigir certificado/hostname do provider SMTP (sem `InsecureSkipVerify`).

## Incidentes

| ID | Descrição | Estado |
|---|---|---|
| INC-OPS-008-1 | Helper host sem suporte `smtp.env` → restore envs | **RESOLVIDO** (sync helper + redeploy) |
| INC-OPS-008-2 | `smtp-send-test` → `smtp_tls_failed` | **ABERTO** (config TLS do SMTP externo) |

## Evidências

- PR #122 — https://github.com/storesace-cv/bwb-modulo-fiscal/pull/122
- Docs — [smtp-notifications.md](smtp-notifications.md)
