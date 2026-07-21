#!/usr/bin/env bash
# Staging update orchestrator (fail-closed).
# DEPLOY_DRY_RUN=1: local policy/migration mocks without remote I/O.
# DEPLOY_MOCK_REMOTE=1: full live path with ssh/scp from PATH (tests; sudo/systemctl only via remote).
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
REMOTE_UPLOAD=""
PREV_SHA=""
ACTIVE_RELEASE="none"
ENV_BACKUP_ID=""
EXPECTED_SCHEMA_VERSION="${EXPECTED_SCHEMA_VERSION:-${DEPLOY_EXPECTED_SCHEMA_VERSION_DEFAULT}}"

report() {
  printf 'report %s\n' "$*"
}

die() {
  echo "error: $*" >&2
  exit 1
}

cleanup_remote_upload() {
  if [[ -n "${REMOTE_UPLOAD:-}" && -n "${DEPLOY_HOST:-}" && ${#SSH_BASE[@]} -gt 0 ]]; then
    set +e
    remote_sh "sudo -n rm -rf -- '${REMOTE_UPLOAD}'" >/dev/null 2>&1
    set -e
    REMOTE_UPLOAD=""
  fi
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
      DEPLOY_HOST | DEPLOY_USER | DEPLOY_SSH_KEY | DEPLOY_KNOWN_HOSTS | EXPECTED_COMMIT | DEPLOY_GOARCH | DEPLOY_DRY_RUN | DEPLOY_MOCK_REMOTE | DEPLOY_N1_COMPAT_PROVEN | DEPLOY_MOCK_MIGRATE_VERSION_BEFORE | DEPLOY_MOCK_MIGRATE_VERSION_AFTER | DEPLOY_MOCK_MIGRATE_DIRTY | DEPLOY_SIMULATE_HEALTH_FAIL | HEALTH_URL | OUT_DIR | DEPLOY_TEST_OUT_ROOT | EXPECTED_SCHEMA_VERSION) ;;
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

remote_sudo() {
  # Encode the snippet so quoting survives SSH/mock remapping.
  local snippet b64
  snippet="set -Eeuo pipefail; $*"
  b64="$(printf '%s' "${snippet}" | base64 | tr -d '\n')"
  # shellcheck disable=SC2029
  remote_sh "echo '${b64}' | (base64 -d 2>/dev/null || base64 -D) | sudo -n bash"
}

read_active_release() {
  remote_sh "if [[ -e '${REMOTE_OPT}/current' ]]; then basename \"\$(readlink '${REMOTE_OPT}/current')\"; else printf 'none'; fi"
}

promote_symlink() {
  local sha="$1"
  deploy_assert_sha1 "promote sha" "${sha}"
  remote_sudo "ln -sfn '${REMOTE_OPT}/releases/${sha}' '${REMOTE_OPT}/current.new' && mv -f '${REMOTE_OPT}/current.new' '${REMOTE_OPT}/current'"
  ACTIVE_RELEASE="${sha}"
}

backup_remote_envs() {
  ENV_BACKUP_ID="$(date -u +%Y%m%dT%H%M%SZ)-${HEAD}"
  remote_sudo "install -d -m 0750 -o root -g root '${REMOTE_ETC}/backups' && \
    if [[ -f '${REMOTE_ETC}/fiscal.env' ]]; then install -m 0600 -o root -g root '${REMOTE_ETC}/fiscal.env' '${REMOTE_ETC}/backups/fiscal.env.${ENV_BACKUP_ID}'; fi && \
    if [[ -f '${REMOTE_ETC}/migrate.env' ]]; then install -m 0600 -o root -g root '${REMOTE_ETC}/migrate.env' '${REMOTE_ETC}/backups/migrate.env.${ENV_BACKUP_ID}'; fi"
  report "env_backup=ok id=${ENV_BACKUP_ID}"
}

restore_remote_envs() {
  if [[ -z "${ENV_BACKUP_ID}" ]]; then
    return 0
  fi
  remote_sudo "\
    if [[ -f '${REMOTE_ETC}/backups/fiscal.env.${ENV_BACKUP_ID}' ]]; then install -m 0600 -o root -g root '${REMOTE_ETC}/backups/fiscal.env.${ENV_BACKUP_ID}' '${REMOTE_ETC}/fiscal.env'; fi && \
    if [[ -f '${REMOTE_ETC}/backups/migrate.env.${ENV_BACKUP_ID}' ]]; then install -m 0600 -o root -g root '${REMOTE_ETC}/backups/migrate.env.${ENV_BACKUP_ID}' '${REMOTE_ETC}/migrate.env'; fi"
  report "env_restore=ok id=${ENV_BACKUP_ID}"
}

install_env_remote() {
  local local_file="$1"
  local remote_name="$2"
  # Place under the already-random 0700 upload dir (no nested absolute mktemp paths).
  local remote_tmp="${REMOTE_UPLOAD}/env.${remote_name}.$$"
  "${SCP_BASE[@]}" "${local_file}" "${DEPLOY_USER}@${DEPLOY_HOST}:${remote_tmp}"
  remote_sudo "install -d -m 0750 -o root -g root '${REMOTE_ETC}' && install -m 0600 -o root -g root '${remote_tmp}' '${REMOTE_ETC}/${remote_name}' && rm -f -- '${remote_tmp}'"
}

restart_unit() {
  remote_sudo "systemctl restart '${REMOTE_UNIT}'"
  report "restart=ok"
}

run_remote_health() {
  if [[ "${DEPLOY_SIMULATE_HEALTH_FAIL:-0}" == "1" ]]; then
    # Fail exactly once so post-rollback health can succeed in tests and real recovery paths.
    DEPLOY_SIMULATE_HEALTH_FAIL=0
    export DEPLOY_SIMULATE_HEALTH_FAIL
    report "health=fail"
    return 1
  fi
  DEPLOY_DRY_RUN=0 DEPLOY_MOCK_REMOTE="${DEPLOY_MOCK_REMOTE}" \
    bash "${SCRIPT_DIR}/healthcheck.sh"
  report "health=ok"
}

report_active() {
  report "active_release=${ACTIVE_RELEASE}"
}

if [[ -f "${ENV_LOCAL}" ]]; then
  load_operator_env "${ENV_LOCAL}"
fi

: "${EXPECTED_COMMIT:?EXPECTED_COMMIT required}"
: "${DEPLOY_GOARCH:?DEPLOY_GOARCH required}"
deploy_assert_sha1 "EXPECTED_COMMIT" "${EXPECTED_COMMIT}"

if [[ -f "${ENV_DEPLOY}" ]]; then
  deploy_validate_allowlisted_file "${ROOT}/deploy/env.allowlist" "${ENV_DEPLOY}" \
    || die "runtime env allowlist validation failed"
  report "runtime_env_allowlist=ok"
else
  [[ "${DEPLOY_DRY_RUN}" == "1" && "${DEPLOY_MOCK_REMOTE}" != "1" ]] || die "missing deploy env file"
fi

if [[ -f "${ENV_MIGRATE}" ]]; then
  deploy_validate_exact_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${ENV_MIGRATE}" \
    || die "migrate env allowlist validation failed"
  report "migrate_env_allowlist=ok"
else
  [[ "${DEPLOY_DRY_RUN}" == "1" && "${DEPLOY_MOCK_REMOTE}" != "1" ]] || die "missing migrate env file"
fi

HEAD="$(git rev-parse HEAD)"
deploy_assert_sha1 "HEAD" "${HEAD}"
[[ "${HEAD}" == "${EXPECTED_COMMIT}" ]] || die "EXPECTED_COMMIT does not match HEAD"

# Live path never builds from a dirty tree (DEPLOY_TEST_OUT_ROOT only for tests).
if [[ "${DEPLOY_DRY_RUN}" != "1" || "${DEPLOY_MOCK_REMOTE}" == "1" ]]; then
  if [[ -z "${DEPLOY_TEST_OUT_ROOT:-}" ]]; then
    deploy_assert_clean_worktree
  fi
fi

export EXPECTED_COMMIT DEPLOY_GOARCH EXPECTED_SCHEMA_VERSION
OUT_DIR="${OUT_DIR:-${ROOT}/dist/releases/${HEAD}}"
export OUT_DIR
bash "${SCRIPT_DIR}/build-linux-release.sh"
report "build=ok commit=${HEAD} arch=${DEPLOY_GOARCH} schema=${EXPECTED_SCHEMA_VERSION}"

RELEASE_DIR="$(deploy_release_dir_for_sha "${HEAD}")"
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
  if [[ "${migration_after}" != "${EXPECTED_SCHEMA_VERSION}" ]]; then
    report "promote=blocked reason=schema_mismatch expected=${EXPECTED_SCHEMA_VERSION} got=${migration_after}"
    die "migration version mismatch after up"
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
      report "active_release=previous_or_unchanged"
    else
      report "action=roll_forward_or_manual"
      report "active_release=new_or_partial"
      die "post-migration health fail without N-1 proof: no automatic binary rollback"
    fi
  else
    report "promote=ok"
  fi

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

deploy_assert_restricted_file "ENV_DEPLOY" "${ENV_DEPLOY}"
deploy_assert_restricted_file "ENV_MIGRATE" "${ENV_MIGRATE}"

deploy_ssh_base
trap cleanup_remote_upload EXIT

report "mode=live mock_remote=${DEPLOY_MOCK_REMOTE}"

# Ephemeral remote upload dir (0700); never predictable /tmp/bwb-release-<sha>.
REMOTE_UPLOAD="$(remote_sh "d=\$(mktemp -d /tmp/bwb-upload.XXXXXX); chmod 0700 \"\$d\"; printf '%s' \"\$d\"")"
[[ -n "${REMOTE_UPLOAD}" ]] || die "failed to create remote upload directory"
report "upload_dir=ok"

"${SCP_BASE[@]}" -r \
  "${OUT_DIR}/fiscal-api" \
  "${OUT_DIR}/fiscal-migrate" \
  "${OUT_DIR}/COMMIT" \
  "${OUT_DIR}/EXPECTED_SCHEMA_VERSION" \
  "${OUT_DIR}/SHA256SUMS" \
  "${OUT_DIR}/remote-migrate-run.sh" \
  "${OUT_DIR}/lib" \
  "${DEPLOY_USER}@${DEPLOY_HOST}:${REMOTE_UPLOAD}/"
report "upload=ok"

remote_sh "cd '${REMOTE_UPLOAD}' && test \"\$(tr -d '[:space:]' <COMMIT)\" = '${HEAD}' && test \"\$(tr -d '[:space:]' <EXPECTED_SCHEMA_VERSION)\" = '${EXPECTED_SCHEMA_VERSION}' && (command -v sha256sum >/dev/null && sha256sum -c SHA256SUMS >/dev/null || shasum -a 256 -c SHA256SUMS >/dev/null)"
report "verify=ok commit=${HEAD} schema=${EXPECTED_SCHEMA_VERSION}"

# Immutable root-owned install under releases/<sha>
remote_sudo "install -d -m 0755 -o root -g root '${REMOTE_OPT}/releases'"
remote_sudo "rm -rf -- '${RELEASE_DIR}.partial'"
remote_sudo "mkdir -p '${RELEASE_DIR}.partial'"
remote_sudo "cp -a '${REMOTE_UPLOAD}/.' '${RELEASE_DIR}.partial/'"
remote_sudo "chown -R root:root '${RELEASE_DIR}.partial'"
remote_sudo "chmod 0755 '${RELEASE_DIR}.partial/fiscal-api' '${RELEASE_DIR}.partial/fiscal-migrate' '${RELEASE_DIR}.partial/remote-migrate-run.sh'"
remote_sudo "chmod 0644 '${RELEASE_DIR}.partial/COMMIT' '${RELEASE_DIR}.partial/EXPECTED_SCHEMA_VERSION' '${RELEASE_DIR}.partial/SHA256SUMS' '${RELEASE_DIR}.partial/lib/allowlist.sh' '${RELEASE_DIR}.partial/lib/migrate.env.allowlist'"
if remote_sh "[[ -d '${RELEASE_DIR}' ]]"; then
  remote_sudo "test \"\$(tr -d '[:space:]' <'${RELEASE_DIR}/COMMIT')\" = '${HEAD}'"
  remote_sudo "rm -rf -- '${RELEASE_DIR}.partial'"
else
  remote_sudo "mv '${RELEASE_DIR}.partial' '${RELEASE_DIR}'"
fi
report "install_release=ok path=${RELEASE_DIR} owner=root"

PREV_RAW="$(read_active_release)"
if [[ "${PREV_RAW}" == "none" || -z "${PREV_RAW}" ]]; then
  PREV_SHA=""
  report "previous_release=none"
else
  deploy_assert_sha1 "PREV_SHA" "${PREV_RAW}"
  PREV_SHA="${PREV_RAW}"
  report "previous_release=${PREV_SHA}"
  ACTIVE_RELEASE="${PREV_SHA}"
fi

backup_remote_envs
install_env_remote "${ENV_DEPLOY}" "fiscal.env"
install_env_remote "${ENV_MIGRATE}" "migrate.env"
report "install_env=ok mode=0600 owner=root"

phase="pre_migrate"
rollback_allowed="true"
report "phase=${phase} rollback_allowed=${rollback_allowed}"

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
  report_active
  die "migration dirty after up; promotion blocked"
fi
if [[ "${migration_after}" != "${EXPECTED_SCHEMA_VERSION}" ]]; then
  report "promote=blocked reason=schema_mismatch expected=${EXPECTED_SCHEMA_VERSION} got=${migration_after}"
  report_active
  die "migration version mismatch after up; activation blocked"
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

promote_symlink "${HEAD}"
restart_unit

if run_remote_health; then
  report "promote=ok symlink=current->${HEAD}"
  report_active
  cleanup_remote_upload
  report "done"
  exit 0
fi

# Health failed after promote.
if [[ "${rollback_allowed}" == "true" ]]; then
  if [[ -n "${PREV_SHA}" && "${PREV_SHA}" != "${HEAD}" ]]; then
    promote_symlink "${PREV_SHA}"
    restore_remote_envs
    restart_unit
    if run_remote_health; then
      report "action=restore_previous_binary previous=${PREV_SHA}"
      report "health=ok_after_rollback"
      report_active
      die "new release health failed; rolled back to previous release"
    fi
    report "action=rollback_health_failed"
    report_active
    die "rollback completed but health still failing; manual intervention required"
  fi
  report "action=restore_previous_binary previous=unavailable"
  report "active_release=${HEAD}"
  die "rollback allowed but previous release missing; new release remains active"
fi

report "action=roll_forward_or_manual"
report "active_release=${HEAD}"
die "post-migration health fail without N-1 proof: no automatic binary rollback"
