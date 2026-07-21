#!/usr/bin/env bash
# Deploy D1 self-tests (no SSH, no server, no real secrets printed).
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

# Secrets must not appear when validating a file that contains a canary secret
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

# Force unknown key with canary in value of another line — error path
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

# --- build + checksums ---
HEAD="$(git rev-parse HEAD)"
export EXPECTED_COMMIT="${HEAD}"
export DEPLOY_GOARCH=amd64
export OUT_DIR="${TMP}/release"
if bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/build.out" 2>"${TMP}/build.err"; then
  if [[ -f "${OUT_DIR}/fiscal-api" && -f "${OUT_DIR}/fiscal-migrate" && -f "${OUT_DIR}/COMMIT" && -f "${OUT_DIR}/SHA256SUMS" ]]; then
    ok "linux release artifacts present"
  else
    bad "release artifacts missing"
  fi
  if ! grep -q CANARY "${TMP}/build.out" "${TMP}/build.err" 2>/dev/null; then
    ok "build output free of canary secrets"
  fi
else
  bad "build-linux-release failed"
  cat "${TMP}/build.err" >&2 || true
fi

# --- reject wrong commit ---
if EXPECTED_COMMIT=0000000000000000000000000000000000000000 \
  DEPLOY_GOARCH=amd64 OUT_DIR="${TMP}/badrel" \
  bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/bad.out" 2>"${TMP}/bad.err"; then
  bad "should reject mismatched EXPECTED_COMMIT"
else
  if grep -q 'does not match EXPECTED_COMMIT' "${TMP}/bad.err"; then
    ok "rejects incorrect EXPECTED_COMMIT"
  else
    bad "wrong error for EXPECTED_COMMIT"
  fi
fi

# --- update dry-run: post-migration without N-1 refuses health-fail rollback ---
if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" \
  DEPLOY_GOARCH=amd64 \
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
  OUT_DIR="${TMP}/rel3" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY=true \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/dirty.out" 2>"${TMP}/dirty.err"; then
  bad "dirty migration should block"
else
  ok "dirty migration blocks update"
fi

# --- antipatterns ---
if bash "${ROOT}/scripts/deploy/check-antipatterns.sh"; then
  ok "no forbidden SSH antipatterns"
else
  bad "antipattern check failed"
fi

# --- nginx deny-all documents ---
if grep -q 'deny all' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-http.conf" \
  && grep -q 'deny all' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-tls.conf" \
  && ! grep -E 'allow[[:space:]]+[0-9]' "${ROOT}/deploy/nginx/"*.conf; then
  ok "nginx documents deny-all without IP allow placeholders"
else
  bad "nginx documents ACL incorrect"
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
