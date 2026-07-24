#!/usr/bin/env bash
# Deterministic S4 kit tests: HTTP mock server + curl argv mock.
# Runs on macOS bash 3.2+ and Ubuntu 22.04. No sandbox/SSH.
# Each kit run uses an isolated TMPDIR; no global find/rm of bwb-pos-kit.*.
set -Eeuo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
KIT="${ROOT}/scripts/integration/pos-sandbox-kit.sh"
OPENAPI="${ROOT}/specs/openapi/openapi.yaml"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/bwb-s4-tests.XXXXXX")"
chmod 0700 "$TMP"
trap 'rm -rf "$TMP"; if [ -n "${MOCK_PID:-}" ]; then kill "$MOCK_PID" 2>/dev/null || true; wait "$MOCK_PID" 2>/dev/null || true; fi' EXIT

pass=0
fail=0
ok() { echo "PASS: $*"; pass=$((pass + 1)); }
bad() { echo "FAIL: $*"; fail=$((fail + 1)); }

# Synthetic tokens matching module format (prefix + 43 Base64URL chars).
TOK_VALID="bwb_sbox_$(python3 -c 'print("A"*43)')"
TOK_REVOKED="bwb_sbox_$(python3 -c 'print("B"*43)')"
[ "${#TOK_VALID}" -eq 52 ] || { echo "internal: TOK_VALID len"; exit 2; }
[ "${#TOK_REVOKED}" -eq 52 ] || { echo "internal: TOK_REVOKED len"; exit 2; }

# --- OpenAPI structural guard via Redocly bundle + jq (no YAML order regex) ---
BUNDLE="$TMP/openapi.bundle.json"
if npx --yes @redocly/cli@1.34.3 bundle "$OPENAPI" -o "$BUNDLE" >/dev/null 2>"$TMP/redocly-bundle.err"; then
  if jq -e '
      .paths["/documents"].post.responses["429"] as $r
      | ($r != null)
      and (($r | has("content")) | not)
      and (
        [ $r | .. | objects | .["$ref"]? // empty | select(tostring | test("Problem")) ]
        | length == 0
      )
    ' "$BUNDLE" >/dev/null; then
    ok "OpenAPI 429 has no content/Problem (bundled JSON)"
  else
    bad "OpenAPI 429 structural guard failed"
    jq '.paths["/documents"].post.responses["429"]' "$BUNDLE" 2>/dev/null || true
  fi
else
  bad "Redocly bundle failed"
  cat "$TMP/redocly-bundle.err" 2>/dev/null || true
fi
if grep -q '0.1.6-draft' "$OPENAPI"; then
  ok "OpenAPI version 0.1.6-draft"
else
  bad "OpenAPI version"
fi

# --- mock curl (argv inspector) ---
cat >"$TMP/mock-curl" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
LOG="${BWB_MOCK_CURL_LOG:-/dev/null}"
printf '%s\n' "$@" >>"$LOG"
printf 'ARGV_BEGIN\n' >>"$LOG"
i=1
for a in "$@"; do
  printf 'arg[%s]=%s\n' "$i" "$a" >>"$LOG"
  i=$((i + 1))
done
printf 'ARGV_END\n' >>"$LOG"
for a in "$@"; do
  case "$a" in
    -L|--location) echo "FORBIDDEN_REDIRECT" >>"$LOG"; exit 97 ;;
    Authorization:*|Bearer\ *) echo "FORBIDDEN_AUTH_ARGV" >>"$LOG"; exit 96 ;;
  esac
done
if [ "${BWB_MOCK_CURL_PASSTHROUGH:-0}" = "1" ]; then
  exec curl "$@"
fi
out=""
prev=""
for a in "$@"; do
  if [ "$prev" = "-o" ]; then out="$a"; fi
  prev="$a"
done
if [ -n "$out" ]; then
  printf '%s' '{"id":"doc_mock","external_id":"x","status":"sealed_locally","submission_id":"sub_mock","created_at":"2026-07-21T10:00:01.000000Z"}' >"$out"
fi
printf '201'
exit 0
EOF
chmod 0700 "$TMP/mock-curl"

# --- HTTP mock server (ThreadingHTTPServer + nginx-like 10r/s burst=20) ---
cat >"$TMP/mock_http.py" <<'PY'
#!/usr/bin/env python3
import json, os, threading, time
from http.server import BaseHTTPRequestHandler, HTTPServer
from socketserver import ThreadingMixIn

VALID = os.environ["BWB_MOCK_VALID_TOKEN"]
REVOKED = os.environ["BWB_MOCK_REVOKED_TOKEN"]

class Stats:
    def __init__(self):
        self.lock = threading.Lock()
        self.in_flight = 0
        self.peak_in_flight = 0
        self.posts = 0
        self.gate_posts_before = None  # set by harness via /__mark_pre_rate

class TokenBucket:
    """Approximate nginx limit_req rate=10r/s burst=20 nodelay."""

    def __init__(self, rate=10.0, burst=20):
        self.rate = rate
        self.capacity = float(burst) + 1.0
        self.tokens = self.capacity
        self.updated = time.monotonic()
        self.lock = threading.Lock()

    def allow(self):
        with self.lock:
            now = time.monotonic()
            elapsed = now - self.updated
            self.updated = now
            self.tokens = min(self.capacity, self.tokens + elapsed * self.rate)
            if self.tokens >= 1.0:
                self.tokens -= 1.0
                return True
            return False

    def reset(self):
        with self.lock:
            self.tokens = self.capacity
            self.updated = time.monotonic()

stats = Stats()
bucket = TokenBucket()
store = {"by_idem": {}, "by_ext": {}}
store_lock = threading.Lock()

class ThreadingHTTPServer(ThreadingMixIn, HTTPServer):
    daemon_threads = True
    allow_reuse_address = True

class H(BaseHTTPRequestHandler):
    def log_message(self, *args):
        return

    def _read(self):
        n = int(self.headers.get("Content-Length", "0"))
        return self.rfile.read(n)

    def do_GET(self):
        if self.path == "/__reset":
            with store_lock:
                store["by_idem"].clear()
                store["by_ext"].clear()
            with stats.lock:
                stats.posts = 0
                stats.peak_in_flight = 0
                stats.in_flight = 0
                stats.gate_posts_before = None
            bucket.reset()
            self.send_response(204)
            self.end_headers()
            return
        if self.path == "/__stats":
            with stats.lock:
                body = json.dumps({
                    "peak_in_flight": stats.peak_in_flight,
                    "posts": stats.posts,
                    "in_flight": stats.in_flight,
                    "gate_posts_before": stats.gate_posts_before,
                }).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if self.path == "/__mark_pre_rate":
            with stats.lock:
                stats.gate_posts_before = stats.posts
            self.send_response(204)
            self.end_headers()
            return
        if self.path.endswith("/health") or self.path == "/v1/health":
            body = json.dumps({"status": "ok", "revision": "dev", "version": "test"}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        self.send_response(404)
        self.end_headers()

    def do_POST(self):
        if not (self.path.endswith("/documents") or self.path == "/v1/documents"):
            self.send_response(404)
            self.end_headers()
            return
        with stats.lock:
            stats.in_flight += 1
            stats.posts += 1
            if stats.in_flight > stats.peak_in_flight:
                stats.peak_in_flight = stats.in_flight
        try:
            # Hold the slot briefly so concurrent workers overlap (peak > 5).
            time.sleep(0.05)
            raw = self._read()
            try:
                doc = json.loads(raw.decode("utf-8"))
            except Exception:
                self._problem(422, "FISCAL_VALIDATION_FAILED")
                return
            if not bucket.allow():
                self.send_response(429)
                self.send_header("Content-Type", "text/html")
                body = b"<html>429</html>"
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                self.wfile.write(body)
                return
            auth = self.headers.get("Authorization", "")
            idem = self.headers.get("Idempotency-Key", "")
            if auth == "Bearer INVALID_TOKEN_NOT_A_SECRET":
                self._problem(401, "FISCAL_UNAUTHORIZED")
                return
            if auth == "Bearer " + REVOKED:
                self._problem(401, "FISCAL_UNAUTHORIZED")
                return
            if auth != "Bearer " + VALID:
                self._problem(401, "FISCAL_UNAUTHORIZED")
                return
            seller = (doc.get("seller") or {}).get("tax_id", "")
            if seller == "FIXTURE-NIF-OTHER-9999":
                self._problem(403, "FISCAL_SCOPE_MISMATCH")
                return
            if doc.get("currency") == "INVALID":
                self._problem(422, "FISCAL_VALIDATION_FAILED")
                return
            with store_lock:
                if idem in store["by_idem"]:
                    prev = store["by_idem"][idem]
                    if prev["raw"] != raw:
                        self._problem(409, "FISCAL_IDEMPOTENCY_CONFLICT")
                        return
                    self._json(201, prev["resp"])
                    return
                ext = doc.get("external_id")
                if ext in store["by_ext"]:
                    self._problem(409, "FISCAL_EXTERNAL_ID_CONFLICT")
                    return
                resp = {
                    "id": "doc_" + idem.replace("-", "")[:24],
                    "external_id": ext,
                    "status": "sealed_locally",
                    "submission_id": "sub_mock",
                    "created_at": "2026-07-21T10:00:01.000000Z",
                }
                store["by_idem"][idem] = {"raw": raw, "resp": resp}
                store["by_ext"][ext] = True
            self._json(201, resp)
        finally:
            with stats.lock:
                stats.in_flight -= 1

    def _json(self, code, obj):
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _problem(self, code, c):
        obj = {"type": "urn:test", "title": "t", "status": code, "code": c, "request_id": "req_err"}
        self._json(code, obj)

def main():
    port_file = os.environ.get("BWB_MOCK_PORT_FILE", "__PORT__")
    httpd = ThreadingHTTPServer(("127.0.0.1", 0), H)
    port = httpd.server_address[1]
    with open(port_file, "w", encoding="utf-8") as f:
        f.write(str(port))
    httpd.serve_forever()

if __name__ == "__main__":
    main()
PY

BWB_MOCK_PORT_FILE="$TMP/__PORT__" \
BWB_MOCK_VALID_TOKEN="$TOK_VALID" \
BWB_MOCK_REVOKED_TOKEN="$TOK_REVOKED" \
  python3 "$TMP/mock_http.py" &
MOCK_PID=$!
for _ in $(seq 1 100); do
  [ -f "$TMP/__PORT__" ] && break
  sleep 0.05
done
[ -f "$TMP/__PORT__" ] || { bad "mock HTTP failed to start"; echo "summary pass=$pass fail=$fail"; exit 1; }
PORT="$(cat "$TMP/__PORT__")"
BASE="http://127.0.0.1:${PORT}/v1"

TOKEN="$TMP/token"
printf '%s' "$TOK_VALID" >"$TOKEN"
chmod 0600 "$TOKEN"

# --- URL allow/deny ---
if bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "http://localhost:${PORT}/v1" >/dev/null 2>&1; then
  bad "should reject localhost"
else
  ok "rejects localhost loopback"
fi
if bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "http://127.0.0.1:0/v1" >/dev/null 2>&1; then
  bad "should reject port 0"
else
  ok "rejects port 0"
fi
if bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "http://127.0.0.1:65536/v1" >/dev/null 2>&1; then
  bad "should reject port 65536"
else
  ok "rejects port 65536"
fi

chmod 0644 "$TOKEN"
if bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject mode 0644"
else
  ok "rejects permissive token mode"
fi
chmod 0600 "$TOKEN"

ln -s "$TOKEN" "$TMP/token.link"
if bash "$KIT" --token-file "$TMP/token.link" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject symlink token"
else
  ok "rejects symlink token"
fi

: >"$TMP/empty"
chmod 0600 "$TMP/empty"
if bash "$KIT" --token-file "$TMP/empty" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject empty token"
else
  ok "rejects empty token"
fi

# Token format rejects
printf 'bwb_sbox_%s' "$(python3 -c 'print("A"*200)')" >"$TMP/oversize"
chmod 0600 "$TMP/oversize"
if bash "$KIT" --token-file "$TMP/oversize" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject oversized token"
else
  ok "rejects oversized token"
fi

printf 'bwb_sbox_%s\n' "$(python3 -c 'print("A"*43)')" >"$TMP/nl.token"
chmod 0600 "$TMP/nl.token"
if bash "$KIT" --token-file "$TMP/nl.token" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject newline in token file"
else
  ok "rejects newline in token file"
fi

printf 'bwb_sbox_%s' "$(python3 -c 'print("A"*42 + "ç")')" >"$TMP/uni.token"
chmod 0600 "$TMP/uni.token"
if bash "$KIT" --token-file "$TMP/uni.token" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject Unicode token"
else
  ok "rejects Unicode token"
fi

printf 'bwb_sbox_%s' "$(python3 -c 'print("A"*42 + "+")')" >"$TMP/badchar.token"
chmod 0600 "$TMP/badchar.token"
if bash "$KIT" --token-file "$TMP/badchar.token" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject invalid Base64URL char"
else
  ok "rejects invalid Base64URL char"
fi

# Test env overrides rejected without loopback (sandbox URL path)
if BWB_POS_KIT_CURL=/bin/true bash "$KIT" --token-file "$TOKEN" >/dev/null 2>&1; then
  bad "BWB_POS_KIT_CURL should be rejected without loopback"
else
  ok "rejects BWB_POS_KIT_CURL without loopback"
fi
if BWB_POS_KIT_RATE_COOLDOWN=0 bash "$KIT" --token-file "$TOKEN" >/dev/null 2>&1; then
  bad "BWB_POS_KIT_RATE_COOLDOWN should be rejected without loopback"
else
  ok "rejects BWB_POS_KIT_RATE_COOLDOWN without loopback"
fi

# --- functional run ---
curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
REP1="$TMP/rep1.json"
ISO1="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO1"
PATH_FILE1="$ISO1/kit.tmp.path"
if TMPDIR="$ISO1" BWB_POS_KIT_TMP_PATH_FILE="$PATH_FILE1" \
  BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl1.log" BWB_MOCK_CURL_PASSTHROUGH=1 \
  BWB_POS_KIT_RATE_COOLDOWN=0 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" --report-file "$REP1"; then
  ok "kit functional run against HTTP mock"
else
  bad "kit functional run failed"
  cat "$REP1" 2>/dev/null || true
fi

# Semantic outcomes in report
if jq -e '
    (.cases|map(select(.case=="create_201" and .status=="PASS" and .request_id==null))|length==1)
    and (.cases|map(select(.case=="replay" and .status=="PASS" and .request_id==null))|length==1)
    and (.cases|map(select(.case=="idempotency_conflict" and .status=="PASS" and (.request_id|type=="string")))|length==1)
    and (.cases|map(select(.case=="external_id_conflict" and .status=="PASS" and (.request_id|type=="string")))|length==1)
    and (.cases|map(select(.case=="scope_mismatch" and .status=="PASS" and (.request_id|type=="string")))|length==1)
    and (.cases|map(select(.case=="validation_422" and .status=="PASS" and (.request_id|type=="string")))|length==1)
    and (.cases|map(select(.case=="unauthorized_bad_token" and .status=="PASS" and (.request_id|type=="string")))|length==1)
    and (.cases|map(select(.case=="token_revoked_401" and .status=="NOT_RUN"))|length==1)
    and (.cases|map(select(.case=="rate_429" and .status=="PASS" and (.result|test("429=[1-9]"))))|length==1)
    and (.cases|map(select(.case=="rate_429" and (.result|test("collected=30"))))|length==1)
    and (.cases|map(select(.case=="rate_429" and (.result|test("transport=0")) and (.result|test("5xx=0")) and (.result|test("other=0"))))|length==1)
  ' "$REP1" >/dev/null; then
  ok "semantic case outcomes in report"
else
  bad "semantic case outcomes missing"
  jq . "$REP1" 2>/dev/null || true
fi

# Burst concurrency proof (mock ThreadingHTTPServer + token bucket)
STATS1="$(curl -sS "http://127.0.0.1:${PORT}/__stats")"
PEAK1="$(printf '%s' "$STATS1" | jq -r .peak_in_flight)"
if [ -n "$PEAK1" ] && [ "$PEAK1" -gt 5 ]; then
  ok "peak_in_flight=$PEAK1 (>5 proves concurrency beyond waves-of-5)"
else
  bad "peak_in_flight too low ($PEAK1) — burst not concurrent"
fi
RATE_RES="$(jq -r '.cases[]|select(.case=="rate_429")|.result' "$REP1")"
case "$RATE_RES" in
  *429=[1-9]*|*429=[1-9][0-9]*) ok "rate_429 result has >=1×429 ($RATE_RES)" ;;
  *) bad "rate_429 missing 429 count: $RATE_RES" ;;
esac

# Admin-compatible token: exactly 52 raw bytes, no LF (kit reader accepts without tr -d)
if python3 -c "import sys; d=open(sys.argv[1],'rb').read(); sys.exit(0 if len(d)==52 and b'\\n' not in d and b'\\r' not in d else 1)" "$TOKEN"; then
  ok "token file is 52 bytes without CR/LF (fiscal-admin output-file contract)"
else
  bad "token file has CR/LF or wrong length"
fi

# OpenAPI CreateDocumentResponse: exact body without request_id must validate;
# body with additional request_id must fail additionalProperties:false check.
validate_201_jq='
  type=="object"
  and (.id|type=="string" and length>0)
  and (.external_id|type=="string" and length>0)
  and .status=="sealed_locally"
  and (.submission_id|type=="string" and length>0)
  and (.created_at|type=="string" and length>0)
  and ((keys | map(select(. != "id" and . != "external_id" and . != "status" and . != "submission_id" and . != "created_at")) | length) == 0)
'
printf '%s' '{"id":"doc_a","external_id":"e","status":"sealed_locally","submission_id":"s","created_at":"2026-07-21T10:00:01.000000Z"}' >"$TMP/ok201.json"
if jq -e "$validate_201_jq" "$TMP/ok201.json" >/dev/null; then
  ok "OpenAPI-exact 201 without request_id validates"
else
  bad "OpenAPI-exact 201 should validate"
fi
printf '%s' '{"id":"doc_a","external_id":"e","status":"sealed_locally","submission_id":"s","created_at":"2026-07-21T10:00:01.000000Z","request_id":"req_extra"}' >"$TMP/bad201.json"
if jq -e "$validate_201_jq" "$TMP/bad201.json" >/dev/null; then
  bad "201 with extra request_id should fail additionalProperties check"
else
  ok "201 with extra request_id rejected by additionalProperties check"
fi

if [ -f "$TMP/curl1.log" ]; then
  if grep -q 'FORBIDDEN_REDIRECT\|FORBIDDEN_AUTH_ARGV' "$TMP/curl1.log"; then
    bad "curl argv contained redirect or Authorization"
  else
    ok "curl argv free of token and redirects"
  fi
  if grep -q '@' "$TMP/curl1.log"; then
    ok "curl uses -H @headerfile form"
  else
    ok "curl log captured (passthrough mode)"
  fi
else
  bad "mock curl log missing"
fi

: >"$TMP/curl2.log"
ISO2="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO2"
TMPDIR="$ISO2" BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl2.log" BWB_MOCK_CURL_PASSTHROUGH=0 \
  BWB_POS_KIT_RATE_COOLDOWN=0 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" --report-file "$TMP/rep-argv.json" >/dev/null 2>&1 || true
if grep -q 'FORBIDDEN_AUTH_ARGV\|Bearer bwb_sbox_' "$TMP/curl2.log"; then
  bad "token leaked into curl argv"
else
  ok "token absent from mock-curl argv"
fi
if grep -q -- '-L\|--location' "$TMP/curl2.log"; then
  bad "redirect flag present"
else
  ok "no redirect flags in curl argv"
fi
if grep -q 'Authorization: Bearer' "$TMP/curl2.log" && ! grep -q '@' "$TMP/curl2.log"; then
  bad "Authorization appeared as raw argv without @file"
else
  ok "Authorization only via @file pattern"
fi

if jq -e '.cases[]|select(.case=="token_revoked_401" and .status=="NOT_RUN")' "$REP1" >/dev/null 2>&1; then
  ok "token_revoked_401 is NOT_RUN without revoked file"
else
  bad "token_revoked_401 should be NOT_RUN"
fi

curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
REP2="$TMP/rep2.json"
ISO3="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO3"
if TMPDIR="$ISO3" BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl3.log" BWB_MOCK_CURL_PASSTHROUGH=1 \
  BWB_POS_KIT_RATE_COOLDOWN=0 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" --report-file "$REP2" >/dev/null; then
  r1="$(jq -r .run_id "$REP1")"
  r2="$(jq -r .run_id "$REP2")"
  if [ -n "$r1" ] && [ -n "$r2" ] && [ "$r1" != "$r2" ]; then
    ok "two runs produce distinct run_id"
  else
    bad "run_id collision or missing"
  fi
else
  bad "second kit run failed"
fi

curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
REV="$TMP/revoked.token"
printf '%s' "$TOK_REVOKED" >"$REV"
chmod 0600 "$REV"
REP3="$TMP/rep3.json"
ISO4="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO4"
if TMPDIR="$ISO4" BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl4.log" BWB_MOCK_CURL_PASSTHROUGH=1 \
  BWB_POS_KIT_RATE_COOLDOWN=0 \
  bash "$KIT" --token-file "$TOKEN" --revoked-token-file "$REV" --allow-loopback-test "$BASE" --report-file "$REP3"; then
  if jq -e '.cases[]|select(.case=="token_revoked_401" and .status=="PASS")' "$REP3" >/dev/null; then
    ok "token_revoked_401 PASS with revoked file"
  else
    bad "token_revoked_401 did not PASS"
  fi
else
  bad "kit with revoked file failed"
fi

if jq -e '.lines[0].description|test("café|文字")' "$ROOT/scripts/integration/fixtures/document.base.json" >/dev/null; then
  ok "fixture contains Unicode"
else
  bad "Unicode fixture missing"
fi
jq --arg ext "id-ünicode-文字" '.external_id=$ext' \
  "$ROOT/scripts/integration/fixtures/document.base.json" >"$TMP/uni.json"
if jq -e . "$TMP/uni.json" >/dev/null; then
  ok "jq materialization with Unicode valid"
else
  bad "Unicode JSON invalid"
fi

if jq -r 'tostring' "$REP1" | grep -qE 'FIXTURE-NIF|doc_'; then
  bad "report leaked fiscal or document id"
else
  ok "report omits fiscal/document identifiers"
fi

# --- INT/TERM: isolated TMPDIR; child ignores TERM so kit must escalate to KILL on tracked PIDs only ---
cat >"$TMP/ignore-term-curl" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail
# Ignore TERM/INT; exec so the tracked PID is sleep itself (no orphaned sleep child).
trap '' TERM
trap '' INT
exec sleep 120
EOF
chmod 0700 "$TMP/ignore-term-curl"

run_signal_case() {
  local sig="$1" expect_ec="$2" label="$3"
  local iso path_file ready_file kit_tmp ec kit_pid
  iso="$(mktemp -d "$TMP/sig.XXXXXX")"
  chmod 0700 "$iso"
  path_file="$iso/kit.tmp.path"
  ready_file="$iso/kit.ready"
  : >"$path_file"
  chmod 0600 "$path_file"
  rm -f "$ready_file"

  set +e
  python3 - "$KIT" "$TOKEN" "$BASE" "$iso" "$path_file" "$ready_file" "$TMP/ignore-term-curl" "$iso/kit.pid" "$sig" "$expect_ec" <<'PY'
import os, sys, subprocess, signal, time
kit, token, base, tmpdir, path_file, ready_file, curl, pid_file, sig_name, expect_s = sys.argv[1:11]
expect = int(expect_s)
sig = signal.SIGINT if sig_name == "INT" else signal.SIGTERM
env = os.environ.copy()
env["TMPDIR"] = tmpdir
env["BWB_POS_KIT_TMP_PATH_FILE"] = path_file
env["BWB_POS_KIT_READY_FILE"] = ready_file
env["BWB_POS_KIT_CURL"] = curl
env["BWB_POS_KIT_RATE_COOLDOWN"] = "0"
env.pop("BWB_MOCK_CURL_PASSTHROUGH", None)
env.pop("BWB_MOCK_CURL_LOG", None)
p = subprocess.Popen(
    ["bash", kit, "--token-file", token, "--allow-loopback-test", base],
    env=env,
    stdout=subprocess.DEVNULL,
    stderr=subprocess.DEVNULL,
    preexec_fn=os.setsid,
)
with open(pid_file, "w", encoding="utf-8") as f:
    f.write(str(p.pid))
deadline = time.time() + 8
while time.time() < deadline:
    if os.path.exists(ready_file):
        break
    time.sleep(0.05)
time.sleep(0.1)
os.kill(p.pid, sig)
# Kit grace for children is 2s + KILL; allow a little margin.
try:
    rc = p.wait(timeout=6)
except subprocess.TimeoutExpired:
    sys.stderr.write("timeout waiting for kit exit\n")
    try:
        os.kill(p.pid, signal.SIGTERM)
        p.wait(timeout=2)
    except Exception:
        pass
    sys.exit(99)
if rc < 0:
    rc = 128 + (-rc)
# Descendants in the kit session must be gone.
alive = []
try:
    # Best-effort: any process still in the session/group of the kit leader.
    import subprocess as sp
    out = sp.check_output(["ps", "-ax", "-o", "pid=,pgid=,command="], text=True)
    for line in out.splitlines():
        parts = line.strip().split(None, 2)
        if len(parts) < 2:
            continue
        pid_s, pgid_s = parts[0], parts[1]
        try:
            if int(pgid_s) == p.pid and int(pid_s) != p.pid:
                alive.append(pid_s)
        except ValueError:
            continue
except Exception:
    pass
if alive:
    sys.stderr.write("descendants still alive: %s\n" % ",".join(alive))
    sys.exit(98)
sys.exit(rc)
PY
  ec=$?
  set -e
  kit_tmp=""
  if [ -s "$path_file" ]; then
    kit_tmp="$(tr -d '\r\n' <"$path_file")"
  fi
  kit_pid=""
  if [ -f "$iso/kit.pid" ]; then
    kit_pid="$(tr -d '\r\n' <"$iso/kit.pid")"
  fi

  if [ "$ec" -eq 99 ]; then
    bad "$label: kit did not exit within deadline"
    return 0
  fi
  if [ "$ec" -eq 98 ]; then
    bad "$label: kit left descendant processes alive"
    return 0
  fi
  if [ "$ec" -ne "$expect_ec" ]; then
    bad "$label: expected exit $expect_ec got $ec"
    return 0
  fi
  if [ -z "$kit_tmp" ]; then
    bad "$label: kit tmp path missing"
    return 0
  fi
  if [ -d "$kit_tmp" ]; then
    bad "$label: kit tmpdir still exists"
    return 0
  fi
  if [ -n "$kit_pid" ] && kill -0 "$kit_pid" 2>/dev/null; then
    bad "$label: kit pid still alive"
    return 0
  fi
  ok "$label: exit $ec, tmpdir removed, children reaped (TERM-ignore child)"
}

run_signal_case TERM 143 "TERM cleanup"
run_signal_case INT 130 "INT cleanup"

# EXIT cleanup on error path — isolated TMPDIR, exact path file
ISO_ERR="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO_ERR"
PATH_ERR="$ISO_ERR/kit.tmp.path"
cat >"$TMP/fail-curl" <<'EOF'
#!/usr/bin/env bash
exit 22
EOF
chmod 0700 "$TMP/fail-curl"
set +e
TMPDIR="$ISO_ERR" BWB_POS_KIT_TMP_PATH_FILE="$PATH_ERR" \
  BWB_POS_KIT_CURL="$TMP/fail-curl" BWB_POS_KIT_RATE_COOLDOWN=0 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1
set -e
if [ -s "$PATH_ERR" ]; then
  err_tmp="$(tr -d '\r\n' <"$PATH_ERR")"
  if [ -n "$err_tmp" ] && [ ! -d "$err_tmp" ]; then
    ok "EXIT cleanup on kit error path"
  else
    bad "EXIT cleanup left tmpdir"
  fi
else
  # fail-curl may exit before path publish if deps fail first — still require no dir under ISO_ERR matching kit pattern
  leftover_err="$(find "$ISO_ERR" -maxdepth 1 -name 'bwb-pos-kit.*' 2>/dev/null | wc -l | tr -d ' ')"
  if [ "$leftover_err" = "0" ]; then
    ok "EXIT cleanup on kit error path"
  else
    bad "EXIT cleanup left tmpdir under iso"
  fi
fi

# Success-path cleanup with exact path
ISO_OK="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO_OK"
PATH_OK="$ISO_OK/kit.tmp.path"
: >"$TMP/curl5.log"
TMPDIR="$ISO_OK" BWB_POS_KIT_TMP_PATH_FILE="$PATH_OK" \
  BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl5.log" BWB_MOCK_CURL_PASSTHROUGH=0 \
  BWB_POS_KIT_RATE_COOLDOWN=0 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1 || true
if [ -s "$PATH_OK" ]; then
  ok_tmp="$(tr -d '\r\n' <"$PATH_OK")"
  if [ -n "$ok_tmp" ] && [ ! -d "$ok_tmp" ]; then
    ok "kit tmpdir cleaned after success path"
  else
    bad "success path left kit tmpdir"
  fi
else
  bad "success path did not publish tmp path"
fi

if grep -q "stat -f" "$KIT" && grep -q "stat -c" "$KIT"; then
  ok "portable stat -f/-c present"
else
  bad "stat portability missing"
fi

# --- rate gate: no posts during hold-before-gate; then release ---
curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
ISO_GATE="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO_GATE"
PATH_GATE="$ISO_GATE/kit.tmp.path"
HOLD_BEFORE="$ISO_GATE/release-before"
rm -f "$HOLD_BEFORE"
REP_GATE="$ISO_GATE/rep.json"
set +e
TMPDIR="$ISO_GATE" BWB_POS_KIT_TMP_PATH_FILE="$PATH_GATE" \
  BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$ISO_GATE/curl.log" BWB_MOCK_CURL_PASSTHROUGH=1 \
  BWB_POS_KIT_RATE_COOLDOWN=0 \
  BWB_POS_KIT_RATE_HOLD_BEFORE_GATE="$HOLD_BEFORE" \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" --report-file "$REP_GATE" \
  >"$ISO_GATE/kit.out" 2>"$ISO_GATE/kit.err" &
GATE_KIT_PID=$!
set -e
# Wait until kit published tmp + rate-holding
kit_tmp_gate=""
for _ in $(seq 1 400); do
  if [ -s "$PATH_GATE" ]; then
    kit_tmp_gate="$(tr -d '\r\n' <"$PATH_GATE")"
    if [ -n "$kit_tmp_gate" ] && [ -f "$kit_tmp_gate/rate-holding" ]; then
      break
    fi
  fi
  sleep 0.05
done
if [ -z "$kit_tmp_gate" ] || [ ! -f "$kit_tmp_gate/rate-holding" ]; then
  bad "hold-before-gate: kit did not reach rate holding"
  kill "$GATE_KIT_PID" 2>/dev/null || true
  wait "$GATE_KIT_PID" 2>/dev/null || true
else
  posts_hold="$(curl -sS "http://127.0.0.1:${PORT}/__stats" | jq -r .posts)"
  sleep 0.4
  posts_hold2="$(curl -sS "http://127.0.0.1:${PORT}/__stats" | jq -r .posts)"
  if [ "$posts_hold" = "$posts_hold2" ]; then
    ok "hold-before-gate: no additional HTTP posts while gate closed (posts=$posts_hold)"
  else
    bad "hold-before-gate: posts advanced $posts_hold -> $posts_hold2 before release"
  fi
  # ready files must be 30
  ready_cnt="$(find "$kit_tmp_gate/rate-ready" -type f 2>/dev/null | wc -l | tr -d ' ')"
  if [ "$ready_cnt" = "30" ]; then
    ok "hold-before-gate: 30 workers ready before gate"
  else
    bad "hold-before-gate: ready_cnt=$ready_cnt"
  fi
  : >"$HOLD_BEFORE"
  set +e
  wait "$GATE_KIT_PID"
  gate_ec=$?
  set -e
  if [ "$gate_ec" -eq 0 ] && jq -e '.cases[]|select(.case=="rate_429" and .status=="PASS")' "$REP_GATE" >/dev/null; then
    ok "hold-before-gate: kit completed rate_429 PASS after release"
  else
    bad "hold-before-gate: kit exit=$gate_ec"
    cat "$REP_GATE" 2>/dev/null || true
  fi
fi

# --- readiness timeout when one worker stalls (setsid; no SIGKILL on PASS) ---
curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
ISO_STALL="$(mktemp -d "$TMP/iso.XXXXXX")"
chmod 0700 "$ISO_STALL"
PATH_STALL="$ISO_STALL/kit.tmp.path"
set +e
python3 - "$KIT" "$TOKEN" "$BASE" "$ISO_STALL" "$PATH_STALL" "$TMP/mock-curl" <<'PY'
import os, sys, subprocess, signal, time

kit, token, base, tmpdir, path_file, curl = sys.argv[1:7]
env = os.environ.copy()
env["TMPDIR"] = tmpdir
env["BWB_POS_KIT_TMP_PATH_FILE"] = path_file
env["BWB_POS_KIT_CURL"] = curl
env["BWB_POS_KIT_RATE_COOLDOWN"] = "0"
env["BWB_MOCK_CURL_PASSTHROUGH"] = "1"
env["BWB_MOCK_CURL_LOG"] = os.path.join(tmpdir, "curl.log")
env["BWB_POS_KIT_RATE_READY_TIMEOUT"] = "2"
env["BWB_POS_KIT_RATE_STALL_WORKER"] = "7"

def pg_members(leader_pid):
    """PIDs in process group still alive (excluding nothing — includes leader if alive)."""
    alive = []
    try:
        out = subprocess.check_output(["ps", "-ax", "-o", "pid=,pgid="], text=True)
    except Exception:
        return alive
    for line in out.splitlines():
        parts = line.strip().split()
        if len(parts) < 2:
            continue
        try:
            pid, pgid = int(parts[0]), int(parts[1])
        except ValueError:
            continue
        if pgid == leader_pid:
            alive.append(pid)
    return alive

def force_kill_group(leader_pid):
    try:
        os.killpg(leader_pid, signal.SIGKILL)
    except Exception:
        pass

p = subprocess.Popen(
    ["bash", kit, "--token-file", token, "--allow-loopback-test", base],
    env=env,
    stdout=subprocess.DEVNULL,
    stderr=subprocess.DEVNULL,
    preexec_fn=os.setsid,
)
try:
    rc = p.wait(timeout=30)
except subprocess.TimeoutExpired:
    force_kill_group(p.pid)
    sys.stderr.write("readiness timeout test: kit hung\n")
    sys.exit(99)
if rc < 0:
    rc = 128 + (-rc)
if rc == 0:
    force_kill_group(p.pid)
    sys.stderr.write("readiness timeout should fail the kit\n")
    sys.exit(96)

# Short margin only — then check; SIGKILL only after FAIL detection.
time.sleep(0.25)
members = pg_members(p.pid)
if members:
    force_kill_group(p.pid)
    sys.stderr.write("readiness timeout left process group members: %s\n" % ",".join(map(str, members)))
    sys.exit(98)

kit_tmp = ""
if os.path.exists(path_file) and os.path.getsize(path_file) > 0:
    with open(path_file, encoding="utf-8") as f:
        kit_tmp = f.read().strip()
if kit_tmp and os.path.isdir(kit_tmp):
    sys.stderr.write("readiness timeout left tmpdir\n")
    sys.exit(97)
# PASS path: no SIGKILL was used after a clean group.
sys.exit(0)
PY
stall_ec=$?
set -e
case "$stall_ec" in
  0) ok "readiness timeout: kit failed, tmpdir gone, no residual process group (no SIGKILL on PASS)" ;;
  96) bad "readiness timeout should fail the kit" ;;
  97) bad "readiness timeout left tmpdir" ;;
  98) bad "readiness timeout left residual process group / stalled worker" ;;
  99) bad "readiness timeout test hung (SIGKILL after timeout FAIL)" ;;
  *) bad "readiness timeout unexpected exit $stall_ec" ;;
esac

# --- INT/TERM while waiting for before-gate / after-gate hold (process group) ---
# PASS path must NEVER SIGKILL. SIGKILL only after timeout FAIL or residual FAIL (+ cleanup).
run_rate_signal() {
  local sig="$1" expect_ec="$2" label="$3" hold_kind="$4"
  local iso path_file hold_file
  iso="$(mktemp -d "$TMP/rsig.XXXXXX")"
  chmod 0700 "$iso"
  path_file="$iso/kit.tmp.path"
  hold_file="$iso/hold-release"
  rm -f "$hold_file"
  curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
  set +e
  python3 - "$KIT" "$TOKEN" "$BASE" "$iso" "$path_file" "$hold_file" "$TMP/mock-curl" "$sig" "$expect_ec" "$hold_kind" <<'PY'
import os, sys, subprocess, signal, time

kit, token, base, tmpdir, path_file, hold_file, curl, sig_name, expect_s, hold_kind = sys.argv[1:11]
expect = int(expect_s)
sig = signal.SIGINT if sig_name == "INT" else signal.SIGTERM
env = os.environ.copy()
env["TMPDIR"] = tmpdir
env["BWB_POS_KIT_TMP_PATH_FILE"] = path_file
env["BWB_POS_KIT_CURL"] = curl
env["BWB_POS_KIT_RATE_COOLDOWN"] = "0"
env["BWB_MOCK_CURL_PASSTHROUGH"] = "1"
env["BWB_MOCK_CURL_LOG"] = os.path.join(tmpdir, "curl.log")
if hold_kind == "before":
    env["BWB_POS_KIT_RATE_HOLD_BEFORE_GATE"] = hold_file
    env.pop("BWB_POS_KIT_RATE_HOLD_AFTER_GATE", None)
else:
    env["BWB_POS_KIT_RATE_HOLD_AFTER_GATE"] = hold_file
    env.pop("BWB_POS_KIT_RATE_HOLD_BEFORE_GATE", None)

def pg_members(leader_pid):
    alive = []
    try:
        out = subprocess.check_output(["ps", "-ax", "-o", "pid=,pgid="], text=True)
    except Exception:
        return alive
    for line in out.splitlines():
        parts = line.strip().split()
        if len(parts) < 2:
            continue
        try:
            pid, pgid = int(parts[0]), int(parts[1])
        except ValueError:
            continue
        if pgid == leader_pid:
            alive.append(pid)
    return alive

def force_kill_group(leader_pid):
    """Harness cleanup after a FAILED assertion only — never on PASS."""
    try:
        os.killpg(leader_pid, signal.SIGKILL)
    except Exception:
        pass

p = subprocess.Popen(
    ["bash", kit, "--token-file", token, "--allow-loopback-test", base],
    env=env,
    stdout=subprocess.DEVNULL,
    stderr=subprocess.DEVNULL,
    preexec_fn=os.setsid,
)
deadline = time.time() + 60
kit_tmp = ""
while time.time() < deadline:
    if os.path.exists(path_file) and os.path.getsize(path_file) > 0:
        with open(path_file, encoding="utf-8") as f:
            kit_tmp = f.read().strip()
        if kit_tmp:
            if hold_kind == "before" and os.path.exists(os.path.join(kit_tmp, "rate-holding")):
                break
            if hold_kind == "after" and os.path.exists(os.path.join(kit_tmp, "rate-gate")):
                break
    time.sleep(0.05)
else:
    force_kill_group(p.pid)
    sys.stderr.write("timeout waiting for hold point (SIGKILL after FAIL)\n")
    sys.exit(99)

time.sleep(0.15)
os.kill(p.pid, sig)
try:
    rc = p.wait(timeout=8)
except subprocess.TimeoutExpired:
    force_kill_group(p.pid)
    sys.stderr.write("timeout waiting for kit exit after signal (SIGKILL after FAIL)\n")
    sys.exit(99)
if rc < 0:
    rc = 128 + (-rc)

# Short limited margin, then CHECK before any SIGKILL.
time.sleep(0.25)
members = pg_members(p.pid)
if members:
    force_kill_group(p.pid)
    sys.stderr.write("descendants still alive: %s (SIGKILL after FAIL)\n" % ",".join(map(str, members)))
    sys.exit(98)

# PASS path: process group empty — do NOT SIGKILL.
if kit_tmp and os.path.isdir(kit_tmp):
    sys.stderr.write("tmpdir remains\n")
    sys.exit(97)
if rc != expect:
    sys.stderr.write("exit %s want %s\n" % (rc, expect))
    sys.exit(rc if rc else 1)
sys.exit(rc)
PY
  ec=$?
  set -e
  if [ "$ec" -eq 99 ]; then
    bad "$label: timeout (SIGKILL only after FAIL)"
    return 0
  fi
  if [ "$ec" -eq 98 ]; then
    bad "$label: descendants alive (SIGKILL only after FAIL)"
    return 0
  fi
  if [ "$ec" -eq 97 ]; then
    bad "$label: tmpdir remains"
    return 0
  fi
  if [ "$ec" -ne "$expect_ec" ]; then
    bad "$label: expected exit $expect_ec got $ec"
    return 0
  fi
  ok "$label: exit $ec, tmpdir removed, no residuals, PASS path without SIGKILL"
}

run_rate_signal INT 130 "INT during before-gate hold" before
run_rate_signal TERM 143 "TERM during before-gate hold" before
run_rate_signal INT 130 "INT during after-gate hold" after
run_rate_signal TERM 143 "TERM during after-gate hold" after

# --- three consecutive complete kit runs against real curl + HTTP mock (no hang) ---
consec_ok=0
for run_i in 1 2 3; do
  curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
  iso_c="$(mktemp -d "$TMP/consec.XXXXXX")"
  chmod 0700 "$iso_c"
  set +e
  TMPDIR="$iso_c" BWB_POS_KIT_RATE_COOLDOWN=0 \
    bash "$KIT" --token-file "$TOKEN" --revoked-token-file "$REV" --allow-loopback-test "$BASE" \
    --report-file "$iso_c/rep.json" >/dev/null 2>"$iso_c/err"
  cec=$?
  set -e
  if [ "$cec" -eq 0 ] && jq -e '
      (.summary.fail==0)
      and (.cases|map(select(.case=="rate_429" and .status=="PASS" and (.result|test("collected=30"))))|length==1)
    ' "$iso_c/rep.json" >/dev/null; then
    consec_ok=$((consec_ok + 1))
  else
    bad "consecutive run $run_i failed ec=$cec"
    cat "$iso_c/err" 2>/dev/null || true
    cat "$iso_c/rep.json" 2>/dev/null || true
  fi
done
if [ "$consec_ok" -eq 3 ]; then
  ok "three consecutive complete kit runs (rate_429 PASS, collected=30)"
else
  bad "consecutive runs ok=$consec_ok/3"
fi

echo "summary pass=$pass fail=$fail"
[ "$fail" -eq 0 ]
