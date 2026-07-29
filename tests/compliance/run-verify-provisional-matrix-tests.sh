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

# Fixture: missing banner / admissions must fail
mkdir -p "${TMP}/compliance/derived/requirements"
cat >"${TMP}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md" <<'EOF'
# bad
| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | x |
| AO-ID-001 | `scaffold` | — | x |
| AO-DOC-001 | `scaffold` | — | x |
| AO-DOC-002 | `scaffold` | — | x |
| AO-SEQ-001 | `scaffold` | — | x |
| AO-SEQ-002 | `scaffold` | — | x |
| AO-IDEM-001 | `scaffold` | — | x |
| AO-TAX-001 | `scaffold` | — | x |
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
AO-LEG-RECT-10-19-2019 reviewed b3db14e2. AO-LEG-DP-71-25-2025 4931fd3c 11902–11920; p.21 DE 372/25 @11921.
AO-LEG-DE-74-19-2019 5b63c80e 1576–1586; n.º34 @1582; Rect 1948–1949. C-SIGN-001.
Art.10 @11908–11909. GAP-002. RM-SRC-004. DOCUMENT-TYPES-MATRIX-RM-REQ-001.
DE 683/25: 19164–19227; p.66 Aviso 4/25 @19228.
### Citação D — schema SAF-T AO
SAFTAO1.01_01.xsd e9a938e1 AuditFile InvoiceType InvoiceNo InvoiceStatus References Hash HashControl PaymentType SAFTAOPaymentType pending_validation Development C-DOC-003 C-SIGN-001 L2023 L1004 L1361 L2740
### Citação G — DE 683 Anexos
Anexo I registarFactura taxType 19166; Anexo II 19193; Anexo III 19194 solicitarSerie; Tabelas 19212–19223.
### Citação H — FE HML FE-RNG
AO-FE-SNAP-HML-2026-07-25-REGISTAR eb430954 AO-FE-SNAP-HML-2026-07-25-SOLICITAR f8fb22e7 AO-FE-SNAP-HML-2026-07-25-LISTAR 5729f02c AO-FE-SNAP-HML-2026-07-25-VALIDAR 7ab70629 FE-RNG-051 FE-RNG-033 registarFactura solicitarSerie listarSeries validarDocumento C-FE-001 pending_validation FE-SERVICES-MATRIX-RM-REQ-001
### Citação I — terminal série DE74
terminal informático 1577
### Citação J — GESTAO chaves
de423e66 GAP-013
ARTIGO 4.º — o critério do catálogo **não** fica satisfeito só com ART.4.
| ID | Estado | Fonte candidata | Nota |
|---|---|---|---|
| ASM-REG-001 | `scaffold` | — | x |
| AO-ID-001 | `partial` | AO-LEG-DE-683-25-2025 | taxRegistrationNumber 19166 softwareValidationNumber 19167 estabelecimento terminal **Não** confirmado; critério **não** fica satisfeito |
| AO-DOC-001 | `scaffold` | — | citação pendente |
| AO-DOC-002 | `promoted` | AO-LEG-DP-71-25-2025 | CONFIRMED-MATRIX confirmed_normative 11904 11907 |
| AO-SEQ-001 | `partial` | AO-LEG-DP-71-25-2025 | Art.10 11908 @1577 critério **não** fica satisfeito; **Não** confirmado |
| AO-SEQ-002 | `partial` | AO-LEG-DE-683-25-2025 | ART. 4 / 19164 solicitarSerie @19183 FE-RNG-051 listarSeries b01e4581 **Não** confirmado |
| AO-IDEM-001 | `scaffold` | — | x |
| AO-TAX-001 | `partial` | AO-LEG-DE-683-25-2025 | taxType 19171 Tabela 19212–19227 critério **não** fica satisfeito; **Não** confirmado |
| AO-CRYPTO-001 | `partial` | AO-LEG-DE-683-25-2025 + AO-FE-SNAP-HML-2026-07-25-ESTRUTURA | jwsDocumentSignature 19168 RS256 pending_validation SAF-T encadeamento **não** citado; critério **não** fica satisfeito; **Não** confirmado |
| AO-KEY-001 | `blocked` | — | x |
| AO-AGT-001 | `pending_validation` | FE | FE-RNG eb430954 Citação H C-FE-001 GAP-006 critério **não** fica satisfeito; **não** confirmado |
| AO-AGT-002 | `partial` | FE | requestID obterEstado f851f512 critério **não** fica satisfeito; **Não** confirmado |
| AO-OFF-001 | `partial` | AO-LEG-DP-71-25-2025 | Art.18 11911 11912 critério **não** fica satisfeito; **Não** confirmado |
| AO-OFF-002 | `partial` | AO-LEG-DE-74-19-2019 | @1580 critério **não** fica satisfeito; **Não** confirmado |
| AO-AUD-001 | `scaffold` | — | x |
| AO-SAF-001 | `pending_validation` | XSD | e9a938e1 InvoiceType L2023 critério **não** fica satisfeito; **não** confirmado |
| AO-SAF-002 | `pending_validation` | XSD | References L1004 @1577 critério **não** fica satisfeito; **não** confirmado |
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
  mkdir -p "${TMP}/compliance/derived/requirements"
  cp "${ROOT}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md" \
    "${TMP}/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md"
  cp "${ROOT}/compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md" \
    "${TMP}/compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md"
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

mutate_real "AO-DOC-001 promovido a partial sem citação" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-DOC-001 | `scaffold`", "| AO-DOC-001 | `partial`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-TAX-001 scaffold indevido" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-TAX-001 | `partial`", "| AO-TAX-001 | `scaffold`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-TAX-001 sem taxType" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8").replace("taxType", "taxKind")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-TAX-001 sem 19212 na linha" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
for line in t.splitlines():
    if line.startswith("| AO-TAX-001 |"):
        bad = line.replace("19212", "xxxxx").replace("19227", "yyyyy")
        t = t.replace(line, bad)
        break
p.write_text(t, encoding="utf-8")
'

mutate_real "Citação G sem Anexo III na secção" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
# Strip Anexo III only inside Citação G; keep later mentions if any.
import re
def strip_g(m):
    return m.group(0).replace("Anexo III", "Anexo X").replace("19194", "xxxxx")
t2 = re.sub(r"###\s+Citação G\b.*?(?=\n###\s|\n##\s|$)", strip_g, t, count=1, flags=re.S)
p.write_text(t2, encoding="utf-8")
'

mutate_real "AO-SEQ-001 scaffold indevido" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-SEQ-001 | `partial`", "| AO-SEQ-001 | `scaffold`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-DOC-002 scaffold indevido" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-DOC-002 | `promoted`", "| AO-DOC-002 | `scaffold`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-OFF-001 blocked indevido" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-OFF-001 | `partial`", "| AO-OFF-001 | `blocked`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-OFF-002 scaffold indevido" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-OFF-002 | `partial`", "| AO-OFF-002 | `scaffold`")
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

mutate_real "AO-SAF-001 sem AuditFile" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8").replace("AuditFile", "RootFile")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-SAF-001 promovido a partial" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-SAF-001 | `pending_validation`", "| AO-SAF-001 | `partial`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-SAF-002 promovido a partial" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-SAF-002 | `pending_validation`", "| AO-SAF-002 | `partial`")
p.write_text(t, encoding="utf-8")
'

mutate_real "Citação D sem InvoiceType na secção" '
from pathlib import Path
import re
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
def strip_d(m):
    return m.group(0).replace("InvoiceType", "DocTypeX").replace("L2023", "Lxxxx")
t2 = re.sub(r"###\s+Citação D\b.*?(?=\n###\s|\n##\s|$)", strip_d, t, count=1, flags=re.S)
p.write_text(t2, encoding="utf-8")
'

mutate_real "Citação D sem Hash exacto (só HashControl)" '
from pathlib import Path
import re
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
def strip_hash(m):
    # Remove standalone Hash but keep HashControl.
    return re.sub(r"(?<![A-Za-z])Hash(?![A-Za-z])", "Digest", m.group(0))
t2 = re.sub(r"###\s+Citação D\b.*?(?=\n###\s|\n##\s|$)", strip_hash, t, count=1, flags=re.S)
p.write_text(t2, encoding="utf-8")
'

mutate_real "AO-AGT-001 blocked indevido" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
t = t.replace("| AO-AGT-001 | `pending_validation`", "| AO-AGT-001 | `blocked`")
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-AGT-001 sem Não confirmado" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
for line in t.splitlines():
    if line.startswith("| AO-AGT-001 |"):
        bad = line.replace("**não** confirmado", "confirmado").replace("não confirmado", "confirmado")
        t = t.replace(line, bad)
        break
p.write_text(t, encoding="utf-8")
'

mutate_real "Citação H sem FE-RNG na secção" '
from pathlib import Path
import re
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
def strip_h(m):
    return m.group(0).replace("FE-RNG-", "ERR-").replace("C-FE-001", "C-XX-001")
t2 = re.sub(r"###\s+Citação H\b.*?(?=\n###\s|\n##\s|$)", strip_h, t, count=1, flags=re.S)
p.write_text(t2, encoding="utf-8")
'

mutate_real "AO-OFF-002 sem gazeta na linha" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
for line in t.splitlines():
    if line.startswith("| AO-OFF-002 |"):
        bad = line.replace("1580", "xxxx").replace("1579", "yyyy")
        t = t.replace(line, bad)
        break
p.write_text(t, encoding="utf-8")
'

mutate_real "AO-SEQ-001 sem 1577 na linha" '
from pathlib import Path
p = Path("'"${TMP}"'/compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")
t = p.read_text(encoding="utf-8")
for line in t.splitlines():
    if line.startswith("| AO-SEQ-001 |"):
        bad = line.replace("1577", "xxxx")
        t = t.replace(line, bad)
        break
p.write_text(t, encoding="utf-8")
'

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
[[ "${fail}" -eq 0 ]]
