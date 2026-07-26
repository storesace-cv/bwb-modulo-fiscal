#!/usr/bin/env bash
# Structural tests for compliance/scripts/verify_document_types_matrix.py
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VERIFY="${ROOT}/compliance/scripts/verify_document_types_matrix.py"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/doc-types.XXXXXX")"
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
cat >"${TMP}/compliance/derived/requirements/DOCUMENT-TYPES-MATRIX-RM-REQ-001.md" <<'EOF'
# incomplete
FT NC
EOF

if python3 "${VERIFY}" --repo-root "${TMP}" >/dev/null 2>&1; then
  bad "fixture incompleta deveria falhar"
else
  ok "fixture incompleta rejeitada"
fi

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
[[ "${fail}" -eq 0 ]]
