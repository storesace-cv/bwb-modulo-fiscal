#!/usr/bin/env bash
# Closed-operation remote deploy helper. Runs as root via sudoers (D2 bootstrap).
# Never invoked as `sudo bash`. No arbitrary shell snippets.
#
# Usage: bwb-fiscal-deploy-helper <operation> [args...]
#
# Operations:
#   backup-envs <backup-id>
#   install-release <sha40> <upload-dir>
#   install-env fiscal.env|migrate.env <temp-file>
#   activate <sha40>
#   restart
#   migrate <sha40> up|version
#   restore-env <backup-id>
#   cleanup-upload <upload-dir>
#
# Test overrides (never used in production sudoers path):
#   BWB_DEPLOY_OPT, BWB_DEPLOY_ETC, BWB_DEPLOY_UNIT
set -Eeuo pipefail

OPT_ROOT="${BWB_DEPLOY_OPT:-/opt/bwb-modulo-fiscal}"
ETC_ROOT="${BWB_DEPLOY_ETC:-/etc/bwb-modulo-fiscal}"
UNIT_NAME="${BWB_DEPLOY_UNIT:-bwb-fiscal-api.service}"
RELEASES="${OPT_ROOT}/releases"
BACKUPS="${ETC_ROOT}/backups"

die() {
  echo "error: $*" >&2
  exit 1
}

assert_sha1() {
  local name="$1" val="$2"
  [[ "${val}" =~ ^[0-9a-f]{40}$ ]] || die "${name} must be 40-char lowercase hex SHA-1"
}

assert_backup_id() {
  local id="$1"
  # UTC stamp + sha1, e.g. 20260721T143043Z-<40hex>
  [[ "${id}" =~ ^[0-9]{8}T[0-9]{6}Z-[0-9a-f]{40}$ ]] || die "invalid backup-id"
}

# Reject symlinks and require a real path under an expected prefix.
assert_real_under() {
  local label="$1"
  local path="$2"
  local prefix="$3"
  local real
  [[ -e "${path}" ]] || die "${label} missing"
  [[ ! -L "${path}" ]] || die "${label} must not be a symlink"
  if command -v realpath >/dev/null 2>&1; then
    real="$(realpath "${path}")"
  else
    real="$(cd "${path}" 2>/dev/null && pwd -P)" || real="$(python3 -c "import os; print(os.path.realpath('${path}'))")"
  fi
  case "${real}" in
    "${prefix}" | "${prefix}"/*) ;;
    *) die "${label} escapes allowed prefix" ;;
  esac
  printf '%s' "${real}"
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
  [[ "${path}" =~ ^/tmp/bwb-upload\.[A-Za-z0-9._-]+/env\.(fiscal|migrate)\.env\.[0-9]+$ ]] || die "env temp path rejected"
  case "${name}" in
    fiscal.env)
      [[ "${path}" == */env.fiscal.env.* ]] || die "env temp name mismatch"
      ;;
    migrate.env)
      [[ "${path}" == */env.migrate.env.* ]] || die "env temp name mismatch"
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

  [[ -f "${upload}/COMMIT" ]] || die "COMMIT missing in upload"
  [[ "$(tr -d '[:space:]' <"${upload}/COMMIT")" == "${sha}" ]] || die "COMMIT mismatch"
  sha256_check "${upload}"

  install -d -m 0755 -o root -g root "${OPT_ROOT}" "${RELEASES}"
  rm -rf -- "${partial}"
  mkdir -p "${partial}"
  cp -a "${upload}/." "${partial}/"
  chown -R root:root "${partial}"
  chmod 0755 "${partial}/fiscal-api" "${partial}/fiscal-migrate" "${partial}/remote-migrate-run.sh"
  chmod 0644 "${partial}/COMMIT" "${partial}/EXPECTED_SCHEMA_VERSION" "${partial}/SHA256SUMS" \
    "${partial}/lib/allowlist.sh" "${partial}/lib/migrate.env.allowlist"

  if [[ -d "${dest}" ]]; then
    [[ "$(tr -d '[:space:]' <"${dest}/COMMIT")" == "${sha}" ]] || die "existing release COMMIT mismatch"
    rm -rf -- "${partial}"
  else
    mv "${partial}" "${dest}"
  fi
  chown -R root:root "${dest}"
  chmod 0755 "${dest}" "${dest}/fiscal-api" "${dest}/fiscal-migrate" "${dest}/remote-migrate-run.sh"
  chmod 0644 "${dest}/COMMIT" "${dest}/EXPECTED_SCHEMA_VERSION" "${dest}/SHA256SUMS" \
    "${dest}/lib/allowlist.sh" "${dest}/lib/migrate.env.allowlist"
  printf 'install_release_ok sha=%s\n' "${sha}"
}

op_install_env() {
  local name="$1"
  local tmp_arg="$2"
  local tmp
  case "${name}" in
    fiscal.env | migrate.env) ;;
    *) die "invalid env name" ;;
  esac
  tmp="$(assert_env_temp "${tmp_arg}" "${name}")"
  install -d -m 0750 -o root -g root "${ETC_ROOT}"
  install -m 0600 -o root -g root "${tmp}" "${ETC_ROOT}/${name}"
  rm -f -- "${tmp}"
  printf 'install_env_ok name=%s\n' "${name}"
}

op_activate() {
  local sha="$1"
  assert_sha1 "sha" "${sha}"
  local dest="${RELEASES}/${sha}"
  [[ -d "${dest}" ]] || die "release missing"
  [[ ! -L "${dest}" ]] || die "release dir must not be a symlink"
  [[ "$(tr -d '[:space:]' <"${dest}/COMMIT")" == "${sha}" ]] || die "release COMMIT mismatch"
  ln -sfn "${dest}" "${OPT_ROOT}/current.new"
  mv -f "${OPT_ROOT}/current.new" "${OPT_ROOT}/current"
  printf 'activate_ok sha=%s\n' "${sha}"
}

op_restart() {
  systemctl restart "${UNIT_NAME}"
  printf 'restart_ok unit=%s\n' "${UNIT_NAME}"
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
  [[ -x "${release}/remote-migrate-run.sh" ]] || die "remote-migrate-run missing"
  # Helper is already root; runner reads root:root 0600 migrate.env.
  bash "${release}/remote-migrate-run.sh" "${release}" "${cmd}"
}

op_restore_env() {
  local backup_id="$1"
  assert_backup_id "${backup_id}"
  local meta="${BACKUPS}/meta.${backup_id}"
  [[ -f "${meta}" ]] || die "backup meta missing"
  [[ ! -L "${meta}" ]] || die "backup meta must not be a symlink"

  local fiscal_state migrate_state line
  fiscal_state=""
  migrate_state=""
  while IFS= read -r line || [[ -n "${line}" ]]; do
    case "${line}" in
      fiscal.env=present | fiscal.env=absent) fiscal_state="${line#fiscal.env=}" ;;
      migrate.env=present | migrate.env=absent) migrate_state="${line#migrate.env=}" ;;
    esac
  done <"${meta}"
  [[ -n "${fiscal_state}" && -n "${migrate_state}" ]] || die "backup meta incomplete"

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
  printf 'restore_env_ok id=%s\n' "${backup_id}"
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
    [[ $# -eq 2 ]] || die "usage: install-env fiscal.env|migrate.env <temp-file>"
    op_install_env "$1" "$2"
    ;;
  activate)
    [[ $# -eq 1 ]] || die "usage: activate <sha40>"
    op_activate "$1"
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
  *)
    die "unknown operation"
    ;;
esac
