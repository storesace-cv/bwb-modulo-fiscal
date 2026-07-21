#!/usr/bin/env bash
# Safe remote migrate runner: reads migrate.env without shell-sourcing values.
# Must run as root (or equivalent) to read root:root 0600 migrate.env.
# Usage: remote-migrate-run.sh <release_dir> <up|version>
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

RELEASE_DIR="${1:?release_dir required}"
CMD="${2:?command required}"
ENV_FILE="${MIGRATE_ENV_FILE:-/etc/bwb-modulo-fiscal/migrate.env}"
RELEASE_PREFIX="/opt/bwb-modulo-fiscal/releases/"
sha=""

case "${CMD}" in
  up | version) ;;
  *)
    echo "usage: remote-migrate-run.sh <release_dir> <up|version>" >&2
    exit 2
    ;;
esac

# Production path: /opt/bwb-modulo-fiscal/releases/<40-hex-sha>
# Test override only when MIGRATE_ENV_FILE points away from production.
if [[ "${RELEASE_DIR}" == "${RELEASE_PREFIX}"* ]]; then
  sha="${RELEASE_DIR#"${RELEASE_PREFIX}"}"
  if [[ "${sha}" == */* || -z "${sha}" ]]; then
    echo "error: RELEASE_DIR must be /opt/bwb-modulo-fiscal/releases/<40-hex-sha>" >&2
    exit 1
  fi
  deploy_assert_sha1 "RELEASE_DIR sha" "${sha}"
elif [[ -n "${MIGRATE_ENV_FILE:-}" && "${MIGRATE_ENV_FILE}" != "/etc/bwb-modulo-fiscal/migrate.env" ]]; then
  sha=""
else
  echo "error: RELEASE_DIR must be /opt/bwb-modulo-fiscal/releases/<40-hex-sha>" >&2
  exit 1
fi

if [[ ! -d "${RELEASE_DIR}" ]]; then
  echo "error: release dir missing" >&2
  exit 1
fi

deploy_verify_release_manifest "${RELEASE_DIR}" "${sha}"

if [[ ! -x "${RELEASE_DIR}/fiscal-migrate" ]]; then
  echo "error: fiscal-migrate missing in release_dir" >&2
  exit 1
fi
if [[ ! -f "${ENV_FILE}" ]]; then
  echo "error: migrate env file missing" >&2
  exit 1
fi
if [[ -L "${ENV_FILE}" ]]; then
  echo "error: migrate env must not be a symlink" >&2
  exit 1
fi

allowlist="${RELEASE_DIR}/lib/migrate.env.allowlist"
deploy_validate_exact_allowlisted_file "${allowlist}" "${ENV_FILE}"

driver="$(deploy_read_env_value "${ENV_FILE}" FISCAL_DATABASE_DRIVER)"
url="$(deploy_read_env_value "${ENV_FILE}" FISCAL_DATABASE_URL)"
printf -v FISCAL_DATABASE_DRIVER '%s' "${driver}"
printf -v FISCAL_DATABASE_URL '%s' "${url}"
export FISCAL_DATABASE_DRIVER FISCAL_DATABASE_URL

exec "${RELEASE_DIR}/fiscal-migrate" "${CMD}"
