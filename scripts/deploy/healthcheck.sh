#!/usr/bin/env bash
# Health check. Live path always probes server loopback; never trusts HEALTH_URL.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

# Fixed production/live probe target. Not overridable on the live path.
LIVE_HEALTH_URL="http://127.0.0.1:8080/v1/health"

# Accept only when the JSON field "status" is exactly "ok".
# Presence of "ok" in any other field must not count as success.
check_body() {
  local body="$1"
  if [[ ! "${body}" =~ \"status\"[[:space:]]*:[[:space:]]*\"ok\" ]]; then
    echo "error: health status not ok" >&2
    return 1
  fi
}

if [[ "${DEPLOY_DRY_RUN:-0}" == "1" && "${DEPLOY_MOCK_REMOTE:-0}" != "1" ]]; then
  echo "dry_run health_skip url_set=1"
  exit 0
fi

# Live / mock-remote: hardcoded loopback URL only (no interpolation of operator env).
if [[ -n "${DEPLOY_HOST:-}" && -n "${DEPLOY_USER:-}" ]]; then
  if [[ -n "${HEALTH_URL:-}" ]]; then
    echo "error: HEALTH_URL is forbidden on the live health path" >&2
    exit 1
  fi
  deploy_ssh_base
  # URL is a fixed literal in this script — never from operator input.
  # shellcheck disable=SC2029
  body="$(deploy_ssh_run "${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" \
    'set -Eeuo pipefail; curl -fsS --max-time 5 http://127.0.0.1:8080/v1/health')"
  check_body "${body}"
  echo "health_ok remote=1 url=${LIVE_HEALTH_URL}"
  exit 0
fi

# Local non-live path (unit tests / operator loopback). Override only via explicit test var.
url="${DEPLOY_TEST_HEALTH_URL:-${LIVE_HEALTH_URL}}"
if [[ "${url}" != "http://127.0.0.1:8080/v1/health" && "${url}" != "http://127.0.0.1:"*"/v1/health" ]]; then
  # Allow only loopback health paths in tests.
  if [[ "${DEPLOY_TEST_HEALTH_URL:-}" != "${url}" ]]; then
    echo "error: refusing non-loopback health URL" >&2
    exit 1
  fi
  case "${url}" in
    http://127.0.0.1:* | http://localhost:*) ;;
    *)
      echo "error: DEPLOY_TEST_HEALTH_URL must target loopback" >&2
      exit 1
      ;;
  esac
fi

deploy_require_cmds curl
body="$(curl -fsS --max-time 5 "${url}")"
check_body "${body}"
echo "health_ok"
