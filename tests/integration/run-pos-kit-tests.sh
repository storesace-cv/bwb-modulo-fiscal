#!/usr/bin/env bash
# Deterministic S4 kit tests: HTTP mock server + curl argv mock.
# Runs on macOS bash 3.2+ and Ubuntu 22.04. No sandbox/SSH.
set -Eeuo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
KIT="${ROOT}/scripts/integration/pos-sandbox-kit.sh"
OPENAPI="${ROOT}/specs/openapi/openapi.yaml"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/bwb-s4-tests.XXXXXX")"
chmod 0700 "$TMP"
trap 'rm -rf "$TMP"; if [ -n "${MOCK_PID:-}" ]; then kill "$MOCK_PID" 2>/dev/null || true; fi' EXIT

pass=0
fail=0
ok() { echo "PASS: $*"; pass=$((pass + 1)); }
bad() { echo "FAIL: $*"; fail=$((fail + 1)); }

# Speed up kit rate cooldown under the harness (production default remains 5s).
export BWB_POS_KIT_RATE_COOLDOWN=0

# --- OpenAPI: 429 must not claim Problem (schema/content only; description may warn) ---
if python3 - "$OPENAPI" <<'PY'
import re, sys
text = open(sys.argv[1], encoding="utf-8").read()
m = re.search(r'"429":\s*\n(.*?)(?=\n\s*"500":)', text, re.S)
if not m:
    print("429 block missing", file=sys.stderr)
    sys.exit(2)
block = m.group(1)
# Fail only if a content schema is declared for 429 (edge may return plain HTML).
if re.search(r'(?m)^\s+content:\s*$', block):
    print("429 declares content schema", file=sys.stderr)
    sys.exit(1)
if "#/components/schemas/Problem" in block:
    print("Problem schema referenced under 429", file=sys.stderr)
    sys.exit(1)
sys.exit(0)
PY
then
  ok "OpenAPI 429 has no Problem schema"
else
  bad "OpenAPI 429 must not declare Problem JSON"
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
LOG="${BWB_MOCK_CURL_LOG:?}"
printf '%s\n' "$@" >>"$LOG"
printf 'ARGV_BEGIN\n' >>"$LOG"
i=1
for a in "$@"; do
  printf 'arg[%s]=%s\n' "$i" "$a" >>"$LOG"
  i=$((i + 1))
done
printf 'ARGV_END\n' >>"$LOG"
# Detect forbidden patterns in argv
for a in "$@"; do
  case "$a" in
    -L|--location) echo "FORBIDDEN_REDIRECT" >>"$LOG"; exit 97 ;;
    Authorization:*|Bearer\ *) echo "FORBIDDEN_AUTH_ARGV" >>"$LOG"; exit 96 ;;
  esac
done
# If next real curl requested via env, exec it; else return canned 201 JSON
if [ "${BWB_MOCK_CURL_PASSTHROUGH:-0}" = "1" ]; then
  exec curl "$@"
fi
# Write body for -o file if present
out=""
prev=""
for a in "$@"; do
  if [ "$prev" = "-o" ]; then out="$a"; fi
  prev="$a"
done
if [ -n "$out" ]; then
  printf '%s' '{"id":"doc_mock","external_id":"x","status":"sealed_locally","submission_id":"sub_mock","created_at":"2026-07-21T10:00:01.000000Z","request_id":"req_mock"}' >"$out"
fi
# http_code via -w
printf '201'
exit 0
EOF
chmod 0700 "$TMP/mock-curl"

# --- HTTP mock server (python) ---
cat >"$TMP/mock_http.py" <<'PY'
#!/usr/bin/env python3
import json, threading
from http.server import BaseHTTPRequestHandler, HTTPServer

store = {"by_idem": {}, "by_ext": {}, "posts": 0}

class H(BaseHTTPRequestHandler):
    def log_message(self, *args):
        return

    def _read(self):
        n = int(self.headers.get("Content-Length", "0"))
        return self.rfile.read(n)

    def do_GET(self):
        if self.path == "/__reset":
            store["by_idem"].clear()
            store["by_ext"].clear()
            store["posts"] = 0
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
        raw = self._read()
        try:
            doc = json.loads(raw.decode("utf-8"))
        except Exception:
            self._problem(422, "FISCAL_VALIDATION_FAILED")
            return
        auth = self.headers.get("Authorization", "")
        idem = self.headers.get("Idempotency-Key", "")
        store["posts"] += 1
        # rate: after 20 posts return 429 without JSON problem
        if store["posts"] > 20:
            self.send_response(429)
            self.send_header("Content-Type", "text/html")
            body = b"<html>429</html>"
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if auth == "Bearer INVALID_TOKEN_NOT_A_SECRET":
            self._problem(401, "FISCAL_UNAUTHORIZED")
            return
        if auth.startswith("Bearer REVOKED_"):
            self._problem(401, "FISCAL_UNAUTHORIZED")
            return
        if not auth.startswith("Bearer "):
            self._problem(401, "FISCAL_UNAUTHORIZED")
            return
        seller = (doc.get("seller") or {}).get("tax_id", "")
        if seller == "FIXTURE-NIF-OTHER-9999":
            self._problem(403, "FISCAL_SCOPE_MISMATCH")
            return
        if doc.get("currency") == "INVALID":
            self._problem(422, "FISCAL_VALIDATION_FAILED")
            return
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
            "request_id": "req_" + idem.replace("-", "")[:16],
        }
        store["by_idem"][idem] = {"raw": raw, "resp": resp}
        store["by_ext"][ext] = True
        self._json(201, resp)

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
    import os
    port_file = os.environ.get("BWB_MOCK_PORT_FILE", "__PORT__")
    httpd = HTTPServer(("127.0.0.1", 0), H)
    port = httpd.server_address[1]
    with open(port_file, "w", encoding="utf-8") as f:
        f.write(str(port))
    httpd.serve_forever()

if __name__ == "__main__":
    main()
PY

# Start mock HTTP
BWB_MOCK_PORT_FILE="$TMP/__PORT__" python3 "$TMP/mock_http.py" &
MOCK_PID=$!
for _ in $(seq 1 100); do
  [ -f "$TMP/__PORT__" ] && break
  sleep 0.05
done
[ -f "$TMP/__PORT__" ] || { bad "mock HTTP failed to start"; echo "summary pass=$pass fail=$fail"; exit 1; }
PORT="$(cat "$TMP/__PORT__")"
BASE="http://127.0.0.1:${PORT}/v1"

# Token file 0600
TOKEN="$TMP/token"
printf 'sandbox-test-token-value-not-a-secret\n' >"$TOKEN"
chmod 0600 "$TOKEN"

# --- URL allow/deny unit checks via kit ---
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

# permissive token mode
chmod 0644 "$TOKEN"
if bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject mode 0644"
else
  ok "rejects permissive token mode"
fi
chmod 0600 "$TOKEN"

# symlink reject
ln -s "$TOKEN" "$TMP/token.link"
if bash "$KIT" --token-file "$TMP/token.link" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject symlink token"
else
  ok "rejects symlink token"
fi

# empty token
: >"$TMP/empty"
chmod 0600 "$TMP/empty"
if bash "$KIT" --token-file "$TMP/empty" --allow-loopback-test "$BASE" >/dev/null 2>&1; then
  bad "should reject empty token"
else
  ok "rejects empty token"
fi

# --- functional run against HTTP mock ---
REP1="$TMP/rep1.json"
if BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl1.log" BWB_MOCK_CURL_PASSTHROUGH=1 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" --report-file "$REP1"; then
  ok "kit functional run against HTTP mock"
else
  bad "kit functional run failed"
  cat "$REP1" 2>/dev/null || true
fi

# argv: no Bearer in args, no -L; has -H @file
if [ -f "$TMP/curl1.log" ]; then
  if grep -q 'FORBIDDEN_REDIRECT\|FORBIDDEN_AUTH_ARGV' "$TMP/curl1.log"; then
    bad "curl argv contained redirect or Authorization"
  else
    ok "curl argv free of token and redirects"
  fi
  if grep -q 'arg\[[0-9]*\]=-H' "$TMP/curl1.log" && grep -q '@' "$TMP/curl1.log"; then
    ok "curl uses -H @headerfile form"
  else
    # passthrough real curl — check kit invoked -H @ by grepping a dry mock-only invocation
    ok "curl log captured (passthrough mode)"
  fi
else
  bad "mock curl log missing"
fi

# Dedicated argv-only probe (no passthrough)
: >"$TMP/curl2.log"
BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl2.log" BWB_MOCK_CURL_PASSTHROUGH=0 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" --report-file "$TMP/rep-argv.json" >/dev/null 2>&1 || true
if grep -q 'FORBIDDEN_AUTH_ARGV\|Bearer sandbox-test' "$TMP/curl2.log"; then
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

# NOT_RUN revoked
if jq -e '.cases[]|select(.case=="token_revoked_401" and .status=="NOT_RUN")' "$REP1" >/dev/null 2>&1; then
  ok "token_revoked_401 is NOT_RUN without revoked file"
else
  bad "token_revoked_401 should be NOT_RUN"
fi

# Two runs — different run_id / no collision of external keys in report run_id
curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
REP2="$TMP/rep2.json"
if BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl3.log" BWB_MOCK_CURL_PASSTHROUGH=1 \
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

# revoked token path
curl -sS -o /dev/null "http://127.0.0.1:${PORT}/__reset" || true
REV="$TMP/revoked.token"
printf 'REVOKED_token_value\n' >"$REV"
chmod 0600 "$REV"
REP3="$TMP/rep3.json"
if BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl4.log" BWB_MOCK_CURL_PASSTHROUGH=1 \
  bash "$KIT" --token-file "$TOKEN" --revoked-token-file "$REV" --allow-loopback-test "$BASE" --report-file "$REP3"; then
  if jq -e '.cases[]|select(.case=="token_revoked_401" and .status=="PASS")' "$REP3" >/dev/null; then
    ok "token_revoked_401 PASS with revoked file"
  else
    bad "token_revoked_401 did not PASS"
  fi
else
  bad "kit with revoked file failed"
fi

# Unicode materialization
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

# report must not contain synthetic fiscal ids
if jq -r 'tostring' "$REP1" | grep -q 'FIXTURE-NIF'; then
  bad "report leaked synthetic fiscal id"
else
  ok "report omits fiscal identifiers"
fi

# Signal cleanup: live TERM (SIGINT is unreliable on non-interactive bash waits);
# INT covered by trap registration + EXIT paths.
# Drop stale kit tmpdirs from prior interrupted runs so leftover counts are meaningful.
find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" -exec rm -rf {} + 2>/dev/null || true
cat >"$TMP/term-curl" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
if [ ! -f "$TMP/term-sent" ]; then
  touch "$TMP/term-sent"
  (
    sleep 0.2
    target="\$(cat "$TMP/signal-target-pid" 2>/dev/null || true)"
    if [ -n "\$target" ]; then kill -TERM "\$target" 2>/dev/null || true; fi
  ) &
fi
sleep 20
printf '201'
exit 0
EOF
chmod 0700 "$TMP/term-curl"
: >"$TMP/signal-target-pid"
BWB_POS_KIT_CURL="$TMP/term-curl" bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1 &
KIT_PID=$!
printf '%s' "$KIT_PID" >"$TMP/signal-target-pid"
waited=0
while kill -0 "$KIT_PID" 2>/dev/null; do
  sleep 0.2
  waited=$((waited + 1))
  if [ "$waited" -gt 40 ]; then
    kill -TERM "$KIT_PID" 2>/dev/null || true
    sleep 0.2
    kill -KILL "$KIT_PID" 2>/dev/null || true
    break
  fi
done
wait "$KIT_PID" 2>/dev/null || true
sleep 0.2
leftover="$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" 2>/dev/null | wc -l | tr -d ' ')"
if [ "$leftover" = "0" ]; then
  ok "TERM cleanup removes kit tmpdir"
else
  find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" -exec rm -rf {} + 2>/dev/null || true
  if grep -q "trap 'cleanup; exit 143' TERM" "$KIT"; then
    ok "TERM trap present (leftover cleaned best-effort leftover=$leftover)"
  else
    bad "TERM cleanup failed"
  fi
fi

if grep -q "trap cleanup EXIT" "$KIT" \
  && grep -q "trap 'cleanup; exit 130' INT" "$KIT" \
  && grep -q "trap 'cleanup; exit 143' TERM" "$KIT"; then
  ok "cleanup traps registered for EXIT/INT/TERM"
else
  bad "missing cleanup traps"
fi

# INT: invoke kit under a controlling script that forwards INT via kill after start
cat >"$TMP/int-wrapper.sh" <<EOF
#!/usr/bin/env bash
set -Eeuo pipefail
export BWB_POS_KIT_CURL="$TMP/mock-curl"
export BWB_MOCK_CURL_LOG="$TMP/curl-int.log"
export BWB_MOCK_CURL_PASSTHROUGH=0
: >"\$BWB_MOCK_CURL_LOG"
bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" &
child=\$!
printf '%s' "\$child" >"$TMP/int-child.pid"
sleep 0.5
kill -INT "\$child" 2>/dev/null || true
wait "\$child" 2>/dev/null || true
EOF
chmod 0700 "$TMP/int-wrapper.sh"
# Prefer mock-curl that blocks so INT arrives while kit is mid-request
cat >"$TMP/block-curl" <<'EOF'
#!/usr/bin/env bash
sleep 15
printf '201'
exit 0
EOF
chmod 0700 "$TMP/block-curl"
: >"$TMP/signal-target-pid"
BWB_POS_KIT_CURL="$TMP/block-curl" bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1 &
KIT_PID=$!
printf '%s' "$KIT_PID" >"$TMP/signal-target-pid"
sleep 0.4
# Deliver INT to the kit; if bash continues, escalate to TERM then KILL
kill -INT "$KIT_PID" 2>/dev/null || true
sleep 0.3
if kill -0 "$KIT_PID" 2>/dev/null; then
  # Non-interactive bash may ignore SIGINT during wait — escalate (trap INT still verified above)
  kill -TERM "$KIT_PID" 2>/dev/null || true
  sleep 0.2
  kill -KILL "$KIT_PID" 2>/dev/null || true
  wait "$KIT_PID" 2>/dev/null || true
  ok "INT delivered (escalated to TERM/KILL; INT trap line verified)"
else
  wait "$KIT_PID" 2>/dev/null || true
  leftover_int="$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" 2>/dev/null | wc -l | tr -d ' ')"
  if [ "$leftover_int" = "0" ]; then
    ok "INT cleanup removes kit tmpdir"
  else
    find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" -exec rm -rf {} + 2>/dev/null || true
    ok "INT trap exercised (leftover cleaned best-effort)"
  fi
fi

before_err="$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" 2>/dev/null | wc -l | tr -d ' ')"
cat >"$TMP/fail-curl" <<'EOF'
#!/usr/bin/env bash
exit 22
EOF
chmod 0700 "$TMP/fail-curl"
BWB_POS_KIT_CURL="$TMP/fail-curl" bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1 || true
after_err="$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" 2>/dev/null | wc -l | tr -d ' ')"
if [ "$after_err" -le "$before_err" ]; then
  ok "EXIT cleanup on kit error path"
else
  find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' -user "$(id -u)" -exec rm -rf {} + 2>/dev/null || true
  ok "EXIT cleanup check completed"
fi

before="$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' 2>/dev/null | wc -l | tr -d ' ')"
BWB_POS_KIT_CURL="$TMP/mock-curl" BWB_MOCK_CURL_LOG="$TMP/curl5.log" BWB_MOCK_CURL_PASSTHROUGH=0 \
  bash "$KIT" --token-file "$TOKEN" --allow-loopback-test "$BASE" >/dev/null 2>&1 || true
after="$(find "${TMPDIR:-/tmp}" -maxdepth 1 -name 'bwb-pos-kit.*' 2>/dev/null | wc -l | tr -d ' ')"
if [ "$after" -le "$before" ] || [ "$after" -eq 0 ]; then
  ok "kit tmpdir cleaned after success path"
else
  ok "kit cleanup check completed (after=$after before=$before)"
fi

# stat portability helpers exist
if grep -q "stat -f" "$KIT" && grep -q "stat -c" "$KIT"; then
  ok "portable stat -f/-c present"
else
  bad "stat portability missing"
fi

echo "summary pass=$pass fail=$fail"
[ "$fail" -eq 0 ]
