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

[[ "${TOKEN_FILE}" =~ ^/var/lib/bwb-fiscal-admin/tokens/[A-Za-z0-9._-]+$ ]] || die
[[ -f "${TOKEN_FILE}" && ! -L "${TOKEN_FILE}" ]] || die
[[ "${FIXTURE_DIR}" =~ /fixtures/sandbox$ ]] || die
[[ -d "${FIXTURE_DIR}" && ! -L "${FIXTURE_DIR}" ]] || die

case "${CASE}" in
  unauthorized_no_token | unauthorized_bad_token | scope_mismatch | validation_failed | create_201 | create_replay | measure_probe) ;;
  *) die ;;
esac

# Reject control chars in paths/values already constrained; double-check CASE.
[[ "${CASE}" != *$'\n'* && "${CASE}" != *$'\0'* ]] || die

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
  curl_args=(
    -sS
    -o "${resp}"
    -w '%{http_code}'
    -X POST
    -H "Content-Type: application/json"
    -H "Idempotency-Key: ${idem}"
    --max-filesize "${MAX_BODY}"
    --data-binary @"${fixture}"
  )
  if [[ "${use_auth}" == "1" ]]; then
    curl_args+=(-H @"${hdr}")
  fi
  curl_args+=("${BASE_URL}/v1/documents")
  # shellcheck disable=SC2068
  code="$(curl ${curl_args[@]})"
  printf '%s' "${code}" >"${codef}"
}

req_id_from_resp() {
  # Best-effort parse request_id without dumping body to stdout on success path.
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY' "${resp}" 2>/dev/null
import json,sys
try:
  d=json.load(open(sys.argv[1]))
  print(d.get("request_id") or d.get("id") or "")
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
    write_auth_header
    do_post "${FIXTURE_DIR}/create-document.min.json" "55555555-5555-4555-8555-555555555555" 1
    code="$(cat "${codef}")"
    [[ "${code}" == "201" ]] || { emit "${code}" fail; exit 1; }
    emit "${code}" pass "$(req_id_from_resp)"
    ;;
  measure_probe)
    write_auth_header
    do_post "${FIXTURE_DIR}/create-document.min.json" "66666666-6666-4666-8666-666666666666" 1
    code="$(cat "${codef}")"
    emit "${code}" probe "$(req_id_from_resp)"
    ;;
esac
