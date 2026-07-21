#!/usr/bin/env bash
# Deploy D1 self-tests (no real SSH/network; no secret values printed).
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"
# shellcheck source=../../scripts/deploy/lib/allowlist.sh
# shellcheck disable=SC1091
source "${ROOT}/scripts/deploy/lib/allowlist.sh"

TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

pass=0
fail=0

ok() {
  echo "PASS: $*"
  pass=$((pass + 1))
}

bad() {
  echo "FAIL: $*" >&2
  fail=$((fail + 1))
}

chmod_restrict() {
  chmod 0600 "$1"
}

# --- allowlist validation ---
cat >"${TMP}/good.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://runtime:secret-should-not-leak@127.0.0.1:5432/fiscal?sslmode=require
EOF
chmod_restrict "${TMP}/good.env"

if deploy_validate_exact_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/good.env"; then
  ok "exact allowlist accepts valid migrate env"
else
  bad "exact allowlist rejected valid migrate env"
fi

cat >"${TMP}/unknown.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://x
EVIL_KEY=1
EOF
if deploy_validate_exact_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/unknown.env" 2>"${TMP}/err"; then
  bad "allowlist should reject unknown key"
else
  if grep -q 'EVIL_KEY' "${TMP}/err" && ! grep -q 'secret' "${TMP}/err"; then
    ok "allowlist rejects unknown key without leaking secrets"
  else
    bad "unknown key error mishandled"
  fi
fi

cat >"${TMP}/dup.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://x
FISCAL_DATABASE_DRIVER=sqlite
EOF
if deploy_validate_exact_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/dup.env" 2>/dev/null; then
  bad "allowlist should reject duplicate"
else
  ok "allowlist rejects duplicate"
fi

cat >"${TMP}/incomplete.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
EOF
if deploy_validate_exact_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/incomplete.env" 2>/dev/null; then
  bad "exact allowlist should require all keys"
else
  ok "exact allowlist rejects missing keys"
fi

# --- env parser specials / no-exec ---
cat >"${TMP}/special.env" <<'EOF'
# full line comment only
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:p@h/db?x=1#frag and spaces $HOME "q" 's' a=b
EOF
FISCAL_DATABASE_DRIVER=""
FISCAL_DATABASE_URL=""
deploy_load_allowlisted_env "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/special.env"
expected_url="$(deploy_read_env_value "${TMP}/special.env" FISCAL_DATABASE_URL)"
# shellcheck disable=SC2016
if [[ "${FISCAL_DATABASE_DRIVER}" == "postgres" \
  && "${FISCAL_DATABASE_URL}" == "${expected_url}" \
  && "${FISCAL_DATABASE_URL}" == *'#frag'* \
  && "${FISCAL_DATABASE_URL}" == *'$HOME'* \
  && "${FISCAL_DATABASE_URL}" == *'a=b'* ]]; then
  ok "parser preserves special characters in values"
else
  bad "parser corrupted special values"
fi

PWNED="${TMP}/pwned"
rm -f "${PWNED}"
cat >"${TMP}/exec.env" <<EOF
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=\$(touch ${PWNED})
EOF
FISCAL_DATABASE_URL=""
deploy_load_allowlisted_env "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/exec.env"
if [[ -e "${PWNED}" ]]; then
  bad "env value was executed"
else
  ok "env value not executed"
fi

# --- SHA-1 validation ---
if deploy_assert_sha1 "x" "abc" 2>/dev/null; then
  bad "short sha should fail"
else
  ok "rejects non-40-char sha"
fi
if deploy_assert_sha1 "x" "$(printf 'a%.0s' {1..40})"; then
  ok "accepts 40-char hex sha"
else
  bad "valid sha rejected"
fi

# --- build + checksums ---
HEAD="$(git rev-parse HEAD)"
export EXPECTED_COMMIT="${HEAD}"
export DEPLOY_GOARCH=amd64
export DEPLOY_TEST_OUT_ROOT="${TMP}"
export OUT_DIR="${TMP}/release"
export EXPECTED_SCHEMA_VERSION=2
if bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/build.out" 2>"${TMP}/build.err"; then
  if deploy_verify_release_manifest "${OUT_DIR}" "${HEAD}"; then
    ok "release manifest verifies"
  else
    bad "release manifest verify failed"
  fi
  if grep -q 'remote-migrate-run.sh' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'lib/allowlist.sh' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'lib/migrate.env.allowlist' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'EXPECTED_SCHEMA_VERSION' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'COMMIT' "${OUT_DIR}/SHA256SUMS"; then
    ok "SHA256SUMS covers helpers and schema metadata"
  else
    bad "SHA256SUMS incomplete"
  fi
  if [[ "$(tr -d '[:space:]' <"${OUT_DIR}/EXPECTED_SCHEMA_VERSION")" == "2" ]]; then
    ok "EXPECTED_SCHEMA_VERSION metadata present"
  else
    bad "EXPECTED_SCHEMA_VERSION missing"
  fi
else
  bad "build-linux-release failed"
  cat "${TMP}/build.err" >&2 || true
fi

# Corrupt helper checksum must fail verify
cp -a "${OUT_DIR}" "${TMP}/corrupt-rel"
echo 'tamper' >>"${TMP}/corrupt-rel/lib/allowlist.sh"
if ( cd "${TMP}/corrupt-rel" && deploy_sha256_check SHA256SUMS ) 2>"${TMP}/corrupt.err"; then
  bad "corrupt helper should fail checksum"
  cat "${TMP}/corrupt.err" >&2 || true
else
  ok "invalid helper checksum rejected"
fi

# Reject OUT_DIR outside allowlist
if EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 DEPLOY_TEST_OUT_ROOT="${TMP}" \
  OUT_DIR="/tmp/bwb-forbidden-out-$$" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/forbid.out" 2>"${TMP}/forbid.err"; then
  bad "should reject OUT_DIR outside dist/releases"
  rm -rf "/tmp/bwb-forbidden-out-$$"
else
  ok "rejects OUT_DIR outside authorized roots"
fi

# Dirty worktree refusal without test root
DIRTY_MARKER="${ROOT}/.deploy-dirty-test-$$"
touch "${DIRTY_MARKER}"
if env -u DEPLOY_TEST_OUT_ROOT -u DEPLOY_ALLOW_DIRTY_WORKTREE \
  EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 \
  OUT_DIR="${ROOT}/dist/releases/${HEAD}-dirtytest" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/dirtyb.out" 2>"${TMP}/dirtyb.err"; then
  bad "should refuse dirty worktree on real path"
  rm -rf "${ROOT}/dist/releases/${HEAD}-dirtytest"
else
  ok "refuses dirty worktree on real path"
fi
rm -f "${DIRTY_MARKER}"

# --- remote-migrate-run unit (test env override) ---
mkdir -p "${TMP}/rel/lib"
cp -a "${OUT_DIR}/." "${TMP}/rel/"
cat >"${TMP}/migrate.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:p@127.0.0.1/db?x=1#keep
EOF
chmod_restrict "${TMP}/migrate.env"
cat >"${TMP}/rel/fiscal-migrate" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
[[ -n "${FISCAL_DATABASE_DRIVER:-}" && -n "${FISCAL_DATABASE_URL:-}" ]] || exit 1
[[ "${FISCAL_DATABASE_URL}" == *'#keep' ]] || exit 1
echo "version=2 dirty=false"
EOF
chmod 0755 "${TMP}/rel/fiscal-migrate"
# Rebuild checksums after stubbing fiscal-migrate
(
  cd "${TMP}/rel"
  if command -v sha256sum >/dev/null; then
    sha256sum fiscal-api fiscal-migrate remote-migrate-run.sh lib/allowlist.sh lib/migrate.env.allowlist COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
  else
    shasum -a 256 fiscal-api fiscal-migrate remote-migrate-run.sh lib/allowlist.sh lib/migrate.env.allowlist COMMIT EXPECTED_SCHEMA_VERSION | awk '{print $1"  "$2}' >SHA256SUMS
  fi
)
if MIGRATE_ENV_FILE="${TMP}/migrate.env" bash "${TMP}/rel/remote-migrate-run.sh" "${TMP}/rel" version >"${TMP}/rm.out" 2>"${TMP}/rm.err"; then
  if grep -q 'version=2 dirty=false' "${TMP}/rm.out" && ! grep -q 'postgres://' "${TMP}/rm.out" "${TMP}/rm.err"; then
    ok "remote-migrate-run loads keys without printing DSN"
  else
    bad "remote-migrate-run output incorrect"
  fi
else
  bad "remote-migrate-run failed"
  cat "${TMP}/rm.err" >&2 || true
fi

# --- dry-run policy ---
if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" \
  DEPLOY_GOARCH=amd64 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" \
  OUT_DIR="${TMP}/rel2" \
  EXPECTED_SCHEMA_VERSION=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=1 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  DEPLOY_N1_COMPAT_PROVEN=0 \
  DEPLOY_SIMULATE_HEALTH_FAIL=1 \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/up.out" 2>"${TMP}/up.err"; then
  bad "update should fail health without N-1 after migration"
else
  if grep -q 'roll_forward_or_manual' "${TMP}/up.out" "${TMP}/up.err" \
    && grep -q 'rollback_allowed=false' "${TMP}/up.out"; then
    ok "post-migration without N-1 blocks automatic binary rollback"
  else
    bad "N-1 rollback policy not enforced"
  fi
fi

if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 \
  OUT_DIR="${TMP}/rel3" EXPECTED_SCHEMA_VERSION=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=true \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/dirty.out" 2>"${TMP}/dirty.err"; then
  bad "dirty migration should block"
else
  ok "dirty migration blocks update"
fi

if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 \
  OUT_DIR="${TMP}/rel4" EXPECTED_SCHEMA_VERSION=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/ver.out" 2>"${TMP}/ver.err"; then
  bad "unexpected schema version should block"
else
  if grep -q 'schema_mismatch' "${TMP}/ver.out" "${TMP}/ver.err"; then
    ok "unexpected migration version blocks activation"
  else
    bad "schema mismatch not reported"
  fi
fi

# --- live path with PATH mocks ---
MOCK_BIN="${ROOT}/tests/deploy/mocks/bin"
MOCK_FS="${TMP}/mockfs"
MOCK_LOG="${TMP}/mock.log"
mkdir -p "${MOCK_FS}/opt/bwb-modulo-fiscal/releases" "${MOCK_FS}/etc/bwb-modulo-fiscal/backups" "${MOCK_FS}/tmp"
# Previous release (valid sha1-looking 40 hex)
PREV="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
mkdir -p "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${PREV}"
printf '%s\n' "${PREV}" >"${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${PREV}/COMMIT"
ln -sfn "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS}/opt/bwb-modulo-fiscal/current"

cat >"${TMP}/fiscal.live.env" <<'EOF'
FISCAL_HTTP_ADDR=127.0.0.1:8080
FISCAL_APP_VERSION=test
FISCAL_PACKAGE=AO-UNDECLARED
FISCAL_HTTP_READ_TIMEOUT=5s
FISCAL_HTTP_READ_HEADER_TIMEOUT=5s
FISCAL_HTTP_WRITE_TIMEOUT=10s
FISCAL_HTTP_IDLE_TIMEOUT=60s
FISCAL_HTTP_SHUTDOWN_TIMEOUT=10s
FISCAL_ENV=development
FISCAL_AUTH_MODE=dev_static
FISCAL_AUTH_DEV_TOKEN=CANARY_SECRET_TOKEN_VALUE_32CHARS!!
FISCAL_AUTH_DEV_SCOPE_ID=scope-test
FISCAL_SCOPE_TIMEZONE=Africa/Luanda
FISCAL_SERIES_MODE=static
FISCAL_SERIES_EFFECTIVE_CODE=A
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:CANARY_DSN_SECRET@127.0.0.1/db
EOF
cat >"${TMP}/migrate.live.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://mig:CANARY_MIG@127.0.0.1/db?x=1#keep
EOF
chmod_restrict "${TMP}/fiscal.live.env"
chmod_restrict "${TMP}/migrate.live.env"
touch "${TMP}/id_test" "${TMP}/known_hosts"
chmod 0600 "${TMP}/id_test"
chmod 0644 "${TMP}/known_hosts"

# Seed existing envs so backup path runs
mkdir -p "${MOCK_FS}/etc/bwb-modulo-fiscal"
cp "${TMP}/fiscal.live.env" "${MOCK_FS}/etc/bwb-modulo-fiscal/fiscal.env"
cp "${TMP}/migrate.live.env" "${MOCK_FS}/etc/bwb-modulo-fiscal/migrate.env"
chmod 0600 "${MOCK_FS}/etc/bwb-modulo-fiscal/"*.env

run_live() {
  local out="$1" err="$2" log="$3" fs="$4"
  shift 4
  env PATH="${MOCK_BIN}:${PATH}" \
    DEPLOY_MOCK_REMOTE=1 \
    DEPLOY_DRY_RUN=0 \
    DEPLOY_HOST=mock.host \
    DEPLOY_USER=mock \
    DEPLOY_SSH_KEY="${TMP}/id_test" \
    DEPLOY_KNOWN_HOSTS="${TMP}/known_hosts" \
    DEPLOY_MOCK_FS="${fs}" \
    DEPLOY_MOCK_LOG="${log}" \
    EXPECTED_COMMIT="${HEAD}" \
    DEPLOY_GOARCH=amd64 \
    DEPLOY_TEST_OUT_ROOT="${TMP}" \
    EXPECTED_SCHEMA_VERSION=2 \
    ENV_DEPLOY="${TMP}/fiscal.live.env" \
    ENV_MIGRATE="${TMP}/migrate.live.env" \
    "$@" \
    bash "${ROOT}/scripts/deploy/update-staging.sh" >"${out}" 2>"${err}"
}

if run_live "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" "${MOCK_FS}" \
  OUT_DIR="${TMP}/live-rel" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false; then
  if grep -q 'mode=live' "${TMP}/live.out" \
    && grep -q 'binary=new_release' "${TMP}/live.out" \
    && grep -q 'promote=ok' "${TMP}/live.out" \
    && grep -q 'health=ok' "${TMP}/live.out" \
    && grep -q 'install_release=ok' "${TMP}/live.out" \
    && grep -q 'owner=root' "${TMP}/live.out" \
    && grep -q 'env_backup=ok' "${TMP}/live.out" \
    && [[ -d "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${HEAD}" ]] \
    && grep -q 'sudo ' "${MOCK_LOG}" \
    && grep -q 'systemctl restart' "${MOCK_LOG}" \
    && grep -q "bash '.*/remote-migrate-run.sh'" "${MOCK_LOG}" \
    && ! grep -q 'deploy_require_cmds sudo' "${ROOT}/scripts/deploy/update-staging.sh" \
    && ! grep -q 'CANARY' "${TMP}/live.out" "${TMP}/live.err"; then
    ok "live path: root install, sudo remote, health, promote"
  else
    bad "live happy-path assertions failed"
    cat "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" >&2 || true
  fi
  # sudo/systemctl only via ssh remote
  if grep -E '^sudo ' "${MOCK_LOG}" >/dev/null \
    && grep -E '^ssh ' "${MOCK_LOG}" | grep -q 'sudo'; then
    ok "sudo invoked only through remote ssh commands"
  else
    bad "sudo/remote coupling not proven"
  fi
  # upload temps cleaned
  if compgen -G "${MOCK_FS}/tmp/bwb-upload.*" >/dev/null; then
    bad "upload temp dirs not cleaned"
  else
    ok "remote upload temps cleaned"
  fi
else
  bad "live mock update failed"
  cat "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" >&2 || true
fi

# Health fail + rollback + health recheck
MOCK_FS2="${TMP}/mockfs2"
MOCK_LOG2="${TMP}/mock2.log"
mkdir -p "${MOCK_FS2}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS2}/etc/bwb-modulo-fiscal/backups" "${MOCK_FS2}/tmp"
printf '%s\n' "${PREV}" >"${MOCK_FS2}/opt/bwb-modulo-fiscal/releases/${PREV}/COMMIT"
ln -sfn "${MOCK_FS2}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS2}/opt/bwb-modulo-fiscal/current"
cp "${TMP}/fiscal.live.env" "${MOCK_FS2}/etc/bwb-modulo-fiscal/fiscal.env"
cp "${TMP}/migrate.live.env" "${MOCK_FS2}/etc/bwb-modulo-fiscal/migrate.env"
chmod 0600 "${MOCK_FS2}/etc/bwb-modulo-fiscal/"*.env

if run_live "${TMP}/live2.out" "${TMP}/live2.err" "${MOCK_LOG2}" "${MOCK_FS2}" \
  OUT_DIR="${TMP}/live-rel2" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  DEPLOY_N1_COMPAT_PROVEN=1 \
  DEPLOY_SIMULATE_HEALTH_FAIL=1; then
  bad "health fail should not exit 0"
else
  if grep -q 'action=restore_previous_binary' "${TMP}/live2.out" "${TMP}/live2.err" \
    && grep -q 'health=ok_after_rollback' "${TMP}/live2.out" "${TMP}/live2.err" \
    && grep -q "active_release=${PREV}" "${TMP}/live2.out" "${TMP}/live2.err" \
    && grep -q 'env_restore=ok' "${TMP}/live2.out" "${TMP}/live2.err"; then
    ok "rollback restores previous release, envs, restart and health"
  else
    bad "rollback+health path incomplete"
    cat "${TMP}/live2.out" "${TMP}/live2.err" >&2 || true
  fi
fi

# No previous release + health fail
MOCK_FS3="${TMP}/mockfs3"
MOCK_LOG3="${TMP}/mock3.log"
mkdir -p "${MOCK_FS3}/opt/bwb-modulo-fiscal/releases" "${MOCK_FS3}/etc/bwb-modulo-fiscal/backups" "${MOCK_FS3}/tmp"
if run_live "${TMP}/live3.out" "${TMP}/live3.err" "${MOCK_LOG3}" "${MOCK_FS3}" \
  OUT_DIR="${TMP}/live-rel3" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  DEPLOY_SIMULATE_HEALTH_FAIL=1; then
  bad "missing previous should fail"
else
  if grep -q 'previous=unavailable' "${TMP}/live3.out" "${TMP}/live3.err" \
    && grep -q "active_release=${HEAD}" "${TMP}/live3.out" "${TMP}/live3.err"; then
    ok "reports new release active when previous unavailable"
  else
    bad "missing previous reporting incorrect"
    cat "${TMP}/live3.out" "${TMP}/live3.err" >&2 || true
  fi
fi

# Live dirty migration
MOCK_FS4="${TMP}/mockfs4"
MOCK_LOG4="${TMP}/mock4.log"
mkdir -p "${MOCK_FS4}/opt/bwb-modulo-fiscal/releases" "${MOCK_FS4}/etc/bwb-modulo-fiscal/backups" "${MOCK_FS4}/tmp"
if run_live "${TMP}/live4.out" "${TMP}/live4.err" "${MOCK_LOG4}" "${MOCK_FS4}" \
  OUT_DIR="${TMP}/live-rel4" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY_BEFORE=false \
  DEPLOY_MOCK_MIGRATE_DIRTY_AFTER=true; then
  bad "live dirty should fail"
else
  if grep -q 'promote=blocked reason=dirty' "${TMP}/live4.out" "${TMP}/live4.err"; then
    ok "live dirty migration blocks promote"
  else
    bad "live dirty not blocked"
    cat "${TMP}/live4.out" "${TMP}/live4.err" >&2 || true
  fi
fi

# Live unexpected version
MOCK_FS5="${TMP}/mockfs5"
MOCK_LOG5="${TMP}/mock5.log"
mkdir -p "${MOCK_FS5}/opt/bwb-modulo-fiscal/releases" "${MOCK_FS5}/etc/bwb-modulo-fiscal/backups" "${MOCK_FS5}/tmp"
if run_live "${TMP}/live5.out" "${TMP}/live5.err" "${MOCK_LOG5}" "${MOCK_FS5}" \
  OUT_DIR="${TMP}/live-rel5" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=9 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false; then
  bad "live unexpected version should fail"
else
  if grep -q 'schema_mismatch' "${TMP}/live5.out" "${TMP}/live5.err"; then
    ok "live unexpected schema version blocks activation"
  else
    bad "live schema mismatch not blocked"
  fi
fi

# Restricted file validation
cat >"${TMP}/loose.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=x
EOF
chmod 0644 "${TMP}/loose.env"
if deploy_assert_restricted_file "loose" "${TMP}/loose.env" 2>/dev/null; then
  bad "0644 should fail restricted check"
else
  ok "rejects world/group-readable env files"
fi

# --- antipatterns / nginx / systemd ---
if bash "${ROOT}/scripts/deploy/check-antipatterns.sh"; then
  ok "no forbidden SSH/env antipatterns"
else
  bad "antipattern check failed"
fi

if ! grep -E 'source[[:space:]].*migrate\.env' "${ROOT}/scripts/deploy/"*.sh \
  && ! grep -E '^[^#]*\beval\b' "${ROOT}/scripts/deploy/migrate-remote.sh" "${ROOT}/scripts/deploy/remote-migrate-run.sh" "${ROOT}/scripts/deploy/lib/allowlist.sh"; then
  ok "no source/eval on migrate env path"
else
  bad "source/eval still present"
fi

if grep -q 'deny all' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-http.conf" \
  && ! grep -E 'listen[[:space:]]+\[::\]' "${ROOT}/deploy/nginx/"*.conf; then
  ok "nginx deny-all and no IPv6 listen"
else
  bad "nginx ACL/IPv6 incorrect"
fi

if grep -E '^EnvironmentFile=' "${ROOT}/deploy/systemd/bwb-fiscal-api.service" | grep -q 'fiscal.env' \
  && ! grep -E '^EnvironmentFile=' "${ROOT}/deploy/systemd/bwb-fiscal-api.service" | grep -q 'migrate'; then
  ok "systemd uses only fiscal.env"
else
  bad "systemd EnvironmentFile incorrect"
fi

# Local git diff --check against main range (same intent as CI)
if git rev-parse --verify origin/main >/dev/null 2>&1; then
  if git diff --check origin/main...HEAD; then
    ok "git diff --check origin/main...HEAD clean"
  else
    bad "git diff --check origin/main...HEAD failed"
  fi
else
  ok "skip origin/main diff check (ref missing)"
fi

echo "summary pass=${pass} fail=${fail}"
[[ "${fail}" -eq 0 ]]
