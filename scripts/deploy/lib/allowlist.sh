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

deploy_allowlist_has() {
  local allowlist_file="$1"
  local key="$2"
  grep -Exq "${key}" "${allowlist_file}"
}

# Validate file against allowlist without exporting (for tests / dry checks).
deploy_validate_allowlisted_file() {
  local allowlist_file="$1"
  local env_file="$2"
  local line key raw lineno=0
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
    line="${raw%%#*}"
    # trim
    line="$(printf '%s' "${line}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    [[ -z "${line}" ]] && continue
    case "${line}" in
      *=*) ;;
      *)
        rm -f "${seen_file}"
        echo "error: malformed env line ${lineno}" >&2
        return 1
        ;;
    esac
    key="${line%%=*}"
    case "${key}" in
      '' | *[!A-Za-z0-9_]* | [0-9]*)
        rm -f "${seen_file}"
        echo "error: malformed key on line ${lineno}" >&2
        return 1
        ;;
    esac
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

# Load KEY=VALUE file into the current shell after allowlist validation.
deploy_load_allowlisted_env() {
  local allowlist_file="$1"
  local env_file="$2"
  local line key value raw lineno=0

  deploy_validate_allowlisted_file "${allowlist_file}" "${env_file}" || return 1

  while IFS= read -r raw || [[ -n "${raw}" ]]; do
    lineno=$((lineno + 1))
    line="${raw%%#*}"
    line="$(printf '%s' "${line}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    [[ -z "${line}" ]] && continue
    key="${line%%=*}"
    value="${line#*=}"
    # Assign without echoing value.
    eval "${key}=\"\${value}\""
    eval "export ${key}"
  done <"${env_file}"
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

# Parse "version=N dirty=true|false" style output; never logs DSN.
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
