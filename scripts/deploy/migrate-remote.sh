#!/usr/bin/env bash
# Run fiscal-migrate on the remote host using migrate.env via safe runner (never source).
# Requires RELEASE_DIR pointing at the release that owns fiscal-migrate (new release for updates).
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

CMD="${1:-version}"
case "${CMD}" in
  up | version) ;;
  *)
    echo "usage: migrate-remote.sh [up|version]" >&2
    exit 2
    ;;
esac

if [[ "${DEPLOY_DRY_RUN:-0}" == "1" && "${DEPLOY_MOCK_REMOTE:-0}" != "1" ]]; then
  MOCK_VERSION="${DEPLOY_MOCK_MIGRATE_VERSION:-2}"
  MOCK_DIRTY="${DEPLOY_MOCK_MIGRATE_DIRTY:-false}"
  if [[ "${CMD}" == "up" ]]; then
    printf 'ok version=%s dirty=%s\n' "${MOCK_VERSION}" "${MOCK_DIRTY}"
  else
    printf 'version=%s dirty=%s\n' "${MOCK_VERSION}" "${MOCK_DIRTY}"
  fi
  exit 0
fi

: "${DEPLOY_HOST:?DEPLOY_HOST required}"
: "${DEPLOY_USER:?DEPLOY_USER required}"
: "${DEPLOY_SSH_KEY:?DEPLOY_SSH_KEY required}"
: "${DEPLOY_KNOWN_HOSTS:?DEPLOY_KNOWN_HOSTS required}"
: "${RELEASE_DIR:?RELEASE_DIR required (new release path for fiscal-migrate)}"

deploy_ssh_base

# Remote runner reads only allowlisted keys; never sources migrate.env as shell.
# shellcheck disable=SC2029
"${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" \
  "set -Eeuo pipefail; bash '${RELEASE_DIR}/remote-migrate-run.sh' '${RELEASE_DIR}' '${CMD}'"
