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

# Fixture: banner de não-confirmação NÃO pode mascarar afirmação afirmativa posterior
cat >"${TMP}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md" <<'EOF'
# Matriz provisória
**Estado:** `EM_CURSO` — **não** é matriz AO-* confirmada.
AO-LEG-RECT-10-19-2019 excluída. GAP-002. RM-SRC-004 BLOQUEADO.
DE 683/25: 19164–19227; p.66 Aviso 4/25 @19228.
ARTIGO 4.º — o critério do catálogo **não** fica satisfeito só com ART.4.
| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | x |
| AO-ID-001 | `scaffold` | — | x |
| AO-DOC-001 | `blocked` | — | Rect |
| AO-DOC-002 | `blocked` | — | Rect |
| AO-SEQ-001 | `blocked` | — | Rect |
| AO-SEQ-002 | `partial` | AO-LEG-DE-683-25-2025 | ART. 4 / 19164 |
| AO-IDEM-001 | `scaffold` | — | x |
| AO-TAX-001 | `blocked` | — | Rect |
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

Os requisitos AO-* confirmados estão prontos para implementação.
EOF

if python3 "${VERIFY}" --repo-root "${TMP}" >/dev/null 2>&1; then
  bad "afirmação afirmativa deveria falhar apesar do banner"
else
  ok "afirmação afirmativa rejeitada com banner presente"
fi

# Fixture: AO-SEQ-002 scaffold (em vez de partial) deve falhar
cp "${ROOT}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md" \
  "${TMP}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md"
python3 - <<PY
from pathlib import Path
p = Path("${TMP}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-SEQ-002 | \`partial\`", "| AO-SEQ-002 | \`scaffold\`")
p.write_text(t, encoding="utf-8")
PY
if python3 "${VERIFY}" --repo-root "${TMP}" >/dev/null 2>&1; then
  bad "AO-SEQ-002 scaffold deveria falhar"
else
  ok "AO-SEQ-002 scaffold rejeitado"
fi

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
[[ "${fail}" -eq 0 ]]
