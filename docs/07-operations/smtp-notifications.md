# SMTP notifications (sandbox / ops)

**Itens:** `RM-OPS-008` · `RM-OPS-009`
**Âmbito:** notificações operacionais do backoffice (teste autorizado + digest de alertas)
**≠** AGT · **≠** emails com passwords · **≠** IdP · **≠** envio automático em background

## Política

| Regra | Valor |
|---|---|
| Transporte | **Implicit TLS** apenas (`FISCAL_SMTP_TLS_MODE=implicit` ou alias `implicit_tls`) |
| Porta | **465** obrigatória |
| Segredos | ficheiro `/etc/bwb-modulo-fiscal/smtp.env` `root:root` `0600`; nunca em Git, logs ou respostas |
| Destinatário | só `FISCAL_SMTP_ADMIN_NOTIFICATION_EMAIL` (não há recipient arbitrário na API) |
| RBAC HTTP | `notify.test` — **owner-only** |
| Conteúdo | texto sanitizado; sem passwords, tokens, NIF, DSN, PEM ou credenciais AGT |

## Ficheiros

| Path | Papel |
|---|---|
| `.env.smtp.local` | operador (gitignored) → instalado como `smtp.env` |
| `deploy/smtp.env.allowlist` | chaves exactas permitidas |
| systemd `EnvironmentFile=-/etc/bwb-modulo-fiscal/smtp.env` | injecta no `fiscal-api` (ficheiro opcional) |

## Endpoints / CLI

1. `POST /admin/v1/ops/notifications/test` — Bearer owner; devolve `DeliveryStatus` sanitizado (`RM-OPS-008`).
2. `POST /admin/v1/ops/notifications/alerts-digest` — Bearer owner; carrega alertas da fila ops (`RM-BO-017`), envia digest allowlist e devolve `DeliveryStatus` + `alert_count` + `alert_codes` (`RM-OPS-009`).
3. Helper (sandbox sem IdP): `bwb-fiscal-deploy-helper smtp-send-test <sha40>` — corre `fiscal-api smtp-send-test` drop-priv; imprime JSON sanitizado (só teste; digest exige Admin API autenticada).

## Status sanitizado

```json
{"status":"sent","to_domain":"example.com","tls_mode":"implicit","port":465,"request_id":"..."}
```

Digest:

```json
{"status":"sent","to_domain":"example.com","tls_mode":"implicit","port":465,"request_id":"...","alert_count":1,"alert_codes":["ops_retry_backlog"]}
```

Falhas: `status=failed` + `reason` allowlist (`smtp_timeout`, `smtp_auth_failed`, `smtp_tls_failed`, `smtp_connect_failed`, `smtp_send_failed`).

## Testes

`internal/notify/smtp` usa servidor SMTP **fake com TLS** (listener `tls.Listen`); proibido SMTP em claro nos testes.
