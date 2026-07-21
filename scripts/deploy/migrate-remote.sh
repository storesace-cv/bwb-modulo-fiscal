#!/usr/bin/env bash
# Run fiscal-migrate on the remote host using migrate.env only (not fiscal.env).
# Dry-run mocks version output without SSH or DSN.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
source "${SCRIPT_DIR}/lib/allowlist.sh"

ROOT="$(deploy_repo_root)"
CMD="${1:-version}"
case "${CMD}" in
  up | version) ;;
  *)
    echo "usage: migrate-remote.sh [up|version]" >&2
    exit 2
    ;;
esac

if [[ "${DEPLOY_DRY_RUN:-0}" == "1" ]]; then
  # Mock without touching DB or printing secrets.
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

# Expand ~
DEPLOY_SSH_KEY="${DEPLOY_SSH_KEY/#\~/${HOME}}"
DEPLOY_KNOWN_HOSTS="${DEPLOY_KNOWN_HOSTS/#\~/${HOME}}"

deploy_require_cmds ssh

# Remote: source migrate.env with set -a, run binary from current release.
# Values never echoed locally.
ssh \
  -i "${DEPLOY_SSH_KEY}" \
  -o IdentitiesOnly=yes \
  -o StrictHostKeyChecking=yes \
  -o UserKnownHostsFile="${DEPLOY_KNOWN_HOSTS}" \
  -o BatchMode=yes \
  "${DEPLOY_USER}@${DEPLOY_HOST}" \
  "set -Eeuo pipefail; set -a; source /etc/bwb-modulo-fiscal/migrate.env; set +a; /opt/bwb-modulo-fiscal/current/fiscal-migrate ${CMD}"
