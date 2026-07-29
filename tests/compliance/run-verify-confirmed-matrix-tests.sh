#!/usr/bin/env bash
# Structural tests for compliance/scripts/verify_confirmed_matrix.py
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VERIFY="${ROOT}/compliance/scripts/verify_confirmed_matrix.py"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/conf-matrix.XXXXXX")"
trap 'rm -rf "${TMP}"' EXIT

pass=0
fail=0

ok() { pass=$((pass + 1)); printf 'ok %s\n' "$1"; }
bad() { fail=$((fail + 1)); printf 'FAIL %s\n' "$1" >&2; }

if python3 "${VERIFY}" --repo-root "${ROOT}"; then
  ok "repositório real"
else
  bad "repositório real"
fi

mkdir -p "${TMP}/compliance/derived/requirements"
cat >"${TMP}/compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md" <<'EOF'
# bad
sem banner
EOF

if python3 "${VERIFY}" --repo-root "${TMP}" >/dev/null 2>&1; then
  bad "fixture inválida deveria falhar"
else
  ok "fixture inválida rejeitada"
fi

# Mutate real: strip Art.8 exception token
cp -R "${ROOT}/compliance" "${TMP}/compliance"
python3 - <<PY
from pathlib import Path
p = Path("${TMP}/compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t2 = t.replace("n.º8", "n.X8").replace("Fail-closed", "Open-default")
p.write_text(t2, encoding="utf-8")
PY

if python3 "${VERIFY}" --repo-root "${TMP}" >/dev/null 2>&1; then
  bad "mutação Art.8/Fail-closed deveria falhar"
else
  ok "mutação Art.8/Fail-closed rejeitada"
fi

echo "${pass} passed, ${fail} failed"
if [ "${fail}" -ne 0 ]; then
  exit 1
fi
