#!/usr/bin/env bash
# Safe remote migrate runner: reads migrate.env without shell-sourcing values.
# Usage: remote-migrate-run.sh <release_dir> <up|version>
# Env file default: /etc/bwb-modulo-fiscal/migrate.env
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

RELEASE_DIR="${1:?release_dir required}"
CMD="${2:?command required}"
ENV_FILE="${MIGRATE_ENV_FILE:-/etc/bwb-modulo-fiscal/migrate.env}"

case "${CMD}" in
  up | version) ;;
  *)
    echo "usage: remote-migrate-run.sh <release_dir> <up|version>" >&2
    exit 2
    ;;
esac

if [[ ! -x "${RELEASE_DIR}/fiscal-migrate" ]]; then
  echo "error: fiscal-migrate missing in release_dir" >&2
  exit 1
fi
if [[ ! -f "${ENV_FILE}" ]]; then
  echo "error: migrate env file missing" >&2
  exit 1
fi

# Only the two allowlisted keys — never source the file as shell.
driver="$(deploy_read_env_value "${ENV_FILE}" FISCAL_DATABASE_DRIVER)"
url="$(deploy_read_env_value "${ENV_FILE}" FISCAL_DATABASE_URL)"

export FISCAL_DATABASE_DRIVER="${driver}"
export FISCAL_DATABASE_URL="${url}"

exec "${RELEASE_DIR}/fiscal-migrate" "${CMD}"
