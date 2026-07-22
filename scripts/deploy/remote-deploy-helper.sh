#!/usr/bin/env bash
# Closed-operation remote deploy helper. Runs as root via sudoers (D2 bootstrap).
# Never invoked as `sudo bash`. Never executes release scripts/binaries as root.
#
# Usage: bwb-fiscal-deploy-helper <operation> [args...]
#
# Operations:
#   backup-envs <backup-id>
#   install-release <sha40> <upload-dir>
#   install-env fiscal.env|migrate.env|admin.env <temp-file>
#   activate <sha40>
#   current-sha
#   restart
#   migrate <sha40> up|version
#   restore-env <backup-id>
#   cleanup-upload <upload-dir>
#   admin-scope-create <sha40> <scope-id> <taxpayer-nif> <timezone> <series> <environment>
#   admin-credential-issue <sha40> <scope-id> <created-by> [expires-at]
#   admin-credential-rotate <sha40> <scope-id> <created-by> <grace-until> [expires-at]
#   admin-credential-revoke <sha40> <scope-id> <credential-id> [reason-code]
#   admin-sandbox-e2e <sha40> <case> [base-url] [token-basename]
#   admin-sandbox-measure <sha40> <sustained|burst|replay>
#   admin-sandbox-ab-revoke-gate <sha40> <scope-id> <created-by>
#
# Never installs Nginx open candidate. Never runs fiscal-admin as root.
# activate requires GNU coreutils `mv -T` (Ubuntu 22.04 staging). BSD mv is not supported.
#
# D2 bootstrap also installs:
#   /usr/local/lib/bwb-fiscal-deploy/{allowlist.sh,migrate.env.allowlist,admin.env.allowlist}
#   users bwb-fiscal-migrate and bwb-fiscal-admin (nologin)
set -Eeuo pipefail

# Test overrides are forbidden when running as root.
if [[ "${EUID}" -eq 0 ]]; then
  if [[ -n "${BWB_DEPLOY_OPT:-}" || -n "${BWB_DEPLOY_ETC:-}" || -n "${BWB_DEPLOY_UNIT:-}" \
    || -n "${BWB_MOCK_TMP:-}" || -n "${BWB_HELPER_LIB:-}" || -n "${BWB_MIGRATE_USER:-}" \
    || -n "${BWB_ADMIN_USER:-}" || -n "${BWB_ADMIN_TOKEN_DIR:-}" ]]; then
    echo "error: BWB_* test overrides are forbidden when EUID=0" >&2
    exit 1
  fi
fi

OPT_ROOT="${BWB_DEPLOY_OPT:-/opt/bwb-modulo-fiscal}"
ETC_ROOT="${BWB_DEPLOY_ETC:-/etc/bwb-modulo-fiscal}"
UNIT_NAME="${BWB_DEPLOY_UNIT:-bwb-fiscal-api.service}"
MIGRATE_USER="${BWB_MIGRATE_USER:-bwb-fiscal-migrate}"
ADMIN_USER="${BWB_ADMIN_USER:-bwb-fiscal-admin}"
ADMIN_TOKEN_DIR="${BWB_ADMIN_TOKEN_DIR:-/var/lib/bwb-fiscal-admin/tokens}"
HELPER_LIB="${BWB_HELPER_LIB:-/usr/local/lib/bwb-fiscal-deploy}"
RELEASES="${OPT_ROOT}/releases"
BACKUPS="${ETC_ROOT}/backups"

die() {
  echo "error: $*" >&2
  exit 1
}

# shellcheck source=/dev/null
source "${HELPER_LIB}/allowlist.sh"

assert_sha1() {
  local name="$1" val="$2"
  [[ "${val}" =~ ^[0-9a-f]{40}$ ]] || die "${name} must be 40-char lowercase hex SHA-1"
}

assert_backup_id() {
  local id="$1"
  [[ "${id}" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{40}$ ]] || die "invalid backup-id"
}

assert_upload_dir() {
  local path="$1"
  local real mapped
  [[ "${path}" =~ ^/tmp/bwb-upload\.[A-Za-z0-9._-]+$ ]] || die "upload dir path rejected"
  if [[ -n "${BWB_MOCK_TMP:-}" ]]; then
    mapped="${BWB_MOCK_TMP}/$(basename "${path}")"
    [[ -d "${mapped}" ]] || die "upload dir missing"
    [[ ! -L "${mapped}" ]] || die "upload dir must not be a symlink"
    printf '%s' "${mapped}"
    return 0
  fi
  [[ -d "${path}" ]] || die "upload dir missing"
  [[ ! -L "${path}" ]] || die "upload dir must not be a symlink"
  [[ "$(dirname "${path}")" == "/tmp" ]] || die "upload dir must be directly under /tmp"
  if command -v realpath >/dev/null 2>&1; then
    real="$(realpath "${path}")"
  else
    real="$(cd "${path}" && pwd -P)"
  fi
  case "${real}" in
    /tmp/bwb-upload.*) ;;
    *) die "upload dir realpath rejected" ;;
  esac
  printf '%s' "${path}"
}

assert_env_temp() {
  local path="$1"
  local name="$2"
  local mapped
  [[ "${path}" =~ ^/tmp/bwb-upload\.[A-Za-z0-9._-]+/env\.(fiscal|migrate|admin)\.env\.[0-9]+$ ]] || die "env temp path rejected"
  case "${name}" in
    fiscal.env)
      [[ "${path}" == */env.fiscal.env.* ]] || die "env temp name mismatch"
      ;;
    migrate.env)
      [[ "${path}" == */env.migrate.env.* ]] || die "env temp name mismatch"
      ;;
    admin.env)
      [[ "${path}" == */env.admin.env.* ]] || die "env temp name mismatch"
      ;;
    *) die "invalid env name" ;;
  esac
  if [[ -n "${BWB_MOCK_TMP:-}" ]]; then
    mapped="${BWB_MOCK_TMP}/${path#/tmp/}"
    [[ -f "${mapped}" ]] || die "env temp missing"
    [[ ! -L "${mapped}" ]] || die "env temp must not be a symlink"
    printf '%s' "${mapped}"
    return 0
  fi
  [[ -f "${path}" ]] || die "env temp missing"
  [[ ! -L "${path}" ]] || die "env temp must not be a symlink"
  printf '%s' "${path}"
}

assert_safe_arg() {
  local name="$1" val="$2"
  [[ -n "${val}" ]] || die "${name} required"
  # Bash [[ patterns cannot reliably match NUL; reject CR/LF explicitly.
  # Values are also constrained by the charset regex below (no control chars).
  case "${val}" in
    *$'\n'* | *$'\r'*) die "${name} rejects control chars" ;;
  esac
  [[ "${val}" =~ ^[A-Za-z0-9._:+/@-]+$ ]] || die "${name} charset rejected"
}

assert_token_name() {
  local name="$1"
  [[ "${name}" =~ ^[A-Za-z0-9._-]+$ ]] || die "token name rejected"
}

sha256_check() {
  local dir="$1"
  (
    cd "${dir}"
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c SHA256SUMS >/dev/null
    else
      shasum -a 256 -c SHA256SUMS >/dev/null
    fi
  )
}

# Full release tree validation (never only COMMIT).
verify_release_tree() {
  local dir="$1"
  local sha="$2"
  [[ -d "${dir}" ]] || die "release dir missing"
  [[ ! -L "${dir}" ]] || die "release dir must not be a symlink"
  [[ -f "${dir}/COMMIT" ]] || die "COMMIT missing"
  [[ -f "${dir}/EXPECTED_SCHEMA_VERSION" ]] || die "EXPECTED_SCHEMA_VERSION missing"
  [[ -f "${dir}/SHA256SUMS" ]] || die "SHA256SUMS missing"
  [[ -f "${dir}/fiscal-api" ]] || die "fiscal-api missing"
  [[ -f "${dir}/fiscal-migrate" ]] || die "fiscal-migrate missing"
  [[ -f "${dir}/fiscal-admin" ]] || die "fiscal-admin missing"
  [[ -f "${dir}/fiscal-sandbox-e2e" ]] || die "fiscal-sandbox-e2e missing"
  [[ -f "${dir}/fiscal-sandbox-measure" ]] || die "fiscal-sandbox-measure missing"
  [[ -f "${dir}/lib/allowlist.sh" ]] || die "lib/allowlist.sh missing"
  [[ -f "${dir}/lib/migrate.env.allowlist" ]] || die "lib/migrate.env.allowlist missing"
  [[ -f "${dir}/lib/admin.env.allowlist" ]] || die "lib/admin.env.allowlist missing"
  # Release must not ship an executable runner — migrate is done by this helper.
  [[ ! -e "${dir}/remote-migrate-run.sh" ]] || die "remote-migrate-run.sh must not be in release"
  # Open HTTPS candidate must never ship inside the release tree (cannot be activated).
  [[ ! -e "${dir}/nginx/candidates/bwb-fiscal-sandbox-tls.open.candidate.conf" ]] || die "open candidate must not be in release"
  [[ "$(tr -d '[:space:]' <"${dir}/COMMIT")" == "${sha}" ]] || die "COMMIT mismatch"
  sha256_check "${dir}"
}

op_backup_envs() {
  local backup_id="$1"
  assert_backup_id "${backup_id}"
  install -d -m 0750 -o root -g root "${ETC_ROOT}" "${BACKUPS}"
  local meta="${BACKUPS}/meta.${backup_id}"
  : >"${meta}"
  chmod 0600 "${meta}"
  chown root:root "${meta}"

  if [[ -f "${ETC_ROOT}/fiscal.env" && ! -L "${ETC_ROOT}/fiscal.env" ]]; then
    install -m 0600 -o root -g root "${ETC_ROOT}/fiscal.env" "${BACKUPS}/fiscal.env.${backup_id}"
    printf 'fiscal.env=present\n' >>"${meta}"
  else
    printf 'fiscal.env=absent\n' >>"${meta}"
  fi
  if [[ -f "${ETC_ROOT}/migrate.env" && ! -L "${ETC_ROOT}/migrate.env" ]]; then
    install -m 0600 -o root -g root "${ETC_ROOT}/migrate.env" "${BACKUPS}/migrate.env.${backup_id}"
    printf 'migrate.env=present\n' >>"${meta}"
  else
    printf 'migrate.env=absent\n' >>"${meta}"
  fi
  if [[ -f "${ETC_ROOT}/admin.env" && ! -L "${ETC_ROOT}/admin.env" ]]; then
    install -m 0600 -o root -g root "${ETC_ROOT}/admin.env" "${BACKUPS}/admin.env.${backup_id}"
    printf 'admin.env=present\n' >>"${meta}"
  else
    printf 'admin.env=absent\n' >>"${meta}"
  fi
  printf 'backup_ok id=%s\n' "${backup_id}"
}

op_install_release() {
  local sha="$1"
  local upload="$2"
  local dest partial
  assert_sha1 "sha" "${sha}"
  upload="$(assert_upload_dir "${upload}")"
  dest="${RELEASES}/${sha}"
  partial="${dest}.partial"

  verify_release_tree "${upload}" "${sha}"

  install -d -m 0755 -o root -g root "${OPT_ROOT}" "${RELEASES}"
  rm -rf -- "${partial}"
  mkdir -p "${partial}"
  cp -a "${upload}/." "${partial}/"
  chown -R root:root "${partial}"
  chmod 0755 "${partial}/fiscal-api" "${partial}/fiscal-migrate" "${partial}/fiscal-admin" \
    "${partial}/fiscal-sandbox-e2e" "${partial}/fiscal-sandbox-measure"
  chmod 0644 "${partial}/COMMIT" "${partial}/EXPECTED_SCHEMA_VERSION" "${partial}/SHA256SUMS" \
    "${partial}/lib/allowlist.sh" "${partial}/lib/migrate.env.allowlist" "${partial}/lib/admin.env.allowlist"

  if [[ -d "${dest}" ]]; then
    verify_release_tree "${dest}" "${sha}"
    rm -rf -- "${partial}"
  else
    mv "${partial}" "${dest}"
  fi
  chown -R root:root "${dest}"
  chmod 0755 "${dest}" "${dest}/fiscal-api" "${dest}/fiscal-migrate" "${dest}/fiscal-admin" \
    "${dest}/fiscal-sandbox-e2e" "${dest}/fiscal-sandbox-measure"
  chmod 0644 "${dest}/COMMIT" "${dest}/EXPECTED_SCHEMA_VERSION" "${dest}/SHA256SUMS" \
    "${dest}/lib/allowlist.sh" "${dest}/lib/migrate.env.allowlist" "${dest}/lib/admin.env.allowlist"
  # Drop-priv users need exec on binaries/scripts (not writable).
  chmod 0755 "${dest}/fiscal-migrate" "${dest}/fiscal-api" "${dest}/fiscal-admin" \
    "${dest}/fiscal-sandbox-e2e" "${dest}/fiscal-sandbox-measure"
  printf 'install_release_ok sha=%s\n' "${sha}"
}

op_install_env() {
  local name="$1"
  local tmp_arg="$2"
  local tmp
  case "${name}" in
    fiscal.env | migrate.env | admin.env) ;;
    *) die "invalid env name" ;;
  esac
  tmp="$(assert_env_temp "${tmp_arg}" "${name}")"
  install -d -m 0750 -o root -g root "${ETC_ROOT}"
  # admin.env is root:root 0600 — bwb-fiscal-admin must not read it directly.
  install -m 0600 -o root -g root "${tmp}" "${ETC_ROOT}/${name}"
  rm -f -- "${tmp}"
  printf 'install_env_ok name=%s\n' "${name}"
}

# True when mv supports GNU -T (no-target-directory). Staging hosts are Ubuntu 22.04+.
gnu_mv_supports_T() {
  local probe
  probe="$(mktemp -d "${TMPDIR:-/tmp}/bwb-mvT.XXXXXX")" || return 1
  mkdir -p "${probe}/dir" || {
    rm -rf -- "${probe}"
    return 1
  }
  ln -sfn "${probe}/dir" "${probe}/link"
  ln -sfn "${probe}/dir" "${probe}/link.new"
  if ! mv -Tf "${probe}/link.new" "${probe}/link" 2>/dev/null; then
    rm -rf -- "${probe}"
    return 1
  fi
  if [[ -L "${probe}/link" && ! -e "${probe}/dir/link.new" && ! -e "${probe}/link.new" ]]; then
    rm -rf -- "${probe}"
    return 0
  fi
  rm -rf -- "${probe}"
  return 1
}

# Resolve a symlink to an absolute path. Prefer GNU readlink -f / realpath (Ubuntu).
# Portable fallback is for non-root test harnesses only (e.g. macOS).
resolve_symlink_path() {
  local path="$1"
  local out=""
  local st=0
  set +e
  out="$(readlink -f "${path}" 2>/dev/null)"
  st=$?
  set -e
  if [[ "${st}" -eq 0 && -n "${out}" ]]; then
    printf '%s\n' "${out}"
    return 0
  fi
  if command -v realpath >/dev/null 2>&1; then
    set +e
    out="$(realpath "${path}" 2>/dev/null)"
    st=$?
    set -e
    if [[ "${st}" -eq 0 && -n "${out}" ]]; then
      printf '%s\n' "${out}"
      return 0
    fi
  fi
  set +e
  out="$(readlink "${path}" 2>/dev/null)"
  st=$?
  set -e
  [[ "${st}" -eq 0 && -n "${out}" ]] || return 1
  if [[ "${out}" == /* ]]; then
    printf '%s\n' "${out}"
    return 0
  fi
  printf '%s\n' "$(cd "$(dirname "${path}")" && cd "$(dirname "${out}")" && pwd)/$(basename "${out}")"
}

op_activate() {
  local sha="$1"
  assert_sha1 "sha" "${sha}"
  local dest="${RELEASES}/${sha}"
  verify_release_tree "${dest}" "${sha}"

  # Atomic symlink replace. GNU mv without -T follows an existing symlink-to-directory
  # and moves current.new *into* the old release tree (false activate_ok).
  # Production dependency: GNU coreutils `mv -T` on Ubuntu 22.04+.
  # Non-root test harness (BWB_DEPLOY_OPT) may use ln -sfn when GNU mv is absent —
  # that path must never run as root (overrides are refused when EUID=0).
  # Refuse a real directory at current before any ln/mv (avoids nesting under BSD ln -sfn).
  if [[ -e "${OPT_ROOT}/current" && ! -L "${OPT_ROOT}/current" ]]; then
    die "activate failed: current is not a symlink"
  fi
  if gnu_mv_supports_T; then
    ln -sfn "${dest}" "${OPT_ROOT}/current.new"
    if ! mv -Tf "${OPT_ROOT}/current.new" "${OPT_ROOT}/current"; then
      rm -f -- "${OPT_ROOT}/current.new"
      die "activate failed to replace current symlink"
    fi
  elif [[ -n "${BWB_DEPLOY_OPT:-}" ]]; then
    # Test-only portable replace (does not claim BSD supports mv -T).
    ln -sfn "${dest}" "${OPT_ROOT}/current"
    rm -f -- "${OPT_ROOT}/current.new"
  else
    die "activate requires GNU mv -T (Ubuntu 22.04+ coreutils); BSD mv is not supported"
  fi
  # Fail closed if current does not resolve to the requested release.
  # Compare by SHA/COMMIT (not raw paths): macOS may prefix /private on resolved paths.
  local active=""
  set +e
  active="$(resolve_symlink_path "${OPT_ROOT}/current")"
  local resolve_st=$?
  set -e
  [[ "${resolve_st}" -eq 0 && -n "${active}" && -d "${active}" ]] || die "activate did not switch current to ${sha}"
  local active_sha
  active_sha="$(basename "${active}")"
  [[ "${active_sha}" == "${sha}" ]] || die "activate did not switch current to ${sha}"
  [[ -f "${active}/COMMIT" ]] || die "activate target missing COMMIT"
  local committed
  committed="$(tr -d '[:space:]' <"${active}/COMMIT")"
  [[ "${committed}" == "${sha}" ]] || die "activate COMMIT mismatch"
  # Ensure the two-step did not leave junk inside any release tree.
  [[ ! -e "${active}/current.new" ]] || die "activate left current.new inside release tree"
  printf 'activate_ok sha=%s\n' "${sha}"
}

# Closed read of the active release SHA (basename of current). No arbitrary shell on the host.
op_current_sha() {
  local cur="${OPT_ROOT}/current"
  [[ -L "${cur}" ]] || die "current is missing or not a symlink"
  local active=""
  set +e
  active="$(resolve_symlink_path "${cur}")"
  local resolve_st=$?
  set -e
  [[ "${resolve_st}" -eq 0 && -n "${active}" && -d "${active}" ]] || die "current does not resolve"
  local sha
  sha="$(basename "${active}")"
  assert_sha1 "current-sha" "${sha}"
  [[ -f "${active}/COMMIT" ]] || die "active COMMIT missing"
  local committed
  committed="$(tr -d '[:space:]' <"${active}/COMMIT")"
  [[ "${committed}" == "${sha}" ]] || die "active COMMIT mismatch"
  printf 'current_sha=%s\n' "${sha}"
}

op_restart() {
  systemctl restart "${UNIT_NAME}"
  printf 'restart_ok unit=%s\n' "${UNIT_NAME}"
}

# Read migrate.env as root; execute fiscal-migrate only after dropping privileges.
run_fiscal_migrate_dropped() {
  local bin="$1"
  local cmd="$2"
  local driver="$3"
  local url="$4"

  [[ -f "${bin}" && -x "${bin}" ]] || die "fiscal-migrate missing or not executable"
  [[ ! -L "${bin}" ]] || die "fiscal-migrate must not be a symlink"

  if [[ "${EUID}" -ne 0 ]]; then
    # Non-privileged test mode only.
    env -i \
      PATH="/usr/bin:/bin:/usr/local/bin" \
      FISCAL_DATABASE_DRIVER="${driver}" \
      FISCAL_DATABASE_URL="${url}" \
      "${bin}" "${cmd}"
    return $?
  fi

  if command -v runuser >/dev/null 2>&1; then
    runuser -u "${MIGRATE_USER}" -- env -i \
      PATH="/usr/bin:/bin" \
      HOME="/nonexistent" \
      FISCAL_DATABASE_DRIVER="${driver}" \
      FISCAL_DATABASE_URL="${url}" \
      "${bin}" "${cmd}"
    return $?
  fi
  if command -v setpriv >/dev/null 2>&1; then
    setpriv --reuid="${MIGRATE_USER}" --regid="${MIGRATE_USER}" --init-groups --reset-env \
      env -i \
        PATH="/usr/bin:/bin" \
        HOME="/nonexistent" \
        FISCAL_DATABASE_DRIVER="${driver}" \
        FISCAL_DATABASE_URL="${url}" \
        "${bin}" "${cmd}"
    return $?
  fi
  die "runuser or setpriv required to drop privileges for migrate"
}

op_migrate() {
  local sha="$1"
  local cmd="$2"
  assert_sha1 "sha" "${sha}"
  case "${cmd}" in
    up | version) ;;
    *) die "invalid migrate command" ;;
  esac

  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"

  local env_file="${ETC_ROOT}/migrate.env"
  [[ -f "${env_file}" ]] || die "migrate.env missing"
  [[ ! -L "${env_file}" ]] || die "migrate.env must not be a symlink"

  local allowlist="${HELPER_LIB}/migrate.env.allowlist"
  [[ -f "${allowlist}" ]] || die "helper migrate allowlist missing"
  deploy_validate_exact_allowlisted_file "${allowlist}" "${env_file}"

  local driver url
  driver="$(deploy_read_env_value "${env_file}" FISCAL_DATABASE_DRIVER)"
  url="$(deploy_read_env_value "${env_file}" FISCAL_DATABASE_URL)"

  # Never bash/source release scripts. Only the fiscal-migrate binary, dropped.
  run_fiscal_migrate_dropped "${release}/fiscal-migrate" "${cmd}" "${driver}" "${url}"
}

op_restore_env() {
  local backup_id="$1"
  assert_backup_id "${backup_id}"
  local meta="${BACKUPS}/meta.${backup_id}"
  [[ -f "${meta}" ]] || die "backup meta missing"
  [[ ! -L "${meta}" ]] || die "backup meta must not be a symlink"

  local fiscal_state migrate_state admin_state line
  fiscal_state=""
  migrate_state=""
  admin_state=""
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in
      fiscal.env=present | fiscal.env=absent) fiscal_state="${line#fiscal.env=}" ;;
      migrate.env=present | migrate.env=absent) migrate_state="${line#migrate.env=}" ;;
      admin.env=present | admin.env=absent) admin_state="${line#admin.env=}" ;;
    esac
  done <"${meta}"
  [[ -n "${fiscal_state}" && -n "${migrate_state}" ]] || die "backup meta incomplete"
  # Older backups may omit admin.env; treat missing key as absent.
  [[ -n "${admin_state}" ]] || admin_state="absent"

  if [[ "${fiscal_state}" == "present" ]]; then
    [[ -f "${BACKUPS}/fiscal.env.${backup_id}" ]] || die "fiscal backup missing"
    install -m 0600 -o root -g root "${BACKUPS}/fiscal.env.${backup_id}" "${ETC_ROOT}/fiscal.env"
  else
    rm -f -- "${ETC_ROOT}/fiscal.env"
  fi
  if [[ "${migrate_state}" == "present" ]]; then
    [[ -f "${BACKUPS}/migrate.env.${backup_id}" ]] || die "migrate backup missing"
    install -m 0600 -o root -g root "${BACKUPS}/migrate.env.${backup_id}" "${ETC_ROOT}/migrate.env"
  else
    rm -f -- "${ETC_ROOT}/migrate.env"
  fi
  if [[ "${admin_state}" == "present" ]]; then
    [[ -f "${BACKUPS}/admin.env.${backup_id}" ]] || die "admin backup missing"
    install -m 0600 -o root -g root "${BACKUPS}/admin.env.${backup_id}" "${ETC_ROOT}/admin.env"
  else
    rm -f -- "${ETC_ROOT}/admin.env"
  fi
  printf 'restore_env_ok id=%s\n' "${backup_id}"
}

ensure_admin_token_dir() {
  local parent
  parent="$(dirname "${ADMIN_TOKEN_DIR}")"
  [[ "${parent}" != "/" && -n "${parent}" ]] || die "invalid admin token dir"
  if [[ "${EUID}" -eq 0 ]]; then
    install -d -m 0700 -o "${ADMIN_USER}" -g "${ADMIN_USER}" "${parent}" "${ADMIN_TOKEN_DIR}"
    chown -R "${ADMIN_USER}:${ADMIN_USER}" "${parent}"
    chmod 0700 "${parent}" "${ADMIN_TOKEN_DIR}"
  else
    install -d -m 0700 "${parent}" "${ADMIN_TOKEN_DIR}"
  fi
}

choose_token_path() {
  local label="$1"
  assert_token_name "${label}"
  ensure_admin_token_dir
  local path="${ADMIN_TOKEN_DIR}/${label}.token"
  [[ ! -e "${path}" ]] || die "token path already exists"
  printf '%s' "${path}"
}

read_admin_env_pair() {
  local env_file="${ETC_ROOT}/admin.env"
  local allowlist="${HELPER_LIB}/admin.env.allowlist"
  [[ -f "${env_file}" ]] || die "admin.env missing"
  [[ ! -L "${env_file}" ]] || die "admin.env must not be a symlink"
  [[ -f "${allowlist}" ]] || die "helper admin allowlist missing"
  deploy_validate_exact_allowlisted_file "${allowlist}" "${env_file}"
  ADMIN_DRIVER="$(deploy_read_env_value "${env_file}" FISCAL_DATABASE_DRIVER)"
  ADMIN_URL="$(deploy_read_env_value "${env_file}" FISCAL_DATABASE_URL)"
  [[ -n "${ADMIN_DRIVER}" && -n "${ADMIN_URL}" ]] || die "admin.env incomplete"
}

# Root reads admin.env; child gets env -i with only DRIVER/URL. Never log DSN.
run_admin_dropped() {
  local -a cmd=("$@")
  [[ "${#cmd[@]}" -ge 1 ]] || die "admin command required"
  [[ -f "${cmd[0]}" && -x "${cmd[0]}" ]] || die "admin binary missing or not executable"
  [[ ! -L "${cmd[0]}" ]] || die "admin binary must not be a symlink"

  read_admin_env_pair

  if [[ "${EUID}" -ne 0 ]]; then
    env -i \
      PATH="/usr/bin:/bin:/usr/local/bin" \
      HOME="/nonexistent" \
      FISCAL_DATABASE_DRIVER="${ADMIN_DRIVER}" \
      FISCAL_DATABASE_URL="${ADMIN_URL}" \
      "${cmd[@]}"
    return $?
  fi

  if command -v runuser >/dev/null 2>&1; then
    runuser -u "${ADMIN_USER}" -- env -i \
      PATH="/usr/bin:/bin" \
      HOME="/nonexistent" \
      FISCAL_DATABASE_DRIVER="${ADMIN_DRIVER}" \
      FISCAL_DATABASE_URL="${ADMIN_URL}" \
      "${cmd[@]}"
    return $?
  fi
  if command -v setpriv >/dev/null 2>&1; then
    setpriv --reuid="${ADMIN_USER}" --regid="${ADMIN_USER}" --init-groups --reset-env \
      env -i \
        PATH="/usr/bin:/bin" \
        HOME="/nonexistent" \
        FISCAL_DATABASE_DRIVER="${ADMIN_DRIVER}" \
        FISCAL_DATABASE_URL="${ADMIN_URL}" \
        "${cmd[@]}"
    return $?
  fi
  die "runuser or setpriv required to drop privileges for admin"
}

op_admin_scope_create() {
  local sha="$1" scope_id="$2" nif="$3" tz="$4" series="$5" envn="$6"
  assert_sha1 "sha" "${sha}"
  assert_safe_arg "scope_id" "${scope_id}"
  assert_safe_arg "taxpayer_nif" "${nif}"
  assert_safe_arg "timezone" "${tz}"
  assert_safe_arg "series" "${series}"
  assert_safe_arg "environment" "${envn}"
  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"
  run_admin_dropped "${release}/fiscal-admin" scope create \
    --scope-id "${scope_id}" \
    --taxpayer-nif "${nif}" \
    --timezone "${tz}" \
    --series "${series}" \
    --environment "${envn}"
  printf 'admin_scope_create_ok scope_id=%s\n' "${scope_id}"
}

op_admin_credential_issue() {
  local sha="$1" scope_id="$2" created_by="$3"
  local expires_at="${4:-}"
  assert_sha1 "sha" "${sha}"
  assert_safe_arg "scope_id" "${scope_id}"
  assert_safe_arg "created_by" "${created_by}"
  [[ -z "${expires_at}" ]] || assert_safe_arg "expires_at" "${expires_at}"
  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"
  local token_path
  token_path="$(choose_token_path "issue-${scope_id}-$(date -u +%Y%m%dT%H%M%SZ)")"
  local -a args=(
    "${release}/fiscal-admin" credential issue
    --scope-id "${scope_id}"
    --created-by "${created_by}"
    --output-file "${token_path}"
  )
  if [[ -n "${expires_at}" ]]; then
    args+=(--expires-at "${expires_at}")
  fi
  run_admin_dropped "${args[@]}"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${ADMIN_USER}:${ADMIN_USER}" "${token_path}"
    chmod 0600 "${token_path}"
  else
    chmod 0600 "${token_path}"
  fi
  install -m 0600 "${token_path}" "${ADMIN_TOKEN_DIR}/current.token"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${ADMIN_USER}:${ADMIN_USER}" "${ADMIN_TOKEN_DIR}/current.token"
  fi
  printf 'admin_credential_issue_ok scope_id=%s token_path=%s\n' "${scope_id}" "${token_path}"
}

op_admin_credential_rotate() {
  local sha="$1" scope_id="$2" created_by="$3" grace_until="$4"
  local expires_at="${5:-}"
  assert_sha1 "sha" "${sha}"
  assert_safe_arg "scope_id" "${scope_id}"
  assert_safe_arg "created_by" "${created_by}"
  assert_safe_arg "grace_until" "${grace_until}"
  [[ -z "${expires_at}" ]] || assert_safe_arg "expires_at" "${expires_at}"
  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"
  local token_path
  token_path="$(choose_token_path "rotate-${scope_id}-$(date -u +%Y%m%dT%H%M%SZ)")"
  local -a args=(
    "${release}/fiscal-admin" credential rotate
    --scope-id "${scope_id}"
    --created-by "${created_by}"
    --grace-until "${grace_until}"
    --output-file "${token_path}"
  )
  if [[ -n "${expires_at}" ]]; then
    args+=(--expires-at "${expires_at}")
  fi
  run_admin_dropped "${args[@]}"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${ADMIN_USER}:${ADMIN_USER}" "${token_path}"
    chmod 0600 "${token_path}"
  else
    chmod 0600 "${token_path}"
  fi
  install -m 0600 "${token_path}" "${ADMIN_TOKEN_DIR}/current.token"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${ADMIN_USER}:${ADMIN_USER}" "${ADMIN_TOKEN_DIR}/current.token"
  fi
  printf 'admin_credential_rotate_ok scope_id=%s token_path=%s\n' "${scope_id}" "${token_path}"
}

op_admin_credential_revoke() {
  local sha="$1" scope_id="$2" cred_id="$3"
  local reason="${4:-}"
  assert_sha1 "sha" "${sha}"
  assert_safe_arg "scope_id" "${scope_id}"
  assert_safe_arg "credential_id" "${cred_id}"
  [[ -z "${reason}" ]] || assert_safe_arg "reason_code" "${reason}"
  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"
  local -a args=(
    "${release}/fiscal-admin" credential revoke
    --scope-id "${scope_id}"
    --credential-id "${cred_id}"
  )
  if [[ -n "${reason}" ]]; then
    args+=(--reason-code "${reason}")
  fi
  run_admin_dropped "${args[@]}"
  printf 'admin_credential_revoke_ok scope_id=%s credential_id=%s\n' "${scope_id}" "${cred_id}"
}

op_admin_sandbox_e2e() {
  local sha="$1" case_name="$2"
  local base_url="${3:-http://127.0.0.1:8080}"
  local token_base="${4:-current.token}"
  assert_sha1 "sha" "${sha}"
  assert_safe_arg "case" "${case_name}"
  case "${base_url}" in
    http://127.0.0.1:8080 | http://127.0.0.1:18080 | https://sandbox.fiscalmod.bwb.pt) ;;
    *) die "base-url not allowlisted" ;;
  esac
  assert_token_name "${token_base%.token}"
  [[ "${token_base}" =~ ^[A-Za-z0-9._-]+$ ]] || die "token basename rejected"
  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"
  local token_path="${ADMIN_TOKEN_DIR}/${token_base}"
  [[ -f "${token_path}" && ! -L "${token_path}" ]] || die "token missing"
  run_admin_dropped "${release}/fiscal-sandbox-e2e" \
    --base-url "${base_url}" \
    --token-file "${token_path}" \
    --fixture-dir "${release}/fixtures/sandbox" \
    --case "${case_name}"
}

# A→B revoke gate: A usable → revoke A → A 401 → issue B → 201 + real replay.
# Tokens never appear in argv/stdout/stderr/logs; only credential_id/scope_id/status.
op_admin_sandbox_ab_revoke_gate() {
  local sha="$1" scope_id="$2" created_by="$3"
  assert_sha1 "sha" "${sha}"
  assert_safe_arg "scope_id" "${scope_id}"
  assert_safe_arg "created_by" "${created_by}"
  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"

  local token_a cred_a issue_out
  token_a="$(choose_token_path "gate-a-${scope_id}-$(date -u +%Y%m%dT%H%M%SZ)")"
  issue_out="$(
    run_admin_dropped "${release}/fiscal-admin" credential issue \
      --scope-id "${scope_id}" \
      --created-by "${created_by}" \
      --output-file "${token_a}"
  )"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${ADMIN_USER}:${ADMIN_USER}" "${token_a}"
    chmod 0600 "${token_a}"
  else
    chmod 0600 "${token_a}"
  fi
  [[ "${issue_out}" != *postgres://* ]] || die "DSN leak in issue output"
  [[ "${issue_out}" != *bwb_sbox_* ]] || die "token leak in issue output"
  cred_a="$(printf '%s\n' "${issue_out}" | sed -n 's/^credential_id=\([^ ]*\).*/\1/p' | head -1)"
  assert_safe_arg "credential_id" "${cred_a}"

  # A must succeed before revoke (fixture/key A).
  local a_out doc_a
  a_out="$(
    run_admin_dropped "${release}/fiscal-sandbox-e2e" \
      --base-url "http://127.0.0.1:8080" \
      --token-file "${token_a}" \
      --fixture-dir "${release}/fixtures/sandbox" \
      --case "create_201"
  )"
  [[ "${a_out}" != *postgres://* && "${a_out}" != *bwb_sbox_* ]] || die "secret leak in A e2e"
  doc_a="$(printf '%s\n' "${a_out}" | sed -n 's/.*document_id=\([A-Za-z0-9._-]*\).*/\1/p' | head -1)"
  assert_safe_arg "document_id_a" "${doc_a}"
  printf 'ab_gate_a_usable=ok credential_id=%s document_id=%s\n' "${cred_a}" "${doc_a}"

  run_admin_dropped "${release}/fiscal-admin" credential revoke \
    --scope-id "${scope_id}" \
    --credential-id "${cred_a}"
  printf 'ab_gate_a_revoked=ok credential_id=%s\n' "${cred_a}"

  # Same token A file must now yield 401 (real revoked credential, not artificial bad token).
  run_admin_dropped "${release}/fiscal-sandbox-e2e" \
    --base-url "http://127.0.0.1:8080" \
    --token-file "${token_a}" \
    --fixture-dir "${release}/fixtures/sandbox" \
    --case "token_revoked_401"
  printf 'ab_gate_a_rejected=ok credential_id=%s\n' "${cred_a}"

  local token_b
  token_b="$(choose_token_path "gate-b-${scope_id}-$(date -u +%Y%m%dT%H%M%SZ)")"
  issue_out="$(
    run_admin_dropped "${release}/fiscal-admin" credential issue \
      --scope-id "${scope_id}" \
      --created-by "${created_by}" \
      --output-file "${token_b}"
  )"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${ADMIN_USER}:${ADMIN_USER}" "${token_b}"
    chmod 0600 "${token_b}"
  else
    chmod 0600 "${token_b}"
  fi
  [[ "${issue_out}" != *postgres://* && "${issue_out}" != *bwb_sbox_* ]] || die "secret leak in B issue"
  install -m 0600 "${token_b}" "${ADMIN_TOKEN_DIR}/current.token"
  if [[ "${EUID}" -eq 0 ]]; then
    chown "${ADMIN_USER}:${ADMIN_USER}" "${ADMIN_TOKEN_DIR}/current.token"
  fi

  # B creates a NEW document (fixture/key B) then proves idempotent replay of that same request.
  local b_out doc_b
  b_out="$(
    run_admin_dropped "${release}/fiscal-sandbox-e2e" \
      --base-url "http://127.0.0.1:8080" \
      --token-file "${token_b}" \
      --fixture-dir "${release}/fixtures/sandbox" \
      --case "create_replay"
  )"
  [[ "${b_out}" != *postgres://* && "${b_out}" != *bwb_sbox_* ]] || die "secret leak in B e2e"
  doc_b="$(printf '%s\n' "${b_out}" | sed -n 's/.*document_id=\([A-Za-z0-9._-]*\).*/\1/p' | head -1)"
  assert_safe_arg "document_id_b" "${doc_b}"
  [[ "${doc_a}" != "${doc_b}" ]] || die "A and B document_id must differ"
  printf 'ab_gate_b_replay=ok scope_id=%s document_id=%s\n' "${scope_id}" "${doc_b}"
  printf 'ab_gate_docs_distinct=ok\n'
}

op_admin_sandbox_measure() {
  local sha="$1"
  local profile="$2"
  assert_sha1 "sha" "${sha}"
  case "${profile}" in
    sustained | burst | replay) ;;
    *) die "usage: admin-sandbox-measure <sha40> sustained|burst|replay" ;;
  esac
  local release="${RELEASES}/${sha}"
  verify_release_tree "${release}" "${sha}"
  local token_path="${ADMIN_TOKEN_DIR}/measure.token"
  [[ -f "${token_path}" && ! -L "${token_path}" ]] || die "measure token missing"
  # Closed profile only: URL, token path, rate/concurrency are fixed inside the binary.
  run_admin_dropped "${release}/fiscal-sandbox-measure" --profile "${profile}"
}

op_cleanup_upload() {
  local upload="$1"
  upload="$(assert_upload_dir "${upload}")"
  rm -rf -- "${upload}"
  printf 'cleanup_upload_ok\n'
}

OP="${1:-}"
if [[ $# -gt 0 ]]; then
  shift
fi

case "${OP}" in
  backup-envs)
    [[ $# -eq 1 ]] || die "usage: backup-envs <backup-id>"
    op_backup_envs "$1"
    ;;
  install-release)
    [[ $# -eq 2 ]] || die "usage: install-release <sha40> <upload-dir>"
    op_install_release "$1" "$2"
    ;;
  install-env)
    [[ $# -eq 2 ]] || die "usage: install-env fiscal.env|migrate.env|admin.env <temp-file>"
    op_install_env "$1" "$2"
    ;;
  activate)
    [[ $# -eq 1 ]] || die "usage: activate <sha40>"
    op_activate "$1"
    ;;
  current-sha)
    [[ $# -eq 0 ]] || die "usage: current-sha"
    op_current_sha
    ;;
  restart)
    [[ $# -eq 0 ]] || die "usage: restart"
    op_restart
    ;;
  migrate)
    [[ $# -eq 2 ]] || die "usage: migrate <sha40> up|version"
    op_migrate "$1" "$2"
    ;;
  restore-env)
    [[ $# -eq 1 ]] || die "usage: restore-env <backup-id>"
    op_restore_env "$1"
    ;;
  cleanup-upload)
    [[ $# -eq 1 ]] || die "usage: cleanup-upload <upload-dir>"
    op_cleanup_upload "$1"
    ;;
  admin-scope-create)
    [[ $# -eq 6 ]] || die "usage: admin-scope-create <sha40> <scope-id> <nif> <tz> <series> <env>"
    op_admin_scope_create "$1" "$2" "$3" "$4" "$5" "$6"
    ;;
  admin-credential-issue)
    [[ $# -eq 3 || $# -eq 4 ]] || die "usage: admin-credential-issue <sha40> <scope-id> <created-by> [expires-at]"
    op_admin_credential_issue "$1" "$2" "$3" "${4:-}"
    ;;
  admin-credential-rotate)
    [[ $# -eq 4 || $# -eq 5 ]] || die "usage: admin-credential-rotate <sha40> <scope-id> <created-by> <grace-until> [expires-at]"
    op_admin_credential_rotate "$1" "$2" "$3" "$4" "${5:-}"
    ;;
  admin-credential-revoke)
    [[ $# -eq 3 || $# -eq 4 ]] || die "usage: admin-credential-revoke <sha40> <scope-id> <credential-id> [reason]"
    op_admin_credential_revoke "$1" "$2" "$3" "${4:-}"
    ;;
  admin-sandbox-e2e)
    [[ $# -eq 2 || $# -eq 3 || $# -eq 4 ]] || die "usage: admin-sandbox-e2e <sha40> <case> [base-url] [token-basename]"
    op_admin_sandbox_e2e "$1" "$2" "${3:-http://127.0.0.1:8080}" "${4:-current.token}"
    ;;
  admin-sandbox-measure)
    [[ $# -eq 2 ]] || die "usage: admin-sandbox-measure <sha40> sustained|burst|replay"
    op_admin_sandbox_measure "$1" "$2"
    ;;
  admin-sandbox-ab-revoke-gate)
    [[ $# -eq 3 ]] || die "usage: admin-sandbox-ab-revoke-gate <sha40> <scope-id> <created-by>"
    op_admin_sandbox_ab_revoke_gate "$1" "$2" "$3"
    ;;
  install-nginx-open | activate-open-candidate | nginx-open)
    die "open HTTPS candidate cannot be activated by this helper"
    ;;
  *)
    die "unknown operation"
    ;;
esac
