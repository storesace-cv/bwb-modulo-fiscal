#!/usr/bin/env bash
# POS / software-house sandbox integration kit (S4).
# Token never on argv/env/logs. Exact sandbox URL only (or loopback under test flag).
# Compatible with bash 3.2+ (macOS). Requires curl>=7.55, jq>=1.6, openssl>=1.1.1.
set -Eeuo pipefail

KIT_ROOT="$(cd "$(dirname "$0")" && pwd)"
FIXTURE_DIR="${KIT_ROOT}/fixtures"
SANDBOX_BASE="https://sandbox.fiscalmod.bwb.pt/v1"

die() { echo "error: $*" >&2; exit 1; }

usage() {
  cat <<'EOF'
Usage:
  pos-sandbox-kit.sh --token-file PATH [--revoked-token-file PATH] [--report-file PATH]
  pos-sandbox-kit.sh --token-stdin [--revoked-token-file PATH] [--report-file PATH]
  pos-sandbox-kit.sh --allow-loopback-test http://127.0.0.1:PORT/v1 --token-file PATH ...

Mutually exclusive: --token-file | --token-stdin
EOF
}

# --- dependency versions ---
version_ge() {
  # version_ge A B → 0 if A >= B (dotted numeric)
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

validate_token_file() {
  local f="$1" label="$2"
  [ -n "$f" ] || die "$label path empty"
  [ -e "$f" ] || die "$label missing"
  [ ! -L "$f" ] || die "$label must not be a symlink"
  [ -f "$f" ] || die "$label must be a regular file"
  local mode owner
  mode="$(file_mode_octal "$f")"
  owner="$(file_owner_uid "$f")"
  # normalize mode to 3 digits when possible
  mode="$(printf '%s' "$mode" | sed 's/^0*//')"
  [ -z "$mode" ] && mode="0"
  [ "$mode" = "600" ] || die "$label mode must be 0600 (got $mode)"
  [ "$owner" = "$(id -u)" ] || die "$label owner must be euid"
  local tok
  tok="$(tr -d '\r\n' <"$f")"
  [ -n "$tok" ] || die "$label empty"
  printf '%s' "$tok" | grep -q '[^[:space:]]' || die "$label whitespace-only"
}

csprng_hex() {
  local n="$1"
  openssl rand -hex "$n" 2>/dev/null || die "openssl rand failed"
}

uuid_v4() {
  local hex
  hex="$(csprng_hex 16)"
  # Set version nibble (byte 6 high) to 4 and variant (byte 8 high) to 10xx
  local b6 b8
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
  [ "$port" -ge 1 ] && [ "$port" -le 65535 ] || die "loopback port out of range"
  # reject userinfo/query/fragment already by pattern
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
  # Exact match only — no env override, no lookalikes.
  [ "$u" = "$SANDBOX_BASE" ] || die "BASE must be exactly ${SANDBOX_BASE}"
}

TOKEN_FILE=""
TOKEN_STDIN=0
REVOKED_FILE=""
REPORT_FILE=""
ALLOW_LOOPBACK=0
BASE_URL="$SANDBOX_BASE"
CURL_BIN="${BWB_POS_KIT_CURL:-curl}"

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

if [ "$TOKEN_STDIN" -eq 1 ] && [ -n "$TOKEN_FILE" ]; then
  die "--token-file and --token-stdin are mutually exclusive"
fi
if [ "$TOKEN_STDIN" -eq 0 ] && [ -z "$TOKEN_FILE" ]; then
  die "--token-file is required (or --token-stdin)"
fi

TMP="$(mktemp -d "${TMPDIR:-/tmp}/bwb-pos-kit.XXXXXX")"
chmod 0700 "$TMP"
# shellcheck disable=SC2329 # invoked via trap
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

HDR="$TMP/auth.hdr"
RESP="$TMP/resp.body"
CODEF="$TMP/resp.code"
REPORT_JSON="$TMP/report.json"
DOCS="$TMP/docs"
mkdir -p "$DOCS"
chmod 0700 "$DOCS"

# Load primary token into header file only (never echo).
if [ "$TOKEN_STDIN" -eq 1 ]; then
  tok="$(tr -d '\r\n' </dev/stdin)"
  [ -n "$tok" ] || die "stdin token empty"
  printf '%s' "$tok" | grep -q '[^[:space:]]' || die "stdin token whitespace-only"
  printf 'Authorization: Bearer %s\n' "$tok" >"$HDR"
  unset tok
else
  validate_token_file "$TOKEN_FILE" "token-file"
  tok="$(tr -d '\r\n' <"$TOKEN_FILE")"
  printf 'Authorization: Bearer %s\n' "$tok" >"$HDR"
  unset tok
fi
chmod 0600 "$HDR"

if [ -n "$REVOKED_FILE" ]; then
  validate_token_file "$REVOKED_FILE" "revoked-token-file"
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
# second doc same external_id for conflict (distinct body from base via description tweak already in alt — use base again with new idem only)
# For external_id conflict we POST a different body (alt) after first create with same EXT_ID and new key — wait:
# create_201 uses base+IDEM_A; external_id conflict: new key + same external_id + different semantic body is OK for EXTERNAL_ID_CONFLICT
# Actually EXTERNAL_ID_CONFLICT: same external_id, different Idempotency-Key
# IDEMPOTENCY_CONFLICT: same Idempotency-Key, different body

curl_post() {
  local body="$1" idem="$2" use_auth="$3" hdr_override="${4:-}"
  local -a args
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
  # never -L
  if [ "$use_auth" = "1" ]; then
    if [ -n "$hdr_override" ]; then
      args+=(-H @"$hdr_override")
    else
      args+=(-H @"$HDR")
    fi
  fi
  args+=("${BASE_URL}/documents")
  local code
  code="$("$CURL_BIN" "${args[@]}")" || code="000"
  printf '%s' "$code" >"$CODEF"
  printf '%s' "$code"
}

request_id_from_resp() {
  if [ ! -s "$RESP" ]; then
    printf ''
    return 0
  fi
  jq -r 'if type=="object" then (.request_id // empty) else empty end' "$RESP" 2>/dev/null || true
}

# Report accumulator as JSON lines then jq -s
: >"$TMP/results.ndjson"
record() {
  local case_name="$1" status="$2" http_code="$3" result="$4" rid="${5:-}"
  jq -nc --arg c "$case_name" --arg s "$status" --arg h "$http_code" --arg r "$result" --arg id "$rid" \
    '{case:$c, status:$s, http_code:$h, result:$r, request_id:(if $id=="" then null else $id end)}' \
    >>"$TMP/results.ndjson"
}

# --- cases ---
code="$(curl_post "$DOCS/base.json" "$IDEM_A" 1)"
rid="$(request_id_from_resp)"
if [ "$code" = "201" ]; then
  record create_201 PASS "$code" pass "$rid"
else
  record create_201 FAIL "$code" fail "$rid"
fi

# replay same key+body
code="$(curl_post "$DOCS/base.json" "$IDEM_A" 1)"
rid="$(request_id_from_resp)"
if [ "$code" = "201" ]; then
  record replay PASS "$code" pass "$rid"
else
  record replay FAIL "$code" fail "$rid"
fi

# idempotency conflict: same key, different body
code="$(curl_post "$DOCS/alt.json" "$IDEM_A" 1)"
rid="$(request_id_from_resp)"
if [ "$code" = "409" ]; then
  record idempotency_conflict PASS "$code" pass "$rid"
else
  record idempotency_conflict FAIL "$code" fail "$rid"
fi

# external_id conflict: new key, same external_id (alt body)
code="$(curl_post "$DOCS/alt.json" "$IDEM_B" 1)"
rid="$(request_id_from_resp)"
if [ "$code" = "409" ]; then
  record external_id_conflict PASS "$code" pass "$rid"
else
  record external_id_conflict FAIL "$code" fail "$rid"
fi

code="$(curl_post "$DOCS/mismatch.json" "$(uuid_v4)" 1)"
rid="$(request_id_from_resp)"
if [ "$code" = "403" ]; then
  record scope_mismatch PASS "$code" pass "$rid"
else
  record scope_mismatch FAIL "$code" fail "$rid"
fi

code="$(curl_post "$DOCS/invalid.json" "$(uuid_v4)" 1)"
rid="$(request_id_from_resp)"
if [ "$code" = "422" ]; then
  record validation_422 PASS "$code" pass "$rid"
else
  record validation_422 FAIL "$code" fail "$rid"
fi

# bad token: temporary header with invalid constant (not from secrets)
BADHDR="$TMP/bad.hdr"
printf 'Authorization: Bearer INVALID_TOKEN_NOT_A_SECRET\n' >"$BADHDR"
chmod 0600 "$BADHDR"
code="$(curl_post "$DOCS/base.json" "$(uuid_v4)" 1 "$BADHDR")"
rid="$(request_id_from_resp)"
if [ "$code" = "401" ]; then
  record unauthorized_bad_token PASS "$code" pass "$rid"
else
  record unauthorized_bad_token FAIL "$code" fail "$rid"
fi

# revoked
if [ -z "$REVOKED_FILE" ]; then
  record token_revoked_401 NOT_RUN "" NOT_RUN ""
else
  RHDR="$TMP/revoked.hdr"
  rtok="$(tr -d '\r\n' <"$REVOKED_FILE")"
  printf 'Authorization: Bearer %s\n' "$rtok" >"$RHDR"
  unset rtok
  chmod 0600 "$RHDR"
  code="$(curl_post "$DOCS/base.json" "$(uuid_v4)" 1 "$RHDR")"
  rid="$(request_id_from_resp)"
  if [ "$code" = "401" ]; then
    record token_revoked_401 PASS "$code" pass "$rid"
  else
    record token_revoked_401 FAIL "$code" fail "$rid"
  fi
fi

# rate_429 last — limited concurrency, max 30
c201=0
c429=0
c5xx=0
cerr=0
i=1
while [ "$i" -le 30 ]; do
  body="$DOCS/rate-$i.json"
  jq --arg ext "FIXTURE-RATE-${RUN_ID}-$i" '.external_id = $ext' "$FIXTURE_DIR/document.base.json" >"$body"
  jq -e . "$body" >/dev/null
  idem="$(uuid_v4)"
  (
    code="$("$CURL_BIN" -sS --max-time 15 -o /dev/null -w '%{http_code}' -X POST \
      -H "Content-Type: application/json" \
      -H "Idempotency-Key: ${idem}" \
      -H @"$HDR" \
      --data-binary @"$body" \
      "${BASE_URL}/documents" || echo 000)"
    printf '%s\n' "$code" >"$TMP/rate-code-$i"
  ) &
  # limit concurrency to 5
  if [ $((i % 5)) -eq 0 ]; then
    wait
  fi
  i=$((i + 1))
done
wait
i=1
while [ "$i" -le 30 ]; do
  code="$(tr -d '\r\n' <"$TMP/rate-code-$i" 2>/dev/null || echo 000)"
  case "$code" in
    201) c201=$((c201 + 1)) ;;
    429) c429=$((c429 + 1)) ;;
    5[0-9][0-9]) c5xx=$((c5xx + 1)) ;;
    000) cerr=$((cerr + 1)) ;;
  esac
  i=$((i + 1))
done
# cooldown (tests may set BWB_POS_KIT_RATE_COOLDOWN=0)
sleep "${BWB_POS_KIT_RATE_COOLDOWN:-5}"
if [ "$c429" -ge 1 ] && [ "$c5xx" -eq 0 ] && [ "$cerr" -eq 0 ]; then
  record rate_429 PASS "429x${c429}" "201=${c201};429=${c429};5xx=${c5xx};transport=${cerr}" ""
else
  record rate_429 FAIL "mixed" "201=${c201};429=${c429};5xx=${c5xx};transport=${cerr}" ""
fi

jq -s --arg run "$RUN_ID" --arg base "$BASE_URL" \
  '{run_id:$run, base_url:$base, cases:., summary:{pass:([.[]|select(.status=="PASS")]|length), fail:([.[]|select(.status=="FAIL")]|length), not_run:([.[]|select(.status=="NOT_RUN")]|length)}}' \
  "$TMP/results.ndjson" >"$REPORT_JSON"

# Sanitized stdout (no fiscal ids, tokens, bodies)
jq -c '{run_id, base_url, summary, cases:[.cases[]|{case,status,http_code,result,request_id}]}' "$REPORT_JSON"

if [ -n "$REPORT_FILE" ]; then
  jq -c '{run_id, base_url, summary, cases:[.cases[]|{case,status,http_code,result,request_id}]}' "$REPORT_JSON" >"$REPORT_FILE"
fi

fails="$(jq '.summary.fail' "$REPORT_JSON")"
[ "$fails" = "0" ] || exit 1
exit 0
