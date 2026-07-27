# SMTP notifications (sandbox / ops)

**Item:** `RM-OPS-008`
**Âmbito:** notificações operacionais do backoffice (teste autorizado)
**≠** AGT · **≠** emails com passwords · **≠** IdP

## Política

| Regra | Valor |
|---|---|
| Transporte | **Implicit TLS** apenas (`FISCAL_SMTP_TLS_MODE=implicit` ou alias `implicit_tls`) |
| Porta | **465** obrigatória |
| Segredos | ficheiro `/etc/bwb-modulo-fiscal/smtp.env` `root:root` `0600`; nunca em Git, logs ou respostas |
| Destinatário do teste | só `FISCAL_SMTP_ADMIN_NOTIFICATION_EMAIL` (não há recipient arbitrário na API) |
| RBAC HTTP | `notify.test` — **owner-only** |
| Conteúdo | texto sanitizado; sem passwords, tokens, NIF, DSN, PEM ou credenciais AGT |

## Ficheiros

| Path | Papel |
|---|---|
| `.env.smtp.local` | operador (gitignored) → instalado como `smtp.env` |
| `deploy/smtp.env.allowlist` | chaves exactas permitidas |
| systemd `EnvironmentFile=-/etc/bwb-modulo-fiscal/smtp.env` | injecta no `fiscal-api` (ficheiro opcional) |

## Endpoints / CLI

1. `POST /admin/v1/ops/notifications/test` — Bearer owner; devolve `DeliveryStatus` sanitizado.
2. Helper (sandbox sem IdP): `bwb-fiscal-deploy-helper smtp-send-test <sha40>` — corre `fiscal-api smtp-send-test` drop-priv; imprime JSON sanitizado.

## Status sanitizado

```json
{"status":"sent","to_domain":"example.com","tls_mode":"implicit","port":465,"request_id":"..."}
```

Falhas: `status=failed` + `reason` allowlist (`smtp_timeout`, `smtp_auth_failed`, `smtp_tls_failed`, `smtp_connect_failed`, `smtp_send_failed`).

## Testes

`internal/notify/smtp` usa servidor SMTP **fake com TLS** (listener `tls.Listen`); proibido SMTP em claro nos testes.
