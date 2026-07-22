# Nginx artefacts shipped in the Linux release (S3C2).

| File | Role |
|---|---|
| `tls.deny.conf` | Immutable deny-all site (rollback / default safe) |
| `tls.open.conf` | Public documents open with `limit_req` rate=10r/s burst=20 |
| `limit-req-documents.conf` | `limit_req_zone` for `bwb_documents` |

Installed on the host **only** by closed helper ops:

- `nginx-open-arm <sha40>` — install open + arm 5‑minute systemd rollback timer; disable `:18080`
- `nginx-open-confirm <sha40>` — cancel timer after successful gates
- `nginx-deny-all <sha40>` — restore deny-all, cancel timer, verify 403
- `nginx-open-rollback-fire` — timer target (no operator args)

Updater/`install-release` never activates open. Operator cannot pass paths/URLs/commands.
