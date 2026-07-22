#!/usr/bin/env bash
# Closed sandbox E2E runner. Must run as bwb-fiscal-admin (never root).
# Reads token from file; curl via header file — never puts token in argv/stdout/stderr.
set -Eeuo pipefail

die() { echo "error: e2e_failed" >&2; exit 1; }

[[ "${EUID}" -ne 0 ]] || die

BASE_URL=""
TOKEN_FILE=""
FIXTURE_DIR=""
CASE=""
MAX_BODY=1048576

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-url)
      BASE_URL="${2:-}"
      shift 2
      ;;
    --token-file)
      TOKEN_FILE="${2:-}"
      shift 2
      ;;
    --fixture-dir)
      FIXTURE_DIR="${2:-}"
      shift 2
      ;;
    --case)
      CASE="${2:-}"
      shift 2
      ;;
    *)
      die
      ;;
  esac
done

[[ -n "${BASE_URL}" && -n "${TOKEN_FILE}" && -n "${FIXTURE_DIR}" && -n "${CASE}" ]] || die

# Allowlisted bases only (loopback API, loopback measure, public sandbox HTTPS).
case "${BASE_URL}" in
  http://127.0.0.1:8080 | http://127.0.0.1:18080 | https://sandbox.fiscalmod.bwb.pt) ;;
  *) die ;;
esac

# Token must live under a directory named tokens/; basename allowlisted (no spaces).
token_base="$(basename -- "${TOKEN_FILE}")"
token_dir="$(dirname -- "${TOKEN_FILE}")"
[[ "${token_base}" =~ ^[A-Za-z0-9._-]+$ ]] || die
[[ "$(basename -- "${token_dir}")" == "tokens" ]] || die
[[ -f "${TOKEN_FILE}" && ! -L "${TOKEN_FILE}" ]] || die
[[ -d "${token_dir}" && ! -L "${token_dir}" ]] || die

[[ "${FIXTURE_DIR}" =~ /fixtures/sandbox$ ]] || die
[[ -d "${FIXTURE_DIR}" && ! -L "${FIXTURE_DIR}" ]] || die

case "${CASE}" in
  unauthorized_no_token | unauthorized_bad_token | scope_mismatch | validation_failed \
  | create_201 | create_replay | token_revoked_401 | measure_probe) ;;
  *) die ;;
esac

[[ "${CASE}" != *$'\n'* && "${CASE}" != *$'\r'* ]] || die

TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

hdr="${TMP}/headers"
resp="${TMP}/resp"
codef="${TMP}/code"

write_auth_header() {
  local tok
  tok="$(tr -d '\r\n' <"${TOKEN_FILE}")"
  [[ -n "${tok}" ]] || die
  # Header file for curl -H @file — never echo token.
  printf 'Authorization: Bearer %s\n' "${tok}" >"${hdr}"
  chmod 0600 "${hdr}"
}

do_post() {
  local fixture="$1"
  local idem="$2"
  local use_auth="$3"
  local -a curl_args
  [[ -f "${fixture}" && ! -L "${fixture}" ]] || die
  curl_args=(
    -sS
    -o "${resp}"
    -w '%{http_code}'
    -X POST
    -H "Content-Type: application/json"
    -H "Idempotency-Key: ${idem}"
    --max-filesize "${MAX_BODY}"
    "--data-binary@${fixture}"
  )
  if [[ "${use_auth}" == "1" ]]; then
    curl_args+=(-H @"${hdr}")
  fi
  curl_args+=("${BASE_URL}/v1/documents")
  code="$(curl "${curl_args[@]}")"
  printf '%s' "${code}" >"${codef}"
}

# Extract stable CreateDocumentResponse fields for idempotent replay comparison.
# OpenAPI 0.1.4-draft: id, external_id, status, submission_id, created_at (no fiscal_seq / fiscal_number).
# Idempotency indicator = byte-identical stable field set (same as internal/httpapi replay test).
compare_replay_stable() {
  local first="$1" second="$2"
  python3 - "$first" "$second" <<'PY' || return 1
import json, sys
keys = ("id", "external_id", "status", "submission_id", "created_at")
forbidden = ("fiscal_number", "authority_request_id", "token", "authorization")
a = json.load(open(sys.argv[1], encoding="utf-8"))
b = json.load(open(sys.argv[2], encoding="utf-8"))
for k in forbidden:
    if k in a or k in b:
        sys.exit(2)
for k in keys:
    if k not in a or k not in b:
        sys.exit(3)
    if a[k] != b[k]:
        sys.exit(4)
if a.get("status") != "sealed_locally":
    sys.exit(5)
# Full required-object equality (idempotent replay must not mint a new document).
for k in keys:
    if a[k] != b[k]:
        sys.exit(4)
sys.exit(0)
PY
}

req_id_from_resp() {
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY' "${resp}" 2>/dev/null
import json,sys
try:
  d=json.load(open(sys.argv[1], encoding="utf-8"))
  v=d.get("request_id") or ""
  if isinstance(v, str) and all(c.isalnum() or c in "_-" for c in v):
    print(v)
  else:
    print("")
except Exception:
  print("")
PY
  fi
}

emit() {
  local status="$1" result="$2" rid="${3:-}"
  if [[ -n "${rid}" ]]; then
    printf 'status=%s result=%s request_id=%s\n' "${status}" "${result}" "${rid}"
  else
    printf 'status=%s result=%s\n' "${status}" "${result}"
  fi
}

case "${CASE}" in
  unauthorized_no_token)
    do_post "${FIXTURE_DIR}/create-document.min.json" "11111111-1111-4111-8111-111111111111" 0
    code="$(cat "${codef}")"
    [[ "${code}" == "401" ]] || { emit "${code}" fail; exit 1; }
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  unauthorized_bad_token)
    # Artificial invalid token constant — not a previously issued credential.
    printf 'Authorization: Bearer %s\n' "bwb_sbox_AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" >"${hdr}"
    chmod 0600 "${hdr}"
    do_post "${FIXTURE_DIR}/create-document.min.json" "22222222-2222-4222-8222-222222222222" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "401" ]] || { emit "${code}" fail; exit 1; }
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  scope_mismatch)
    write_auth_header
    do_post "${FIXTURE_DIR}/create-document.nif-mismatch.json" "33333333-3333-4333-8333-333333333333" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "403" ]] || { emit "${code}" fail; exit 1; }
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  validation_failed)
    write_auth_header
    do_post "${FIXTURE_DIR}/create-document.invalid.json" "44444444-4444-4444-8444-444444444444" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "422" ]] || { emit "${code}" fail; exit 1; }
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  create_201)
    write_auth_header
    do_post "${FIXTURE_DIR}/create-document.min.json" "55555555-5555-4555-8555-555555555555" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "201" ]] || { emit "${code}" fail; exit 1; }
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  create_replay)
    # Real idempotent replay: same Idempotency-Key + body → identical stable fields.
    write_auth_header
    idem="55555555-5555-4555-8555-555555555555"
    fixture="${FIXTURE_DIR}/create-document.min.json"
    do_post "${fixture}" "${idem}" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "201" ]] || { emit "${code}" fail; exit 1; }
    cp "${resp}" "${TMP}/first.json"
    do_post "${fixture}" "${idem}" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "201" ]] || { emit "${code}" fail; exit 1; }
    cp "${resp}" "${TMP}/second.json"
    if ! compare_replay_stable "${TMP}/first.json" "${TMP}/second.json"; then
      emit "${code}" replay_mismatch
      exit 1
    fi
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  token_revoked_401)
    # Expects TOKEN_FILE to hold a previously issued token whose credential was revoked.
    # Distinct from unauthorized_bad_token (artificial invalid constant).
    write_auth_header
    do_post "${FIXTURE_DIR}/create-document.min.json" "77777777-7777-4777-8777-777777777777" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "401" ]] || { emit "${code}" fail; exit 1; }
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  measure_probe)
    write_auth_header
    do_post "${FIXTURE_DIR}/create-document.min.json" "66666666-6666-4666-8666-666666666666" 1
    code="$(cat "${codef}")"
    emit "${code}" probe "$(req_id_from_resp)"
    ;;
esac
