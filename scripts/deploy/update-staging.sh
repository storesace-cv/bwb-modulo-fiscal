#!/usr/bin/env bash
# Staging update orchestrator (fail-closed). Dry-run exercises logic without SSH.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

ROOT="$(deploy_repo_root)"
cd "${ROOT}"

DEPLOY_DRY_RUN="${DEPLOY_DRY_RUN:-0}"
DEPLOY_N1_COMPAT_PROVEN="${DEPLOY_N1_COMPAT_PROVEN:-0}"
ENV_LOCAL="${ENV_LOCAL:-${ROOT}/.env.local}"
ENV_DEPLOY="${ENV_DEPLOY:-${ROOT}/.env.deploy.local}"
ENV_MIGRATE="${ENV_MIGRATE:-${ROOT}/.env.migrate.local}"

report() {
  printf 'report %s\n' "$*"
}

die() {
  echo "error: $*" >&2
  exit 1
}

load_operator_env() {
  local f="$1"
  local line key value lineno=0
  [[ -f "${f}" ]] || die "operator env missing"
  while IFS= read -r line || [[ -n "${line}" ]]; do
    lineno=$((lineno + 1))
    line="${line%%#*}"
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -z "${line}" ]] && continue
    [[ "${line}" == *=* ]] || die "malformed operator env line ${lineno}"
    key="${line%%=*}"
    value="${line#*=}"
    case "${key}" in
      DEPLOY_HOST | DEPLOY_USER | DEPLOY_SSH_KEY | DEPLOY_KNOWN_HOSTS | EXPECTED_COMMIT | DEPLOY_GOARCH | DEPLOY_DRY_RUN | DEPLOY_N1_COMPAT_PROVEN | DEPLOY_MOCK_MIGRATE_VERSION_BEFORE | DEPLOY_MOCK_MIGRATE_VERSION_AFTER | DEPLOY_MOCK_MIGRATE_DIRTY | DEPLOY_SIMULATE_HEALTH_FAIL | HEALTH_URL | OUT_DIR) ;;
      *) die "unknown operator key: ${key}" ;;
    esac
    printf -v "${key}" '%s' "${value}"
    export "${key?}"
  done <"${f}"
}

if [[ -f "${ENV_LOCAL}" ]]; then
  load_operator_env "${ENV_LOCAL}"
fi

: "${EXPECTED_COMMIT:?EXPECTED_COMMIT required}"
: "${DEPLOY_GOARCH:?DEPLOY_GOARCH required}"

if [[ -f "${ENV_DEPLOY}" ]]; then
  deploy_validate_allowlisted_file "${ROOT}/deploy/env.allowlist" "${ENV_DEPLOY}" \
    || die "runtime env allowlist validation failed"
  report "runtime_env_allowlist=ok"
else
  [[ "${DEPLOY_DRY_RUN}" == "1" ]] || die "missing deploy env file"
fi

if [[ -f "${ENV_MIGRATE}" ]]; then
  deploy_validate_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${ENV_MIGRATE}" \
    || die "migrate env allowlist validation failed"
  report "migrate_env_allowlist=ok"
else
  [[ "${DEPLOY_DRY_RUN}" == "1" ]] || die "missing migrate env file"
fi

HEAD="$(git rev-parse HEAD)"
[[ "${HEAD}" == "${EXPECTED_COMMIT}" ]] || die "EXPECTED_COMMIT does not match HEAD"

export EXPECTED_COMMIT DEPLOY_GOARCH
OUT_DIR="${OUT_DIR:-${ROOT}/dist/releases/${HEAD}}"
export OUT_DIR
bash "${SCRIPT_DIR}/build-linux-release.sh"
report "build=ok commit=${HEAD} arch=${DEPLOY_GOARCH}"

MOCK_BEFORE="${DEPLOY_MOCK_MIGRATE_VERSION_BEFORE:-2}"
MOCK_AFTER="${DEPLOY_MOCK_MIGRATE_VERSION_AFTER:-2}"
MOCK_DIRTY="${DEPLOY_MOCK_MIGRATE_DIRTY:-false}"

migration_before_out="$(
  DEPLOY_DRY_RUN="${DEPLOY_DRY_RUN}" \
    DEPLOY_MOCK_MIGRATE_VERSION="${MOCK_BEFORE}" \
    DEPLOY_MOCK_MIGRATE_DIRTY="${MOCK_DIRTY}" \
    bash "${SCRIPT_DIR}/migrate-remote.sh" version
)"
deploy_parse_migrate_version "${migration_before_out}"
migration_before="${DEPLOY_MIG_VERSION}"
migration_before_dirty="${DEPLOY_MIG_DIRTY}"
report "migration_before=${migration_before} dirty=${migration_before_dirty}"
[[ "${migration_before_dirty}" == "false" ]] || die "migration dirty before update; refusing"

phase="pre_migrate"
rollback_allowed="true"
report "phase=${phase} rollback_allowed=${rollback_allowed}"

if [[ "${DEPLOY_DRY_RUN}" != "1" ]]; then
  die "live remote update is D2-only; set DEPLOY_DRY_RUN=1 for repository tests"
fi

migration_after_out="$(
  DEPLOY_DRY_RUN=1 \
    DEPLOY_MOCK_MIGRATE_VERSION="${MOCK_AFTER}" \
    DEPLOY_MOCK_MIGRATE_DIRTY="${MOCK_DIRTY}" \
    bash "${SCRIPT_DIR}/migrate-remote.sh" up
)"
deploy_parse_migrate_version "${migration_after_out}"
migration_after="${DEPLOY_MIG_VERSION}"
migration_after_dirty="${DEPLOY_MIG_DIRTY}"
report "migration_after=${migration_after} dirty=${migration_after_dirty}"

if [[ "${migration_after_dirty}" != "false" ]]; then
  report "promote=blocked reason=dirty"
  die "migration dirty after up; promotion blocked"
fi

phase="post_migrate"
if [[ "${migration_after}" != "${migration_before}" ]]; then
  if [[ "${DEPLOY_N1_COMPAT_PROVEN}" == "1" ]]; then
    rollback_allowed="true"
  else
    rollback_allowed="false"
  fi
else
  rollback_allowed="true"
fi
report "phase=${phase} rollback_allowed=${rollback_allowed}"

DEPLOY_DRY_RUN=1 bash "${SCRIPT_DIR}/healthcheck.sh"
report "health=checked"

if [[ "${DEPLOY_SIMULATE_HEALTH_FAIL:-0}" == "1" ]]; then
  report "health=fail"
  if [[ "${rollback_allowed}" == "true" ]]; then
    report "action=restore_previous_binary"
  else
    report "action=roll_forward_or_manual"
    die "post-migration health fail without N-1 proof: no automatic binary rollback"
  fi
fi

report "promote=ok"
report "incidentes=none"
report "done"
