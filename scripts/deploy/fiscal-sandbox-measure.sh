#!/usr/bin/env bash
# Closed rate-limit measurement against loopback Nginx :18080 only.
# Caps: <=60 requests, <=5 concurrent, <=60 seconds. Never prints token/DSN/NIF.
set -Eeuo pipefail

die() { echo "error: measure_failed" >&2; exit 1; }

[[ "${EUID}" -ne 0 ]] || die

TOKEN_FILE=""
FIXTURE_DIR=""
E2E_BIN=""
CONCURRENCY=5
TOTAL=60
DURATION_SEC=60

while [[ $# -gt 0 ]]; do
  case "$1" in
    --token-file) TOKEN_FILE="${2:-}"; shift 2 ;;
    --fixture-dir) FIXTURE_DIR="${2:-}"; shift 2 ;;
    --e2e-bin) E2E_BIN="${2:-}"; shift 2 ;;
    --concurrency) CONCURRENCY="${2:-}"; shift 2 ;;
    --total) TOTAL="${2:-}"; shift 2 ;;
    --duration-sec) DURATION_SEC="${2:-}"; shift 2 ;;
    *) die ;;
  esac
done

[[ -n "${TOKEN_FILE}" && -n "${FIXTURE_DIR}" && -n "${E2E_BIN}" ]] || die
[[ "${TOKEN_FILE}" =~ ^/var/lib/bwb-fiscal-admin/tokens/[A-Za-z0-9._-]+$ ]] || die
[[ -x "${E2E_BIN}" && ! -L "${E2E_BIN}" ]] || die

[[ "${CONCURRENCY}" =~ ^[1-5]$ ]] || die
[[ "${TOTAL}" =~ ^[1-9][0-9]?$ ]] || die
[[ "${TOTAL}" -le 60 ]] || die
[[ "${DURATION_SEC}" =~ ^[1-9][0-9]?$ ]] || die
[[ "${DURATION_SEC}" -le 60 ]] || die

BASE_URL="http://127.0.0.1:18080"
start="${SECONDS}"
ok=0
r429=0
other=0
sent=0
pids=()

WORKDIR="$(mktemp -d)"
trap 'rm -rf "${WORKDIR}"' EXIT

run_one() {
  local out=""
  set +e
  out="$("${E2E_BIN}" --base-url "${BASE_URL}" --token-file "${TOKEN_FILE}" \
    --fixture-dir "${FIXTURE_DIR}" --case measure_probe 2>/dev/null)"
  set -e
  case "${out}" in
    status=429*) printf '429\n' ;;
    status=201*|status=409*) printf 'ok\n' ;;
    *) printf 'other\n' ;;
  esac
  return 0
}

while [[ "${sent}" -lt "${TOTAL}" && $((SECONDS - start)) -lt "${DURATION_SEC}" ]]; do
  pids=()
  batch=0
  while [[ "${batch}" -lt "${CONCURRENCY}" && "${sent}" -lt "${TOTAL}" && $((SECONDS - start)) -lt "${DURATION_SEC}" ]]; do
    run_one >"${WORKDIR}/r.${sent}" &
    pids+=("$!")
    sent=$((sent + 1))
    batch=$((batch + 1))
  done
  for pid in "${pids[@]}"; do
    wait "${pid}"
  done
  for ((i = sent - batch; i < sent; i++)); do
    f="${WORKDIR}/r.${i}"
    case "$(cat "${f}" 2>/dev/null)" in
      429) r429=$((r429 + 1)) ;;
      ok) ok=$((ok + 1)) ;;
      *) other=$((other + 1)) ;;
    esac
    rm -f -- "${f}"
  done
done

printf 'measure_ok total_sent=%s concurrency=%s duration_cap_s=%s status_okish=%s status_429=%s status_other=%s\n' \
  "${sent}" "${CONCURRENCY}" "${DURATION_SEC}" "${ok}" "${r429}" "${other}"
[[ "${r429}" -ge 1 ]] || exit 1
