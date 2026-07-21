#!/usr/bin/env bash
# Staging update orchestrator (fail-closed).
# DEPLOY_DRY_RUN=1: local policy/migration mocks without remote I/O.
# DEPLOY_MOCK_REMOTE=1: full live path with ssh/scp/sudo/systemctl from PATH (tests).
# Neither: real remote update (D2 execution).
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

ROOT="$(deploy_repo_root)"
cd "${ROOT}"

DEPLOY_DRY_RUN="${DEPLOY_DRY_RUN:-0}"
DEPLOY_MOCK_REMOTE="${DEPLOY_MOCK_REMOTE:-0}"
DEPLOY_N1_COMPAT_PROVEN="${DEPLOY_N1_COMPAT_PROVEN:-0}"
ENV_LOCAL="${ENV_LOCAL:-${ROOT}/.env.local}"
ENV_DEPLOY="${ENV_DEPLOY:-${ROOT}/.env.deploy.local}"
ENV_MIGRATE="${ENV_MIGRATE:-${ROOT}/.env.migrate.local}"
REMOTE_OPT="/opt/bwb-modulo-fiscal"
REMOTE_ETC="/etc/bwb-modulo-fiscal"
REMOTE_UNIT="bwb-fiscal-api.service"

report() {
  printf 'report %s\n' "$*"
}

die() {
  echo "error: $*" >&2
  exit 1
}

load_operator_env() {
  local f="$1"
  local line key value lineno=0 trimmed
  [[ -f "${f}" ]] || die "operator env missing"
  while IFS= read -r line || [[ -n "${line}" ]]; do
    lineno=$((lineno + 1))
    trimmed="$(printf '%s' "${line}" | deploy_trim)"
    [[ -z "${trimmed}" || "${trimmed}" == \#* ]] && continue
    case "${line}" in
      *=*) ;;
      *) die "malformed operator env line ${lineno}" ;;
    esac
    key="${line%%=*}"
    value="${line#*=}"
    key="$(printf '%s' "${key}" | deploy_trim)"
    case "${key}" in
      DEPLOY_HOST | DEPLOY_USER | DEPLOY_SSH_KEY | DEPLOY_KNOWN_HOSTS | EXPECTED_COMMIT | DEPLOY_GOARCH | DEPLOY_DRY_RUN | DEPLOY_MOCK_REMOTE | DEPLOY_N1_COMPAT_PROVEN | DEPLOY_MOCK_MIGRATE_VERSION_BEFORE | DEPLOY_MOCK_MIGRATE_VERSION_AFTER | DEPLOY_MOCK_MIGRATE_DIRTY | DEPLOY_SIMULATE_HEALTH_FAIL | HEALTH_URL | OUT_DIR | DEPLOY_ALLOW_DIRTY_WORKTREE | DEPLOY_TEST_OUT_ROOT) ;;
      *) die "unknown operator key: ${key}" ;;
    esac
    printf -v "${key}" '%s' "${value}"
    export "${key?}"
  done <"${f}"
}

remote_sh() {
  # shellcheck disable=SC2029
  "${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "set -Eeuo pipefail; $*"
}

install_env_remote() {
  local local_file="$1"
  local remote_name="$2"
  local remote_tmp
  remote_tmp="/tmp/bwb-env-${remote_name}.$$"
  "${SCP_BASE[@]}" "${local_file}" "${DEPLOY_USER}@${DEPLOY_HOST}:${remote_tmp}"
  if [[ "${DEPLOY_MOCK_REMOTE}" == "1" ]]; then
    remote_sh "mkdir -p '${REMOTE_ETC}' && cp '${remote_tmp}' '${REMOTE_ETC}/${remote_name}' && chmod 0600 '${REMOTE_ETC}/${remote_name}' && rm -f '${remote_tmp}'"
  else
    remote_sh "install -d -m 0750 -o root -g root '${REMOTE_ETC}' && install -m 0600 -o root -g root '${remote_tmp}' '${REMOTE_ETC}/${remote_name}' && rm -f '${remote_tmp}'"
  fi
}

promote_symlink() {
  local sha="$1"
  # Atomic symlink swap without GNU-only mv -T (portable for mocked macOS tests).
  remote_sh "ln -sfn '${REMOTE_OPT}/releases/${sha}' '${REMOTE_OPT}/current.new' && mv -f '${REMOTE_OPT}/current.new' '${REMOTE_OPT}/current'"
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
  [[ "${DEPLOY_DRY_RUN}" == "1" && "${DEPLOY_MOCK_REMOTE}" != "1" ]] || die "missing deploy env file"
fi

if [[ -f "${ENV_MIGRATE}" ]]; then
  deploy_validate_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${ENV_MIGRATE}" \
    || die "migrate env allowlist validation failed"
  report "migrate_env_allowlist=ok"
else
  [[ "${DEPLOY_DRY_RUN}" == "1" && "${DEPLOY_MOCK_REMOTE}" != "1" ]] || die "missing migrate env file"
fi

HEAD="$(git rev-parse HEAD)"
[[ "${HEAD}" == "${EXPECTED_COMMIT}" ]] || die "EXPECTED_COMMIT does not match HEAD"

export EXPECTED_COMMIT DEPLOY_GOARCH
OUT_DIR="${OUT_DIR:-${ROOT}/dist/releases/${HEAD}}"
export OUT_DIR
bash "${SCRIPT_DIR}/build-linux-release.sh"
report "build=ok commit=${HEAD} arch=${DEPLOY_GOARCH}"

RELEASE_DIR="${REMOTE_OPT}/releases/${HEAD}"
export RELEASE_DIR

MOCK_BEFORE="${DEPLOY_MOCK_MIGRATE_VERSION_BEFORE:-2}"
MOCK_AFTER="${DEPLOY_MOCK_MIGRATE_VERSION_AFTER:-2}"
MOCK_DIRTY="${DEPLOY_MOCK_MIGRATE_DIRTY:-false}"

# --- Local dry-run (no remote path) ---
if [[ "${DEPLOY_DRY_RUN}" == "1" && "${DEPLOY_MOCK_REMOTE}" != "1" ]]; then
  migration_before_out="$(
    DEPLOY_DRY_RUN=1 \
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
  report "mode=dry_run"
  report "done"
  exit 0
fi

# --- Live path (real or mocked remote commands) ---
: "${DEPLOY_HOST:?DEPLOY_HOST required}"
: "${DEPLOY_USER:?DEPLOY_USER required}"
: "${DEPLOY_SSH_KEY:?DEPLOY_SSH_KEY required}"
: "${DEPLOY_KNOWN_HOSTS:?DEPLOY_KNOWN_HOSTS required}"
[[ -f "${ENV_DEPLOY}" ]] || die "missing deploy env file"
[[ -f "${ENV_MIGRATE}" ]] || die "missing migrate env file"

deploy_ssh_base
deploy_require_cmds sudo systemctl

REMOTE_UPLOAD="/tmp/bwb-release-${HEAD}"
PREV_SHA=""

report "mode=live mock_remote=${DEPLOY_MOCK_REMOTE}"

# Upload to remote temporary directory
remote_sh "rm -rf '${REMOTE_UPLOAD}' && mkdir -p '${REMOTE_UPLOAD}'"
"${SCP_BASE[@]}" -r \
  "${OUT_DIR}/fiscal-api" \
  "${OUT_DIR}/fiscal-migrate" \
  "${OUT_DIR}/COMMIT" \
  "${OUT_DIR}/SHA256SUMS" \
  "${OUT_DIR}/remote-migrate-run.sh" \
  "${OUT_DIR}/lib" \
  "${DEPLOY_USER}@${DEPLOY_HOST}:${REMOTE_UPLOAD}/"
report "upload=ok"

# Verify COMMIT + SHA256SUMS before install
remote_sh "cd '${REMOTE_UPLOAD}' && test -f COMMIT && test -f SHA256SUMS && test \"\$(tr -d '[:space:]' <COMMIT)\" = '${HEAD}' && (command -v sha256sum >/dev/null && sha256sum -c SHA256SUMS >/dev/null || shasum -a 256 -c SHA256SUMS >/dev/null) && test -f fiscal-api && test -f fiscal-migrate && test -x remote-migrate-run.sh"
report "verify=ok commit=${HEAD}"

# Immutable install under releases/<sha>
remote_sh "if [[ -d '${RELEASE_DIR}' ]]; then test \"\$(tr -d '[:space:]' <'${RELEASE_DIR}/COMMIT')\" = '${HEAD}'; else install -d -m 0755 '${REMOTE_OPT}/releases'; fi"
remote_sh "rm -rf '${RELEASE_DIR}.partial' && mkdir -p '${RELEASE_DIR}.partial' && cp -a '${REMOTE_UPLOAD}/.' '${RELEASE_DIR}.partial/' && chmod 0755 '${RELEASE_DIR}.partial/fiscal-api' '${RELEASE_DIR}.partial/fiscal-migrate' '${RELEASE_DIR}.partial/remote-migrate-run.sh' && mv '${RELEASE_DIR}.partial' '${RELEASE_DIR}' && rm -rf '${REMOTE_UPLOAD}'"
report "install_release=ok path=${RELEASE_DIR}"

# Atomic env install (0600)
install_env_remote "${ENV_DEPLOY}" "fiscal.env"
install_env_remote "${ENV_MIGRATE}" "migrate.env"
report "install_env=ok mode=0600"

# Capture previous current (for N-1 rollback only)
PREV_SHA="$(remote_sh "if [[ -e '${REMOTE_OPT}/current' ]]; then basename \"\$(readlink '${REMOTE_OPT}/current')\"; else printf ''; fi")"
if [[ -n "${PREV_SHA}" ]]; then
  report "previous_release=${PREV_SHA}"
else
  report "previous_release=none"
fi

phase="pre_migrate"
rollback_allowed="true"
report "phase=${phase} rollback_allowed=${rollback_allowed}"

# migration_before with NEW release binary
migration_before_out="$(
  RELEASE_DIR="${RELEASE_DIR}" \
    DEPLOY_DRY_RUN=0 \
    DEPLOY_MOCK_REMOTE="${DEPLOY_MOCK_REMOTE}" \
    bash "${SCRIPT_DIR}/migrate-remote.sh" version
)"
deploy_parse_migrate_version "${migration_before_out}"
migration_before="${DEPLOY_MIG_VERSION}"
migration_before_dirty="${DEPLOY_MIG_DIRTY}"
report "migration_before=${migration_before} dirty=${migration_before_dirty} binary=new_release"
[[ "${migration_before_dirty}" == "false" ]] || die "migration dirty before update; refusing"

# up with NEW release fiscal-migrate (never current)
migration_after_out="$(
  RELEASE_DIR="${RELEASE_DIR}" \
    DEPLOY_DRY_RUN=0 \
    DEPLOY_MOCK_REMOTE="${DEPLOY_MOCK_REMOTE}" \
    bash "${SCRIPT_DIR}/migrate-remote.sh" up
)"
deploy_parse_migrate_version "${migration_after_out}"
migration_after="${DEPLOY_MIG_VERSION}"
migration_after_dirty="${DEPLOY_MIG_DIRTY}"
report "migration_after=${migration_after} dirty=${migration_after_dirty} binary=new_release"

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

# Atomic symlink promotion
promote_symlink "${HEAD}"
report "promote=ok symlink=current->${HEAD}"

# Restart + health
remote_sh "sudo systemctl restart '${REMOTE_UNIT}'"
report "restart=ok"

if [[ "${DEPLOY_SIMULATE_HEALTH_FAIL:-0}" == "1" ]]; then
  report "health=fail"
  if [[ "${rollback_allowed}" == "true" ]]; then
    if [[ -n "${PREV_SHA}" && "${PREV_SHA}" != "${HEAD}" ]]; then
      promote_symlink "${PREV_SHA}"
      remote_sh "sudo systemctl restart '${REMOTE_UNIT}'"
      report "action=restore_previous_binary previous=${PREV_SHA}"
    else
      report "action=restore_previous_binary previous=unavailable"
      die "rollback allowed but previous release missing"
    fi
  else
    report "action=roll_forward_or_manual"
    die "post-migration health fail without N-1 proof: no automatic binary rollback"
  fi
else
  if [[ "${DEPLOY_MOCK_REMOTE}" == "1" ]]; then
    report "health=checked mock=1"
  else
    DEPLOY_DRY_RUN=0 bash "${SCRIPT_DIR}/healthcheck.sh"
    report "health=checked"
  fi
fi

report "done"
