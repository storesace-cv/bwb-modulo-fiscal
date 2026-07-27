# RM-OPS-008 — SMTP foundation + sandbox deploy

**Host:** `sandbox.fiscalmod.bwb.pt` (nunca apex produção)
**Código live:** `38f86fb52367937e6595192ab39da68097ff4a53` (`origin/main`)

## Relatório separado

| # | Declaração | Estado |
|---|---|---|
| 1 | Código online (`health.revision` = `current-sha` = `main`) | **SIM** — `38f86fb…` / `0.2.74-staging` |
| 2 | Admin UI alcançável | **SIM** — `/admin/ui/login` 200; ready `fail_closed` |
| 3 | Login interactivo indisponível | **SIM** — `admin_interactive_login=unavailable`; OIDC `not_configured` |

## Deploy (updater endurecido)

| Campo | Valor |
|---|---|
| Schema | `12` / `dirty=false` |
| `smtp.env` | `root:root` `0600` |
| systemd | `EnvironmentFile=-/etc/bwb-modulo-fiscal/smtp.env` |
| Documents sem token | **401** |
| PG/API externos | fechados |

## Diagnóstico TLS estrito (pós-renovação do certificado do provider)

Sem autenticação; sem `InsecureSkipVerify`; sem plaintext/oportunista.

| Porta | Modo | TCP | Cadeia + hostname (host configurado) |
|---|---|---|---|
| 465 | implicit | ok | **ok** (TLSv1.2; cert válido ~89 dias) |
| 587 | STARTTLS obrigatório | ok | **ok** (`AUTH` após TLS) |
| 25 | STARTTLS obrigatório | ok | **ok** (`AUTH` após TLS) |

**Recomendação:** `port=465` `mode=implicit` (host configurado inalterado; `host_changed=false`).

Config local / `smtp.env` **não** alterados (já alinhados com a combinação recomendada).

## Teste SMTP autorizado (sanitizado)

### Tentativa inicial (cert expirado / mismatch) — 2026-07-27

```json
{"status":"failed","reason":"smtp_tls_failed","to_domain":"bwb.pt","tls_mode":"implicit","port":465,"request_id":"cli_smtp_send_test"}
```

### Re-teste após renovação do certificado do provider — 2026-07-27

Via `bwb-fiscal-deploy-helper smtp-send-test` (sem IdP; sem expor password):

```json
{"status":"sent","to_domain":"bwb.pt","tls_mode":"implicit","port":465,"request_id":"cli_smtp_send_test"}
```

**Confirmação de entrega (sanitizada):** owner confirmou recepção do email de teste no domínio `bwb.pt` (2026-07-27). Sem reenvio; sem alteração de configuração SMTP.

## Incidentes

| ID | Descrição | Estado |
|---|---|---|
| INC-OPS-008-1 | Helper host sem suporte `smtp.env` → restore envs | **RESOLVIDO** |
| INC-OPS-008-2 | `smtp_tls_failed` (cert expirado + hostname) | **RESOLVIDO** — provider renovou cert; 465 implicit validado; teste `sent` + confirmação owner |

## Evidências

- PR #122 — https://github.com/storesace-cv/bwb-modulo-fiscal/pull/122
- Docs — [smtp-notifications.md](smtp-notifications.md)
