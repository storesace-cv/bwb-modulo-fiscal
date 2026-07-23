#!/usr/bin/env bash
# POS / software-house sandbox integration kit (S4).
# Token never on argv/env/logs. Exact sandbox URL only (or loopback under test flag).
# Compatible with bash 3.2+ (macOS). Requires curl>=7.55, jq>=1.6, openssl>=1.1.1.
set -Eeuo pipefail

KIT_ROOT="$(cd "$(dirname "$0")" && pwd)"
FIXTURE_DIR="${KIT_ROOT}/fixtures"
SANDBOX_BASE="https://sandbox.fiscalmod.bwb.pt/v1"

# Sandbox API token: prefix + RawURLEncoding(32 bytes) without padding (matches module).
TOKEN_PREFIX="bwb_sbox_"
TOKEN_EXACT_LEN=52
TOKEN_MAX_READ=64

die() { echo "error: $*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage:
  pos-sandbox-kit.sh --token-file PATH [--revoked-token-file PATH] [--report-file PATH]
  pos-sandbox-kit.sh --token-stdin [--revoked-token-file PATH] [--report-file PATH]
  pos-sandbox-kit.sh --allow-loopback-test http://127.0.0.1:PORT/v1 --token-file PATH ...

Mutually exclusive: --token-file | --token-stdin

Test-only (rejected unless --allow-loopback-test): BWB_POS_KIT_CURL,
BWB_POS_KIT_RATE_COOLDOWN, BWB_POS_KIT_TMP_PATH_FILE.
EOF
}

version_ge() {
  local a="$1" b="$2"
  local IFS=.
  # shellcheck disable=SC2086
  set -- $a
  local a1="${1:-0}" a2="${2:-0}" a3="${3:-0}"
  # shellcheck disable=SC2086
  set -- $b
  local b1="${1:-0}" b2="${2:-0}" b3="${3:-0}"
  if [ "$a1" -gt "$b1" ]; then return 0; fi
  if [ "$a1" -lt "$b1" ]; then return 1; fi
  if [ "$a2" -gt "$b2" ]; then return 0; fi
  if [ "$a2" -lt "$b2" ]; then return 1; fi
  [ "$a3" -ge "$b3" ]
}

require_deps() {
  command -v curl >/dev/null 2>&1 || die "curl missing (need >= 7.55.0)"
  command -v jq >/dev/null 2>&1 || die "jq missing (need >= 1.6)"
  command -v openssl >/dev/null 2>&1 || die "openssl missing (need >= 1.1.1)"
  local cv jv ov
  cv="$(curl --version 2>/dev/null | head -n1 | awk '{print $2}')"
  jv="$(jq --version 2>/dev/null | sed 's/^jq-//')"
  ov="$(openssl version 2>/dev/null | awk '{print $2}' | sed 's/[a-zA-Z].*$//')"
  version_ge "$cv" "7.55.0" || die "curl $cv < 7.55.0"
  version_ge "$jv" "1.6" || die "jq $jv < 1.6"
  version_ge "$ov" "1.1.1" || die "openssl $ov < 1.1.1"
}

file_mode_octal() {
  local f="$1"
  if stat -f '%Mp%Lp' "$f" >/dev/null 2>&1; then
    stat -f '%Mp%Lp' "$f"
  else
    stat -c '%a' "$f"
  fi
}

file_owner_uid() {
  local f="$1"
  if stat -f '%u' "$f" >/dev/null 2>&1; then
    stat -f '%u' "$f"
  else
    stat -c '%u' "$f"
  fi
}

file_size_bytes() {
  local f="$1"
  if stat -f '%z' "$f" >/dev/null 2>&1; then
    stat -f '%z' "$f"
  else
    stat -c '%s' "$f"
  fi
}

# Read exact bytes without command-substitution stripping of trailing newlines.
# Result in LAST_BYTES (must not call this via $(...)).
read_bytes_file() {
  local f="$1" max="$2"
  local raw
  LAST_BYTES=""
  raw="$(head -c "$max" "$f" | tr -d '\0'; printf x)"
  LAST_BYTES="${raw%x}"
}

read_bytes_stdin() {
  local max="$1"
  local raw
  LAST_BYTES=""
  raw="$(head -c "$max" | tr -d '\0'; printf x)"
  LAST_BYTES="${raw%x}"
}

# Validate sandbox token bytes: exact length, prefix, Base64URL ASCII, no WS/CR/LF.
validate_token_value() {
  local tok="$1" label="$2"
  local len="${#tok}"
  [ "$len" -eq "$TOKEN_EXACT_LEN" ] || die "$label length must be ${TOKEN_EXACT_LEN} (got ${len})"
  case "$tok" in
    "${TOKEN_PREFIX}"*) ;;
    *) die "$label must start with ${TOKEN_PREFIX}" ;;
  esac
  case "$tok" in
    *[!A-Za-z0-9_-]*) die "$label contains invalid characters (ASCII Base64URL only)" ;;
  esac
}

read_token_from_file() {
  # Sets LAST_TOKEN. Must not be invoked inside $(...) — die/exit must reach the main shell.
  local f="$1" label="$2"
  local sz
  LAST_TOKEN=""
  sz="$(file_size_bytes "$f")"
  [ "$sz" -le "$TOKEN_MAX_READ" ] || die "$label file too large (max ${TOKEN_MAX_READ} bytes)"
  [ "$sz" -gt 0 ] || die "$label empty"
  read_bytes_file "$f" "$TOKEN_MAX_READ"
  case "$LAST_BYTES" in
    *$'\n'*|*$'\r'*|*[[:space:]]*) die "$label must not contain CR/LF or whitespace" ;;
  esac
  validate_token_value "$LAST_BYTES" "$label"
  LAST_TOKEN="$LAST_BYTES"
  unset LAST_BYTES
}

read_token_from_stdin() {
  LAST_TOKEN=""
  read_bytes_stdin $((TOKEN_MAX_READ + 1))
  [ "${#LAST_BYTES}" -le "$TOKEN_MAX_READ" ] || die "stdin token exceeds max ${TOKEN_MAX_READ} bytes"
  case "$LAST_BYTES" in
    *$'\n'*|*$'\r'*|*[[:space:]]*) die "stdin token must not contain CR/LF or whitespace" ;;
  esac
  [ -n "$LAST_BYTES" ] || die "stdin token empty"
  validate_token_value "$LAST_BYTES" "stdin-token"
  LAST_TOKEN="$LAST_BYTES"
  unset LAST_BYTES
}

validate_token_file_meta() {
  local f="$1" label="$2"
  [ -n "$f" ] || die "$label path empty"
  [ -e "$f" ] || die "$label missing"
  [ ! -L "$f" ] || die "$label must not be a symlink"
  [ -f "$f" ] || die "$label must be a regular file"
  local mode owner
  mode="$(file_mode_octal "$f")"
  owner="$(file_owner_uid "$f")"
  mode="$(printf '%s' "$mode" | sed 's/^0*//')"
  [ -z "$mode" ] && mode="0"
  [ "$mode" = "600" ] || die "$label mode must be 0600 (got $mode)"
  [ "$owner" = "$(id -u)" ] || die "$label owner must be euid"
}

csprng_hex() {
  local n="$1"
  openssl rand -hex "$n" 2>/dev/null || die "openssl rand failed"
}

uuid_v4() {
  local hex b6 b8
  hex="$(csprng_hex 16)"
  b6="$(printf '%s' "$hex" | cut -c13-14)"
  b8="$(printf '%s' "$hex" | cut -c17-18)"
  b6="$(printf '%02x' $(( (0x$b6 & 0x0f) | 0x40 )) )"
  b8="$(printf '%02x' $(( (0x$b8 & 0x3f) | 0x80 )) )"
  printf '%s-%s-%s%s-%s%s-%s\n' \
    "$(printf '%s' "$hex" | cut -c1-8)" \
    "$(printf '%s' "$hex" | cut -c9-12)" \
    "$b6" "$(printf '%s' "$hex" | cut -c15-16)" \
    "$b8" "$(printf '%s' "$hex" | cut -c19-20)" \
    "$(printf '%s' "$hex" | cut -c21-32)"
}

parse_loopback_base() {
  local u="$1"
  case "$u" in
    http://127.0.0.1:*) ;;
    *) die "loopback URL must be http://127.0.0.1:PORT/v1" ;;
  esac
  local rest port path
  rest="${u#http://127.0.0.1:}"
  port="${rest%%/*}"
  path="/${rest#*/}"
  [ "$path" = "/v1" ] || die "loopback path must be exactly /v1"
  case "$port" in
    ''|*[!0-9]*) die "invalid loopback port" ;;
  esac
  [ "$port" -ge 1 ] || die "loopback port out of range"
  [ "$port" -le 65535 ] || die "loopback port out of range"
  case "$u" in
    *@*|*\?*|*\#*) die "loopback URL rejects userinfo/query/fragment" ;;
  esac
}

assert_base_url() {
  local u="$1"
  if [ "$ALLOW_LOOPBACK" = "1" ]; then
    parse_loopback_base "$u"
    return 0
  fi
  [ "$u" = "$SANDBOX_BASE" ] || die "BASE must be exactly ${SANDBOX_BASE}"
}

reject_test_env_outside_loopback() {
  if [ "$ALLOW_LOOPBACK" = "1" ]; then
    return 0
  fi
  if [ -n "${BWB_POS_KIT_CURL:-}" ]; then
    die "BWB_POS_KIT_CURL is test-only; requires --allow-loopback-test"
  fi
  if [ -n "${BWB_POS_KIT_RATE_COOLDOWN:-}" ]; then
    die "BWB_POS_KIT_RATE_COOLDOWN is test-only; requires --allow-loopback-test"
  fi
  if [ -n "${BWB_POS_KIT_TMP_PATH_FILE:-}" ]; then
    die "BWB_POS_KIT_TMP_PATH_FILE is test-only; requires --allow-loopback-test"
  fi
  if [ -n "${BWB_POS_KIT_READY_FILE:-}" ]; then
    die "BWB_POS_KIT_READY_FILE is test-only; requires --allow-loopback-test"
  fi
}

TOKEN_FILE=""
TOKEN_STDIN=0
REVOKED_FILE=""
REPORT_FILE=""
ALLOW_LOOPBACK=0
BASE_URL="$SANDBOX_BASE"

while [ $# -gt 0 ]; do
  case "$1" in
    --token-file)
      TOKEN_FILE="${2:-}"
      shift 2
      ;;
    --token-stdin)
      TOKEN_STDIN=1
      shift
      ;;
    --revoked-token-file)
      REVOKED_FILE="${2:-}"
      shift 2
      ;;
    --report-file)
      REPORT_FILE="${2:-}"
      shift 2
      ;;
    --allow-loopback-test)
      ALLOW_LOOPBACK=1
      BASE_URL="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      die "unknown arg: $1"
      ;;
  esac
done

require_deps
assert_base_url "$BASE_URL"
reject_test_env_outside_loopback

if [ "$ALLOW_LOOPBACK" = "1" ]; then
  CURL_BIN="${BWB_POS_KIT_CURL:-curl}"
  RATE_COOLDOWN="${BWB_POS_KIT_RATE_COOLDOWN:-5}"
else
  CURL_BIN="curl"
  RATE_COOLDOWN=5
fi

if [ "$TOKEN_STDIN" -eq 1 ] && [ -n "$TOKEN_FILE" ]; then
  die "--token-file and --token-stdin are mutually exclusive"
fi
if [ "$TOKEN_STDIN" -eq 0 ] && [ -z "$TOKEN_FILE" ]; then
  die "--token-file is required (or --token-stdin)"
fi

TMP="$(mktemp -d "${TMPDIR:-/tmp}/bwb-pos-kit.XXXXXX")"
chmod 0700 "$TMP"
CHILD_PIDS=""
CLEANED=0

# shellcheck disable=SC2317,SC2329 # invoked via trap / cleanup
kill_children() {
  local pid
  for pid in $CHILD_PIDS; do
    if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done
  for pid in $CHILD_PIDS; do
    if [ -n "$pid" ]; then
      while kill -0 "$pid" 2>/dev/null; do
        sleep 0.05
      done
      wait "$pid" 2>/dev/null || true
    fi
  done
  CHILD_PIDS=""
}

# shellcheck disable=SC2317,SC2329 # invoked via trap
cleanup() {
  [ "$CLEANED" = "1" ] && return 0
  CLEANED=1
  kill_children
  rm -rf "$TMP"
}

trap 'ec=$?; cleanup; exit "$ec"' EXIT
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

if [ -n "${BWB_POS_KIT_TMP_PATH_FILE:-}" ]; then
  printf '%s' "$TMP" >"$BWB_POS_KIT_TMP_PATH_FILE"
  chmod 0600 "$BWB_POS_KIT_TMP_PATH_FILE"
fi

HDR="$TMP/auth.hdr"
RESP="$TMP/resp.body"
CODEF="$TMP/resp.code"
CURLEC="$TMP/curl.ec"
REPORT_JSON="$TMP/report.json"
DOCS="$TMP/docs"
mkdir -p "$DOCS"
chmod 0700 "$DOCS"

if [ "$TOKEN_STDIN" -eq 1 ]; then
  read_token_from_stdin
  printf 'Authorization: Bearer %s\n' "$LAST_TOKEN" >"$HDR"
  unset LAST_TOKEN
else
  validate_token_file_meta "$TOKEN_FILE" "token-file"
  read_token_from_file "$TOKEN_FILE" "token-file"
  printf 'Authorization: Bearer %s\n' "$LAST_TOKEN" >"$HDR"
  unset LAST_TOKEN
fi
chmod 0600 "$HDR"

if [ -n "$REVOKED_FILE" ]; then
  validate_token_file_meta "$REVOKED_FILE" "revoked-token-file"
  read_token_from_file "$REVOKED_FILE" "revoked-token-file"
  unset LAST_TOKEN
fi

RUN_ID="$(csprng_hex 16)"
EXT_ID="FIXTURE-SBOX-${RUN_ID}"
IDEM_A="$(uuid_v4)"
IDEM_B="$(uuid_v4)"

materialize() {
  local template="$1" out="$2" ext="$3"
  jq --arg ext "$ext" '.external_id = $ext' "$template" >"$out"
  jq -e . "$out" >/dev/null || die "materialized JSON invalid: $out"
}

materialize "$FIXTURE_DIR/document.base.json" "$DOCS/base.json" "$EXT_ID"
materialize "$FIXTURE_DIR/document.alt-body.json" "$DOCS/alt.json" "$EXT_ID"
materialize "$FIXTURE_DIR/document.nif-mismatch.json" "$DOCS/mismatch.json" "FIXTURE-MISMATCH-${RUN_ID}"
materialize "$FIXTURE_DIR/document.invalid.json" "$DOCS/invalid.json" "FIXTURE-INVALID-${RUN_ID}"

# Run curl in the main shell (never via $(...)) so INT/TERM traps are not deferred.
# Sets LAST_HTTP_CODE and writes CURLEC; body in RESP.
curl_post() {
  local body="$1" idem="$2" use_auth="$3" hdr_override="${4:-}"
  local -a args
  local cpid code ec
  LAST_HTTP_CODE="000"
  args=(
    -sS
    --max-time 30
    -o "$RESP"
    -w '%{http_code}'
    -X POST
    -H "Content-Type: application/json"
    -H "Idempotency-Key: ${idem}"
    --data-binary
    "@${body}"
  )
  if [ "$use_auth" = "1" ]; then
    if [ -n "$hdr_override" ]; then
      args+=(-H @"$hdr_override")
    else
      args+=(-H @"$HDR")
    fi
  fi
  args+=("${BASE_URL}/documents")
  if [ -n "${BWB_POS_KIT_READY_FILE:-}" ]; then
    : >"$BWB_POS_KIT_READY_FILE"
  fi
  set +e
  "$CURL_BIN" "${args[@]}" >"$CODEF" &
  cpid=$!
  CHILD_PIDS="${CHILD_PIDS} ${cpid}"
  while kill -0 "$cpid" 2>/dev/null; do
    sleep 0.05
  done
  wait "$cpid"
  ec=$?
  set -e
  CHILD_PIDS="$(printf '%s' "$CHILD_PIDS" | sed "s/ ${cpid}//g")"
  printf '%s' "$ec" >"$CURLEC"
  if [ "$ec" -ne 0 ]; then
    LAST_HTTP_CODE="000"
    return 0
  fi
  code="$(tr -d '\r\n' <"$CODEF")"
  case "$code" in
    [1-9][0-9][0-9]) LAST_HTTP_CODE="$code" ;;
    *) LAST_HTTP_CODE="000" ;;
  esac
}

curl_ec() {
  tr -d '\r\n' <"$CURLEC" 2>/dev/null || printf '1'
}

request_id_from_resp() {
  if [ ! -s "$RESP" ]; then
    printf ''
    return 0
  fi
  jq -r 'if type=="object" then (.request_id // empty) else empty end' "$RESP" 2>/dev/null || true
}

problem_code_from_resp() {
  if [ ! -s "$RESP" ]; then
    printf ''
    return 0
  fi
  jq -r 'if type=="object" then (.code // empty) else empty end' "$RESP" 2>/dev/null || true
}

: >"$TMP/results.ndjson"
# Sanitized record only — no bodies, fiscal ids, or document ids.
record() {
  local case_name="$1" status="$2" http_code="$3" result="$4" rid="${5:-}"
  jq -nc --arg c "$case_name" --arg s "$status" --arg h "$http_code" --arg r "$result" --arg id "$rid" \
    '{case:$c, status:$s, http_code:$h, result:$r, request_id:(if $id=="" then null else $id end)}' \
    >>"$TMP/results.ndjson"
}

stable_doc_fields() {
  jq -c '{id, external_id, status, submission_id, created_at}' "$1"
}

LAST_HTTP_CODE="000"

# --- cases ---
curl_post "$DOCS/base.json" "$IDEM_A" 1
rid="$(request_id_from_resp)"
if [ "$(curl_ec)" -ne 0 ]; then
  record create_201 FAIL "000" transport "$rid"
elif [ "$LAST_HTTP_CODE" = "201" ] \
  && jq -e '
      type=="object"
      and (.id|type=="string" and length>0)
      and (.external_id|type=="string" and length>0)
      and .status=="sealed_locally"
      and (.submission_id|type=="string" and length>0)
      and (.created_at|type=="string" and length>0)
      and (.request_id|type=="string" and length>0)
    ' "$RESP" >/dev/null 2>&1; then
  cp "$RESP" "$TMP/create_resp.json"
  record create_201 PASS "$LAST_HTTP_CODE" pass "$rid"
else
  record create_201 FAIL "$LAST_HTTP_CODE" semantic_or_status "$rid"
fi

curl_post "$DOCS/base.json" "$IDEM_A" 1
rid="$(request_id_from_resp)"
if [ "$(curl_ec)" -ne 0 ]; then
  record replay FAIL "000" transport "$rid"
elif [ "$LAST_HTTP_CODE" = "201" ] && [ -f "$TMP/create_resp.json" ]; then
  cp "$RESP" "$TMP/replay_resp.json"
  if [ "$(stable_doc_fields "$TMP/create_resp.json")" = "$(stable_doc_fields "$TMP/replay_resp.json")" ]; then
    record replay PASS "$LAST_HTTP_CODE" pass "$rid"
  else
    record replay FAIL "$LAST_HTTP_CODE" stable_fields_mismatch "$rid"
  fi
else
  record replay FAIL "$LAST_HTTP_CODE" semantic_or_status "$rid"
fi

curl_post "$DOCS/alt.json" "$IDEM_A" 1
rid="$(request_id_from_resp)"
pc="$(problem_code_from_resp)"
if [ "$(curl_ec)" -ne 0 ]; then
  record idempotency_conflict FAIL "000" transport "$rid"
elif [ "$LAST_HTTP_CODE" = "409" ] && [ "$pc" = "FISCAL_IDEMPOTENCY_CONFLICT" ]; then
  record idempotency_conflict PASS "$LAST_HTTP_CODE" pass "$rid"
else
  record idempotency_conflict FAIL "$LAST_HTTP_CODE" "code=${pc}" "$rid"
fi

curl_post "$DOCS/alt.json" "$IDEM_B" 1
rid="$(request_id_from_resp)"
pc="$(problem_code_from_resp)"
if [ "$(curl_ec)" -ne 0 ]; then
  record external_id_conflict FAIL "000" transport "$rid"
elif [ "$LAST_HTTP_CODE" = "409" ] && [ "$pc" = "FISCAL_EXTERNAL_ID_CONFLICT" ]; then
  record external_id_conflict PASS "$LAST_HTTP_CODE" pass "$rid"
else
  record external_id_conflict FAIL "$LAST_HTTP_CODE" "code=${pc}" "$rid"
fi

curl_post "$DOCS/mismatch.json" "$(uuid_v4)" 1
rid="$(request_id_from_resp)"
pc="$(problem_code_from_resp)"
if [ "$(curl_ec)" -ne 0 ]; then
  record scope_mismatch FAIL "000" transport "$rid"
elif [ "$LAST_HTTP_CODE" = "403" ] && [ "$pc" = "FISCAL_SCOPE_MISMATCH" ]; then
  record scope_mismatch PASS "$LAST_HTTP_CODE" pass "$rid"
else
  record scope_mismatch FAIL "$LAST_HTTP_CODE" "code=${pc}" "$rid"
fi

curl_post "$DOCS/invalid.json" "$(uuid_v4)" 1
rid="$(request_id_from_resp)"
pc="$(problem_code_from_resp)"
if [ "$(curl_ec)" -ne 0 ]; then
  record validation_422 FAIL "000" transport "$rid"
elif [ "$LAST_HTTP_CODE" = "422" ] && [ "$pc" = "FISCAL_VALIDATION_FAILED" ]; then
  record validation_422 PASS "$LAST_HTTP_CODE" pass "$rid"
else
  record validation_422 FAIL "$LAST_HTTP_CODE" "code=${pc}" "$rid"
fi

BADHDR="$TMP/bad.hdr"
printf 'Authorization: Bearer INVALID_TOKEN_NOT_A_SECRET\n' >"$BADHDR"
chmod 0600 "$BADHDR"
curl_post "$DOCS/base.json" "$(uuid_v4)" 1 "$BADHDR"
rid="$(request_id_from_resp)"
pc="$(problem_code_from_resp)"
if [ "$(curl_ec)" -ne 0 ]; then
  record unauthorized_bad_token FAIL "000" transport "$rid"
elif [ "$LAST_HTTP_CODE" = "401" ] && [ "$pc" = "FISCAL_UNAUTHORIZED" ]; then
  record unauthorized_bad_token PASS "$LAST_HTTP_CODE" pass "$rid"
else
  record unauthorized_bad_token FAIL "$LAST_HTTP_CODE" "code=${pc}" "$rid"
fi

if [ -z "$REVOKED_FILE" ]; then
  record token_revoked_401 NOT_RUN "" NOT_RUN ""
else
  RHDR="$TMP/revoked.hdr"
  read_token_from_file "$REVOKED_FILE" "revoked-token-file"
  printf 'Authorization: Bearer %s\n' "$LAST_TOKEN" >"$RHDR"
  unset LAST_TOKEN
  chmod 0600 "$RHDR"
  curl_post "$DOCS/base.json" "$(uuid_v4)" 1 "$RHDR"
  rid="$(request_id_from_resp)"
  pc="$(problem_code_from_resp)"
  if [ "$(curl_ec)" -ne 0 ]; then
    record token_revoked_401 FAIL "000" transport "$rid"
  elif [ "$LAST_HTTP_CODE" = "401" ] && [ "$pc" = "FISCAL_UNAUTHORIZED" ]; then
    record token_revoked_401 PASS "$LAST_HTTP_CODE" pass "$rid"
  else
    record token_revoked_401 FAIL "$LAST_HTTP_CODE" "code=${pc}" "$rid"
  fi
fi

# rate_429 last — track each child PID; separate curl exit from http_code.
c201=0
c429=0
c5xx=0
ctransport=0
cother=0
RATE_PIDS=""
i=1
while [ "$i" -le 30 ]; do
  body="$DOCS/rate-$i.json"
  jq --arg ext "FIXTURE-RATE-${RUN_ID}-$i" '.external_id = $ext' "$FIXTURE_DIR/document.base.json" >"$body"
  jq -e . "$body" >/dev/null
  idem="$(uuid_v4)"
  (
    set +e
    code="$("$CURL_BIN" -sS --max-time 15 -o /dev/null -w '%{http_code}' -X POST \
      -H "Content-Type: application/json" \
      -H "Idempotency-Key: ${idem}" \
      -H @"$HDR" \
      --data-binary @"$body" \
      "${BASE_URL}/documents")"
    ec=$?
    set -e
    if [ "$ec" -ne 0 ]; then
      printf 'transport\n' >"$TMP/rate-result-$i"
    else
      printf '%s\n' "$code" >"$TMP/rate-result-$i"
    fi
  ) &
  rpid=$!
  RATE_PIDS="${RATE_PIDS} ${rpid}"
  CHILD_PIDS="${CHILD_PIDS} ${rpid}"
  if [ $((i % 5)) -eq 0 ]; then
    for p in $RATE_PIDS; do
      while kill -0 "$p" 2>/dev/null; do
        sleep 0.05
      done
      wait "$p" 2>/dev/null || true
    done
    for p in $RATE_PIDS; do
      CHILD_PIDS="$(printf '%s' "$CHILD_PIDS" | sed "s/ ${p}//g")"
    done
    RATE_PIDS=""
  fi
  i=$((i + 1))
done
for p in $RATE_PIDS; do
  while kill -0 "$p" 2>/dev/null; do
    sleep 0.05
  done
  wait "$p" 2>/dev/null || true
  CHILD_PIDS="$(printf '%s' "$CHILD_PIDS" | sed "s/ ${p}//g")"
done
RATE_PIDS=""

# All children for rate must be gone; exactly 30 results.
alive=0
for p in $CHILD_PIDS; do
  if [ -n "$p" ] && kill -0 "$p" 2>/dev/null; then
    alive=$((alive + 1))
  fi
done

collected=0
i=1
while [ "$i" -le 30 ]; do
  if [ -f "$TMP/rate-result-$i" ]; then
    collected=$((collected + 1))
    code="$(tr -d '\r\n' <"$TMP/rate-result-$i")"
    case "$code" in
      201) c201=$((c201 + 1)) ;;
      429) c429=$((c429 + 1)) ;;
      5[0-9][0-9]) c5xx=$((c5xx + 1)) ;;
      transport) ctransport=$((ctransport + 1)) ;;
      *) cother=$((cother + 1)) ;;
    esac
  fi
  i=$((i + 1))
done

sleep "$RATE_COOLDOWN"

rate_result="201=${c201};429=${c429};5xx=${c5xx};transport=${ctransport};other=${cother};collected=${collected};alive=${alive}"
if [ "$c429" -ge 1 ] && [ "$c5xx" -eq 0 ] && [ "$ctransport" -eq 0 ] && [ "$cother" -eq 0 ] \
  && [ "$collected" -eq 30 ] && [ "$alive" -eq 0 ]; then
  record rate_429 PASS "429x${c429}" "$rate_result" ""
else
  record rate_429 FAIL "mixed" "$rate_result" ""
fi

jq -s --arg run "$RUN_ID" --arg base "$BASE_URL" \
  '{run_id:$run, base_url:$base, cases:., summary:{pass:([.[]|select(.status=="PASS")]|length), fail:([.[]|select(.status=="FAIL")]|length), not_run:([.[]|select(.status=="NOT_RUN")]|length)}}' \
  "$TMP/results.ndjson" >"$REPORT_JSON"

jq -c '{run_id, base_url, summary, cases:[.cases[]|{case,status,http_code,result,request_id}]}' "$REPORT_JSON"

if [ -n "$REPORT_FILE" ]; then
  jq -c '{run_id, base_url, summary, cases:[.cases[]|{case,status,http_code,result,request_id}]}' "$REPORT_JSON" >"$REPORT_FILE"
fi

fails="$(jq '.summary.fail' "$REPORT_JSON")"
[ "$fails" = "0" ] || exit 1
exit 0
