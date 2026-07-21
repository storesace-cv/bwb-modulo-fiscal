#!/usr/bin/env bash
# Health check. Live path runs curl on the remote host against 127.0.0.1:8080.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/v1/health}"

check_body() {
  local body="$1"
  if [[ "${body}" != *'"status"'* ]]; then
    echo "error: health response missing status" >&2
    return 1
  fi
  if [[ "${body}" != *'"ok"'* && "${body}" != *'"OK"'* ]]; then
    if [[ ! "${body}" =~ \"status\"[[:space:]]*:[[:space:]]*\"ok\" ]]; then
      echo "error: health status not ok" >&2
      return 1
    fi
  fi
}

if [[ "${DEPLOY_DRY_RUN:-0}" == "1" && "${DEPLOY_MOCK_REMOTE:-0}" != "1" ]]; then
  echo "dry_run health_skip url_set=1"
  exit 0
fi

# Live / mock-remote: health must be evaluated on the server loopback.
if [[ -n "${DEPLOY_HOST:-}" && -n "${DEPLOY_USER:-}" ]]; then
  deploy_ssh_base
  # shellcheck disable=SC2029
  body="$("${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" \
    "set -Eeuo pipefail; curl -fsS --max-time 5 '${HEALTH_URL}'")"
  check_body "${body}"
  echo "health_ok remote=1"
  exit 0
fi

deploy_require_cmds curl
body="$(curl -fsS --max-time 5 "${HEALTH_URL}")"
check_body "${body}"
echo "health_ok"
