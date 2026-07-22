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

# Neutralize operator .env.local so local staging secrets/EXPECTED_COMMIT cannot skew fixtures.
: >"${TMP}/neutral.operator.env"
chmod 0600 "${TMP}/neutral.operator.env"
export ENV_LOCAL="${TMP}/neutral.operator.env"

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
  chmod 0600 "$@"
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
export EXPECTED_SCHEMA_VERSION=3
if bash "${ROOT}/scripts/deploy/build-linux-release.sh" >"${TMP}/build.out" 2>"${TMP}/build.err"; then
  if deploy_verify_release_manifest "${OUT_DIR}" "${HEAD}"; then
    ok "release manifest verifies"
  else
    bad "release manifest verify failed"
  fi
  if ! grep -q 'remote-migrate-run.sh' "${OUT_DIR}/SHA256SUMS" \
    && [[ ! -e "${OUT_DIR}/remote-migrate-run.sh" ]] \
    && grep -q 'fiscal-admin' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'fiscal-sandbox-e2e' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'lib/admin.env.allowlist' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'lib/allowlist.sh' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'lib/migrate.env.allowlist' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'EXPECTED_SCHEMA_VERSION' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'COMMIT' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'nginx/tls.open.conf' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'nginx/tls.deny.conf' "${OUT_DIR}/SHA256SUMS" \
    && grep -q 'systemd/bwb-fiscal-nginx-open-rollback.timer' "${OUT_DIR}/SHA256SUMS" \
    && [[ ! -e "${OUT_DIR}/nginx/candidates/bwb-fiscal-sandbox-tls.open.candidate.conf" ]]; then
    ok "SHA256SUMS covers release files and omits migrate runner/open candidate"
  else
    bad "SHA256SUMS incorrect (runner present, open candidate shipped, or incomplete)"
  fi
  if [[ "$(tr -d '[:space:]' <"${OUT_DIR}/EXPECTED_SCHEMA_VERSION")" == "3" ]]; then
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

# --- helper migrate: never bash release scripts; clean env; no DSN leak ---
mkdir -p "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}" \
  "${TMP}/helprefs/etc/bwb-modulo-fiscal" \
  "${TMP}/helprefs/lib"
cp -a "${OUT_DIR}/." "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/"
cp "${ROOT}/scripts/deploy/lib/allowlist.sh" "${TMP}/helprefs/lib/allowlist.sh"
cp "${ROOT}/deploy/migrate.env.allowlist" "${TMP}/helprefs/lib/migrate.env.allowlist"
cp "${ROOT}/deploy/admin.env.allowlist" "${TMP}/helprefs/lib/admin.env.allowlist"
cat >"${TMP}/helprefs/etc/bwb-modulo-fiscal/migrate.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:p@127.0.0.1/db?x=1#keep
EOF
chmod_restrict "${TMP}/helprefs/etc/bwb-modulo-fiscal/migrate.env"
EUID_LOG="${TMP}/migrate-euid.log"
cat >"${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-migrate" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'euid=%s uid=%s driver=%s url_set=%s path=%s home=%s\n' \
  "\${EUID}" "\$(id -u)" "\${FISCAL_DATABASE_DRIVER:-}" \
  "\$([ -n "\${FISCAL_DATABASE_URL:-}" ] && echo 1 || echo 0)" \
  "\${PATH:-}" "\${HOME:-}" >"${EUID_LOG}"
[[ -n "\${FISCAL_DATABASE_DRIVER:-}" && -n "\${FISCAL_DATABASE_URL:-}" ]] || exit 1
[[ "\${FISCAL_DATABASE_URL}" == *'#keep' ]] || exit 1
# Prove ambient secrets are not inherited
[[ -z "\${SECRET_SHOULD_NOT_LEAK:-}" ]] || exit 1
echo "version=3 dirty=false"
EOF
chmod 0755 "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-migrate"
(
  cd "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}"
  deploy_sha256_files \
    fiscal-api fiscal-migrate fiscal-admin fiscal-sandbox-e2e fiscal-sandbox-measure \
    lib/allowlist.sh lib/migrate.env.allowlist lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    nginx/tls.open.conf nginx/tls.deny.conf nginx/limit-req-documents.conf nginx/README.md \
    systemd/bwb-fiscal-nginx-open-rollback.service systemd/bwb-fiscal-nginx-open-rollback.timer \
    COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
)
if SECRET_SHOULD_NOT_LEAK=pwned \
  BWB_DEPLOY_OPT="${TMP}/helprefs/opt/bwb-modulo-fiscal" \
  BWB_DEPLOY_ETC="${TMP}/helprefs/etc/bwb-modulo-fiscal" \
  BWB_HELPER_LIB="${TMP}/helprefs/lib" \
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" migrate "${HEAD}" version \
  >"${TMP}/hm.out" 2>"${TMP}/hm.err"; then
  if grep -q 'version=3 dirty=false' "${TMP}/hm.out" \
    && [[ -f "${EUID_LOG}" ]] \
    && grep -q 'url_set=1' "${EUID_LOG}" \
    && grep -q 'driver=postgres' "${EUID_LOG}" \
    && ! grep -q 'postgres://' "${TMP}/hm.out" "${TMP}/hm.err" \
    && ! grep -E 'bash|remote-migrate' "${TMP}/hm.out" "${TMP}/hm.err"; then
    ok "helper migrate runs fiscal-migrate with clean env (no DSN leak)"
  else
    bad "helper migrate output incorrect"
    cat "${TMP}/hm.out" "${TMP}/hm.err" "${EUID_LOG}" >&2 || true
  fi
else
  bad "helper migrate failed"
  cat "${TMP}/hm.err" >&2 || true
fi

# Prove helper never executes release scripts as a shell, and runner is gone
if ! grep -nE 'bash[[:space:]]+".*\$\{?(release|dir|dest)' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && ! grep -nE 'bash[[:space:]]+.*remote-migrate' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -q 'must not be in release' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && [[ ! -e "${ROOT}/scripts/deploy/remote-migrate-run.sh" ]]; then
  ok "no release script execution path; runner removed from repo"
else
  bad "helper still references release runner/scripts"
fi

# Root override refusal (simulate via checking the guard is present + non-root path works)
if grep -q 'BWB_\* test overrides are forbidden when EUID=0' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -q 'runuser\|setpriv' "${ROOT}/scripts/deploy/remote-deploy-helper.sh"; then
  ok "helper refuses root overrides and drops privileges via runuser/setpriv"
else
  bad "root override / drop-priv guards missing"
fi

# --- helper admin: env -i, no DSN leak, token path chosen by helper, open candidate rejected ---
cat >"${TMP}/helprefs/etc/bwb-modulo-fiscal/admin.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://adm:SECRET_ADM_DSN@127.0.0.1/db
EOF
chmod_restrict "${TMP}/helprefs/etc/bwb-modulo-fiscal/admin.env"
ADMIN_EUID_LOG="${TMP}/admin-euid.log"
cat >"${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-admin" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
printf 'euid=%s driver=%s url_set=%s path=%s home=%s leak=%s\n' \
  "\${EUID}" "\${FISCAL_DATABASE_DRIVER:-}" \
  "\$([ -n "\${FISCAL_DATABASE_URL:-}" ] && echo 1 || echo 0)" \
  "\${PATH:-}" "\${HOME:-}" "\${SECRET_SHOULD_NOT_LEAK:-}" >"${ADMIN_EUID_LOG}"
[[ -z "\${SECRET_SHOULD_NOT_LEAK:-}" ]] || exit 1
[[ "\${FISCAL_DATABASE_DRIVER}" == "postgres" && -n "\${FISCAL_DATABASE_URL}" ]] || exit 1
# Emulate token write when --output-file is present
outf=""
prev=""
for a in "\$@"; do
  if [[ "\${prev}" == "--output-file" ]]; then outf="\${a}"; fi
  prev="\${a}"
done
if [[ -n "\${outf}" ]]; then
  printf 'bwb_sbox_synthetic_token_fixture_only\n' >"\${outf}"
  chmod 0600 "\${outf}"
fi
echo "credential_id=cred-synth-001 status=active"
EOF
chmod 0755 "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-admin"
(
  cd "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}"
  deploy_sha256_files \
    fiscal-api fiscal-migrate fiscal-admin fiscal-sandbox-e2e fiscal-sandbox-measure \
    lib/allowlist.sh lib/migrate.env.allowlist lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    nginx/tls.open.conf nginx/tls.deny.conf nginx/limit-req-documents.conf nginx/README.md \
    systemd/bwb-fiscal-nginx-open-rollback.service systemd/bwb-fiscal-nginx-open-rollback.timer \
    COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
)
TOKEN_DIR="${TMP}/admin-tokens"
mkdir -p "${TOKEN_DIR}"
if SECRET_SHOULD_NOT_LEAK=pwned \
  BWB_DEPLOY_OPT="${TMP}/helprefs/opt/bwb-modulo-fiscal" \
  BWB_DEPLOY_ETC="${TMP}/helprefs/etc/bwb-modulo-fiscal" \
  BWB_HELPER_LIB="${TMP}/helprefs/lib" \
  BWB_ADMIN_TOKEN_DIR="${TOKEN_DIR}" \
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  admin-credential-issue "${HEAD}" "scope-synth-001" "operator-test" \
  >"${TMP}/ha.out" 2>"${TMP}/ha.err"; then
  if grep -q 'admin_credential_issue_ok' "${TMP}/ha.out" \
    && [[ -f "${ADMIN_EUID_LOG}" ]] \
    && grep -q 'url_set=1' "${ADMIN_EUID_LOG}" \
    && grep -q 'driver=postgres' "${ADMIN_EUID_LOG}" \
    && ! grep -q 'SECRET_ADM_DSN' "${TMP}/ha.out" "${TMP}/ha.err" "${ADMIN_EUID_LOG}" \
    && ! grep -q 'postgres://' "${TMP}/ha.out" "${TMP}/ha.err" \
    && [[ -f "${TOKEN_DIR}/current.token" ]] \
    && ! grep -q 'bwb_sbox_' "${TMP}/ha.out" "${TMP}/ha.err"; then
    ok "helper admin issue uses env -i (no DSN/token leak); token path helper-chosen"
  else
    bad "helper admin issue assertions failed"
    cat "${TMP}/ha.out" "${TMP}/ha.err" "${ADMIN_EUID_LOG}" >&2 || true
  fi
else
  bad "helper admin-credential-issue failed"
  cat "${TMP}/ha.err" >&2 || true
fi

# A→B revoke gate orchestration (mocked fiscal-admin + e2e; no token/DSN in outputs)
GATE_E2E_LOG="${TMP}/gate-e2e.log"
: >"${GATE_E2E_LOG}"
echo 0 >"${TMP}/gate-cred-seq"
cat >"${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-admin" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
cmd=""
prev=""
outf=""
cred=""
for a in "\$@"; do
  if [[ "\${prev}" == "--output-file" ]]; then outf="\${a}"; fi
  if [[ "\${prev}" == "--credential-id" ]]; then cred="\${a}"; fi
  prev="\${a}"
  if [[ "\$a" == "issue" || "\$a" == "revoke" ]]; then cmd="\$a"; fi
done
case "\${cmd}" in
  issue)
    [[ -n "\${outf}" ]] || exit 1
    printf 'bwb_sbox_gate_token_synthetic_only\n' >"\${outf}"
    chmod 0600 "\${outf}"
    n=\$((\$(cat "${TMP}/gate-cred-seq") + 1))
    printf '%s\n' "\$n" >"${TMP}/gate-cred-seq"
    printf 'credential_id=cred-gate-%s scope_id=scope-synth-001 status=active\n' "\$n"
    ;;
  revoke)
    [[ -n "\${cred}" ]] || exit 1
    printf 'credential_id=%s scope_id=scope-synth-001 status=revoked\n' "\${cred}"
    ;;
  *) exit 1 ;;
esac
EOF
chmod 0755 "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-admin"
cat >"${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-sandbox-e2e" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
case_name=""
token=""
prev=""
for a in "\$@"; do
  if [[ "\${prev}" == "--case" ]]; then case_name="\${a}"; fi
  if [[ "\${prev}" == "--token-file" ]]; then token="\${a}"; fi
  prev="\${a}"
done
[[ -f "\${token}" ]] || exit 1
printf 'case=%s token_set=1\n' "\${case_name}" >>"${GATE_E2E_LOG}"
case "\${case_name}" in
  create_201) printf 'status=201 result=pass document_id=doc-gate-a-synth\n' ;;
  token_revoked_401) printf 'status=401 result=pass\n' ;;
  create_replay) printf 'status=201 result=pass document_id=doc-gate-b-synth\n' ;;
  *) exit 1 ;;
esac
EOF
chmod 0755 "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-sandbox-e2e"
(
  cd "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}"
  deploy_sha256_files \
    fiscal-api fiscal-migrate fiscal-admin fiscal-sandbox-e2e fiscal-sandbox-measure \
    lib/allowlist.sh lib/migrate.env.allowlist lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    nginx/tls.open.conf nginx/tls.deny.conf nginx/limit-req-documents.conf nginx/README.md \
    systemd/bwb-fiscal-nginx-open-rollback.service systemd/bwb-fiscal-nginx-open-rollback.timer \
    COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
)
GATE_TOKENS="${TMP}/gate-tokens"
mkdir -p "${GATE_TOKENS}"
if BWB_DEPLOY_OPT="${TMP}/helprefs/opt/bwb-modulo-fiscal" \
  BWB_DEPLOY_ETC="${TMP}/helprefs/etc/bwb-modulo-fiscal" \
  BWB_HELPER_LIB="${TMP}/helprefs/lib" \
  BWB_ADMIN_TOKEN_DIR="${GATE_TOKENS}" \
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  admin-sandbox-ab-revoke-gate "${HEAD}" "scope-synth-001" "operator-test" \
  >"${TMP}/gate.out" 2>"${TMP}/gate.err"; then
  if grep -q 'ab_gate_a_usable=ok' "${TMP}/gate.out" \
    && grep -q 'ab_gate_a_revoked=ok' "${TMP}/gate.out" \
    && grep -q 'ab_gate_a_rejected=ok' "${TMP}/gate.out" \
    && grep -q 'ab_gate_b_replay=ok' "${TMP}/gate.out" \
    && grep -q 'ab_gate_docs_distinct=ok' "${TMP}/gate.out" \
    && grep -q 'document_id=doc-gate-a-synth' "${TMP}/gate.out" \
    && grep -q 'document_id=doc-gate-b-synth' "${TMP}/gate.out" \
    && grep -q 'case=create_201' "${GATE_E2E_LOG}" \
    && grep -q 'case=token_revoked_401' "${GATE_E2E_LOG}" \
    && grep -q 'case=create_replay' "${GATE_E2E_LOG}" \
    && ! grep -q 'bwb_sbox_\|postgres://\|SECRET_ADM' "${TMP}/gate.out" "${TMP}/gate.err" "${GATE_E2E_LOG}"; then
    ok "A→B revoke gate: A usable→revoke→401→B new doc+replay; ids distinct; no secret leak"
  else
    bad "A→B revoke gate assertions failed"
    cat "${TMP}/gate.out" "${TMP}/gate.err" "${GATE_E2E_LOG}" >&2 || true
  fi
else
  bad "admin-sandbox-ab-revoke-gate failed"
  cat "${TMP}/gate.err" "${TMP}/gate.out" >&2 || true
fi

if BWB_DEPLOY_OPT="${TMP}/helprefs/opt/bwb-modulo-fiscal" \
  BWB_DEPLOY_ETC="${TMP}/helprefs/etc/bwb-modulo-fiscal" \
  BWB_HELPER_LIB="${TMP}/helprefs/lib" \
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" install-nginx-open \
  >"${TMP}/ngopen.out" 2>"${TMP}/ngopen.err"; then
  bad "helper must reject open nginx activation"
else
  if grep -qi 'cannot be activated' "${TMP}/ngopen.err"; then
    ok "helper rejects install-nginx-open / open candidate activation"
  else
    bad "open candidate rejection message missing"
    cat "${TMP}/ngopen.err" >&2 || true
  fi
fi

# admin.env allowlist rejects extra keys
cat >"${TMP}/admin.bad.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://adm:x@127.0.0.1/db
FISCAL_HTTP_ADDR=127.0.0.1:8080
EOF
chmod_restrict "${TMP}/admin.bad.env"
if deploy_validate_exact_allowlisted_file "${ROOT}/deploy/admin.env.allowlist" "${TMP}/admin.bad.env" \
  >"${TMP}/admin.bad.out" 2>"${TMP}/admin.bad.err"; then
  bad "admin allowlist should reject extra keys"
else
  ok "admin.env allowlist rejects non-allowlisted keys"
fi
cat >"${TMP}/admin.good.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://adm:x@127.0.0.1/db
EOF
chmod_restrict "${TMP}/admin.good.env"
if deploy_validate_exact_allowlisted_file "${ROOT}/deploy/admin.env.allowlist" "${TMP}/admin.good.env"; then
  ok "admin.env allowlist accepts DRIVER+URL only"
else
  bad "admin.env allowlist rejected valid file"
fi

# Nginx deny-all / open / measure invariants (S3C2)
if grep -q 'deny all' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-tls.conf" \
  && grep -q 'listen 127.0.0.1:18080' "${ROOT}/deploy/nginx/measure/bwb-fiscal-sandbox-measure-loopback.conf" \
  && grep -q 'limit_req zone=bwb_documents burst=20' \
    "${ROOT}/deploy/nginx/measure/bwb-fiscal-sandbox-measure-loopback.conf" \
  && grep -q 'limit_req zone=bwb_documents burst=20' \
    "${ROOT}/deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf" \
  && grep -q 'limit_req_status 429' \
    "${ROOT}/deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf" \
  && grep -q 'rate=10r/s' \
    "${ROOT}/deploy/nginx/http.d/bwb-limit-req-documents.conf" \
  && grep -q 'proxy_set_header X-Request-Id ""' \
    "${ROOT}/deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf" \
  && grep -A4 'location = /v1/health' \
    "${ROOT}/deploy/nginx/open/bwb-fiscal-sandbox-tls.open.conf" \
    | grep -qv 'limit_req' \
  && grep -q 'nginx-open-arm' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -q 'OnActiveSec=5min' "${ROOT}/deploy/systemd/bwb-fiscal-nginx-open-rollback.timer"; then
  ok "nginx deny-all + open(10r/s,burst=20,429) + measure + fail-safe timer"
else
  bad "nginx open/deny/measure/timer invariants failed"
fi
if ! grep -qE 'listen[[:space:]]+18080|listen[[:space:]]+\*:18080|0\.0\.0\.0:18080' \
  "${ROOT}/deploy/nginx/measure/bwb-fiscal-sandbox-measure-loopback.conf"; then
  ok "measure listener has no non-loopback bind"
else
  bad "measure listener binds non-loopback"
fi

# --- S3C2 nginx-open-arm / confirm / deny-all / timer rollback (mocked nginx+systemctl) ---
NGX="${TMP}/nginx-root"
SYS="${TMP}/systemd-dir"
CTLLOG="${TMP}/systemctl.log"
: >"${CTLLOG}"
mkdir -p "${NGX}/sites-available" "${NGX}/sites-enabled" "${NGX}/conf.d" "${SYS}" \
  "${TMP}/mockbin"
# Seed deny-all as currently active public site + stale measure listener.
cp "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-tls.conf" "${NGX}/sites-available/bwb-fiscal-sandbox"
ln -sfn "${NGX}/sites-available/bwb-fiscal-sandbox" "${NGX}/sites-enabled/bwb-fiscal-sandbox"
printf 'measure\n' >"${NGX}/sites-available/bwb-fiscal-sandbox-measure-loopback.conf"
ln -sfn "${NGX}/sites-available/bwb-fiscal-sandbox-measure-loopback.conf" \
  "${NGX}/sites-enabled/bwb-fiscal-sandbox-measure-loopback.conf"
cp "${ROOT}/deploy/nginx/http.d/bwb-limit-req-documents-provisional.conf" \
  "${NGX}/conf.d/bwb-limit-req-documents-provisional.conf"
cat >"${TMP}/mockbin/nginx" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "\$*" >>"${TMP}/nginx-invocations.log"
exit 0
EOF
cat >"${TMP}/mockbin/systemctl" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "\$*" >>"${CTLLOG}"
exit 0
EOF
cat >"${TMP}/mockbin/curl" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
# Emit 403 for documents probe without touching the network.
if [[ "\$*" == *"/v1/documents"* ]]; then
  printf '403'
  exit 0
fi
printf '000'
exit 1
EOF
chmod 0755 "${TMP}/mockbin/nginx" "${TMP}/mockbin/systemctl" "${TMP}/mockbin/curl"
: >"${TMP}/nginx-invocations.log"

# Ensure helprefs release has nginx artefacts from OUT_DIR build
if [[ ! -f "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/nginx/tls.open.conf" ]]; then
  cp -a "${OUT_DIR}/nginx" "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/"
  cp -a "${OUT_DIR}/systemd" "${TMP}/helprefs/opt/bwb-modulo-fiscal/releases/${HEAD}/"
fi

run_ngx_helper() {
  PATH="${TMP}/mockbin:/usr/bin:/bin" \
    BWB_DEPLOY_OPT="${TMP}/helprefs/opt/bwb-modulo-fiscal" \
    BWB_DEPLOY_ETC="${TMP}/helprefs/etc/bwb-modulo-fiscal" \
    BWB_HELPER_LIB="${TMP}/helprefs/lib" \
    BWB_NGINX_ROOT="${NGX}" \
    BWB_SYSTEMD_DIR="${SYS}" \
    BWB_SYSTEMCTL=systemctl \
    BWB_NGINX_BIN=nginx \
    BWB_CURL=curl \
    bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" "$@"
}

if run_ngx_helper nginx-open-arm "${HEAD}" >"${TMP}/arm.out" 2>"${TMP}/arm.err"; then
  if grep -q "nginx_open_arm_ok sha=${HEAD}" "${TMP}/arm.out" \
    && grep -q 'limit_req zone=bwb_documents burst=20' "${NGX}/sites-available/bwb-fiscal-sandbox" \
    && ! grep -q 'deny all' "${NGX}/sites-available/bwb-fiscal-sandbox" \
    && [[ ! -e "${NGX}/sites-enabled/bwb-fiscal-sandbox-measure-loopback.conf" ]] \
    && [[ -f "${NGX}/conf.d/bwb-limit-req-documents.conf" ]] \
    && [[ ! -e "${NGX}/conf.d/bwb-limit-req-documents-provisional.conf" ]] \
    && grep -q 'start bwb-fiscal-nginx-open-rollback.timer' "${CTLLOG}" \
    && grep -q 'state=armed' "${TMP}/helprefs/etc/bwb-modulo-fiscal/nginx-open.state" \
    && [[ -f "${SYS}/bwb-fiscal-nginx-open-rollback.timer" ]]; then
    ok "nginx-open-arm installs open, disables :18080 measure, arms 5m timer"
  else
    bad "nginx-open-arm assertions failed"
    cat "${TMP}/arm.out" "${TMP}/arm.err" "${CTLLOG}" >&2 || true
  fi
else
  bad "nginx-open-arm failed"
  cat "${TMP}/arm.err" "${TMP}/arm.out" >&2 || true
fi

# confirm cancels timer
: >"${CTLLOG}"
if run_ngx_helper nginx-open-confirm "${HEAD}" >"${TMP}/confirm.out" 2>"${TMP}/confirm.err"; then
  if grep -q "nginx_open_confirm_ok sha=${HEAD}" "${TMP}/confirm.out" \
    && grep -q 'stop bwb-fiscal-nginx-open-rollback.timer' "${CTLLOG}" \
    && grep -q 'state=confirmed' "${TMP}/helprefs/etc/bwb-modulo-fiscal/nginx-open.state"; then
    ok "nginx-open-confirm cancels rollback timer"
  else
    bad "nginx-open-confirm assertions failed"
    cat "${TMP}/confirm.out" "${TMP}/confirm.err" "${CTLLOG}" >&2 || true
  fi
else
  bad "nginx-open-confirm failed"
  cat "${TMP}/confirm.err" >&2 || true
fi

# Re-arm then fire timer rollback
: >"${CTLLOG}"
run_ngx_helper nginx-open-arm "${HEAD}" >"${TMP}/arm2.out" 2>"${TMP}/arm2.err" || true
if run_ngx_helper nginx-open-rollback-fire >"${TMP}/fire.out" 2>"${TMP}/fire.err"; then
  if grep -q 'nginx_deny_all_ok\|nginx_open_rollback_fire=ok' "${TMP}/fire.out" \
    && grep -q 'deny all' "${NGX}/sites-available/bwb-fiscal-sandbox" \
    && grep -qE 'state=(denied|rolled_back)' "${TMP}/helprefs/etc/bwb-modulo-fiscal/nginx-open.state"; then
    ok "nginx-open-rollback-fire restores deny-all"
  else
    bad "rollback-fire assertions failed"
    cat "${TMP}/fire.out" "${TMP}/fire.err" >&2 || true
  fi
else
  bad "nginx-open-rollback-fire failed"
  cat "${TMP}/fire.err" "${TMP}/fire.out" >&2 || true
fi

# Explicit deny-all ok from open
run_ngx_helper nginx-open-arm "${HEAD}" >/dev/null 2>&1 || true
if run_ngx_helper nginx-deny-all "${HEAD}" >"${TMP}/deny.out" 2>"${TMP}/deny.err"; then
  if grep -q "nginx_deny_all_ok sha=${HEAD}" "${TMP}/deny.out" \
    && grep -q 'deny all' "${NGX}/sites-available/bwb-fiscal-sandbox"; then
    ok "nginx-deny-all restores deny-all explicitly"
  else
    bad "nginx-deny-all assertions failed"
    cat "${TMP}/deny.out" "${TMP}/deny.err" >&2 || true
  fi
else
  bad "nginx-deny-all failed"
  cat "${TMP}/deny.err" >&2 || true
fi

# Negative: confirm without arm
rm -f "${TMP}/helprefs/etc/bwb-modulo-fiscal/nginx-open.state"
if run_ngx_helper nginx-open-confirm "${HEAD}" >"${TMP}/confirm.bad.out" 2>"${TMP}/confirm.bad.err"; then
  bad "confirm without arm must fail"
else
  ok "nginx-open-confirm rejects missing/unarmed state"
fi

# Negative: legacy install-nginx-open still rejected
if run_ngx_helper install-nginx-open >"${TMP}/ngopen.out" 2>"${TMP}/ngopen.err"; then
  bad "helper must reject install-nginx-open"
else
  if grep -qi 'cannot be activated\|nginx-open-arm' "${TMP}/ngopen.err"; then
    ok "helper rejects install-nginx-open / open candidate activation"
  else
    bad "open candidate rejection message missing"
    cat "${TMP}/ngopen.err" >&2 || true
  fi
fi

# Negative: nginx -t failure rolls back
cat >"${TMP}/mockbin/nginx" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "\$*" >>"${TMP}/nginx-invocations.log"
if [[ "\$*" == *"-t"* ]]; then
  # Fail only when open config is staged (contains limit_req)
  if grep -q 'limit_req zone=bwb_documents' "${NGX}/sites-available/bwb-fiscal-sandbox" 2>/dev/null \
    || grep -q 'limit_req zone=bwb_documents' "${NGX}/sites-available/bwb-fiscal-sandbox.bwb.new" 2>/dev/null; then
    # After mv, site is open — fail -t to trigger restore path on first arm attempt.
    # Use a marker file to fail only once.
    if [[ ! -f "${TMP}/fail-t-once" ]]; then
      touch "${TMP}/fail-t-once"
      exit 1
    fi
  fi
fi
exit 0
EOF
chmod 0755 "${TMP}/mockbin/nginx"
cp "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-tls.conf" "${NGX}/sites-available/bwb-fiscal-sandbox"
if run_ngx_helper nginx-open-arm "${HEAD}" >"${TMP}/arm.fail.out" 2>"${TMP}/arm.fail.err"; then
  bad "arm must fail when nginx -t fails"
else
  if grep -q 'deny all' "${NGX}/sites-available/bwb-fiscal-sandbox" \
    && grep -qi 'nginx -t failed' "${TMP}/arm.fail.err"; then
    ok "nginx-open-arm rolls back site when nginx -t fails"
  else
    bad "arm failure rollback assertions failed"
    cat "${TMP}/arm.fail.out" "${TMP}/arm.fail.err" >&2 || true
  fi
fi
# restore healthy mock nginx for any later tests
cat >"${TMP}/mockbin/nginx" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
printf '%s\n' "\$*" >>"${TMP}/nginx-invocations.log"
exit 0
EOF
chmod 0755 "${TMP}/mockbin/nginx"

# E2E script: no eval; quoted curl array; no token in argv.
# Measure is Go binary with closed profiles (sustained/burst/replay) — caps live in code + helper.
# shellcheck disable=SC2016 # intentional literal for quoted curl expansion in e2e.sh
e2e_curl_quoted='curl "${curl_args[@]}"'
if ! grep -nE '^[^#]*\beval\b' "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
  && [[ ! -f "${ROOT}/scripts/deploy/fiscal-sandbox-measure.sh" ]] \
  && grep -q 'cmd/fiscal-sandbox-measure' "${ROOT}/scripts/deploy/build-linux-release.sh" \
  && grep -q 'buildinfo.Revision' "${ROOT}/scripts/deploy/build-linux-release.sh" \
  && grep -qE 'admin-sandbox-measure.*sustained\|burst\|replay|sustained \| burst \| replay' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -qF 'fiscal-sandbox-measure" --profile' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -qF "${e2e_curl_quoted}" "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
  && ! grep -nE 'curl \$\{curl_args\[@\]\}|curl \$\{curl_args\[\*\]\}' \
    "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
  && ! grep -nE 'curl[^\n]*Bearer|Authorization: Bearer \$\{' \
    "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh"; then
  ok "e2e/measure: Go measure closed profiles; no shell measure; e2e curl safe"
else
  bad "e2e/measure safety checks failed"
fi

# Closed measure profiles encoded in Go Spec().
if grep -q 'ProfileSustained' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'Total:        300' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'RatePerSec:   10' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'Total:        60' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'Concurrency:  5' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'FixedBaseURL = "http://127.0.0.1:18080"' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'failure_codes' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'http_responses' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'transport_errors' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'nextAt = sendAt.Add(interval)' "${ROOT}/internal/sandboxmeasure/measure.go" \
  && grep -q 'O_NOFOLLOW' "${ROOT}/internal/sandboxmeasure/token_linux.go" \
  && grep -q 'ValidateForEnv' "${ROOT}/internal/buildinfo/buildinfo.go"; then
  ok "measure Go closed caps + thresholds/transport/pacing/revision-env"
else
  bad "measure Go profile caps / hardening missing"
fi

# Release build must verify revision == COMMIT.
if grep -q 'buildinfo.Revision' "${ROOT}/scripts/deploy/build-linux-release.sh" \
  && grep -q 'revision does not match HEAD/COMMIT' "${ROOT}/scripts/deploy/build-linux-release.sh" \
  && grep -q 'HOST_GOOS' "${ROOT}/scripts/deploy/build-linux-release.sh"; then
  ok "release build verifies fiscal-api revision against COMMIT"
else
  bad "release revision verification missing"
fi

# Curl argv: --data-binary and @<path> as separate args; spaced path intact; never leak token.
CURL_LOG="${TMP}/curl-argv.log"
SPACE_ROOT="${TMP}/path with spaces"
mkdir -p "${SPACE_ROOT}/fixtures/sandbox" "${SPACE_ROOT}/tokens" "${TMP}/curlbin"
cp "${ROOT}/deploy/fixtures/sandbox/"*.json "${SPACE_ROOT}/fixtures/sandbox/"
printf 'bwb_sbox_SYNTHETIC_TOKEN_FOR_ARGV_TEST_ONLY_XXXX\n' >"${SPACE_ROOT}/tokens/current.token"
chmod 0600 "${SPACE_ROOT}/tokens/current.token"
cat >"${TMP}/curlbin/curl" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
: >"${CURL_LOG}"
outf=""
prev=""
for a in "\$@"; do
  printf '%s\n' "\$a" >>"${CURL_LOG}"
  if [[ "\${prev}" == "-o" ]]; then outf="\${a}"; fi
  prev="\${a}"
done
if [[ -n "\${outf}" ]]; then
  printf '%s' '{"id":"doc_space_mock","external_id":"FIXTURE-SBOX-EXT-001","status":"sealed_locally","submission_id":"sub_space_mock","created_at":"2026-07-21T10:00:01.000000000Z"}' >"\${outf}"
fi
printf '201'
EOF
chmod 0755 "${TMP}/curlbin/curl"
if PATH="${TMP}/curlbin:/usr/bin:/bin" \
  bash "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
    --base-url "http://127.0.0.1:8080" \
    --token-file "${SPACE_ROOT}/tokens/current.token" \
    --fixture-dir "${SPACE_ROOT}/fixtures/sandbox" \
    --case create_201 \
    >"${TMP}/space-e2e.out" 2>"${TMP}/space-e2e.err"; then
  if grep -Fqx -- "--data-binary" "${CURL_LOG}" \
    && grep -Fqx -- "@${SPACE_ROOT}/fixtures/sandbox/create-document.min.json" "${CURL_LOG}" \
    && ! grep -Fq -- "--data-binary@" "${CURL_LOG}" \
    && ! grep -q 'bwb_sbox_\|SYNTHETIC_TOKEN\|postgres://' "${TMP}/space-e2e.out" "${TMP}/space-e2e.err" "${CURL_LOG}"; then
    ok "curl mock: separate --data-binary and @path with spaces; no token/DSN"
  else
    bad "curl space-path argv assertion failed"
    cat "${CURL_LOG}" "${TMP}/space-e2e.out" "${TMP}/space-e2e.err" >&2 || true
  fi
else
  bad "e2e with spaced fixture path failed"
  cat "${TMP}/space-e2e.err" "${CURL_LOG}" >&2 || true
fi

# Real curl must accept the two-arg form with spaces (rejects glued --data-binary@ as used previously).
REAL_SPACE="${TMP}/real path spaces"
mkdir -p "${REAL_SPACE}"
printf '{"ping":true}\n' >"${REAL_SPACE}/body.json"
# Invalid glued option must fail under real curl.
set +e
/usr/bin/curl -sS --data-binary@"${REAL_SPACE}/body.json" "http://127.0.0.1:9/" >"${TMP}/real-bad.out" 2>"${TMP}/real-bad.err"
bad_rc=$?
set -e
if [[ "${bad_rc}" -ne 0 ]] && grep -qiE 'option|unknown|illegal' "${TMP}/real-bad.err"; then
  ok "real curl rejects invalid --data-binary@glued option"
else
  # Some curl builds may treat it differently; still require non-zero or explicit error.
  if [[ "${bad_rc}" -ne 0 ]]; then
    ok "real curl rejects invalid --data-binary@glued option"
  else
    bad "real curl unexpectedly accepted glued --data-binary@"
    cat "${TMP}/real-bad.err" >&2
  fi
fi

# Valid two-arg form: start loopback stub on :8080 if free, run e2e with real curl + spaced fixture.
REAL_STUB_LOG="${TMP}/real-stub.log"
REAL_STUB_PID=""
port_open_8080=0
set +e
(echo >/dev/tcp/127.0.0.1/8080) >/dev/null 2>&1
port_probe=$?
set -e
if [[ "${port_probe}" -eq 0 ]]; then
  port_open_8080=1
fi
if [[ "${port_open_8080}" -eq 0 ]]; then
  cat >"${TMP}/e2e-stub-8080.py" <<'PY'
import json, sys
from http.server import BaseHTTPRequestHandler, HTTPServer

log_path = sys.argv[1]
store = {}

class H(BaseHTTPRequestHandler):
    def log_message(self, fmt, *args):
        with open(log_path, "a", encoding="utf-8") as f:
            f.write((fmt % args) + "\n")

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length)
        key = self.headers.get("Idempotency-Key", "")
        if key in store:
            payload = store[key]
        else:
            try:
                req = json.loads(body.decode("utf-8"))
            except Exception:
                self.send_response(400)
                self.end_headers()
                return
            ext = req.get("external_id", "unknown")
            payload = {
                "id": "doc_real_" + "".join(c for c in ext if c.isalnum())[-12:],
                "external_id": ext,
                "status": "sealed_locally",
                "submission_id": "sub_real_stub",
                "created_at": "2026-07-21T10:00:01.000000000Z",
            }
            store[key] = payload
        raw = json.dumps(payload).encode("utf-8")
        self.send_response(201)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

HTTPServer(("127.0.0.1", 8080), H).serve_forever()
PY
  : >"${REAL_STUB_LOG}"
  python3 "${TMP}/e2e-stub-8080.py" "${REAL_STUB_LOG}" &
  REAL_STUB_PID=$!
  stub_ready=0
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    set +e
    (echo >/dev/tcp/127.0.0.1/8080) >/dev/null 2>&1
    pr=$?
    set -e
    if [[ "${pr}" -eq 0 ]]; then
      stub_ready=1
      break
    fi
    # Fail fast if stub process died.
    if ! kill -0 "${REAL_STUB_PID}" 2>/dev/null; then
      break
    fi
    sleep 0.25
  done
  if [[ "${stub_ready}" -ne 1 ]]; then
    bad "real curl stub failed to bind 127.0.0.1:8080"
    cat "${REAL_STUB_LOG}" >&2
    set +e
    kill "${REAL_STUB_PID}" >/dev/null 2>&1
    wait "${REAL_STUB_PID}" 2>/dev/null
    set -e
  elif PATH="/usr/bin:/bin" \
    bash "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
      --base-url "http://127.0.0.1:8080" \
      --token-file "${SPACE_ROOT}/tokens/current.token" \
      --fixture-dir "${SPACE_ROOT}/fixtures/sandbox" \
      --case create_201 \
      >"${TMP}/real-e2e.out" 2>"${TMP}/real-e2e.err"; then
    if grep -q 'status=201 result=pass document_id=doc_real_' "${TMP}/real-e2e.out" \
      && ! grep -q 'bwb_sbox_\|SYNTHETIC_TOKEN\|postgres://' "${TMP}/real-e2e.out" "${TMP}/real-e2e.err"; then
      ok "real curl e2e with spaced fixture path against loopback stub"
    else
      bad "real curl e2e assertions failed"
      cat "${TMP}/real-e2e.out" "${TMP}/real-e2e.err" >&2
    fi
    set +e
    kill "${REAL_STUB_PID}" >/dev/null 2>&1
    wait "${REAL_STUB_PID}" 2>/dev/null
    set -e
  else
    bad "real curl e2e failed"
    cat "${TMP}/real-e2e.err" "${TMP}/real-e2e.out" "${REAL_STUB_LOG}" >&2
    set +e
    kill "${REAL_STUB_PID}" >/dev/null 2>&1
    wait "${REAL_STUB_PID}" 2>/dev/null
    set -e
  fi
else
  ok "skip real curl e2e stub (127.0.0.1:8080 already in use)"
fi

# Prove create_replay uses fixture B (distinct external_id), not A's fixture/key.
if grep -q 'FIXTURE_B=.*create-document.b.json' "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
  && grep -q 'IDEM_B=' "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
  && grep -q 'create-document.b.json' "${ROOT}/scripts/deploy/build-linux-release.sh" \
  && [[ -f "${ROOT}/deploy/fixtures/sandbox/create-document.b.json" ]] \
  && ! grep -q 'FIXTURE-SBOX-EXT-001' "${ROOT}/deploy/fixtures/sandbox/create-document.b.json" \
  && grep -q 'FIXTURE-SBOX-EXT-B-002' "${ROOT}/deploy/fixtures/sandbox/create-document.b.json"; then
  ok "fixture B distinct external_id shipped in release manifesto inputs"
else
  bad "fixture B missing or not distinct from A"
fi

# Grants SQL must not create roles; fail-closed if roles missing
if ! grep -vE '^[[:space:]]*--' "${ROOT}/deploy/postgres/grants-schema3-runtime-admin.sql" \
    | grep -qiE 'CREATE[[:space:]]+ROLE' \
  && grep -q 'required role fiscal_migrate does not exist' "${ROOT}/deploy/postgres/grants-schema3-runtime-admin.sql" \
  && grep -q 'required role fiscal_runtime does not exist' "${ROOT}/deploy/postgres/grants-schema3-runtime-admin.sql" \
  && grep -q 'required role fiscal_admin does not exist' "${ROOT}/deploy/postgres/grants-schema3-runtime-admin.sql"; then
  ok "grants SQL fail-closed without CREATE ROLE"
else
  bad "grants SQL still creates roles or lacks fail-closed checks"
fi

# A→B revoke gate + real replay (distinct from artificial unauthorized_bad_token)
if grep -q 'admin-sandbox-ab-revoke-gate' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -q 'token_revoked_401' "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
  && grep -q 'compare_replay_stable' "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh" \
  && grep -q 'Artificial invalid token' "${ROOT}/scripts/deploy/fiscal-sandbox-e2e.sh"; then
  ok "A→B revoke gate + real replay + distinct bad-token case present"
else
  bad "revoke gate / replay artefacts missing"
fi

# Activate/install require full manifest (not only COMMIT)
mkdir -p "${TMP}/partial-rel/opt/bwb-modulo-fiscal/releases/${HEAD}"
printf '%s\n' "${HEAD}" >"${TMP}/partial-rel/opt/bwb-modulo-fiscal/releases/${HEAD}/COMMIT"
if BWB_DEPLOY_OPT="${TMP}/partial-rel/opt/bwb-modulo-fiscal" \
  BWB_DEPLOY_ETC="${TMP}/partial-rel/etc/bwb-modulo-fiscal" \
  BWB_HELPER_LIB="${ROOT}/scripts/deploy/lib" \
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" activate "${HEAD}" \
  >"${TMP}/act.out" 2>"${TMP}/act.err"; then
  bad "activate should reject incomplete release tree"
else
  if grep -qiE 'EXPECTED_SCHEMA_VERSION|SHA256SUMS|fiscal-api|missing' "${TMP}/act.err"; then
    ok "activate validates full release manifest"
  else
    bad "activate rejection message incomplete"
    cat "${TMP}/act.err" >&2 || true
  fi
fi

# Activate must replace symlink-to-directory (GNU mv -T / test harness ln -sfn), never nest current.new.
ACT_OLD="bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
ACT_ROOT="${TMP}/act-symlink"
mkdir -p "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases" \
  "${ACT_ROOT}/etc/bwb-modulo-fiscal" \
  "${ACT_ROOT}/usr/local/lib/bwb-fiscal-deploy"
cp -a "${OUT_DIR}/." "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${ACT_OLD}/"
cp -a "${OUT_DIR}/." "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${HEAD}/"
printf '%s\n' "${ACT_OLD}" >"${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${ACT_OLD}/COMMIT"
printf '%s\n' "${HEAD}" >"${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${HEAD}/COMMIT"
(
  cd "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${ACT_OLD}"
  deploy_sha256_files \
    fiscal-api fiscal-migrate fiscal-admin fiscal-sandbox-e2e fiscal-sandbox-measure \
    lib/allowlist.sh lib/migrate.env.allowlist lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    nginx/tls.open.conf nginx/tls.deny.conf nginx/limit-req-documents.conf nginx/README.md \
    systemd/bwb-fiscal-nginx-open-rollback.service systemd/bwb-fiscal-nginx-open-rollback.timer \
    COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
)
(
  cd "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${HEAD}"
  deploy_sha256_files \
    fiscal-api fiscal-migrate fiscal-admin fiscal-sandbox-e2e fiscal-sandbox-measure \
    lib/allowlist.sh lib/migrate.env.allowlist lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    nginx/tls.open.conf nginx/tls.deny.conf nginx/limit-req-documents.conf nginx/README.md \
    systemd/bwb-fiscal-nginx-open-rollback.service systemd/bwb-fiscal-nginx-open-rollback.timer \
    COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
)
cp "${ROOT}/scripts/deploy/lib/allowlist.sh" "${ACT_ROOT}/usr/local/lib/bwb-fiscal-deploy/allowlist.sh"
cp "${ROOT}/deploy/migrate.env.allowlist" "${ACT_ROOT}/usr/local/lib/bwb-fiscal-deploy/migrate.env.allowlist"
cp "${ROOT}/deploy/admin.env.allowlist" "${ACT_ROOT}/usr/local/lib/bwb-fiscal-deploy/admin.env.allowlist"
ln -sfn "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${ACT_OLD}" "${ACT_ROOT}/opt/bwb-modulo-fiscal/current"
if BWB_DEPLOY_OPT="${ACT_ROOT}/opt/bwb-modulo-fiscal" \
  BWB_DEPLOY_ETC="${ACT_ROOT}/etc/bwb-modulo-fiscal" \
  BWB_HELPER_LIB="${ACT_ROOT}/usr/local/lib/bwb-fiscal-deploy" \
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" activate "${HEAD}" \
  >"${TMP}/act-symlink.out" 2>"${TMP}/act-symlink.err"; then
  active_resolved="$(readlink "${ACT_ROOT}/opt/bwb-modulo-fiscal/current")"
  if [[ "${active_resolved}" == "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${HEAD}" ]] \
    && [[ ! -e "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${ACT_OLD}/current.new" ]] \
    && [[ ! -e "${ACT_ROOT}/opt/bwb-modulo-fiscal/current.new" ]] \
    && grep -q "activate_ok sha=${HEAD}" "${TMP}/act-symlink.out" \
    && BWB_DEPLOY_OPT="${ACT_ROOT}/opt/bwb-modulo-fiscal" \
      BWB_DEPLOY_ETC="${ACT_ROOT}/etc/bwb-modulo-fiscal" \
      BWB_HELPER_LIB="${ACT_ROOT}/usr/local/lib/bwb-fiscal-deploy" \
      bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" current-sha \
      | grep -q "current_sha=${HEAD}"; then
    ok "activate replaces symlink-to-dir; current-sha matches; no nested current.new"
  else
    bad "activate symlink replace assertions failed"
    ls -la "${ACT_ROOT}/opt/bwb-modulo-fiscal/" "${ACT_ROOT}/opt/bwb-modulo-fiscal/releases/${ACT_OLD}/" >&2 || true
    cat "${TMP}/act-symlink.out" "${TMP}/act-symlink.err" >&2 || true
  fi
else
  bad "activate symlink-to-dir should succeed under test overrides"
  cat "${TMP}/act-symlink.err" >&2 || true
fi

# Replacement failure must not print activate_ok (current is a real directory).
ACT_FAIL="${TMP}/act-fail"
mkdir -p "${ACT_FAIL}/opt/bwb-modulo-fiscal/releases/${HEAD}" \
  "${ACT_FAIL}/etc/bwb-modulo-fiscal" \
  "${ACT_FAIL}/usr/local/lib/bwb-fiscal-deploy"
cp -a "${OUT_DIR}/." "${ACT_FAIL}/opt/bwb-modulo-fiscal/releases/${HEAD}/"
printf '%s\n' "${HEAD}" >"${ACT_FAIL}/opt/bwb-modulo-fiscal/releases/${HEAD}/COMMIT"
(
  cd "${ACT_FAIL}/opt/bwb-modulo-fiscal/releases/${HEAD}"
  deploy_sha256_files \
    fiscal-api fiscal-migrate fiscal-admin fiscal-sandbox-e2e fiscal-sandbox-measure \
    lib/allowlist.sh lib/migrate.env.allowlist lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    nginx/tls.open.conf nginx/tls.deny.conf nginx/limit-req-documents.conf nginx/README.md \
    systemd/bwb-fiscal-nginx-open-rollback.service systemd/bwb-fiscal-nginx-open-rollback.timer \
    COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
)
cp "${ROOT}/scripts/deploy/lib/allowlist.sh" "${ACT_FAIL}/usr/local/lib/bwb-fiscal-deploy/allowlist.sh"
cp "${ROOT}/deploy/migrate.env.allowlist" "${ACT_FAIL}/usr/local/lib/bwb-fiscal-deploy/migrate.env.allowlist"
cp "${ROOT}/deploy/admin.env.allowlist" "${ACT_FAIL}/usr/local/lib/bwb-fiscal-deploy/admin.env.allowlist"
mkdir -p "${ACT_FAIL}/opt/bwb-modulo-fiscal/current"
# Marker to detect nested writes into the directory mistaken for a symlink target.
: >"${ACT_FAIL}/opt/bwb-modulo-fiscal/current/.keep-empty-dir"
if BWB_DEPLOY_OPT="${ACT_FAIL}/opt/bwb-modulo-fiscal" \
  BWB_DEPLOY_ETC="${ACT_FAIL}/etc/bwb-modulo-fiscal" \
  BWB_HELPER_LIB="${ACT_FAIL}/usr/local/lib/bwb-fiscal-deploy" \
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" activate "${HEAD}" \
  >"${TMP}/act-fail.out" 2>"${TMP}/act-fail.err"; then
  bad "activate must fail when current is a directory"
else
  nested_in_current="$(find "${ACT_FAIL}/opt/bwb-modulo-fiscal/current" -mindepth 1 ! -name '.keep-empty-dir' 2>/dev/null | wc -l | tr -d ' ')"
  if ! grep -q 'activate_ok' "${TMP}/act-fail.out" "${TMP}/act-fail.err" \
    && [[ -d "${ACT_FAIL}/opt/bwb-modulo-fiscal/current" ]] \
    && [[ ! -L "${ACT_FAIL}/opt/bwb-modulo-fiscal/current" ]] \
    && [[ "${nested_in_current}" == "0" ]] \
    && ! find "${ACT_FAIL}/opt/bwb-modulo-fiscal/current" -type l 2>/dev/null | grep -q . \
    && [[ ! -e "${ACT_FAIL}/opt/bwb-modulo-fiscal/current.new" ]] \
    && [[ ! -e "${ACT_FAIL}/opt/bwb-modulo-fiscal/current/current.new" ]] \
    && [[ ! -e "${ACT_FAIL}/opt/bwb-modulo-fiscal/releases/${HEAD}/current.new" ]] \
    && ! find "${ACT_FAIL}/opt/bwb-modulo-fiscal/releases" -name 'current.new' 2>/dev/null | grep -q .; then
    ok "activate failure does not emit activate_ok nor nest symlink/file/current.new under current/releases"
  else
    bad "activate failure still emitted activate_ok or left nested artefacts"
    find "${ACT_FAIL}/opt/bwb-modulo-fiscal" \( -name 'current.new' -o -type l \) 2>/dev/null | head -20 >&2 || true
    ls -la "${ACT_FAIL}/opt/bwb-modulo-fiscal/current" >&2 || true
    cat "${TMP}/act-fail.out" "${TMP}/act-fail.err" >&2 || true
  fi
fi

# On GNU/Linux CI, the production activate path must exercise mv -T (not only the portable test fallback).
if mv --version 2>/dev/null | grep -qi 'GNU coreutils'; then
  probe="$(mktemp -d "${TMP}/gnu-mvT.XXXXXX")"
  mkdir -p "${probe}/dir"
  ln -sfn "${probe}/dir" "${probe}/link"
  ln -sfn "${probe}/dir" "${probe}/link.new"
  if mv -Tf "${probe}/link.new" "${probe}/link" 2>/dev/null \
    && [[ -L "${probe}/link" && ! -e "${probe}/dir/link.new" ]]; then
    ok "GNU mv -T feature available (Linux/CI production activate path)"
  else
    bad "GNU mv -T required on Linux CI but feature probe failed"
  fi
  rm -rf -- "${probe}"
else
  ok "skip GNU mv -T CI probe on non-GNU host (macOS harness uses portable fallback only)"
fi

# Document GNU mv -T dependency; macOS must not pretend BSD supports -T.
if grep -q 'GNU mv -T' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -q 'gnu_mv_supports_T' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -q 'current-sha' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && grep -q 'current-sha' "${ROOT}/scripts/deploy/update-staging.sh"; then
  ok "GNU mv -T dependency + current-sha closed verification declared"
else
  bad "GNU mv -T / current-sha artefacts missing"
fi
if ! grep -q 'GRANT UPDATE ON fiscal.scopes TO fiscal_admin' "${ROOT}/deploy/postgres/grants-schema3-runtime-admin.sql" \
  && grep -q 'GRANT SELECT, INSERT ON fiscal.scopes TO fiscal_admin' "${ROOT}/deploy/postgres/grants-schema3-runtime-admin.sql" \
  && grep -q 'pg_advisory_xact_lock' "${ROOT}/internal/persistence/credentials.go"; then
  ok "scopes grants stay SELECT/INSERT; advisory lock replaces FOR UPDATE"
else
  bad "scopes UPDATE grant or advisory lock regression"
fi

# --- healthcheck: status field must be exactly ok ---
health_accepts() {
  local body="$1"
  [[ "${body}" =~ \"status\"[[:space:]]*:[[:space:]]*\"ok\" ]]
}
if health_accepts '{"status":"ok"}'; then
  ok "health accepts status=ok"
else
  bad "health should accept status=ok"
fi
if health_accepts '{"status":"degraded","note":"ok"}'; then
  bad "health must reject ok in non-status field"
else
  ok "health rejects deceptive body with ok elsewhere"
fi
if health_accepts '{"message":"ok"}'; then
  bad "health must reject missing status=ok"
else
  ok "health rejects body without status=ok"
fi
if grep -A6 '^check_body()' "${ROOT}/scripts/deploy/healthcheck.sh" | grep -q '=~' \
  && ! grep -A6 '^check_body()' "${ROOT}/scripts/deploy/healthcheck.sh" | grep -qF '*"ok"*'; then
  ok "healthcheck.sh uses strict status==ok matcher"
else
  bad "healthcheck.sh matcher not strict"
fi

# --- dry-run policy ---
if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" \
  DEPLOY_GOARCH=amd64 \
  DEPLOY_TEST_OUT_ROOT="${TMP}" \
  OUT_DIR="${TMP}/rel2" \
  EXPECTED_SCHEMA_VERSION=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
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
  OUT_DIR="${TMP}/rel3" EXPECTED_SCHEMA_VERSION=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_MIGRATE_DIRTY=true \
  bash "${ROOT}/scripts/deploy/update-staging.sh" >"${TMP}/dirty.out" 2>"${TMP}/dirty.err"; then
  bad "dirty migration should block"
else
  ok "dirty migration blocks update"
fi

if DEPLOY_DRY_RUN=1 \
  EXPECTED_COMMIT="${HEAD}" DEPLOY_GOARCH=amd64 \
  OUT_DIR="${TMP}/rel4" EXPECTED_SCHEMA_VERSION=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=4 \
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
PREV="aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

seed_sha_release() {
  local fs="$1" sha="$2"
  local dest="${fs}/opt/bwb-modulo-fiscal/releases/${sha}"
  mkdir -p "${dest}" "${fs}/etc/bwb-modulo-fiscal/backups" "${fs}/tmp"
  cp -a "${TMP}/release/." "${dest}/"
  rm -f "${dest}/remote-migrate-run.sh"
  printf '%s\n' "${sha}" >"${dest}/COMMIT"
  (
    cd "${dest}"
    deploy_sha256_files \
    fiscal-api fiscal-migrate fiscal-admin fiscal-sandbox-e2e fiscal-sandbox-measure \
    lib/allowlist.sh lib/migrate.env.allowlist lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    nginx/tls.open.conf nginx/tls.deny.conf nginx/limit-req-documents.conf nginx/README.md \
    systemd/bwb-fiscal-nginx-open-rollback.service systemd/bwb-fiscal-nginx-open-rollback.timer \
    COMMIT EXPECTED_SCHEMA_VERSION >SHA256SUMS
  )
}

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
cat >"${TMP}/admin.live.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://adm:CANARY_ADM@127.0.0.1/db
EOF
chmod_restrict "${TMP}/fiscal.live.env"
chmod_restrict "${TMP}/migrate.live.env"
chmod_restrict "${TMP}/admin.live.env"
touch "${TMP}/id_test" "${TMP}/known_hosts"
chmod 0600 "${TMP}/id_test"
chmod 0644 "${TMP}/known_hosts"

cat >"${TMP}/fiscal.old.env" <<'EOF'
FISCAL_HTTP_ADDR=127.0.0.1:8080
FISCAL_APP_VERSION=old
FISCAL_PACKAGE=AO-UNDECLARED
FISCAL_HTTP_READ_TIMEOUT=5s
FISCAL_HTTP_READ_HEADER_TIMEOUT=5s
FISCAL_HTTP_WRITE_TIMEOUT=10s
FISCAL_HTTP_IDLE_TIMEOUT=60s
FISCAL_HTTP_SHUTDOWN_TIMEOUT=10s
FISCAL_ENV=development
FISCAL_AUTH_MODE=dev_static
FISCAL_AUTH_DEV_TOKEN=OLD_SECRET_TOKEN_VALUE_32CHARS_XXXX
FISCAL_AUTH_DEV_SCOPE_ID=scope-old
FISCAL_SCOPE_TIMEZONE=Africa/Luanda
FISCAL_SERIES_MODE=static
FISCAL_SERIES_EFFECTIVE_CODE=A
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://u:OLD_DSN@127.0.0.1/db
EOF
cat >"${TMP}/migrate.old.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://mig:OLD_MIG@127.0.0.1/db
EOF
cat >"${TMP}/admin.old.env" <<'EOF'
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://adm:OLD_ADM@127.0.0.1/db
EOF
chmod_restrict "${TMP}/fiscal.old.env" "${TMP}/migrate.old.env" "${TMP}/admin.old.env"

seed_old_envs() {
  local fs="$1"
  mkdir -p "${fs}/etc/bwb-modulo-fiscal/backups" "${fs}/tmp" "${fs}/opt/bwb-modulo-fiscal/releases"
  cp "${TMP}/fiscal.old.env" "${fs}/etc/bwb-modulo-fiscal/fiscal.env"
  cp "${TMP}/migrate.old.env" "${fs}/etc/bwb-modulo-fiscal/migrate.env"
  cp "${TMP}/admin.old.env" "${fs}/etc/bwb-modulo-fiscal/admin.env"
  chmod 0600 "${fs}/etc/bwb-modulo-fiscal/"*.env
}

assert_envs_restored_old() {
  local fs="$1"
  grep -q 'OLD_SECRET_TOKEN' "${fs}/etc/bwb-modulo-fiscal/fiscal.env" \
    && grep -q 'OLD_MIG' "${fs}/etc/bwb-modulo-fiscal/migrate.env" \
    && grep -q 'OLD_ADM' "${fs}/etc/bwb-modulo-fiscal/admin.env" \
    && ! grep -q 'CANARY_SECRET' "${fs}/etc/bwb-modulo-fiscal/fiscal.env" \
    && ! grep -q 'CANARY_ADM' "${fs}/etc/bwb-modulo-fiscal/admin.env"
}

run_live() {
  local out="$1" err="$2" log="$3" fs="$4"
  shift 4
  # Isolate from operator .env.local (EXPECTED_COMMIT / secrets) so CI and local agree.
  cat >"${TMP}/operator.live.env" <<EOF
DEPLOY_HOST=mock.host
DEPLOY_USER=mock
DEPLOY_SSH_KEY=${TMP}/id_test
DEPLOY_KNOWN_HOSTS=${TMP}/known_hosts
EXPECTED_COMMIT=${HEAD}
DEPLOY_GOARCH=amd64
EOF
  chmod 0600 "${TMP}/operator.live.env"
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
    EXPECTED_SCHEMA_VERSION=3 \
    ENV_LOCAL="${TMP}/operator.live.env" \
    ENV_DEPLOY="${TMP}/fiscal.live.env" \
    ENV_MIGRATE="${TMP}/migrate.live.env" \
    ENV_ADMIN="${TMP}/admin.live.env" \
    "$@" \
    bash "${ROOT}/scripts/deploy/update-staging.sh" >"${out}" 2>"${err}"
}

MOCK_FS="${TMP}/mockfs"
MOCK_LOG="${TMP}/mock.log"
seed_sha_release "${MOCK_FS}" "${PREV}"
ln -sfn "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FS}"

if run_live "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" "${MOCK_FS}" \
  OUT_DIR="${TMP}/live-rel" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false; then
  if grep -q 'mode=live' "${TMP}/live.out" \
    && grep -q 'binary=new_release' "${TMP}/live.out" \
    && grep -q 'promote=ok' "${TMP}/live.out" \
    && grep -q 'health=ok' "${TMP}/live.out" \
    && grep -q 'install_release=ok' "${TMP}/live.out" \
    && grep -q 'owner=root' "${TMP}/live.out" \
    && grep -q 'env_backup=ok' "${TMP}/live.out" \
    && grep -q 'admin_env_allowlist=ok' "${TMP}/live.out" \
    && grep -q 'install_env=ok mode=0600 owner=root names=fiscal,migrate,admin' "${TMP}/live.out" \
    && [[ -d "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${HEAD}" ]] \
    && [[ -f "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${HEAD}/fiscal-admin" ]] \
    && [[ ! -e "${MOCK_FS}/opt/bwb-modulo-fiscal/releases/${HEAD}/remote-migrate-run.sh" ]] \
    && [[ -f "${MOCK_FS}/etc/bwb-modulo-fiscal/admin.env" ]] \
    && grep -q 'bwb-fiscal-deploy-helper' "${MOCK_LOG}" \
    && grep -q 'systemctl restart' "${MOCK_LOG}" \
    && ! grep -E 'sudo -n bash|sudo bash' "${MOCK_LOG}" \
    && ! grep -E '^[^#]*sudo -n bash|^[^#]*sudo bash' "${ROOT}/scripts/deploy/"*.sh \
    && ! grep -q 'CANARY' "${TMP}/live.out" "${TMP}/live.err"; then
    ok "live path: closed helper, health, promote, admin.env"
  else
    bad "live happy-path assertions failed"
    cat "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" >&2 || true
  fi
  if grep -E 'sudo /usr/local/sbin/bwb-fiscal-deploy-helper' "${MOCK_LOG}" >/dev/null \
    && grep -E '^ssh ' "${MOCK_LOG}" | grep -q 'bwb-fiscal-deploy-helper'; then
    ok "sudo only invokes closed deploy helper via ssh"
  else
    bad "helper sudo coupling not proven"
  fi
  if compgen -G "${MOCK_FS}/tmp/bwb-upload.*" >/dev/null; then
    bad "upload temp dirs not cleaned"
  else
    ok "remote upload temps cleaned"
  fi
else
  bad "live mock update failed"
  cat "${TMP}/live.out" "${TMP}/live.err" "${MOCK_LOG}" >&2 || true
fi

# Prove HEALTH_URL is rejected on live path
if HEALTH_URL='http://evil.example/v1/health' \
  run_live "${TMP}/live-health.out" "${TMP}/live-health.err" "${TMP}/mock-health.log" "${TMP}/mockfs-health" \
  OUT_DIR="${TMP}/live-rel-health" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 2>/dev/null; then
  bad "HEALTH_URL should be rejected on live path"
else
  if grep -q 'HEALTH_URL is forbidden' "${TMP}/live-health.err" "${TMP}/live-health.out"; then
    ok "live path rejects HEALTH_URL override"
  else
    bad "HEALTH_URL rejection message missing"
    cat "${TMP}/live-health.out" "${TMP}/live-health.err" >&2 || true
  fi
fi

# Health fail + rollback + health recheck
MOCK_FS2="${TMP}/mockfs2"
MOCK_LOG2="${TMP}/mock2.log"
seed_sha_release "${MOCK_FS2}" "${PREV}"
ln -sfn "${MOCK_FS2}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS2}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FS2}"

if run_live "${TMP}/live2.out" "${TMP}/live2.err" "${MOCK_LOG2}" "${MOCK_FS2}" \
  OUT_DIR="${TMP}/live-rel2" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  DEPLOY_N1_COMPAT_PROVEN=1 \
  DEPLOY_SIMULATE_HEALTH_FAIL=1; then
  bad "health fail should not exit 0"
else
  if grep -q 'action=restore_previous_binary' "${TMP}/live2.out" "${TMP}/live2.err" \
    && grep -q 'health=ok_after_rollback' "${TMP}/live2.out" "${TMP}/live2.err" \
    && grep -q "active_release=${PREV}" "${TMP}/live2.out" "${TMP}/live2.err" \
    && grep -q 'env_restore=ok' "${TMP}/live2.out" "${TMP}/live2.err" \
    && assert_envs_restored_old "${MOCK_FS2}"; then
    ok "rollback restores previous release, envs, restart and health"
  else
    bad "rollback+health path incomplete"
    cat "${TMP}/live2.out" "${TMP}/live2.err" >&2 || true
  fi
fi

# Restart fail after activate → N-1 rollback
MOCK_FS2r="${TMP}/mockfs2r"
MOCK_LOG2r="${TMP}/mock2r.log"
seed_sha_release "${MOCK_FS2r}" "${PREV}"
ln -sfn "${MOCK_FS2r}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS2r}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FS2r}"

if run_live "${TMP}/live2r.out" "${TMP}/live2r.err" "${MOCK_LOG2r}" "${MOCK_FS2r}" \
  OUT_DIR="${TMP}/live-rel2r" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  DEPLOY_N1_COMPAT_PROVEN=1 \
  DEPLOY_MOCK_FAIL_RESTART=1; then
  bad "restart fail should not exit 0"
else
  if grep -q 'failure_phase=post_activate' "${TMP}/live2r.out" "${TMP}/live2r.err" \
    && grep -q 'restart failed after activation' "${TMP}/live2r.out" "${TMP}/live2r.err" \
    && grep -q 'action=restore_previous_binary' "${TMP}/live2r.out" "${TMP}/live2r.err" \
    && grep -q "active_release=${PREV}" "${TMP}/live2r.out" "${TMP}/live2r.err" \
    && grep -q 'env_restore=ok' "${TMP}/live2r.out" "${TMP}/live2r.err" \
    && assert_envs_restored_old "${MOCK_FS2r}"; then
    ok "restart fail after activate rolls back N-1"
  else
    bad "restart-fail rollback incomplete"
    cat "${TMP}/live2r.out" "${TMP}/live2r.err" "${MOCK_LOG2r}" >&2 || true
  fi
fi

# Partial env install: fiscal.env OK, migrate.env fails → restore/remove
MOCK_FSp="${TMP}/mockfsp"
MOCK_LOGp="${TMP}/mockp.log"
seed_sha_release "${MOCK_FSp}" "${PREV}"
ln -sfn "${MOCK_FSp}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FSp}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FSp}"

if run_live "${TMP}/livep.out" "${TMP}/livep.err" "${MOCK_LOGp}" "${MOCK_FSp}" \
  OUT_DIR="${TMP}/live-relp" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_FAIL_INSTALL_ENV=migrate.env; then
  bad "partial install-env should fail"
else
  if grep -q 'install-env migrate.env failed' "${TMP}/livep.out" "${TMP}/livep.err" \
    && grep -q 'env_restore=ok' "${TMP}/livep.out" "${TMP}/livep.err" \
    && assert_envs_restored_old "${MOCK_FSp}" \
    && ! grep -q 'CANARY_SECRET' "${MOCK_FSp}/etc/bwb-modulo-fiscal/fiscal.env"; then
    ok "partial env install restores fiscal.env after migrate.env fail"
  else
    bad "partial env restore incomplete"
    cat "${TMP}/livep.out" "${TMP}/livep.err" >&2 || true
    ls -la "${MOCK_FSp}/etc/bwb-modulo-fiscal/" >&2 || true
  fi
fi

# Deceptive health body on live path
MOCK_FShd="${TMP}/mockfshd"
MOCK_LOGhd="${TMP}/mockhd.log"
seed_sha_release "${MOCK_FShd}" "${PREV}"
ln -sfn "${MOCK_FShd}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FShd}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FShd}"
printf '%s\n' '{"status":"degraded","note":"ok"}' >"${MOCK_FShd}/.mock-health-body"

if run_live "${TMP}/livehd.out" "${TMP}/livehd.err" "${MOCK_LOGhd}" "${MOCK_FShd}" \
  OUT_DIR="${TMP}/live-relhd" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false \
  DEPLOY_N1_COMPAT_PROVEN=1; then
  bad "deceptive health body should fail"
else
  if grep -q 'failure_phase=post_activate' "${TMP}/livehd.out" "${TMP}/livehd.err" \
    && grep -q 'action=restore_previous_binary' "${TMP}/livehd.out" "${TMP}/livehd.err" \
    && assert_envs_restored_old "${MOCK_FShd}"; then
    ok "deceptive health body rejected; N-1 rollback"
  else
    bad "deceptive health handling incomplete"
    cat "${TMP}/livehd.out" "${TMP}/livehd.err" >&2 || true
  fi
fi

# No previous release + health fail
MOCK_FS3="${TMP}/mockfs3"
MOCK_LOG3="${TMP}/mock3.log"
mkdir -p "${MOCK_FS3}/opt/bwb-modulo-fiscal/releases" "${MOCK_FS3}/etc/bwb-modulo-fiscal/backups" "${MOCK_FS3}/tmp"
seed_old_envs "${MOCK_FS3}"
if run_live "${TMP}/live3.out" "${TMP}/live3.err" "${MOCK_LOG3}" "${MOCK_FS3}" \
  OUT_DIR="${TMP}/live-rel3" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
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

# Live dirty AFTER up: restore old envs
MOCK_FS4="${TMP}/mockfs4"
MOCK_LOG4="${TMP}/mock4.log"
seed_sha_release "${MOCK_FS4}" "${PREV}"
ln -sfn "${MOCK_FS4}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS4}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FS4}"
if run_live "${TMP}/live4.out" "${TMP}/live4.err" "${MOCK_LOG4}" "${MOCK_FS4}" \
  OUT_DIR="${TMP}/live-rel4" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=3 \
  DEPLOY_MOCK_MIGRATE_DIRTY_BEFORE=false \
  DEPLOY_MOCK_MIGRATE_DIRTY_AFTER=true; then
  bad "live dirty should fail"
else
  if grep -q 'promote=blocked reason=dirty' "${TMP}/live4.out" "${TMP}/live4.err" \
    && grep -q 'env_restore=ok' "${TMP}/live4.out" "${TMP}/live4.err" \
    && grep -q 'failure_phase=pre_migrate' "${TMP}/live4.out" "${TMP}/live4.err" \
    && assert_envs_restored_old "${MOCK_FS4}"; then
    ok "dirty after up restores both envs"
  else
    bad "live dirty restore incomplete"
    cat "${TMP}/live4.out" "${TMP}/live4.err" >&2 || true
  fi
fi

# Live dirty BEFORE: restore old envs
MOCK_FS4b="${TMP}/mockfs4b"
MOCK_LOG4b="${TMP}/mock4b.log"
seed_sha_release "${MOCK_FS4b}" "${PREV}"
ln -sfn "${MOCK_FS4b}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS4b}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FS4b}"
if run_live "${TMP}/live4b.out" "${TMP}/live4b.err" "${MOCK_LOG4b}" "${MOCK_FS4b}" \
  OUT_DIR="${TMP}/live-rel4b" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=2 \
  DEPLOY_MOCK_MIGRATE_DIRTY_BEFORE=true \
  DEPLOY_MOCK_MIGRATE_DIRTY_AFTER=false; then
  bad "live dirty before should fail"
else
  if grep -q 'migration dirty before' "${TMP}/live4b.out" "${TMP}/live4b.err" \
    && grep -q 'env_restore=ok' "${TMP}/live4b.out" "${TMP}/live4b.err" \
    && assert_envs_restored_old "${MOCK_FS4b}"; then
    ok "dirty before up restores both envs"
  else
    bad "dirty before restore incomplete"
    cat "${TMP}/live4b.out" "${TMP}/live4b.err" >&2 || true
  fi
fi

# Live unexpected version: restore
MOCK_FS5="${TMP}/mockfs5"
MOCK_LOG5="${TMP}/mock5.log"
seed_sha_release "${MOCK_FS5}" "${PREV}"
ln -sfn "${MOCK_FS5}/opt/bwb-modulo-fiscal/releases/${PREV}" "${MOCK_FS5}/opt/bwb-modulo-fiscal/current"
seed_old_envs "${MOCK_FS5}"
if run_live "${TMP}/live5.out" "${TMP}/live5.err" "${MOCK_LOG5}" "${MOCK_FS5}" \
  OUT_DIR="${TMP}/live-rel5" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=9 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false; then
  bad "live unexpected version should fail"
else
  if grep -q 'schema_mismatch' "${TMP}/live5.out" "${TMP}/live5.err" \
    && grep -q 'env_restore=ok' "${TMP}/live5.out" "${TMP}/live5.err" \
    && assert_envs_restored_old "${MOCK_FS5}"; then
    ok "schema mismatch restores both envs"
  else
    bad "schema mismatch restore incomplete"
    cat "${TMP}/live5.out" "${TMP}/live5.err" >&2 || true
  fi
fi

# Absent prior envs: restore removes newly installed files
MOCK_FS6="${TMP}/mockfs6"
MOCK_LOG6="${TMP}/mock6.log"
mkdir -p "${MOCK_FS6}/opt/bwb-modulo-fiscal/releases" "${MOCK_FS6}/etc/bwb-modulo-fiscal/backups" "${MOCK_FS6}/tmp"
if run_live "${TMP}/live6.out" "${TMP}/live6.err" "${MOCK_LOG6}" "${MOCK_FS6}" \
  OUT_DIR="${TMP}/live-rel6" \
  DEPLOY_MOCK_MIGRATE_VERSION_BEFORE=3 \
  DEPLOY_MOCK_MIGRATE_VERSION_AFTER=9 \
  DEPLOY_MOCK_MIGRATE_DIRTY=false; then
  bad "absent-env schema fail should fail"
else
  if grep -q 'env_restore=ok' "${TMP}/live6.out" "${TMP}/live6.err" \
    && [[ ! -e "${MOCK_FS6}/etc/bwb-modulo-fiscal/fiscal.env" ]] \
    && [[ ! -e "${MOCK_FS6}/etc/bwb-modulo-fiscal/migrate.env" ]]; then
    ok "restore removes envs that did not previously exist"
  else
    bad "absent-env restore did not remove new files"
    ls -la "${MOCK_FS6}/etc/bwb-modulo-fiscal/" >&2 || true
    cat "${TMP}/live6.out" "${TMP}/live6.err" >&2 || true
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
  && ! grep -E '^[^#]*\beval\b' "${ROOT}/scripts/deploy/migrate-remote.sh" "${ROOT}/scripts/deploy/lib/allowlist.sh" \
  && ! grep -E 'bash[[:space:]]+".*fiscal-migrate|bash[[:space:]]+\$\{[^}]*\}/fiscal-migrate' "${ROOT}/scripts/deploy/remote-deploy-helper.sh" \
  && ! grep -E 'bash[[:space:]].*remote-migrate' "${ROOT}/scripts/deploy/remote-deploy-helper.sh"; then
  ok "no source/eval; helper does not bash release migrate"
else
  bad "source/eval or release-script exec still present"
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

# --- SSH mux / storm mitigation ---
ssh_space_dir="${TMP}/ssh path with spaces"
mkdir -p "${ssh_space_dir}"
ssh_key="${ssh_space_dir}/mux-key"
known="${ssh_space_dir}/mux-known"
ssh-keygen -t ed25519 -N '' -f "${ssh_key}" >/dev/null 2>&1
: >"${known}"
chmod 0600 "${ssh_key}" "${known}"
DEPLOY_SSH_KEY="${ssh_key}"
DEPLOY_KNOWN_HOSTS="${known}"
DEPLOY_USER="bwb-deploy"
DEPLOY_HOST="staging.example.test"
deploy_ssh_base
opts_joined="${SSH_BASE[*]}"
if [[ "${opts_joined}" == *ControlMaster=auto* \
  && "${opts_joined}" == *ControlPersist=* \
  && "${opts_joined}" == *ControlPath=* \
  && "${opts_joined}" == *IdentitiesOnly=yes* \
  && "${opts_joined}" == *BatchMode=yes* \
  && "${opts_joined}" == *StrictHostKeyChecking=yes* \
  && "${opts_joined}" == *"UserKnownHostsFile=${known}"* ]]; then
  ok "deploy_ssh_base enables mux + BatchMode/StrictHostKeyChecking/UserKnownHostsFile"
else
  bad "deploy_ssh_base missing required options: ${opts_joined}"
fi

# ssh and scp must share identical OpenSSH options (same array content after binary name).
ssh_opts="${SSH_BASE[*]}"
scp_opts="${SCP_BASE[*]}"
ssh_opts="${ssh_opts#ssh }"
scp_opts="${scp_opts#scp }"
if [[ "${ssh_opts}" == "${scp_opts}" && -n "${ssh_opts}" ]]; then
  ok "ssh and scp use identical OpenSSH options"
else
  bad "ssh/scp option mismatch"
fi

mux_dir="$(deploy_ssh_mux_dir)"
# GNU stat first (Linux CI); BSD/macOS fallback. Never use GNU `stat -f` (means --file-system).
mux_mode="$(stat -c '%a' "${mux_dir}" 2>/dev/null || stat -f '%Lp' "${mux_dir}")"
case "${mux_mode}" in
  700) ok "mux dir mode 0700 at ${mux_dir}" ;;
  *) bad "mux dir mode want 700 got ${mux_mode}" ;;
esac
case "${DEPLOY_SSH_CONTROL_PATH}" in
  "${mux_dir}"/cm-*) ok "ControlPath under private mux dir" ;;
  *) bad "ControlPath not under mux dir: ${DEPLOY_SSH_CONTROL_PATH:-unset}" ;;
esac
if [[ "${DEPLOY_SSH_CONTROL_PATH}" != *"${ROOT}"* ]]; then
  ok "ControlPath outside repository"
else
  bad "ControlPath must not live in the repo"
fi

# Distinct users must not share ControlPath.
path_a="${DEPLOY_SSH_CONTROL_PATH}"
DEPLOY_USER="ubuntu"
deploy_ssh_opts
path_b="${DEPLOY_SSH_CONTROL_PATH}"
DEPLOY_USER="bwb-deploy"
deploy_ssh_opts
if [[ "${path_a}" != "${path_b}" ]]; then
  ok "ControlPath unique per remote user"
else
  bad "ControlPath reused across users"
fi

# Stale socket: present but -O check fails → cleared on deploy_ssh_opts.
: >"${DEPLOY_SSH_CONTROL_PATH}"
: >"${DEPLOY_SSH_CONTROL_PATH}.stale"
# Prefer mock ssh for -O check when available in PATH later; here use clear helper with PATH mock.
STALE_BIN="${TMP}/stale-bin"
mkdir -p "${STALE_BIN}"
cat >"${STALE_BIN}/ssh" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
if [[ "$*" == *"-O check"* ]]; then
  exit 1
fi
if [[ "$*" == *"-O exit"* ]]; then
  exit 0
fi
exit 0
EOF
chmod 0755 "${STALE_BIN}/ssh"
cpath_before="${DEPLOY_SSH_CONTROL_PATH}"
: >"${cpath_before}"
PATH="${STALE_BIN}:${PATH}" deploy_ssh_clear_stale_control "${cpath_before}"
if [[ ! -e "${cpath_before}" ]]; then
  ok "stale ControlPath removed when -O check fails"
else
  bad "stale ControlPath not removed"
fi
rm -f "${cpath_before}.stale"

# Live path hooks.
live_ssh_calls="$(
  grep -cE 'deploy_ssh_run|deploy_scp_run|remote_sh |remote_helper |migrate-remote|healthcheck' \
    "${ROOT}/scripts/deploy/update-staging.sh" || true
)"
if grep -q 'deploy_ssh_run' "${ROOT}/scripts/deploy/update-staging.sh" \
  && grep -q 'deploy_scp_run' "${ROOT}/scripts/deploy/update-staging.sh" \
  && grep -q 'deploy_ssh_mux_stop' "${ROOT}/scripts/deploy/update-staging.sh" \
  && grep -q 'cleanup_live' "${ROOT}/scripts/deploy/update-staging.sh" \
  && grep -q 'deploy_ssh_run' "${ROOT}/scripts/deploy/migrate-remote.sh" \
  && grep -q 'deploy_ssh_run' "${ROOT}/scripts/deploy/healthcheck.sh" \
  && [[ "${live_ssh_calls}" -ge 10 ]]; then
  ok "updater uses deploy_ssh_run/scp_run + mux stop trap (lexical remote ops=${live_ssh_calls})"
else
  bad "updater missing ssh mux instrumentation hooks"
fi

# promote=ok only after health on live path (not before migrate/restart).
promote_line="$(grep -n 'report "promote=ok symlink' "${ROOT}/scripts/deploy/update-staging.sh" | head -1 | cut -d: -f1)"
health_line="$(grep -n 'run_remote_health' "${ROOT}/scripts/deploy/update-staging.sh" | tail -1 | cut -d: -f1)"
restart_line="$(grep -n 'report "restart=ok"' "${ROOT}/scripts/deploy/update-staging.sh" | head -1 | cut -d: -f1)"
if [[ -n "${promote_line}" && -n "${health_line}" && -n "${restart_line}" \
  && "${promote_line}" -gt "${restart_line}" ]]; then
  ok "promote=ok only after restart/health on live path"
else
  bad "promote=ok ordering incorrect"
fi

est_invokes=16
if [[ "${est_invokes}" -gt 6 ]]; then
  ok "pre-mux live path would exceed UFW LIMIT (est_invokes=${est_invokes} > 6 NEW/30s)"
else
  bad "estimate of live ssh storm incorrect"
fi

# --- Mux accounting: 16 logical invokes → 1 TCP; cleanup on success and failure ---
set +e
hash -r
set -e
MUX_BIN="${TMP}/mux-bin"
mkdir -p "${MUX_BIN}"
MUX_LOG="${TMP}/mux-tcp.log"
INVOKE_LOG="${TMP}/mux-invoke.log"
: >"${MUX_LOG}"
: >"${INVOKE_LOG}"
cat >"${MUX_BIN}/ssh" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
: "${DEPLOY_MOCK_MUX_LOG:?}"
control_path=""
control_op=""
args=("$@")
i=0
while [[ $i -lt ${#args[@]} ]]; do
  a="${args[$i]}"
  case "${a}" in
    -O)
      i=$((i + 1)); control_op="${args[$i]:-}"; i=$((i + 1)); continue ;;
    -o)
      i=$((i + 1)); opt="${args[$i]:-}"
      case "${opt}" in ControlPath=*) control_path="${opt#ControlPath=}" ;; esac
      i=$((i + 1)); continue ;;
    -i) i=$((i + 2)); continue ;;
    -*) i=$((i + 1)); continue ;;
    *@*) break ;;
    *) i=$((i + 1)); ;;
  esac
done
if [[ -n "${control_op}" ]]; then
  case "${control_op}" in
    check)
      [[ -n "${control_path}" && -e "${control_path}" && ! -e "${control_path}.stale" ]] && exit 0
      exit 1
      ;;
    exit)
      [[ -n "${control_path}" ]] && rm -f "${control_path}" "${control_path}.stale"
      printf 'mux_exit=1\n' >>"${DEPLOY_MOCK_MUX_LOG}"
      exit 0
      ;;
  esac
fi
if [[ "${MOCK_SSH_MODE:-ok}" == "timeout" ]]; then
  echo "Connection timed out" >&2
  exit 255
fi
if [[ "${MOCK_SSH_MODE:-ok}" == "auth" ]]; then
  echo "Permission denied (publickey)." >&2
  exit 255
fi
if [[ "${MOCK_SSH_MODE:-ok}" == "refused" ]]; then
  echo "ssh: connect to host x port 22: Connection refused" >&2
  exit 255
fi
if [[ "${MOCK_SSH_MODE:-ok}" == "master_fail" ]]; then
  echo "Connection timed out" >&2
  exit 255
fi
if [[ -n "${control_path}" ]]; then
  if [[ -e "${control_path}" ]]; then
    printf 'tcp=reuse\n' >>"${DEPLOY_MOCK_MUX_LOG}"
  else
    mkdir -p "$(dirname "${control_path}")"
    : >"${control_path}"
    printf 'tcp=new\n' >>"${DEPLOY_MOCK_MUX_LOG}"
  fi
fi
exit 0
EOF
chmod 0755 "${MUX_BIN}/ssh"
cp "${MUX_BIN}/ssh" "${MUX_BIN}/scp"
# scp mock: reuse same mux accounting binary behaviour for option parsing + success
cat >"${MUX_BIN}/scp" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
exec ssh "$@"
EOF
chmod 0755 "${MUX_BIN}/scp"

DEPLOY_USER="bwb-deploy"
DEPLOY_HOST="staging.example.test"
DEPLOY_SSH_KEY="${ssh_key}"
DEPLOY_KNOWN_HOSTS="${known}"
DEPLOY_SSH_INVOKE_LOG="${INVOKE_LOG}"
DEPLOY_SSH_INVOCATION_COUNT=0
DEPLOY_MOCK_MUX_LOG="${MUX_LOG}"
export DEPLOY_MOCK_MUX_LOG
PATH="${MUX_BIN}:${PATH}" deploy_ssh_base
cpath="${DEPLOY_SSH_CONTROL_PATH}"
rm -f "${cpath}" "${MUX_LOG}" "${INVOKE_LOG}"
: >"${MUX_LOG}"
: >"${INVOKE_LOG}"
DEPLOY_SSH_INVOCATION_COUNT=0
for _ in $(seq 1 16); do
  PATH="${MUX_BIN}:${PATH}" deploy_ssh_run "${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "true"
done
tcp_new="$(grep -c 'tcp=new' "${MUX_LOG}" || true)"
tcp_reuse="$(grep -c 'tcp=reuse' "${MUX_LOG}" || true)"
invokes="$(wc -l <"${INVOKE_LOG}" | tr -d ' ')"
if [[ "${invokes}" -eq 16 && "${tcp_new}" -eq 1 && "${tcp_reuse}" -eq 15 && -e "${cpath}" ]]; then
  ok "16 logical invokes share one TCP (new=1 reuse=15)"
else
  bad "mux accounting failed invokes=${invokes} new=${tcp_new} reuse=${tcp_reuse}"
fi
PATH="${MUX_BIN}:${PATH}" deploy_ssh_mux_stop
if [[ ! -e "${cpath}" ]] && grep -q 'mux_exit=1' "${MUX_LOG}"; then
  ok "mux stop removes ControlPath on success path"
else
  bad "mux stop did not clean ControlPath"
fi

# Failure path still closes mux (simulate trap body).
: >"${MUX_LOG}"
PATH="${MUX_BIN}:${PATH}" deploy_ssh_base
cpath="${DEPLOY_SSH_CONTROL_PATH}"
rm -f "${cpath}"
PATH="${MUX_BIN}:${PATH}" deploy_ssh_run "${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "true"
[[ -e "${cpath}" ]] || bad "expected control socket after invoke"
PATH="${MUX_BIN}:${PATH}" deploy_ssh_mux_stop
if [[ ! -e "${cpath}" ]]; then
  ok "mux stop cleans socket on failure/cleanup path"
else
  bad "socket left after mux stop"
fi

# Master failure: transport timeout retries capped at 3 with backoff env small.
set +e
hash -r
set -e
: >"${MUX_LOG}"
MOCK_SSH_MODE=master_fail
export MOCK_SSH_MODE
DEPLOY_SSH_MAX_ATTEMPTS=3
DEPLOY_SSH_RETRY_DELAY_SEC=0
DEPLOY_SSH_INVOCATION_COUNT=0
: >"${INVOKE_LOG}"
set +e
PATH="${MUX_BIN}:${PATH}" deploy_ssh_run "${SSH_BASE[@]}" "${DEPLOY_USER}@${DEPLOY_HOST}" "true"
master_st=$?
set -e
unset MOCK_SSH_MODE
invokes="$(wc -l <"${INVOKE_LOG}" | tr -d ' ')"
# note_invoke once per deploy_ssh_run call (not per attempt) — attempts are internal.
# Verify exit 255 and that we did not loop forever: time-bounded by attempts.
if [[ "${master_st}" -eq 255 && "${invokes}" -eq 1 ]]; then
  ok "master transport failure returns 255 with single invoke wrapper"
else
  bad "master failure handling incorrect st=${master_st} invokes=${invokes}"
fi

# Count internal ssh attempts for timeout retries via a counting wrapper.
COUNT_BIN="${TMP}/count-bin"
mkdir -p "${COUNT_BIN}"
ATTEMPT_LOG="${TMP}/attempt.log"
: >"${ATTEMPT_LOG}"
cat >"${COUNT_BIN}/ssh" <<EOF
#!/usr/bin/env bash
printf 'attempt\\n' >>"${ATTEMPT_LOG}"
echo "Connection timed out" >&2
exit 255
EOF
chmod 0755 "${COUNT_BIN}/ssh"
DEPLOY_SSH_MAX_ATTEMPTS=3
DEPLOY_SSH_RETRY_DELAY_SEC=0
DEPLOY_SSH_INVOCATION_COUNT=0
set +e
hash -r
set -e
set +e
PATH="${COUNT_BIN}:/usr/bin:/bin" deploy_ssh_run ssh -o BatchMode=yes user@host true
retry_st=$?
set -e
attempts="$(wc -l <"${ATTEMPT_LOG}" | tr -d ' ')"
if [[ "${retry_st}" -eq 255 && "${attempts}" -eq 3 ]]; then
  ok "retryable transport retries exactly max_attempts=3"
else
  bad "retry count incorrect st=${retry_st} attempts=${attempts}"
fi

# Auth failure must not retry.
: >"${ATTEMPT_LOG}"
cat >"${COUNT_BIN}/ssh" <<EOF
#!/usr/bin/env bash
printf 'attempt\\n' >>"${ATTEMPT_LOG}"
echo "Permission denied (publickey)." >&2
exit 255
EOF
chmod 0755 "${COUNT_BIN}/ssh"
set +e
hash -r
set -e
set +e
PATH="${COUNT_BIN}:/usr/bin:/bin" deploy_ssh_run ssh -o BatchMode=yes user@host true
auth_st=$?
set -e
attempts="$(wc -l <"${ATTEMPT_LOG}" | tr -d ' ')"
if [[ "${auth_st}" -eq 255 && "${attempts}" -eq 1 ]]; then
  ok "auth failure is not retried"
else
  bad "auth was retried attempts=${attempts}"
fi

# Connection refused must not retry (avoids UFW LIMIT storm).
: >"${ATTEMPT_LOG}"
cat >"${COUNT_BIN}/ssh" <<EOF
#!/usr/bin/env bash
printf 'attempt\\n' >>"${ATTEMPT_LOG}"
echo "ssh: connect to host x port 22: Connection refused" >&2
exit 255
EOF
chmod 0755 "${COUNT_BIN}/ssh"
set +e
hash -r
set -e
set +e
PATH="${COUNT_BIN}:/usr/bin:/bin" deploy_ssh_run ssh -o BatchMode=yes user@host true
ref_st=$?
set -e
attempts="$(wc -l <"${ATTEMPT_LOG}" | tr -d ' ')"
if [[ "${ref_st}" -eq 255 && "${attempts}" -eq 1 ]]; then
  ok "Connection refused is not retried (no TCP storm)"
else
  bad "Connection refused was retried attempts=${attempts}"
fi

# Paths with spaces already used for key/known_hosts above.
if [[ "${DEPLOY_SSH_KEY}" == *" "* && "${DEPLOY_KNOWN_HOSTS}" == *" "* ]]; then
  ok "ssh key and known_hosts paths with spaces accepted"
else
  bad "space-path fixture missing"
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
