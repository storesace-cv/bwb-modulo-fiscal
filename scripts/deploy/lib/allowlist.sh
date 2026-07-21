#!/usr/bin/env bash
# Shared helpers for staging deploy scripts. Never print env values.
# Compatible with Bash 3.2+ (macOS) and Bash 4+ (CI).
# shellcheck shell=bash

set -Eeuo pipefail

deploy_repo_root() {
  local here
  here="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
  printf '%s' "${here}"
}

deploy_trim() {
  # Trim leading/trailing whitespace from stdin; preserve interior content.
  sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
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

# True if line (after leading spaces) is empty or a full-line comment.
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

# Validate file against allowlist without exporting (for tests / dry checks).
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

# Load KEY=VALUE into the current shell after allowlist validation.
# Comment lines: only when the line (after leading spaces) starts with #.
# Value: everything after the first '=' (including # $ " ' = spaces).
# Never uses eval on values.
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

# Read a single required key from an env file without sourcing.
# Prints only the value to stdout (callers must not log it).
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

deploy_ssh_base() {
  # Expand ~
  DEPLOY_SSH_KEY="${DEPLOY_SSH_KEY/#\~/${HOME}}"
  DEPLOY_KNOWN_HOSTS="${DEPLOY_KNOWN_HOSTS/#\~/${HOME}}"
  deploy_require_cmds ssh scp
  # Populated for callers that source this library (update-staging / migrate-remote).
  # shellcheck disable=SC2034
  SSH_BASE=(
    ssh
    -i "${DEPLOY_SSH_KEY}"
    -o IdentitiesOnly=yes
    -o StrictHostKeyChecking=yes
    -o UserKnownHostsFile="${DEPLOY_KNOWN_HOSTS}"
    -o BatchMode=yes
  )
  # shellcheck disable=SC2034
  SCP_BASE=(
    scp
    -i "${DEPLOY_SSH_KEY}"
    -o IdentitiesOnly=yes
    -o StrictHostKeyChecking=yes
    -o UserKnownHostsFile="${DEPLOY_KNOWN_HOSTS}"
    -o BatchMode=yes
  )
}
