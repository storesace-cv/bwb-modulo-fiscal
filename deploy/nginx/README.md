# Nginx artefacts shipped in the Linux release (S3C2).

| File | Role |
|---|---|
| `tls.deny.conf` | Immutable deny-all site (rollback / default safe); HSTS; ACME-safe redirect |
| `tls.open.conf` | Public `location = /v1/documents` open with `limit_req` rate=10r/s burst=20 |
| `limit-req-documents.conf` | `limit_req_zone` for `bwb_documents` |

Installed on the host **only** by closed helper ops (exclusive flock on `/var/lock/bwb-fiscal-nginx-open.lock`):

- `nginx-open-arm <sha40>` — install open + arm 5‑minute timer (must be **active**); disable `:18080`; on failure: proven `deny_restored`, proven `emergency_nginx_stop` (`is-active != active`), or CRITICAL `emergency_stop_failed`; never `arm_ok` without active timer; failed reload uses the same fail-closed path
- `nginx-open-confirm <sha40>` — persist `state=confirmed` **before** cancelling timer
- `nginx-deny-all <sha40>` — restore deny-all, cancel timer, verify 403
- `nginx-open-rollback-fire` — timer target: deny-all if still `armed`; noop if `confirmed`
- `nginx-open-boot-recovery` — unit `Before=nginx.service`: if `armed`, deny-all on disk + `nginx -t` only; `confirmed` stays open
- Drop-in `systemd/nginx.service.d/bwb-fiscal-open-boot-recovery.conf` — `Requires=`/`After=` boot-recovery so a failed armed recovery **blocks** `nginx.service` start

Updater/`install-release` never activates open. Operator cannot pass paths/URLs/commands.
