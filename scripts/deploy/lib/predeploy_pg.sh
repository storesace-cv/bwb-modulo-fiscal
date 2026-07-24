#!/usr/bin/env bash
# Pre-deploy PostgreSQL backup gate (sourced by remote-deploy-helper.sh).
# shellcheck shell=bash

# Paths (overridable only when EUID!=0 via BWB_* — enforced by helper entry).
PREDEPLOY_RUN_ROOT="${BWB_DEPLOY_RUN:-/run/bwb-fiscal-deploy}"
PREDEPLOY_BACKUP_ROOT="${BWB_PG_BACKUP_ROOT:-/var/backups/bwb-fiscal/pre-deploy}"
PREDEPLOY_LOCKDIR="${PREDEPLOY_RUN_ROOT}/deploy.lockdir"
PREDEPLOY_STALE_SEC=1800
PREDEPLOY_TIMEOUT_DUMP=900
PREDEPLOY_TIMEOUT_LIST=60
PREDEPLOY_TIMEOUT_CREATEDB=60
PREDEPLOY_TIMEOUT_DROPDB=60
PREDEPLOY_TIMEOUT_RESTORE=900
PREDEPLOY_TIMEOUT_PSQL=30
PREDEPLOY_MIGRATE_OWNER="fiscal_migrate"
PREDEPLOY_PG_SOCKET="${BWB_PREDEPLOY_PG_SOCKET:-/var/run/postgresql}"
PREDEPLOY_PG_PORT="${BWB_PREDEPLOY_PG_PORT:-5432}"
PREDEPLOY_PG_USER="${BWB_PREDEPLOY_PG_USER:-}"
PREDEPLOY_PARSE_PY="${HELPER_LIB}/parse_migrate_dsn.py"

predeploy_emit() {
  printf 'pre_deploy_pg_backup'
  local a
  for a in "$@"; do
    printf ' %s' "${a}"
  done
  printf '\n'
}

predeploy_derive_temp_db() {
  local backup_id="$1"
  local ts sha40 sha8 ts14
  ts="${backup_id%%-*}"
  sha40="${backup_id#*-}"
  sha8="${sha40:0:8}"
  ts14="${ts//T/}"
  ts14="${ts14//Z/}"
  printf 'bwb_pd_%s_%s' "${ts14}" "${sha8}"
}

predeploy_assert_temp_db() {
  local name="$1"
  [[ "${name}" =~ ^bwb_pd_[0-9]{14}_[a-f0-9]{8}$ ]] || die "temp_db_invalid"
}

# Run command with timeout; TERM then KILL only the child process tree.
predeploy_run_timeout() {
  local secs="$1"
  shift
  python3 - "$secs" "$@" <<'PY'
import signal
import subprocess
import sys

secs = int(sys.argv[1])
cmd = sys.argv[2:]
proc = subprocess.Popen(cmd)
try:
    rc = proc.wait(timeout=secs)
except subprocess.TimeoutExpired:
    proc.send_signal(signal.SIGTERM)
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait()
    sys.exit(124)
sys.exit(rc)
PY
}

predeploy_lock_meta_path() {
  printf '%s/meta' "${PREDEPLOY_LOCKDIR}"
}

predeploy_lock_read_state() {
  local meta
  meta="$(predeploy_lock_meta_path)"
  [[ -f "${meta}" ]] || { printf 'corrupt'; return 0; }
  local owner="" acquired="" state=""
  while IFS='=' read -r k v || [[ -n "${k}" ]]; do
    case "${k}" in
      owner) owner="${v}" ;;
      acquired_at_utc) acquired="${v}" ;;
      state) state="${v}" ;;
    esac
  done <"${meta}"
  if [[ -z "${owner}" || -z "${acquired}" || -z "${state}" ]]; then
    printf 'corrupt'
    return 0
  fi
  if [[ "${state}" == "poisoned" ]]; then
    printf 'poisoned'
    return 0
  fi
  if [[ "${state}" != "held" ]]; then
    printf 'corrupt'
    return 0
  fi
  local now age
  now="$(date -u +%s)"
  if [[ ! "${acquired}" =~ ^[0-9]+$ ]]; then
    printf 'corrupt'
    return 0
  fi
  age=$((now - acquired))
  if [[ "${age}" -gt "${PREDEPLOY_STALE_SEC}" ]]; then
    printf 'stale'
    return 0
  fi
  printf 'held'
}

op_deploy_lock_acquire() {
  local backup_id="$1"
  assert_backup_id "${backup_id}"
  install -d -m 0755 -o root -g root "${PREDEPLOY_RUN_ROOT}" 2>/dev/null \
    || install -d -m 0755 "${PREDEPLOY_RUN_ROOT}"

  if [[ -e "${PREDEPLOY_LOCKDIR}" ]]; then
    local st
    st="$(predeploy_lock_read_state)"
    case "${st}" in
      held) die "deploy_lock_busy" ;;
      stale) die "deploy_lock_stale_present" ;;
      poisoned) die "deploy_lock_poisoned" ;;
      *) die "deploy_lock_corrupt" ;;
    esac
  fi

  if ! mkdir "${PREDEPLOY_LOCKDIR}" 2>/dev/null; then
    die "deploy_lock_busy"
  fi
  if [[ "${EUID}" -eq 0 ]]; then
    chown root:root "${PREDEPLOY_LOCKDIR}"
    chmod 0700 "${PREDEPLOY_LOCKDIR}"
  else
    chmod 0700 "${PREDEPLOY_LOCKDIR}"
  fi
  local meta now
  now="$(date -u +%s)"
  meta="$(predeploy_lock_meta_path)"
  umask 077
  printf 'owner=%s\nacquired_at_utc=%s\nstate=held\n' "${backup_id}" "${now}" >"${meta}"
  chmod 0600 "${meta}"
  printf 'deploy_lock_acquire_ok backup_id=%s state=held\n' "${backup_id}"
}

op_deploy_lock_release() {
  local backup_id="$1"
  assert_backup_id "${backup_id}"
  [[ -d "${PREDEPLOY_LOCKDIR}" ]] || die "deploy_lock_missing"
  local st owner meta
  st="$(predeploy_lock_read_state)"
  [[ "${st}" == "held" ]] || die "deploy_lock_release_denied state=${st}"
  meta="$(predeploy_lock_meta_path)"
  owner="$(sed -n 's/^owner=//p' "${meta}" | head -1)"
  [[ "${owner}" == "${backup_id}" ]] || die "deploy_lock_owner_mismatch"
  rm -f -- "${meta}"
  rmdir "${PREDEPLOY_LOCKDIR}" || die "deploy_lock_release_failed"
  printf 'deploy_lock_release_ok backup_id=%s\n' "${backup_id}"
}

predeploy_poison_lock() {
  local meta
  meta="$(predeploy_lock_meta_path)"
  [[ -f "${meta}" ]] || return 0
  local owner acquired
  owner="$(sed -n 's/^owner=//p' "${meta}" | head -1)"
  acquired="$(sed -n 's/^acquired_at_utc=//p' "${meta}" | head -1)"
  printf 'owner=%s\nacquired_at_utc=%s\nstate=poisoned\n' "${owner}" "${acquired}" >"${meta}"
  chmod 0600 "${meta}"
}

predeploy_require_lock_owner() {
  local backup_id="$1"
  [[ -d "${PREDEPLOY_LOCKDIR}" ]] || die "deploy_lock_missing"
  local st owner meta
  st="$(predeploy_lock_read_state)"
  [[ "${st}" == "held" ]] || die "deploy_lock_not_held state=${st}"
  meta="$(predeploy_lock_meta_path)"
  owner="$(sed -n 's/^owner=//p' "${meta}" | head -1)"
  [[ "${owner}" == "${backup_id}" ]] || die "deploy_lock_owner_mismatch"
}

predeploy_run_migrate_pg() {
  # Usage: predeploy_run_migrate_pg <timeout> <PGSERVICE> <workdir> <cmd...>
  local timeout="$1" service="$2" workdir="$3"
  shift 3
  if [[ "${EUID}" -eq 0 ]]; then
    if command -v runuser >/dev/null 2>&1; then
      predeploy_run_timeout "${timeout}" runuser -u "${MIGRATE_USER}" -- env -i \
        PATH="/usr/bin:/bin" \
        HOME="/nonexistent" \
        PGSERVICEFILE="${workdir}/pg_service.conf" \
        PGSERVICE="${service}" \
        PGPASSFILE="${workdir}/pgpass" \
        "$@"
      return $?
    fi
    if command -v setpriv >/dev/null 2>&1; then
      predeploy_run_timeout "${timeout}" setpriv --reuid="${MIGRATE_USER}" --regid="${MIGRATE_USER}" \
        --init-groups --reset-env \
        env -i \
          PATH="/usr/bin:/bin" \
          HOME="/nonexistent" \
          PGSERVICEFILE="${workdir}/pg_service.conf" \
          PGSERVICE="${service}" \
          PGPASSFILE="${workdir}/pgpass" \
          "$@"
      return $?
    fi
    die "runuser or setpriv required for predeploy migrate"
  fi
  # Non-root tests: cleaned env; forward only explicit mock instrumentation vars.
  predeploy_run_timeout "${timeout}" env -i \
    PATH="${PATH:-/usr/bin:/bin}" \
    HOME="${HOME:-/tmp}" \
    PGSERVICEFILE="${workdir}/pg_service.conf" \
    PGSERVICE="${service}" \
    PGPASSFILE="${workdir}/pgpass" \
    DEPLOY_MOCK_PREDEPLOY_LOG="${DEPLOY_MOCK_PREDEPLOY_LOG:-}" \
    DEPLOY_MOCK_PREDEPLOY_DBS="${DEPLOY_MOCK_PREDEPLOY_DBS:-}" \
    DEPLOY_MOCK_PG_RESTORE_LIST_FAIL="${DEPLOY_MOCK_PG_RESTORE_LIST_FAIL:-}" \
    DEPLOY_MOCK_PG_RESTORE_FAIL="${DEPLOY_MOCK_PG_RESTORE_FAIL:-}" \
    DEPLOY_MOCK_PSQL_SCHEMA_MODE="${DEPLOY_MOCK_PSQL_SCHEMA_MODE:-}" \
    DEPLOY_MOCK_PSQL_SCHEMA_ROW="${DEPLOY_MOCK_PSQL_SCHEMA_ROW:-}" \
    DEPLOY_MOCK_REQUIRE_CLEAN_PG="${DEPLOY_MOCK_REQUIRE_CLEAN_PG:-}" \
    DEPLOY_MOCK_CREATEDB_FAIL="${DEPLOY_MOCK_CREATEDB_FAIL:-}" \
    DEPLOY_MOCK_DROPDB_FAIL="${DEPLOY_MOCK_DROPDB_FAIL:-}" \
    "$@"
}

predeploy_run_os_postgres() {
  local timeout="$1"
  shift
  if [[ "${EUID}" -eq 0 ]]; then
    if command -v runuser >/dev/null 2>&1; then
      predeploy_run_timeout "${timeout}" runuser -u postgres -- env -i \
        PATH="/usr/bin:/bin" \
        HOME="/var/lib/postgresql" \
        "$@"
      return $?
    fi
    die "runuser required for os postgres predeploy"
  fi
  if [[ -n "${BWB_PREDEPLOY_OS_POSTGRES_WRAP:-}" ]]; then
    # shellcheck disable=SC2086
    predeploy_run_timeout "${timeout}" env -i PATH="${PATH:-/usr/bin:/bin}" \
      DEPLOY_MOCK_PREDEPLOY_LOG="${DEPLOY_MOCK_PREDEPLOY_LOG:-}" \
      DEPLOY_MOCK_PREDEPLOY_DBS="${DEPLOY_MOCK_PREDEPLOY_DBS:-}" \
      DEPLOY_MOCK_CREATEDB_FAIL="${DEPLOY_MOCK_CREATEDB_FAIL:-}" \
      DEPLOY_MOCK_DROPDB_FAIL="${DEPLOY_MOCK_DROPDB_FAIL:-}" \
      ${BWB_PREDEPLOY_OS_POSTGRES_WRAP} "$@"
    return $?
  fi
  predeploy_run_timeout "${timeout}" env -i PATH="${PATH:-/usr/bin:/bin}" \
    DEPLOY_MOCK_PREDEPLOY_LOG="${DEPLOY_MOCK_PREDEPLOY_LOG:-}" \
    DEPLOY_MOCK_PREDEPLOY_DBS="${DEPLOY_MOCK_PREDEPLOY_DBS:-}" \
    DEPLOY_MOCK_CREATEDB_FAIL="${DEPLOY_MOCK_CREATEDB_FAIL:-}" \
    DEPLOY_MOCK_DROPDB_FAIL="${DEPLOY_MOCK_DROPDB_FAIL:-}" \
    "$@"
}

# Atomic durable install: O_EXCL partial + fsync + link (NOREPLACE) + unlink partial.
# Returns 0 on success; non-zero on failure (does not exit).
predeploy_install_durable_atomic() {
  local work_dump="$1" dest="$2"
  python3 - "${work_dump}" "${dest}" <<'PY'
import os, sys, stat

src, dest = sys.argv[1], sys.argv[2]
dest_dir = os.path.dirname(dest)


def fail(code: int) -> None:
    sys.exit(code)


try:
    st_dir = os.lstat(dest_dir)
except FileNotFoundError:
    os.makedirs(dest_dir, mode=0o700, exist_ok=True)
    st_dir = os.lstat(dest_dir)
if stat.S_ISLNK(st_dir.st_mode):
    fail(2)
if not stat.S_ISDIR(st_dir.st_mode):
    fail(2)

try:
    st_dest = os.lstat(dest)
except FileNotFoundError:
    st_dest = None
if st_dest is not None:
    fail(3)  # durable_dump_exists

partial = dest + ".partial." + str(os.getpid())
flags = os.O_CREAT | os.O_EXCL | os.O_WRONLY | os.O_CLOEXEC
try:
    fd = os.open(partial, flags, 0o600)
except FileExistsError:
    fail(4)

try:
    with open(src, "rb") as inf:
        while True:
            chunk = inf.read(1024 * 1024)
            if not chunk:
                break
            os.write(fd, chunk)
    os.fsync(fd)
finally:
    os.close(fd)

try:
    os.link(partial, dest)
except FileExistsError:
    try:
        os.unlink(partial)
    except OSError:
        pass
    fail(3)
except OSError:
    try:
        os.unlink(partial)
    except OSError:
        pass
    fail(5)

try:
    os.unlink(partial)
except OSError:
    pass
os.chmod(dest, 0o600)
sys.exit(0)
PY
}

predeploy_schema_query() {
  local workdir="$1" service="$2"
  predeploy_run_migrate_pg "${PREDEPLOY_TIMEOUT_PSQL}" "${service}" "${workdir}" \
    psql -v ON_ERROR_STOP=1 -Atc \
    "SELECT version, dirty FROM public.bwb_schema_migrations;"
}

predeploy_validate_schema_row() {
  local row="$1" expect_version="${2:-}"
  local n
  n="$(printf '%s\n' "${row}" | grep -c . || :)"
  [[ "${n}" -eq 1 ]] || return 1
  local ver dirty
  ver="${row%%|*}"
  dirty="${row#*|}"
  [[ "${dirty}" == "f" || "${dirty}" == "false" ]] || return 1
  if [[ -n "${expect_version}" && "${ver}" != "${expect_version}" ]]; then
    return 1
  fi
  printf '%s' "${ver}"
}

op_pre_deploy_pg_backup() {
  local sha="$1" backup_id="$2"
  assert_sha1 "sha" "${sha}"
  assert_backup_id "${backup_id}"
  [[ "${backup_id}" == *"-${sha}" ]] || die "backup_id_sha_mismatch"

  local backup_created="false" restore_verified="false"
  local schema_before="" dirty_before="true"
  local temp_db_created="false" temp_db_dropped="false"
  local lock_state="held" deploy_allowed="false" dump_bytes="0"
  local workdir="" temp_db="" durable="" work_dump=""

  predeploy_require_lock_owner "${backup_id}"

  workdir="${PREDEPLOY_RUN_ROOT}/work/${backup_id}"
  temp_db="$(predeploy_derive_temp_db "${backup_id}")"
  predeploy_assert_temp_db "${temp_db}"
  durable="${PREDEPLOY_BACKUP_ROOT}/${backup_id}.dump"
  work_dump="${workdir}/work.dump"

  cleanup_workdir() {
    if [[ -n "${workdir}" && -d "${workdir}" ]]; then
      rm -rf -- "${workdir}"
    fi
  }

  emit_and_fail() {
    local fail_reason="$1"
    predeploy_emit \
      "backup_id=${backup_id}" \
      "backup_created=${backup_created}" \
      "restore_verified=${restore_verified}" \
      "schema_before=${schema_before:-unknown}" \
      "dirty_before=${dirty_before}" \
      "temp_db_created=${temp_db_created}" \
      "temp_db_dropped=${temp_db_dropped}" \
      "lock_state=${lock_state}" \
      "deploy_allowed=${deploy_allowed}" \
      "dump_bytes=${dump_bytes}" \
      "error=${fail_reason}"
    cleanup_workdir
    die "${fail_reason}"
  }

  try_drop_temp() {
    set +e
    if [[ -n "${PREDEPLOY_PG_USER}" ]]; then
      predeploy_run_os_postgres "${PREDEPLOY_TIMEOUT_DROPDB}" \
        dropdb -h "${PREDEPLOY_PG_SOCKET}" -p "${PREDEPLOY_PG_PORT}" -U "${PREDEPLOY_PG_USER}" -- "${temp_db}"
    else
      predeploy_run_os_postgres "${PREDEPLOY_TIMEOUT_DROPDB}" \
        dropdb -h "${PREDEPLOY_PG_SOCKET}" -p "${PREDEPLOY_PG_PORT}" -- "${temp_db}"
    fi
    local dst=$?
    set -e
    if [[ "${dst}" -eq 0 ]]; then
      temp_db_dropped="true"
      return 0
    fi
    lock_state="poisoned"
    predeploy_poison_lock
    temp_db_dropped="false"
    deploy_allowed="false"
    return 1
  }

  rm -rf -- "${workdir}"
  install -d -m 0700 "${workdir}"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${MIGRATE_USER}:${MIGRATE_USER}" "${workdir}"
  fi

  local env_file="${ETC_ROOT}/migrate.env"
  [[ -f "${env_file}" && ! -L "${env_file}" ]] || emit_and_fail "migrate_env_missing"
  local allowlist="${HELPER_LIB}/migrate.env.allowlist"
  [[ -f "${allowlist}" ]] || emit_and_fail "migrate_allowlist_missing"
  deploy_validate_exact_allowlisted_file "${allowlist}" "${env_file}" \
    || emit_and_fail "migrate_env_allowlist"
  local driver url
  driver="$(deploy_read_env_value "${env_file}" FISCAL_DATABASE_DRIVER)"
  url="$(deploy_read_env_value "${env_file}" FISCAL_DATABASE_URL)"
  [[ "${driver}" == "postgres" ]] || emit_and_fail "migrate_driver"

  [[ -f "${PREDEPLOY_PARSE_PY}" ]] || emit_and_fail "parse_script_missing"
  if ! printf '%s' "${url}" | python3 "${PREDEPLOY_PARSE_PY}" \
    --outdir "${workdir}" --temp-db "${temp_db}" >/dev/null; then
    emit_and_fail "dsn_parse_failed"
  fi
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${MIGRATE_USER}:${MIGRATE_USER}" "${workdir}/pg_service.conf" "${workdir}/pgpass"
  fi
  chmod 0600 "${workdir}/pg_service.conf" "${workdir}/pgpass"

  # 1) schema check origin
  local row qst
  set +e
  row="$(predeploy_schema_query "${workdir}" "bwb_predeploy" 2>/dev/null)"
  qst=$?
  set -e
  [[ "${qst}" -eq 0 ]] || emit_and_fail "schema_query_failed"
  set +e
  schema_before="$(predeploy_validate_schema_row "${row}")"
  qst=$?
  set -e
  [[ "${qst}" -eq 0 ]] || emit_and_fail "schema_invalid"
  dirty_before="false"

  # 2-3) pg_dump + size
  set +e
  predeploy_run_migrate_pg "${PREDEPLOY_TIMEOUT_DUMP}" "bwb_predeploy" "${workdir}" \
    pg_dump -Fc --no-owner --no-acl -f "${work_dump}"
  qst=$?
  set -e
  [[ "${qst}" -eq 0 ]] || emit_and_fail "pg_dump_failed"
  [[ -f "${work_dump}" ]] || emit_and_fail "pg_dump_failed"
  dump_bytes="$(wc -c <"${work_dump}" | tr -d ' ')"
  [[ "${dump_bytes}" -gt 0 ]] || emit_and_fail "dump_invalid"

  # 4) pg_restore --list
  set +e
  predeploy_run_migrate_pg "${PREDEPLOY_TIMEOUT_LIST}" "bwb_predeploy" "${workdir}" \
    pg_restore --list "${work_dump}" >/dev/null
  qst=$?
  set -e
  [[ "${qst}" -eq 0 ]] || emit_and_fail "pg_restore_list_failed"

  # 5) durable install (only after list OK)
  set +e
  predeploy_install_durable_atomic "${work_dump}" "${durable}"
  qst=$?
  set -e
  case "${qst}" in
    0) ;;
    3) emit_and_fail "durable_dump_exists" ;;
    *) emit_and_fail "durable_install_failed" ;;
  esac
  backup_created="true"
  dump_bytes="$(wc -c <"${durable}" | tr -d ' ')"

  # 6) createdb (OS postgres)
  set +e
  if [[ -n "${PREDEPLOY_PG_USER}" ]]; then
    predeploy_run_os_postgres "${PREDEPLOY_TIMEOUT_CREATEDB}" \
      createdb -h "${PREDEPLOY_PG_SOCKET}" -p "${PREDEPLOY_PG_PORT}" -U "${PREDEPLOY_PG_USER}" \
      -O "${PREDEPLOY_MIGRATE_OWNER}" \
      -E UTF8 --template=template0 -- "${temp_db}"
  else
    predeploy_run_os_postgres "${PREDEPLOY_TIMEOUT_CREATEDB}" \
      createdb -h "${PREDEPLOY_PG_SOCKET}" -p "${PREDEPLOY_PG_PORT}" \
      -O "${PREDEPLOY_MIGRATE_OWNER}" \
      -E UTF8 --template=template0 -- "${temp_db}"
  fi
  qst=$?
  set -e
  [[ "${qst}" -eq 0 ]] || emit_and_fail "temp_db_create_failed"
  temp_db_created="true"

  # 7) restore into temp (connection via service stanza only)
  set +e
  predeploy_run_migrate_pg "${PREDEPLOY_TIMEOUT_RESTORE}" "bwb_predeploy_temp" "${workdir}" \
    pg_restore --exit-on-error --single-transaction --no-owner --no-privileges \
    -d "service=bwb_predeploy_temp" "${work_dump}"
  qst=$?
  set -e
  if [[ "${qst}" -ne 0 ]]; then
    if ! try_drop_temp; then
      emit_and_fail "temp_db_cleanup_failed"
    fi
    emit_and_fail "pg_restore_failed"
  fi

  # 8) validate temp schema
  set +e
  row="$(predeploy_schema_query "${workdir}" "bwb_predeploy_temp" 2>/dev/null)"
  qst=$?
  set -e
  if [[ "${qst}" -ne 0 ]] || ! predeploy_validate_schema_row "${row}" "${schema_before}" >/dev/null; then
    if ! try_drop_temp; then
      emit_and_fail "temp_db_cleanup_failed"
    fi
    emit_and_fail "restore_schema_mismatch"
  fi
  restore_verified="true"

  # 9) dropdb
  if ! try_drop_temp; then
    emit_and_fail "temp_db_cleanup_failed"
  fi
  deploy_allowed="true"
  cleanup_workdir

  predeploy_emit \
    "backup_id=${backup_id}" \
    "backup_created=${backup_created}" \
    "restore_verified=${restore_verified}" \
    "schema_before=${schema_before}" \
    "dirty_before=${dirty_before}" \
    "temp_db_created=${temp_db_created}" \
    "temp_db_dropped=${temp_db_dropped}" \
    "lock_state=${lock_state}" \
    "deploy_allowed=${deploy_allowed}" \
    "dump_bytes=${dump_bytes}"
}
