#!/usr/bin/env bash
# Structural tests for compliance/scripts/verify_provisional_matrix.py
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VERIFY="${ROOT}/compliance/scripts/verify_provisional_matrix.py"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/prov-matrix.XXXXXX")"
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

# Fixture: missing banner / Rect-dependent not blocked must fail
mkdir -p "${TMP}/compliance/derived/requirements"
cat >"${TMP}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md" <<'EOF'
# bad
| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | x |
| AO-ID-001 | `scaffold` | — | x |
| AO-DOC-001 | `scaffold` | — | should be blocked |
| AO-DOC-002 | `scaffold` | — | should be blocked |
| AO-SEQ-001 | `scaffold` | — | should be blocked |
| AO-SEQ-002 | `scaffold` | — | x |
| AO-IDEM-001 | `scaffold` | — | x |
| AO-TAX-001 | `scaffold` | — | should be blocked |
| AO-CRYPTO-001 | `blocked` | — | x |
| AO-KEY-001 | `blocked` | — | x |
| AO-AGT-001 | `blocked` | — | x |
| AO-AGT-002 | `scaffold` | — | x |
| AO-OFF-001 | `blocked` | — | x |
| AO-OFF-002 | `scaffold` | — | x |
| AO-AUD-001 | `scaffold` | — | x |
| AO-SAF-001 | `pending_validation` | x | x |
| AO-SAF-002 | `pending_validation` | x | x |
| AO-OPS-001 | `scaffold` | — | x |
| AO-UPD-001 | `scaffold` | — | x |
EOF

if python3 "${VERIFY}" --repo-root "${TMP}" >/dev/null 2>&1; then
  bad "fixture inválida deveria falhar"
else
  ok "fixture inválida rejeitada"
fi

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
[[ "${fail}" -eq 0 ]]
