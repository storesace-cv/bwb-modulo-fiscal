#!/usr/bin/env bash
# Health check against loopback (or HEALTH_URL). Does not validate migration version.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/v1/health}"

if [[ "${DEPLOY_DRY_RUN:-0}" == "1" ]]; then
  echo "dry_run health_skip url_set=1"
  exit 0
fi

deploy_require_cmds curl
body="$(curl -fsS --max-time 5 "${HEALTH_URL}")"
# Do not dump full body if it might grow; check status field only.
if [[ "${body}" != *'"status"'* ]]; then
  echo "error: health response missing status" >&2
  exit 1
fi
if [[ "${body}" != *'"ok"'* && "${body}" != *'"OK"'* ]]; then
  # Accept common shapes: {"status":"ok"} or similar
  if [[ ! "${body}" =~ \"status\"[[:space:]]*:[[:space:]]*\"ok\" ]]; then
    echo "error: health status not ok" >&2
    exit 1
  fi
fi
echo "health_ok"
