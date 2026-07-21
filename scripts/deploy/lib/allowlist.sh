#!/usr/bin/env bash
# Shared helpers for staging deploy scripts. Never print env values.
# Compatible with Bash 3.2+ (macOS) and Bash 4+ (CI).
# shellcheck shell=bash

set -Eeuo pipefail

# Schema version expected after successful migrate up for this release line.
# Keep in sync with the highest forward migration in migrations/.
# shellcheck disable=SC2034 # referenced by build/update scripts that source this library
DEPLOY_EXPECTED_SCHEMA_VERSION_DEFAULT=2

deploy_repo_root() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
  printf '%s' "${here}"
}

deploy_trim() {
  sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
}

deploy_assert_sha1() {
  local name="$1"
  local val="$2"
  if [[ ! "${val}" =~ ^[0-9a-f]{40}$ ]]; then
    echo "error: ${name} must be a 40-char lowercase hex SHA-1" >&2
    return 1
  fi
}

deploy_file_mode_octal() {
  local f="$1"
  if mode="$(stat -c '%a' "${f}" 2>/dev/null)"; then
    printf '%s' "${mode}"
    return 0
  fi
  if mode="$(stat -f '%OLp' "${f}" 2>/dev/null)"; then
    printf '%s' "${mode}"
    return 0
  fi
  return 1
}

# Regular file, not a symlink, owner-only access (no group/other bits).
deploy_assert_restricted_file() {
  local label="$1"
  local f="$2"
  local mode
  if [[ ! -e "${f}" ]]; then
    echo "error: ${label} missing" >&2
    return 1
  fi
  if [[ -L "${f}" ]]; then
    echo "error: ${label} must not be a symlink" >&2
    return 1
  fi
  if [[ ! -f "${f}" ]]; then
    echo "error: ${label} must be a regular file" >&2
    return 1
  fi
  mode="$(deploy_file_mode_octal "${f}")" || {
    echo "error: ${label} could not read mode" >&2
    return 1
  }
  case "${mode}" in
    400 | 600 | 0400 | 0600) ;;
    *)
      echo "error: ${label} permissions must be 0600 or 0400 (got ${mode})" >&2
      return 1
      ;;
  esac
}

deploy_allowlist_has() {
  local allowlist_file="$1"
  local key="$2"
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="$(printf '%s' "${line}" | deploy_trim)"
    [[ -z "${line}" ]] && continue
    [[ "${line}" == \#* ]] && continue
    if [[ "${line}" == "${key}" ]]; then
      return 0
    fi
  done <"${allowlist_file}"
  return 1
}

deploy_allowlist_keys() {
  local allowlist_file="$1"
  local line
  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="$(printf '%s' "${line}" | deploy_trim)"
    [[ -z "${line}" ]] && continue
    [[ "${line}" == \#* ]] && continue
    printf '%s\n' "${line}"
  done <"${allowlist_file}"
}

deploy_is_skip_line() {
  local line="$1"
  local trimmed
  trimmed="$(printf '%s' "${line}" | deploy_trim)"
  [[ -z "${trimmed}" || "${trimmed}" == \#* ]]
}

deploy_validate_key_name() {
  local key="$1"
  case "${key}" in
    '' | *[!A-Za-z0-9_]* | [0-9]*)
      return 1
      ;;
  esac
  return 0
}

# Keys present must be in allowlist; no duplicates; well-formed.
deploy_validate_allowlisted_file() {
  local allowlist_file="$1"
  local env_file="$2"
  local raw key lineno=0
  local seen_file
  seen_file="$(mktemp)"

  if [[ ! -f "${allowlist_file}" ]]; then
    echo "error: allowlist file missing" >&2
    return 1
  fi
  if [[ ! -f "${env_file}" ]]; then
    rm -f "${seen_file}"
    echo "error: env file missing" >&2
    return 1
  fi

  while IFS= read -r raw || [[ -n "${raw}" ]]; do
    lineno=$((lineno + 1))
    if deploy_is_skip_line "${raw}"; then
      continue
    fi
    case "${raw}" in
      *=*) ;;
      *)
        rm -f "${seen_file}"
        echo "error: malformed env line ${lineno}" >&2
        return 1
        ;;
    esac
    key="${raw%%=*}"
    key="$(printf '%s' "${key}" | deploy_trim)"
    if ! deploy_validate_key_name "${key}"; then
      rm -f "${seen_file}"
      echo "error: malformed key on line ${lineno}" >&2
      return 1
    fi
    if ! deploy_allowlist_has "${allowlist_file}" "${key}"; then
      rm -f "${seen_file}"
      echo "error: key not in allowlist: ${key}" >&2
      return 1
    fi
    if grep -Fxq "${key}" "${seen_file}"; then
      rm -f "${seen_file}"
      echo "error: duplicate key: ${key}" >&2
      return 1
    fi
    printf '%s\n' "${key}" >>"${seen_file}"
  done <"${env_file}"
  rm -f "${seen_file}"
  return 0
}

# Exactly the allowlisted keys (all required, none extra). Used for migrate.env.
deploy_validate_exact_allowlisted_file() {
  local allowlist_file="$1"
  local env_file="$2"
  local key
  deploy_validate_allowlisted_file "${allowlist_file}" "${env_file}" || return 1
  while IFS= read -r key || [[ -n "${key}" ]]; do
    [[ -z "${key}" ]] && continue
    if ! deploy_read_env_value "${env_file}" "${key}" >/dev/null; then
      echo "error: missing required allowlisted key: ${key}" >&2
      return 1
    fi
  done < <(deploy_allowlist_keys "${allowlist_file}")
  return 0
}

deploy_load_allowlisted_env() {
  local allowlist_file="$1"
  local env_file="$2"
  local raw key value lineno=0

  deploy_validate_allowlisted_file "${allowlist_file}" "${env_file}" || return 1

  while IFS= read -r raw || [[ -n "${raw}" ]]; do
    lineno=$((lineno + 1))
    if deploy_is_skip_line "${raw}"; then
      continue
    fi
    key="${raw%%=*}"
    value="${raw#*=}"
    key="$(printf '%s' "${key}" | deploy_trim)"
    printf -v "${key}" '%s' "${value}"
    export "${key?}"
  done <"${env_file}"
}

deploy_read_env_value() {
  local env_file="$1"
  local want_key="$2"
  local raw key value found=0

  while IFS= read -r raw || [[ -n "${raw}" ]]; do
    if deploy_is_skip_line "${raw}"; then
      continue
    fi
    case "${raw}" in
      *=*) ;;
      *) continue ;;
    esac
    key="${raw%%=*}"
    value="${raw#*=}"
    key="$(printf '%s' "${key}" | deploy_trim)"
    if [[ "${key}" == "${want_key}" ]]; then
      if [[ "${found}" -eq 1 ]]; then
        echo "error: duplicate key: ${want_key}" >&2
        return 1
      fi
      found=1
      printf '%s' "${value}"
    fi
  done <"${env_file}"
  if [[ "${found}" -ne 1 ]]; then
    echo "error: missing required key: ${want_key}" >&2
    return 1
  fi
}

deploy_require_cmds() {
  local c
  for c in "$@"; do
    if ! command -v "${c}" >/dev/null 2>&1; then
      echo "error: required command not found: ${c}" >&2
      return 1
    fi
  done
}

deploy_assert_clean_worktree() {
  local status
  status="$(git status --porcelain)"
  if [[ -n "${status}" ]]; then
    echo "error: refusing build with dirty worktree (tracked/untracked changes present)" >&2
    return 1
  fi
}

deploy_validate_out_dir() {
  local root="$1"
  local out="$2"
  local parent abs test_root
  parent="$(dirname "${out}")"
  mkdir -p "${parent}"
  abs="$(cd "${parent}" && pwd)/$(basename "${out}")"
  case "${abs}" in
    "${root}/dist/releases/"*)
      return 0
      ;;
  esac
  if [[ -n "${DEPLOY_TEST_OUT_ROOT:-}" ]]; then
    test_root="$(cd "${DEPLOY_TEST_OUT_ROOT}" && pwd)"
    case "${abs}" in
      "${test_root}/"*)
        return 0
        ;;
    esac
  fi
  echo "error: OUT_DIR not under dist/releases or DEPLOY_TEST_OUT_ROOT" >&2
  return 1
}

# Parse "version=N dirty=true|false"; never logs DSN.
# Sets globals: DEPLOY_MIG_VERSION DEPLOY_MIG_DIRTY
deploy_parse_migrate_version() {
  local out="$1"
  DEPLOY_MIG_VERSION=""
  DEPLOY_MIG_DIRTY=""
  if [[ "${out}" =~ version=([0-9]+) ]]; then
    DEPLOY_MIG_VERSION="${BASH_REMATCH[1]}"
  fi
  if [[ "${out}" =~ dirty=(true|false) ]]; then
    DEPLOY_MIG_DIRTY="${BASH_REMATCH[1]}"
  elif [[ "${out}" =~ dirty=([01]) ]]; then
    if [[ "${BASH_REMATCH[1]}" == "1" ]]; then
      DEPLOY_MIG_DIRTY="true"
    else
      DEPLOY_MIG_DIRTY="false"
    fi
  fi
  if [[ -z "${DEPLOY_MIG_VERSION}" || -z "${DEPLOY_MIG_DIRTY}" ]]; then
    echo "error: could not parse migration version/dirty from migrate output" >&2
    return 1
  fi
}

deploy_sha256_files() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$@"
  else
    shasum -a 256 "$@" | awk '{print $1"  "$2}'
  fi
}

deploy_sha256_check() {
  local sums="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c "${sums}" >/dev/null
  else
    shasum -a 256 -c "${sums}" >/dev/null
  fi
}

# Verify release directory layout + full SHA256SUMS manifest.
# Release must not ship remote-migrate-run.sh (migrate is drop-priv via closed helper).
deploy_verify_release_manifest() {
  local dir="$1"
  local expected_commit="${2:-}"
  (
    cd "${dir}"
    test -f COMMIT
    test -f EXPECTED_SCHEMA_VERSION
    test -f SHA256SUMS
    test -f fiscal-api
    test -f fiscal-migrate
    test ! -e remote-migrate-run.sh
    test -f lib/allowlist.sh
    test -f lib/migrate.env.allowlist
    deploy_sha256_check SHA256SUMS
    if [[ -n "${expected_commit}" ]]; then
      test "$(tr -d '[:space:]' <COMMIT)" = "${expected_commit}"
    fi
  )
}

deploy_release_dir_for_sha() {
  local sha="$1"
  deploy_assert_sha1 "release sha" "${sha}" || return 1
  printf '/opt/bwb-modulo-fiscal/releases/%s' "${sha}"
}

# Multiplex dir for ControlMaster (one TCP session per deploy host/user/port).
# Keep path short: macOS AF_UNIX sun_path limit is ~104 bytes. Never under the repo.
deploy_ssh_mux_dir() {
  local base="/tmp/bwb-ssh-${UID:-$(id -u)}"
  mkdir -p "${base}"
  chmod 700 "${base}"
  printf '%s' "${base}"
}

# Unique ControlPath per local UID + remote user/host/port (no cross-user reuse).
deploy_ssh_control_path() {
  local mux_dir token
  : "${DEPLOY_USER:?DEPLOY_USER required for ControlPath}"
  : "${DEPLOY_HOST:?DEPLOY_HOST required for ControlPath}"
  mux_dir="$(deploy_ssh_mux_dir)"
  token="$(
    printf '%s@%s:%s' "${DEPLOY_USER}" "${DEPLOY_HOST}" "${DEPLOY_SSH_PORT:-22}" \
      | shasum -a 256 \
      | awk '{ print substr($1, 1, 20) }'
  )"
  printf '%s/cm-%s' "${mux_dir}" "${token}"
}

# Remove stale ControlMaster sockets left by crashed clients.
deploy_ssh_clear_stale_control() {
  local cpath="$1"
  [[ -e "${cpath}" ]] || return 0
  if ! ssh \
    -o BatchMode=yes \
    -o ConnectTimeout=2 \
    -o ControlPath="${cpath}" \
    -O check \
    "${DEPLOY_USER}@${DEPLOY_HOST}" >/dev/null 2>&1; then
    rm -f "${cpath}"
  fi
}

# Shared OpenSSH options: reuse a single TCP connection under UFW LIMIT (6 NEW/30s).
deploy_ssh_opts() {
  local cpath
  cpath="$(deploy_ssh_control_path)"
  deploy_ssh_clear_stale_control "${cpath}"
  DEPLOY_SSH_CONTROL_PATH="${cpath}"
  # shellcheck disable=SC2034
  DEPLOY_SSH_OPTS=(
    -i "${DEPLOY_SSH_KEY}"
    -o IdentitiesOnly=yes
    -o StrictHostKeyChecking=yes
    -o UserKnownHostsFile="${DEPLOY_KNOWN_HOSTS}"
    -o BatchMode=yes
    -o ConnectTimeout=15
    -o ConnectionAttempts=1
    -o ControlMaster=auto
    -o ControlPersist=120
    -o "ControlPath=${cpath}"
  )
}

deploy_ssh_base() {
  DEPLOY_SSH_KEY="${DEPLOY_SSH_KEY/#\~/${HOME}}"
  DEPLOY_KNOWN_HOSTS="${DEPLOY_KNOWN_HOSTS/#\~/${HOME}}"
  deploy_require_cmds ssh scp
  deploy_assert_restricted_file "DEPLOY_SSH_KEY" "${DEPLOY_SSH_KEY}"
  if [[ ! -f "${DEPLOY_KNOWN_HOSTS}" || -L "${DEPLOY_KNOWN_HOSTS}" ]]; then
    echo "error: DEPLOY_KNOWN_HOSTS must be a regular non-symlink file" >&2
    return 1
  fi
  : "${DEPLOY_USER:?DEPLOY_USER required}"
  : "${DEPLOY_HOST:?DEPLOY_HOST required}"
  deploy_ssh_opts
  # shellcheck disable=SC2034
  SSH_BASE=(ssh "${DEPLOY_SSH_OPTS[@]}")
  # shellcheck disable=SC2034
  SCP_BASE=(scp "${DEPLOY_SSH_OPTS[@]}")
  DEPLOY_SSH_INVOCATION_COUNT="${DEPLOY_SSH_INVOCATION_COUNT:-0}"
}

# Count ssh/scp process invocations (not TCP). With ControlMaster, many invokes share one TCP.
# Log fields are counters/timestamps only — never keys, paths, or secrets.
deploy_ssh_note_invoke() {
  local kind="$1"
  DEPLOY_SSH_INVOCATION_COUNT="${DEPLOY_SSH_INVOCATION_COUNT:-0}"
  DEPLOY_SSH_INVOCATION_COUNT=$((DEPLOY_SSH_INVOCATION_COUNT + 1))
  if [[ -n "${DEPLOY_SSH_INVOKE_LOG:-}" ]]; then
    printf 'invoke kind=%s n=%s ts=%s\n' \
      "${kind}" "${DEPLOY_SSH_INVOCATION_COUNT}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
      >>"${DEPLOY_SSH_INVOKE_LOG}"
  fi
}

# True only for transient transport failures. Never auth, host-key, or Connection refused
# (refused often means UFW LIMIT; retrying would open more NEW TCPs and worsen the storm).
deploy_ssh_is_retryable_transport() {
  local errf="$1"
  [[ -s "${errf}" ]] || return 1
  if grep -Eiq \
    'Permission denied|Host key verification failed|Too many authentication failures|Authentication (refused|failed)|Bad permissions|UNPROTECTED PRIVATE KEY|Load key|No such identity|identity file|REMOTE HOST IDENTIFICATION HAS CHANGED|Offending (POSIX )?key|Connection refused' \
    "${errf}"; then
    return 1
  fi
  grep -Eiq \
    'Connection timed out|Connection reset|Network is unreachable|No route to host|Temporary failure in name resolution|Broken pipe|Connection closed by remote host|kex_exchange_identification|Operation timed out' \
    "${errf}"
}

# Retry only retryable transport failures (exit 255), with explicit max + exponential backoff.
deploy_ssh_run() {
  local attempt=1
  local max_attempts="${DEPLOY_SSH_MAX_ATTEMPTS:-3}"
  local delay="${DEPLOY_SSH_RETRY_DELAY_SEC:-2}"
  local st=0
  local errf
  errf="$(mktemp "${TMPDIR:-/tmp}/bwb-ssh-err.XXXXXX")"
  deploy_ssh_note_invoke ssh
  while true; do
    # Use && so set -e does not abort, and $? still reflects the ssh/scp status.
    "$@" 2>"${errf}" && {
      if [[ -s "${errf}" ]]; then
        cat "${errf}" >&2
      fi
      rm -f "${errf}"
      return 0
    }
    st=$?
    if [[ -s "${errf}" ]]; then
      cat "${errf}" >&2
    fi
    if [[ "${st}" -ne 255 || "${attempt}" -ge "${max_attempts}" ]] \
      || ! deploy_ssh_is_retryable_transport "${errf}"; then
      rm -f "${errf}"
      return "${st}"
    fi
    sleep "${delay}"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

deploy_scp_run() {
  local attempt=1
  local max_attempts="${DEPLOY_SSH_MAX_ATTEMPTS:-3}"
  local delay="${DEPLOY_SSH_RETRY_DELAY_SEC:-2}"
  local st=0
  local errf
  errf="$(mktemp "${TMPDIR:-/tmp}/bwb-ssh-err.XXXXXX")"
  deploy_ssh_note_invoke scp
  while true; do
    "$@" 2>"${errf}" && {
      if [[ -s "${errf}" ]]; then
        cat "${errf}" >&2
      fi
      rm -f "${errf}"
      return 0
    }
    st=$?
    if [[ -s "${errf}" ]]; then
      cat "${errf}" >&2
    fi
    if [[ "${st}" -ne 255 || "${attempt}" -ge "${max_attempts}" ]] \
      || ! deploy_ssh_is_retryable_transport "${errf}"; then
      rm -f "${errf}"
      return "${st}"
    fi
    sleep "${delay}"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

deploy_ssh_mux_stop() {
  if [[ ${#SSH_BASE[@]} -eq 0 || -z "${DEPLOY_HOST:-}" || -z "${DEPLOY_USER:-}" ]]; then
    return 0
  fi
  set +e
  "${SSH_BASE[@]}" -O exit "${DEPLOY_USER}@${DEPLOY_HOST}" >/dev/null 2>&1
  set -e
  if [[ -n "${DEPLOY_SSH_CONTROL_PATH:-}" ]]; then
    rm -f "${DEPLOY_SSH_CONTROL_PATH}"
  fi
  return 0
}
