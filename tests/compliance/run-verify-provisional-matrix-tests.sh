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
| AO-ID-001 | `partial` | AO-LEG-DE-683-25-2025 | taxRegistrationNumber 19166 softwareValidationNumber 19167 estabelecimento terminal **Não** confirmado; critério **não** fica satisfeito |
| AO-DOC-001 | `blocked` | — | Rect |
| AO-DOC-002 | `blocked` | — | Rect |
| AO-SEQ-001 | `blocked` | — | Rect |
| AO-SEQ-002 | `partial` | AO-LEG-DE-683-25-2025 | ART. 4 / 19164 b01e4581 **Não** confirmado |
| AO-IDEM-001 | `scaffold` | — | x |
| AO-TAX-001 | `blocked` | — | Rect |
| AO-CRYPTO-001 | `partial` | AO-LEG-DE-683-25-2025 + AO-FE-SNAP-HML-2026-07-25-ESTRUTURA | jwsDocumentSignature 19168 RS256 pending_validation SAF-T encadeamento **não** citado; critério **não** fica satisfeito; **Não** confirmado |
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

mutate_real() {
  local label="$1"
  local py="$2"
  cp "${ROOT}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md" \
    "${TMP}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md"
  python3 -c "$py"
  if python3 "${VERIFY}" --repo-root "${TMP}" >/dev/null 2>&1; then
    bad "${label} deveria falhar"
  else
    ok "${label} rejeitado"
  fi
}

mutate_real "AO-SEQ-002 scaffold" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-SEQ-002 | `partial`", "| AO-SEQ-002 | `scaffold`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-DOC-001 desbloqueado" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-DOC-001 | `blocked`", "| AO-DOC-001 | `partial`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-SEQ-001 desbloqueado" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-SEQ-001 | `blocked`", "| AO-SEQ-001 | `scaffold`")
p.write_text(t, encoding="utf-8")
'

mutate_real "sem caveat não fica satisfeito" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("**não** fica satisfeito", "fica satisfeito")
t = t.replace("não fica satisfeito", "fica satisfeito")
p.write_text(t, encoding="utf-8")
'

mutate_real "sem sha256 original DE 683" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("b01e45813eccc54790ce23ae64bba4564731566476bb3dec0105c15ad4f223ca", "deadbeef")
t = t.replace("b01e4581", "deadbeef")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-SEQ-002 sem Não confirmado" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
for line in t.splitlines():
    if line.startswith("| AO-SEQ-002 |"):
        bad = line.replace("**Não** confirmado", "confirmado").replace("não confirmado", "confirmado")
        t = t.replace(line, bad)
        break
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-ID-001 scaffold" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-ID-001 | `partial`", "| AO-ID-001 | `scaffold`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-ID-001 sem taxRegistrationNumber" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8").replace("taxRegistrationNumber", "taxRegField")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-ID-001 lacuna só fora da linha" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
for line in t.splitlines():
    if line.startswith("| AO-ID-001 |"):
        # Remove gap words from the table row only; keep them elsewhere in the doc.
        bad = (
            line.replace("Estabelecimento/terminal", "Campos FE")
            .replace("estabelecimento/terminal", "campos FE")
            .replace("estabelecimento", "ambito")
            .replace("terminal", "ponto")
        )
        t = t.replace(line, bad)
        break
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-CRYPTO-001 blocked" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-CRYPTO-001 | `partial`", "| AO-CRYPTO-001 | `blocked`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-CRYPTO-001 sem RS256" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8").replace("RS256", "ALG-X")
p.write_text(t, encoding="utf-8")
'

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
[[ "${fail}" -eq 0 ]]
