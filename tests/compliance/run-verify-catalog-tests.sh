#!/usr/bin/env bash
# Deterministic regression tests for compliance/scripts/verify_catalog.py
# No network. Fixtures under TMP; does not require local/ or the archived ZIP.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VERIFY="${ROOT}/compliance/scripts/verify_catalog.py"
REQ="${ROOT}/compliance/scripts/requirements.txt"
TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

pass=0
fail=0
ok() { echo "ok $1"; pass=$((pass + 1)); }
bad() { echo "FAIL $1"; fail=$((fail + 1)); }

# Use project venv if present; otherwise create ephemeral one.
if [[ -x "${ROOT}/compliance/scripts/.venv/bin/python" ]]; then
  PY="${ROOT}/compliance/scripts/.venv/bin/python"
else
  python3 -m venv "${TMP}/venv"
  "${TMP}/venv/bin/pip" install -q -r "${REQ}"
  PY="${TMP}/venv/bin/python"
fi

# Patch verify_catalog REPO_ROOT by copying tree subset into TMP and running with
# PYTHONPATH trick: we invoke the script after rewriting paths via symlink layout.
# Simpler: mutate a copy of the real catalog+schemas under TMP and point script
# by injecting env — but script uses __file__.parents[2]. So copy scripts+catalog+schemas.
setup_fixture() {
  local d="$1"
  mkdir -p "${d}/compliance/scripts" \
    "${d}/compliance/catalog/schema" \
    "${d}/compliance/saft-ao/schemas"
  cp "${VERIFY}" "${d}/compliance/scripts/verify_catalog.py"
  cp "${ROOT}/compliance/catalog/sources.yaml" "${d}/compliance/catalog/sources.yaml"
  cp "${ROOT}/compliance/catalog/schema/sources.schema.json" "${d}/compliance/catalog/schema/sources.schema.json"
  cp -R "${ROOT}/compliance/saft-ao/schemas/." "${d}/compliance/saft-ao/schemas/"
}

run_v() {
  local d="$1"
  "${PY}" "${d}/compliance/scripts/verify_catalog.py"
}

# --- valid ---
setup_fixture "${TMP}/valid"
if run_v "${TMP}/valid" >/dev/null; then ok "git_public válido"; else bad "git_public válido"; fi

# --- path ausente (null) ---
setup_fixture "${TMP}/nop"
"${PY}" - <<'PY' "${TMP}/nop/compliance/catalog/sources.yaml"
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = re.sub(
    r"(id: AO-SAFT-XSD-1.01_01\n(?:.*\n)*?    versioned_path: )compliance/saft-ao/schemas/SAFTAO1.01_01.xsd",
    r"\1null",
    text,
    count=1,
)
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/nop" >/dev/null 2>"${TMP}/nop.err"; then bad "path ausente"; else
  if grep -qi 'git_public exige versioned_path\|versioned_path' "${TMP}/nop.err"; then ok "path ausente"; else bad "path ausente (msg)"; fi
fi

# --- ficheiro inexistente ---
setup_fixture "${TMP}/miss"
rm -f "${TMP}/miss/compliance/saft-ao/schemas/SAFTAO1.01_01.xsd"
if run_v "${TMP}/miss" >/dev/null 2>"${TMP}/miss.err"; then bad "ficheiro inexistente"; else
  if grep -qi 'missing\|ausente\|sha256\|manifesto\|SHA256SUMS\|diverge' "${TMP}/miss.err"; then ok "ficheiro inexistente"; else bad "ficheiro inexistente (msg)"; fi
fi

# --- hash divergente no catálogo ---
setup_fixture "${TMP}/hashcat"
"${PY}" - <<'PY' "${TMP}/hashcat/compliance/catalog/sources.yaml"
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace(
    "sha256: e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631",
    "sha256: " + ("0" * 64),
    1,
)
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/hashcat" >/dev/null 2>"${TMP}/hashcat.err"; then bad "hash divergente"; else
  if grep -qi 'sha256' "${TMP}/hashcat.err"; then ok "hash divergente"; else bad "hash divergente (msg)"; fi
fi

# --- alteração de um byte no XSD ---
setup_fixture "${TMP}/onebyte"
python3 - <<'PY' "${TMP}/onebyte/compliance/saft-ao/schemas/SAFTAO1.01_01.xsd"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
data = bytearray(p.read_bytes())
data[-1] = (data[-1] + 1) % 256
p.write_bytes(data)
PY
if run_v "${TMP}/onebyte" >/dev/null 2>"${TMP}/onebyte.err"; then bad "alteração de um byte"; else
  if grep -qi 'sha256\|manifesto\|diverge' "${TMP}/onebyte.err"; then ok "alteração de um byte"; else bad "alteração de um byte (msg)"; fi
fi

# --- path absoluto ---
setup_fixture "${TMP}/abs"
"${PY}" - <<'PY' "${TMP}/abs/compliance/catalog/sources.yaml"
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace(
    "versioned_path: compliance/saft-ao/schemas/SAFTAO1.01_01.xsd",
    "versioned_path: /tmp/SAFTAO1.01_01.xsd",
    1,
)
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/abs" >/dev/null 2>"${TMP}/abs.err"; then bad "path absoluto"; else
  if grep -qi 'absoluto' "${TMP}/abs.err"; then ok "path absoluto"; else bad "path absoluto (msg)"; fi
fi

# --- traversal ---
setup_fixture "${TMP}/trav"
"${PY}" - <<'PY' "${TMP}/trav/compliance/catalog/sources.yaml"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace(
    "versioned_path: compliance/saft-ao/schemas/SAFTAO1.01_01.xsd",
    "versioned_path: compliance/../compliance/saft-ao/schemas/SAFTAO1.01_01.xsd",
    1,
)
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/trav" >/dev/null 2>"${TMP}/trav.err"; then bad "traversal"; else
  if grep -qi 'traversal\|fora\|normalizado' "${TMP}/trav.err"; then ok "traversal"; else bad "traversal (msg)"; fi
fi

# --- symlink ---
setup_fixture "${TMP}/sym"
python3 - <<'PY' "${TMP}/sym/compliance/saft-ao/schemas"
import os, pathlib, sys
d = pathlib.Path(sys.argv[1])
target = d / "SAFTAO1.01_01.xsd"
backup = d / "SAFTAO1.01_01.xsd.bak"
target.rename(backup)
os.symlink(backup.name, target)
PY
if run_v "${TMP}/sym" >/dev/null 2>"${TMP}/sym.err"; then bad "symlink"; else
  if grep -qi 'symlink' "${TMP}/sym.err"; then ok "symlink"; else bad "symlink (msg)"; fi
fi

# --- local_only com versioned_path (ZIP) ---
setup_fixture "${TMP}/ziplocal"
"${PY}" - <<'PY' "${TMP}/ziplocal/compliance/catalog/sources.yaml"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
# AO-SAFT-ZIP has versioned_path: null — set a fake path
text = text.replace(
    "id: AO-SAFT-ZIP-ASSOFT-MASTER",
    "id: AO-SAFT-ZIP-ASSOFT-MASTER",
    1,
)
# replace first versioned_path: null after ZIP id — crude: replace ZIP block's null
idx = text.find("id: AO-SAFT-ZIP-ASSOFT-MASTER")
assert idx != -1
chunk = text[idx:idx+800]
chunk2 = chunk.replace("versioned_path: null", "versioned_path: compliance/saft-ao/schemas/LICENSE", 1)
text = text[:idx] + chunk2 + text[idx+len(chunk):]
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/ziplocal" >/dev/null 2>"${TMP}/ziplocal.err"; then bad "local_only versioned_path"; else
  if grep -qi 'proíbe versioned_path\|nunca com versioned_path\|versioned_path' "${TMP}/ziplocal.err"; then ok "local_only versioned_path"; else bad "local_only versioned_path (msg)"; fi
fi

# --- private_sync com versioned_path ---
setup_fixture "${TMP}/privvp"
"${PY}" - <<'PY' "${TMP}/privvp/compliance/catalog/sources.yaml"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
idx = text.find("id: AO-LEG-DE-74-19-2019")
chunk = text[idx:idx+1200]
chunk2 = chunk.replace("versioned_path: null", "versioned_path: compliance/saft-ao/schemas/LICENSE", 1)
text = text[:idx] + chunk2 + text[idx+len(chunk):]
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/privvp" >/dev/null 2>"${TMP}/privvp.err"; then bad "private_sync versioned_path"; else
  if grep -qi 'proíbe versioned_path\|versioned_path' "${TMP}/privvp.err"; then ok "private_sync versioned_path"; else bad "private_sync versioned_path (msg)"; fi
fi

# --- git_public sem versioned_path (same as path ausente — already covered; keep explicit) ---
ok "git_public sem versioned_path (coberto por path ausente)"

# --- git_public licença não permitted ---
setup_fixture "${TMP}/badlic"
"${PY}" - <<'PY' "${TMP}/badlic/compliance/catalog/sources.yaml"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
idx = text.find("id: AO-SAFT-XSD-1.01_01")
chunk = text[idx:idx+900]
chunk2 = chunk.replace("license_redistribution: permitted", "license_redistribution: uncertain", 1)
text = text[:idx] + chunk2 + text[idx+len(chunk):]
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/badlic" >/dev/null 2>"${TMP}/badlic.err"; then bad "licença não permitida"; else
  if grep -qi 'license_redistribution=permitted\|permitted' "${TMP}/badlic.err"; then ok "licença não permitida"; else bad "licença não permitida (msg)"; fi
fi

# --- ZIP git_public ---
setup_fixture "${TMP}/zippub"
"${PY}" - <<'PY' "${TMP}/zippub/compliance/catalog/sources.yaml"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
idx = text.find("id: AO-SAFT-ZIP-ASSOFT-MASTER")
chunk = text[idx:idx+900]
chunk2 = chunk.replace("storage: local_only", "storage: git_public", 1)
chunk2 = chunk2.replace("license_redistribution: local_only", "license_redistribution: permitted", 1)
chunk2 = chunk2.replace("versioned_path: null", "versioned_path: compliance/saft-ao/schemas/LICENSE", 1)
text = text[:idx] + chunk2 + text[idx+len(chunk):]
p.write_text(text, encoding="utf-8")
PY
if run_v "${TMP}/zippub" >/dev/null 2>"${TMP}/zippub.err"; then bad "ZIP git_public"; else
  if grep -qi 'ZIP\|archive\|local_only\|git_public' "${TMP}/zippub.err"; then ok "ZIP git_public"; else bad "ZIP git_public (msg)"; fi
fi

# --- ZIP com versioned_path (local_only) ---
# already covered by ziplocal

# --- manifesto válido (real fixture) ---
if run_v "${TMP}/valid" >/dev/null; then ok "manifesto válido"; else bad "manifesto válido"; fi

# --- alteração manifesto ---
setup_fixture "${TMP}/manbad"
echo "0000000000000000000000000000000000000000000000000000000000000000  LICENSE" > "${TMP}/manbad/compliance/saft-ao/schemas/SHA256SUMS.txt"
if run_v "${TMP}/manbad" >/dev/null 2>"${TMP}/manbad.err"; then bad "manifesto alterado"; else
  if grep -qi 'SHA256SUMS\|manifesto\|entradas\|diverge\|LICENSE' "${TMP}/manbad.err"; then ok "manifesto alterado"; else bad "manifesto alterado (msg)"; fi
fi

# --- real repo ---
if "${PY}" "${VERIFY}" >/dev/null; then ok "verificador no repositório real"; else bad "verificador no repositório real"; fi

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
if [[ "${fail}" -ne 0 ]]; then
  exit 1
fi
