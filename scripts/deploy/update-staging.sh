#!/usr/bin/env bash
# Staging update orchestrator (fail-closed).
# Privileged remote work goes only through /usr/local/sbin/bwb-fiscal-deploy-helper.
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
REMOTE_HELPER="/usr/local/sbin/bwb-fiscal-deploy-helper"
REMOTE_UPLOAD=""
PREV_SHA=""
ACTIVE_RELEASE="none"
ENV_BACKUP_ID=""
ENVS_RESTORABLE=0
ENV_RESTORED=0
ACTIVATED=0
EXPECTED_SCHEMA_VERSION="${EXPECTED_SCHEMA_VERSION:-${DEPLOY_EXPECTED_SCHEMA_VERSION_DEFAULT}}"

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
      DEPLOY_HOST | DEPLOY_USER | DEPLOY_SSH_KEY | DEPLOY_KNOWN_HOSTS | EXPECTED_COMMIT | DEPLOY_GOARCH | DEPLOY_DRY_RUN | DEPLOY_MOCK_REMOTE | DEPLOY_N1_COMPAT_PROVEN | DEPLOY_MOCK_MIGRATE_VERSION_BEFORE | DEPLOY_MOCK_MIGRATE_VERSION_AFTER | DEPLOY_MOCK_MIGRATE_DIRTY | DEPLOY_MOCK_MIGRATE_DIRTY_BEFORE | DEPLOY_MOCK_MIGRATE_DIRTY_AFTER | DEPLOY_SIMULATE_HEALTH_FAIL | DEPLOY_SIMULATE_RESTART_FAIL | DEPLOY_MOCK_FAIL_INSTALL_ENV | OUT_DIR | DEPLOY_TEST_OUT_ROOT | EXPECTED_SCHEMA_VERSION | DEPLOY_TEST_HEALTH_URL) ;;
      HEALTH_URL)
        die "HEALTH_URL is forbidden; live health is fixed to http://127.0.0.1:8080/v1/health"
        ;;
      *) die "unknown operator key: ${key}" ;;
    esac
    printf -v "${key}" '%s' "${value}"
    export "${key?}"
  done <"${f}"
}

remote_sh() {
  # shellcheck disable=SC2029
  deploy_ssh_run "${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "set -Eeuo pipefail; $*"
}

# Only the closed helper may be invoked via sudo.
remote_helper() {
  local -a args=("$@")
  local q="" a
  for a in "${args[@]}"; do
    q+=" $(printf '%q' "${a}")"
  done
  # shellcheck disable=SC2029
  remote_sh "sudo -n ${REMOTE_HELPER}${q}"
}

cleanup_remote_upload() {
  if [[ -n "${REMOTE_UPLOAD:-}" && -n "${DEPLOY_HOST:-}" && ${#SSH_BASE[@]} -gt 0 ]]; then
    set +e
    remote_helper cleanup-upload "${REMOTE_UPLOAD}" >/dev/null 2>&1
    set -e
    REMOTE_UPLOAD=""
  fi
}

restore_envs_once() {
  if [[ "${ENVS_RESTORABLE}" -eq 1 && "${ENV_RESTORED}" -eq 0 && -n "${ENV_BACKUP_ID}" ]]; then
    remote_helper restore-env "${ENV_BACKUP_ID}" >/dev/null
    ENV_RESTORED=1
    report "env_restore=ok id=${ENV_BACKUP_ID}"
  fi
}

# Pre-activation failure: restore envs if restorable, report phase, then die.
pre_activate_fail() {
  local msg="$1"
  if [[ "${ACTIVATED}" -ne 0 ]]; then
    die "internal error: pre_activate_fail after activation (${msg})"
  fi
  report "failure_phase=${phase}"
  restore_envs_once
  report_active
  die "${msg}"
}

read_active_release() {
  remote_sh "if [[ -e /opt/bwb-modulo-fiscal/current ]]; then basename \"\$(readlink /opt/bwb-modulo-fiscal/current)\"; else printf 'none'; fi"
}

sync_active_release() {
  local raw
  raw="$(read_active_release)"
  if [[ "${raw}" == "none" || -z "${raw}" ]]; then
    ACTIVE_RELEASE="none"
  else
    deploy_assert_sha1 "active release" "${raw}"
    ACTIVE_RELEASE="${raw}"
  fi
}

run_remote_health() {
  if [[ "${DEPLOY_SIMULATE_HEALTH_FAIL:-0}" == "1" ]]; then
    DEPLOY_SIMULATE_HEALTH_FAIL=0
    export DEPLOY_SIMULATE_HEALTH_FAIL
    report "health=fail"
    return 1
  fi
  # Capture status explicitly: when this function is invoked under `if`, bash
  # disables set -e for the body, so a failed healthcheck subshell must not
  # fall through to reporting health=ok.
  local st
  set +e
  (
    if [[ -n "${HEALTH_URL+x}" ]]; then
      unset HEALTH_URL
    fi
    DEPLOY_DRY_RUN=0 DEPLOY_MOCK_REMOTE="${DEPLOY_MOCK_REMOTE}" \
      bash "${SCRIPT_DIR}/healthcheck.sh" >/dev/null
  )
  st=$?
  set -e
  if [[ "${st}" -ne 0 ]]; then
    report "health=fail"
    return 1
  fi
  report "health=ok"
}

report_active() {
  report "active_release=${ACTIVE_RELEASE}"
}

# After activate has been attempted: re-read current, then N-1 rollback or roll-forward.
post_activate_fail() {
  local msg="$1"
  report "failure_phase=post_activate"
  sync_active_release

  if [[ "${rollback_allowed}" == "true" ]]; then
    if [[ -n "${PREV_SHA}" && "${PREV_SHA}" != "${HEAD}" ]]; then
      local act_st rst_st
      set +e
      remote_helper activate "${PREV_SHA}" >/dev/null
      act_st=$?
      set -e
      sync_active_release
      if [[ "${act_st}" -ne 0 ]]; then
        report "action=rollback_activate_failed"
        report_active
        die "${msg}; rollback activate failed"
      fi
      restore_envs_once
      set +e
      remote_helper restart >/dev/null
      rst_st=$?
      set -e
      if [[ "${rst_st}" -ne 0 ]]; then
        report "action=rollback_restart_failed"
        report_active
        die "${msg}; rollback restart failed"
      fi
      report "restart=ok"
      if run_remote_health; then
        report "action=restore_previous_binary previous=${PREV_SHA}"
        report "health=ok_after_rollback"
        report_active
        die "${msg}; rolled back to previous release"
      fi
      report "action=rollback_health_failed"
      report_active
      die "${msg}; rollback completed but health still failing; manual intervention required"
    fi
    report "action=restore_previous_binary previous=unavailable"
    report_active
    die "${msg}; rollback allowed but previous release missing; new release remains active"
  fi

  report "action=roll_forward_or_manual"
  report_active
  die "${msg}; no automatic binary rollback (N-1 not proven)"
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

# --- Local dry-run ---
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

# --- Live path ---
: "${DEPLOY_HOST:?DEPLOY_HOST required}"
: "${DEPLOY_USER:?DEPLOY_USER required}"
: "${DEPLOY_SSH_KEY:?DEPLOY_SSH_KEY required}"
: "${DEPLOY_KNOWN_HOSTS:?DEPLOY_KNOWN_HOSTS required}"
[[ -f "${ENV_DEPLOY}" ]] || die "missing deploy env file"
[[ -f "${ENV_MIGRATE}" ]] || die "missing migrate env file"
[[ -z "${HEALTH_URL:-}" ]] || die "HEALTH_URL is forbidden on live path"

deploy_assert_restricted_file "ENV_DEPLOY" "${ENV_DEPLOY}"
deploy_assert_restricted_file "ENV_MIGRATE" "${ENV_MIGRATE}"

deploy_ssh_base
# Invoked only via EXIT trap; preserve the script's original exit status.
# shellcheck disable=SC2317,SC2329
cleanup_live() {
  local ec=$?
  set +e
  cleanup_remote_upload
  deploy_ssh_mux_stop
  if [[ "${DEPLOY_SSH_INVOCATION_COUNT:-0}" -gt 0 ]]; then
    report "ssh_invocations=${DEPLOY_SSH_INVOCATION_COUNT} mux=ControlMaster"
  fi
  set -e
  return "${ec}"
}
trap cleanup_live EXIT

report "mode=live mock_remote=${DEPLOY_MOCK_REMOTE}"

REMOTE_UPLOAD="$(remote_sh "d=\$(mktemp -d /tmp/bwb-upload.XXXXXX); chmod 0700 \"\$d\"; printf '%s' \"\$d\"")"
[[ "${REMOTE_UPLOAD}" =~ ^/tmp/bwb-upload\.[A-Za-z0-9._-]+$ ]] || die "invalid remote upload path"
report "upload_dir=ok"

deploy_scp_run "${SCP_BASE[@]}" -r \
  "${OUT_DIR}/fiscal-api" \
  "${OUT_DIR}/fiscal-migrate" \
  "${OUT_DIR}/COMMIT" \
  "${OUT_DIR}/EXPECTED_SCHEMA_VERSION" \
  "${OUT_DIR}/SHA256SUMS" \
  "${OUT_DIR}/lib" \
  "${DEPLOY_USER}@${DEPLOY_HOST}:${REMOTE_UPLOAD}/"
report "upload=ok"

remote_sh "cd '${REMOTE_UPLOAD}' && test \"\$(tr -d '[:space:]' <COMMIT)\" = '${HEAD}' && test \"\$(tr -d '[:space:]' <EXPECTED_SCHEMA_VERSION)\" = '${EXPECTED_SCHEMA_VERSION}' && (command -v sha256sum >/dev/null && sha256sum -c SHA256SUMS >/dev/null || shasum -a 256 -c SHA256SUMS >/dev/null)"
report "verify=ok commit=${HEAD} schema=${EXPECTED_SCHEMA_VERSION}"

remote_helper install-release "${HEAD}" "${REMOTE_UPLOAD}" >/dev/null
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

ENV_BACKUP_ID="$(date -u +%Y%m%dT%H%M%SZ)-${HEAD}"
remote_helper backup-envs "${ENV_BACKUP_ID}" >/dev/null
# Mark restorable immediately after backup, before any install attempt.
ENVS_RESTORABLE=1
phase="env_install"
rollback_allowed="true"
report "env_backup=ok id=${ENV_BACKUP_ID}"

fiscal_tmp="${REMOTE_UPLOAD}/env.fiscal.env.$$"
migrate_tmp="${REMOTE_UPLOAD}/env.migrate.env.$$"

set +e
deploy_scp_run "${SCP_BASE[@]}" "${ENV_DEPLOY}" "${DEPLOY_USER}@${DEPLOY_HOST}:${fiscal_tmp}"
scp_st=$?
set -e
[[ "${scp_st}" -eq 0 ]] || pre_activate_fail "scp fiscal.env failed"

set +e
deploy_scp_run "${SCP_BASE[@]}" "${ENV_MIGRATE}" "${DEPLOY_USER}@${DEPLOY_HOST}:${migrate_tmp}"
scp_st=$?
set -e
[[ "${scp_st}" -eq 0 ]] || pre_activate_fail "scp migrate.env failed"

set +e
remote_helper install-env fiscal.env "${fiscal_tmp}" >/dev/null
inst_st=$?
set -e
[[ "${inst_st}" -eq 0 ]] || pre_activate_fail "install-env fiscal.env failed"

set +e
remote_helper install-env migrate.env "${migrate_tmp}" >/dev/null
inst_st=$?
set -e
[[ "${inst_st}" -eq 0 ]] || pre_activate_fail "install-env migrate.env failed"

report "install_env=ok mode=0600 owner=root"

phase="pre_migrate"
rollback_allowed="true"
report "phase=${phase} rollback_allowed=${rollback_allowed}"

migration_before_out="$(
  RELEASE_DIR="${RELEASE_DIR}" \
    DEPLOY_DRY_RUN=0 \
    DEPLOY_MOCK_REMOTE="${DEPLOY_MOCK_REMOTE}" \
    bash "${SCRIPT_DIR}/migrate-remote.sh" version
)" || pre_activate_fail "migration version command failed"
deploy_parse_migrate_version "${migration_before_out}" || pre_activate_fail "could not parse migration_before"
migration_before="${DEPLOY_MIG_VERSION}"
migration_before_dirty="${DEPLOY_MIG_DIRTY}"
report "migration_before=${migration_before} dirty=${migration_before_dirty} binary=new_release"
if [[ "${migration_before_dirty}" != "false" ]]; then
  pre_activate_fail "migration dirty before update; refusing"
fi

migration_after_out="$(
  RELEASE_DIR="${RELEASE_DIR}" \
    DEPLOY_DRY_RUN=0 \
    DEPLOY_MOCK_REMOTE="${DEPLOY_MOCK_REMOTE}" \
    bash "${SCRIPT_DIR}/migrate-remote.sh" up
)" || pre_activate_fail "migration up failed"
deploy_parse_migrate_version "${migration_after_out}" || pre_activate_fail "could not parse migration_after"
migration_after="${DEPLOY_MIG_VERSION}"
migration_after_dirty="${DEPLOY_MIG_DIRTY}"
report "migration_after=${migration_after} dirty=${migration_after_dirty} binary=new_release"

if [[ "${migration_after_dirty}" != "false" ]]; then
  report "promote=blocked reason=dirty"
  pre_activate_fail "migration dirty after up; promotion blocked"
fi
if [[ "${migration_after}" != "${EXPECTED_SCHEMA_VERSION}" ]]; then
  report "promote=blocked reason=schema_mismatch expected=${EXPECTED_SCHEMA_VERSION} got=${migration_after}"
  pre_activate_fail "migration version mismatch after up; activation blocked"
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

# --- Post-activation path (activate / restart / health) ---
set +e
remote_helper activate "${HEAD}" >/dev/null
act_st=$?
set -e
sync_active_release
if [[ "${act_st}" -ne 0 ]]; then
  if [[ "${ACTIVE_RELEASE}" == "${HEAD}" ]]; then
    ACTIVATED=1
    post_activate_fail "activate failed but current points to new release"
  fi
  # Activate did not switch current — treat as pre-activate for env restore.
  ACTIVATED=0
  pre_activate_fail "activate failed"
fi
ACTIVATED=1

set +e
remote_helper restart >/dev/null
rst_st=$?
set -e
if [[ "${rst_st}" -ne 0 ]] || [[ "${DEPLOY_SIMULATE_RESTART_FAIL:-0}" == "1" ]]; then
  if [[ "${DEPLOY_SIMULATE_RESTART_FAIL:-0}" == "1" ]]; then
    DEPLOY_SIMULATE_RESTART_FAIL=0
    export DEPLOY_SIMULATE_RESTART_FAIL
  fi
  post_activate_fail "restart failed after activation"
fi
report "restart=ok"

if run_remote_health; then
  report "promote=ok symlink=current->${HEAD}"
  report_active
  cleanup_remote_upload
  report "done"
  exit 0
fi

post_activate_fail "health failed after activation"
