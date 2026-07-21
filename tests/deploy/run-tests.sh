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

# --- allowlist validation ---
cat >"${TMP}/good.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://runtime:secret-should-not-leak@127.0.0.1:5432/fiscal?sslmode=require
EOF

if deploy_validate_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/good.env"; then
  ok "allowlist accepts valid migrate env"
else
  bad "allowlist rejected valid migrate env"
fi

cat >"${TMP}/unknown.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://x
EVIL_KEY=1
EOF
if deploy_validate_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/unknown.env" 2>"${TMP}/err"; then
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
if deploy_validate_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/dup.env" 2>"${TMP}/err2"; then
  bad "allowlist should reject duplicate"
else
  ok "allowlist rejects duplicate"
fi

cat >"${TMP}/bad.env" <<'EOF'
NOT_A_VALID_LINE
EOF
if deploy_validate_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/bad.env" 2>/dev/null; then
  bad "allowlist should reject malformed"
else
  ok "allowlist rejects malformed"
fi

cat >"${TMP}/canary.env" <<'EOF'
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
out="$(deploy_validate_allowlisted_file "${ROOT}/deploy/env.allowlist" "${TMP}/canary.env" 2>&1 || true)"
if [[ "${out}" == *CANARY_SECRET* || "${out}" == *CANARY_DSN* ]]; then
  bad "secret leaked in allowlist validation output"
else
  ok "no secret leak on successful allowlist validation"
fi

cat >"${TMP}/canary2.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:CANARY_DSN_SECRET@127.0.0.1/db
UNKNOWN=1
EOF
out2="$(deploy_validate_allowlisted_file "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/canary2.env" 2>&1 || true)"
if [[ "${out2}" == *CANARY_DSN* ]]; then
  bad "secret leaked on allowlist error path"
else
  ok "no secret leak on allowlist error path"
fi

# --- env parser: # only as full-line comment; preserve value after first = ---
cat >"${TMP}/special.env" <<'EOF'
# full line comment only
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:p@h/db?x=1#frag and spaces $HOME "q" 's' a=b
EOF
FISCAL_DATABASE_DRIVER=""
FISCAL_DATABASE_URL=""
deploy_load_allowlisted_env "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/special.env"
expected_url="$(deploy_read_env_value "${TMP}/special.env" FISCAL_DATABASE_URL)"
# shellcheck disable=SC2016 # intentional: assert literal $HOME remains unexpanded
if [[ "${FISCAL_DATABASE_DRIVER}" == "postgres" \
  && "${FISCAL_DATABASE_URL}" == "${expected_url}" \
  && "${FISCAL_DATABASE_URL}" == *'#frag'* \
  && "${FISCAL_DATABASE_URL}" == *'$HOME'* \
  && "${FISCAL_DATABASE_URL}" == *'a=b'* \
  && "${FISCAL_DATABASE_URL}" == *'"q"'* ]]; then
  ok "parser preserves # \$ quotes spaces and = in values"
else
  bad "parser corrupted special values"
fi

# Inline hash in value must stay
cat >"${TMP}/hashval.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=token#not-a-comment
EOF
FISCAL_DATABASE_URL=""
deploy_load_allowlisted_env "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/hashval.env"
if [[ "${FISCAL_DATABASE_URL}" == "token#not-a-comment" ]]; then
  ok "hash inside value preserved"
else
  bad "hash inside value stripped"
fi

# Values must not be executed
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
literal_got="$(deploy_read_env_value "${TMP}/exec.env" FISCAL_DATABASE_URL)"
# shellcheck disable=SC2016 # intentional: assert literal $(touch ...) was not expanded
if [[ "${FISCAL_DATABASE_URL}" == "${literal_got}" && "${literal_got}" == '$(touch '"${PWNED}"')' ]]; then
  ok "literal \$(...) preserved without execution"
else
  bad "unexpected transformed exec value"
fi

# remote-migrate-run must not source; read keys only
cat >"${TMP}/migrate.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:p@127.0.0.1/db?x=1#keep
EOF
mkdir -p "${TMP}/rel"
cat >"${TMP}/rel/fiscal-migrate" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
# Stub: prove env was exported without printing secret URL.
if [[ "${1:-}" == "version" ]]; then
  [[ -n "${FISCAL_DATABASE_DRIVER:-}" && -n "${FISCAL_DATABASE_URL:-}" ]] || exit 1
  [[ "${FISCAL_DATABASE_URL}" == *'#keep' ]] || exit 1
  echo "version=2 dirty=false"
  exit 0
fi
exit 1
EOF
chmod 0755 "${TMP}/rel/fiscal-migrate"
cp "${ROOT}/scripts/deploy/remote-migrate-run.sh" "${TMP}/rel/"
mkdir -p "${TMP}/rel/lib"
cp "${ROOT}/scripts/deploy/lib/allowlist.sh" "${TMP}/rel/lib/"
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

# --- build + checksums ---
HEAD="$(git rev-parse HEAD)"
export EXPECTED_COMMIT="${HEAD}"
export DEPLOY_GOARCH=amd64
export DEPLOY_ALLOW_DIRTY_WORKTREE=1
export DEPLOY_TEST_OUT_ROOT="${TMP}"
export OUT_DIR="${TMP}/release"
if bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/build.out" 2>"${TMP}/build.err"; then
  if [[ -f "${OUT_DIR}/fiscal-api" && -f "${OUT_DIR}/fiscal-migrate" && -f "${OUT_DIR}/COMMIT" && -f "${OUT_DIR}/SHA256SUMS" ]]; then
    ok "linux release artifacts present"
  else
    bad "release artifacts missing"
  fi
  if grep -E 'fiscal-api|fiscal-migrate|^[a-f0-9]{64}  COMMIT$' "${OUT_DIR}/SHA256SUMS" >/dev/null \
    && grep -q 'fiscal-api' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'fiscal-migrate' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'COMMIT' "${OUT_DIR}/SHA256SUMS"; then
    ok "SHA256SUMS covers fiscal-api fiscal-migrate COMMIT"
  else
    bad "SHA256SUMS incomplete"
    cat "${OUT_DIR}/SHA256SUMS" >&2 || true
  fi
  got="$(tr -d '[:space:]' <"${OUT_DIR}/COMMIT")"
  if [[ "${got}" == "${HEAD}" ]]; then
    ok "COMMIT matches HEAD"
  else
    bad "COMMIT mismatch"
  fi
  if ! grep -q CANARY "${TMP}/build.out" "${TMP}/build.err" 2>/dev/null; then
    ok "build output free of canary secrets"
  fi
else
  bad "build-linux-release failed"
  cat "${TMP}/build.err" >&2 || true
fi

# Reject OUT_DIR outside allowlist
if EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  OUT_DIR="/tmp/bwb-forbidden-out-$$" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/forbid.out" 2>"${TMP}/forbid.err"; then
  bad "should reject OUT_DIR outside dist/releases"
  rm -rf "/tmp/bwb-forbidden-out-$$"
else
  ok "rejects OUT_DIR outside authorized roots"
fi

# Reject bad GOARCH
if EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=386 DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" OUT_DIR="${TMP}/badarch" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/arch.out" 2>"${TMP}/arch.err"; then
  bad "should reject GOARCH 386"
else
  ok "rejects unsupported GOARCH"
fi

# Reject wrong commit
if EXPECTED_COMMIT=0000000000000000000000000000000000000000 \
  DEPLOY_GOARCH=amd64 DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" OUT_DIR="${TMP}/badrel" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/bad.out" 2>"${TMP}/bad.err"; then
  bad "should reject mismatched EXPECTED_COMMIT"
else
  if grep -q 'does not match EXPECTED_COMMIT' "${TMP}/bad.err"; then
    ok "rejects incorrect EXPECTED_COMMIT"
  else
    bad "wrong error for EXPECTED_COMMIT"
  fi
fi

# Dirty worktree refusal
DIRTY_MARKER="${ROOT}/.deploy-dirty-test-$$"
touch "${DIRTY_MARKER}"
if EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 DEPLOY_ALLOW_DIRTY_WORKTREE=0 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" OUT_DIR="${TMP}/dirtybuild" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/dirtyb.out" 2>"${TMP}/dirtyb.err"; then
  bad "should refuse dirty worktree"
else
  if grep -q 'dirty worktree' "${TMP}/dirtyb.err"; then
    ok "refuses dirty worktree"
  else
    bad "dirty worktree error incorrect"
  fi
fi
rm -f "${DIRTY_MARKER}"

# Overwrite with different COMMIT refused
mkdir -p "${TMP}/clash"
printf 'deadbeef\n' >"${TMP}/clash/COMMIT"
if EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" OUT_DIR="${TMP}/clash" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/clash.out" 2>"${TMP}/clash.err"; then
  bad "should refuse overwrite of different COMMIT"
else
  ok "refuses overwrite of different COMMIT release"
fi

# --- update dry-run: post-migration without N-1 refuses health-fail rollback ---
if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" \
  DEPLOY_GOARCH=amd64 \
  DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" \
  OUT_DIR="${TMP}/rel2" \
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
    cat "${TMP}/up.out" "${TMP}/up.err" >&2 || true
  fi
fi

# --- dirty blocks promote ---
if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" \
  DEPLOY_GOARCH=amd64 \
  DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" \
  OUT_DIR="${TMP}/rel3" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=true \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/dirty.out" 2>"${TMP}/dirty.err"; then
  bad "dirty migration should block"
else
  ok "dirty migration blocks update"
fi

# --- live path with PATH mocks (zero network) ---
MOCK_BIN="${ROOT}/tests/deploy/mocks/bin"
MOCK_FS="${TMP}/mockfs"
MOCK_LOG="${TMP}/mock.log"
mkdir -p "${MOCK_FS}/opt/bwb-modulo-fiscal/releases" "${MOCK_FS}/etc/bwb-modulo-fiscal" "${MOCK_FS}/tmp"
# Seed a previous release for N-1 path coverage
mkdir -p "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/prevsha"
printf 'prevsha\n' >"${MOCK_FS}/opt/bwb-modulo-fiscal/releases/prevsha/COMMIT"
ln -sfn "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/prevsha" "${MOCK_FS}/opt/bwb-modulo-fiscal/current"

# Minimal allowlisted env files for live mock
cp "${TMP}/canary.env" "${TMP}/fiscal.live.env"
cat >"${TMP}/migrate.live.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://mig:CANARY_MIG@127.0.0.1/db?x=1#keep
EOF

# Dummy key/known_hosts paths (never read for crypto by mocks)
touch "${TMP}/id_test" "${TMP}/known_hosts"

if PATH="${MOCK_BIN}:${PATH}" \
  DEPLOY_MOCK_REMOTE=1 \
  DEPLOY_DRY_RUN=0 \
  DEPLOY_HOST=mock.host \
  DEPLOY_USER=mock \
  DEPLOY_SSH_KEY="${TMP}/id_test" \
  DEPLOY_KNOWN_HOSTS="${TMP}/known_hosts" \
  DEPLOY_MOCK_FS="${MOCK_FS}" \
  DEPLOY_MOCK_LOG="${MOCK_LOG}" \
  EXPECTED_COMMIT="${HEAD}" \
  DEPLOY_GOARCH=amd64 \
  DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" \
  OUT_DIR="${TMP}/live-rel" \
  ENV_DEPLOY="${TMP}/fiscal.live.env" \
  ENV_MIGRATE="${TMP}/migrate.live.env" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/live.out" 2>"${TMP}/live.err"; then
  mode_mig="$(python3 -c "import os; print(f'{os.stat(r'''${MOCK_FS}/etc/bwb-modulo-fiscal/migrate.env''').st_mode & 0o777:03o}')")"
  if grep -q 'mode=live' "${TMP}/live.out" \
    && grep -q 'binary=new_release' "${TMP}/live.out" \
    && grep -q 'promote=ok' "${TMP}/live.out" \
    && grep -q 'restart=ok' "${TMP}/live.out" \
    && [[ -f "${MOCK_FS}/etc/bwb-modulo-fiscal/fiscal.env" ]] \
    && [[ -f "${MOCK_FS}/etc/bwb-modulo-fiscal/migrate.env" ]] \
    && [[ "${mode_mig}" == "600" ]] \
    && [[ -d "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${HEAD}" ]] \
    && grep -q "bash '.*/remote-migrate-run.sh'" "${MOCK_LOG}" \
    && ! grep -q 'current/fiscal-migrate' "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" \
    && ! grep -q 'CANARY' "${TMP}/live.out" "${TMP}/live.err"; then
    ok "live path mocked upload verify install migrate promote restart"
  else
    bad "live mock path assertions failed (mode_mig=${mode_mig})"
    cat "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" >&2 || true
  fi
else
  bad "live mock update failed"
  cat "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" >&2 || true
fi

# Live path: post-migration without N-1 refuses rollback on health fail
if PATH="${MOCK_BIN}:${PATH}" \
  DEPLOY_MOCK_REMOTE=1 \
  DEPLOY_DRY_RUN=0 \
  DEPLOY_HOST=mock.host \
  DEPLOY_USER=mock \
  DEPLOY_SSH_KEY="${TMP}/id_test" \
  DEPLOY_KNOWN_HOSTS="${TMP}/known_hosts" \
  DEPLOY_MOCK_FS="${TMP}/mockfs2" \
  DEPLOY_MOCK_LOG="${TMP}/mock2.log" \
  EXPECTED_COMMIT="${HEAD}" \
  DEPLOY_GOARCH=amd64 \
  DEPLOY_ALLOW_DIRTY_WORKTREE=1 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" \
  OUT_DIR="${TMP}/live-rel2" \
  ENV_DEPLOY="${TMP}/fiscal.live.env" \
  ENV_MIGRATE="${TMP}/migrate.live.env" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=1 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  DEPLOY_N1_COMPAT_PROVEN=0 \
  DEPLOY_SIMULATE_HEALTH_FAIL=1 \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/live2.out" 2>"${TMP}/live2.err"; then
  bad "live mock should fail health without N-1"
else
  if grep -q 'roll_forward_or_manual' "${TMP}/live2.out" "${TMP}/live2.err"; then
    ok "live mock enforces N-1 rollback policy"
  else
    bad "live mock N-1 policy not enforced"
    cat "${TMP}/live2.out" "${TMP}/live2.err" >&2 || true
  fi
fi

# --- antipatterns ---
if bash "${ROOT}/scripts/deploy/check-antipatterns.sh"; then
  ok "no forbidden SSH/env antipatterns"
else
  bad "antipattern check failed"
fi

# No source migrate.env / eval in migrate-remote or update
if ! grep -E 'source[[:space:]].*migrate\.env' "${ROOT}/scripts/deploy/"*.sh \
  && ! grep -E '^[^#]*\beval\b' "${ROOT}/scripts/deploy/migrate-remote.sh" "${ROOT}/scripts/deploy/remote-migrate-run.sh" "${ROOT}/scripts/deploy/lib/allowlist.sh"; then
  ok "no source/eval on migrate env path"
else
  bad "source/eval still present"
fi

# --- nginx deny-all documents + no IPv6 listen in D1 ---
if grep -q 'deny all' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-http.conf" \
  && grep -q 'deny all' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-tls.conf" \
  && ! grep -E 'allow[[:space:]]+[0-9]' "${ROOT}/deploy/nginx/"*.conf \
  && ! grep -E 'listen[[:space:]]+\[::\]' "${ROOT}/deploy/nginx/"*.conf; then
  ok "nginx documents deny-all; no IPv6 listen in D1"
else
  bad "nginx ACL/IPv6 incorrect"
fi

# --- systemd never references migrate.env ---
if grep -E '^EnvironmentFile=' "${ROOT}/deploy/systemd/bwb-fiscal-api.service" | grep -q 'fiscal.env' \
  && ! grep -E '^EnvironmentFile=' "${ROOT}/deploy/systemd/bwb-fiscal-api.service" | grep -q 'migrate'; then
  ok "systemd uses only fiscal.env"
else
  bad "systemd EnvironmentFile incorrect"
fi

# --- http bootstrap has no ssl_certificate directive ---
if ! grep -E '^[[:space:]]*ssl_certificate' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-http.conf"; then
  ok "HTTP bootstrap has no ssl_certificate"
else
  bad "HTTP bootstrap references certificates"
fi

echo "summary pass=${pass} fail=${fail}"
[[ "${fail}" -eq 0 ]]
