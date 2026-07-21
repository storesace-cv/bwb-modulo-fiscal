#!/usr/bin/env bash
# Run fiscal-migrate on the remote host via sudo + safe runner (never source).
# Requires RELEASE_DIR = /opt/bwb-modulo-fiscal/releases/<sha40>.
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

prefix="/opt/bwb-modulo-fiscal/releases/"
[[ "${RELEASE_DIR}" == "${prefix}"* ]] || {
  echo "error: RELEASE_DIR must be under ${prefix}" >&2
  exit 1
}
sha="${RELEASE_DIR#"${prefix}"}"
deploy_assert_sha1 "RELEASE_DIR sha" "${sha}"

deploy_ssh_base

# sudo so root:root 0600 migrate.env is readable; never chmod it for the app user.
# shellcheck disable=SC2029
"${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" \
  "set -Eeuo pipefail; sudo -n bash '${RELEASE_DIR}/remote-migrate-run.sh' '${RELEASE_DIR}' '${CMD}'"
