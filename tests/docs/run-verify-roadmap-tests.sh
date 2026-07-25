#!/usr/bin/env bash
# Structural regression tests for scripts/verify_roadmap.py (no network, stdlib only).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
VERIFY="${ROOT}/scripts/verify_roadmap.py"
TMP="$(mktemp -d "${TMPDIR:-/tmp}/roadmap-verify.XXXXXX")"
trap 'rm -rf "${TMP}"' EXIT

pass=0
fail=0

ok() {
  pass=$((pass + 1))
  printf 'ok %s\n' "$1"
}

bad() {
  fail=$((fail + 1))
  printf 'FAIL %s\n' "$1" >&2
}

# Minimal valid repo fixture
make_base() {
  local d="$1"
  mkdir -p "${d}/docs/06-delivery" "${d}/scripts" "${d}/.github/workflows" "${d}/.cursor/rules" "${d}/tests/docs"
  cp "${VERIFY}" "${d}/scripts/verify_roadmap.py"
  cat >"${d}/docs/06-delivery/implementation-roadmap.md" <<'EOF'
# Roadmap de implementação (apontador)

O roadmap canónico está em [ROADMAP.md](../../ROADMAP.md).
EOF
  cat >"${d}/AGENTS.md" <<'EOF'
# Agents

O [ROADMAP.md](ROADMAP.md) é o roadmap canónico de estado.
EOF
  cat >"${d}/.github/pull_request_template.md" <<'EOF'
## Impacto documental
- [x] documentação atualizada
EOF
  cat >"${d}/.cursor/rules/roadmap-maintenance.mdc" <<'EOF'
---
description: roadmap
alwaysApply: true
---
Actualizar ROADMAP.md
EOF
  cat >"${d}/tests/docs/run-verify-roadmap-tests.sh" <<'EOF'
#!/usr/bin/env bash
echo ok
EOF
  cat >"${d}/.github/workflows/ci.yml" <<'EOF'
name: ci
on: [push]
jobs:
  go:
    runs-on: ubuntu-latest
    steps:
      - run: true
EOF
}

# Skeleton with 18 H2 sections + essential distinctions (fixtures).
skeleton_head() {
  cat <<'EOF'
# ROADMAP

`FISCAL_ENV=homologation` é designação técnica do ambiente sandbox BWB e não é homologação oficial AGT.
SealInTx / `sealed_locally` não constituem emissão fiscal certificada pela AGT.
RM-GOV-002: enquanto não houver ruleset, a política assistida não é tecnicamente impossível de contornar.

<a id="rm-gov-002-ruleset-de-main"></a>

## 1. Visão executiva
## 2. Estado atual
## 3. O que já foi construído
## 4. Caminho crítico para Angola
## 5. Roadmap detalhado por área
## 6. Fontes fiscais e SAF-T AO
## 7. Motor fiscal Angola
## 8. Faturação electrónica AGT
## 9. Backoffice
## 10. Edge/offline
## 11. Integradores e software houses
## 12. Operações de produção
## 13. Certificação AGT
## 14. Cabo Verde
## 15. Bloqueios, decisões e incidentes
## 16. Critérios de conclusão
## 17. Evidências e documentos relacionados
## 18. Regras de manutenção do roadmap

EOF
}

write_roadmap() {
  local d="$1"
  local body="$2"
  {
    skeleton_head
    printf '%s\n' "${body}"
  } >"${d}/ROADMAP.md"
}

run_verify() {
  local d="$1"
  python3 "${d}/scripts/verify_roadmap.py" --repo-root "${d}" --roadmap "${d}/ROADMAP.md"
}

# --- valid item ---
make_base "${TMP}/valid"
write_roadmap "${TMP}/valid" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | Item válido | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done text |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização humana settings GitHub | Ruleset activo |'
if run_verify "${TMP}/valid" >/dev/null; then ok "item válido"; else bad "item válido"; fi

# --- HTML anchor valid ---
if run_verify "${TMP}/valid" >/dev/null; then ok "âncora HTML válida"; else bad "âncora HTML válida"; fi

# --- duplicate ID ---
make_base "${TMP}/dup"
write_roadmap "${TMP}/dup" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [x] | RM-TEST-001 | B | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/dup" >/dev/null 2>"${TMP}/dup.err"; then bad "ID duplicado"; else
  if grep -q 'duplicado' "${TMP}/dup.err"; then ok "ID duplicado"; else bad "ID duplicado (mensagem)"; fi
fi

# --- CONCLUÍDO sem evidência ---
make_base "${TMP}/noev"
write_roadmap "${TMP}/noev" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | — | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/noev" >/dev/null 2>"${TMP}/noev.err"; then bad "concluído sem evidência"; else
  if grep -qi 'evid' "${TMP}/noev.err"; then ok "concluído sem evidência"; else bad "concluído sem evidência (msg)"; fi
fi

# --- check/estado incompatível ---
make_base "${TMP}/mismatch"
write_roadmap "${TMP}/mismatch" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | PENDENTE | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/mismatch" >/dev/null 2>"${TMP}/mismatch.err"; then bad "check/estado incompatível"; else
  if grep -q 'CONCLUÍDO' "${TMP}/mismatch.err"; then ok "check/estado incompatível"; else bad "check/estado (msg)"; fi
fi

# --- BLOQUEADO sem gate ---
make_base "${TMP}/nogate"
write_roadmap "${TMP}/nogate" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/nogate" >/dev/null 2>"${TMP}/nogate.err"; then bad "bloqueado sem gate"; else
  if grep -qi 'gate' "${TMP}/nogate.err"; then ok "bloqueado sem gate"; else bad "bloqueado sem gate (msg)"; fi
fi

# --- ADIADO sem marco ---
make_base "${TMP}/nomarco"
write_roadmap "${TMP}/nomarco" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | Futuro | ADIADO | [AGENTS.md](AGENTS.md) | depois | Done text |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/nomarco" >/dev/null 2>"${TMP}/nomarco.err"; then bad "adiado sem marco"; else
  if grep -qi 'marco' "${TMP}/nomarco.err"; then ok "adiado sem marco"; else bad "adiado sem marco (msg)"; fi
fi

# --- link relativo quebrado ---
make_base "${TMP}/broken"
write_roadmap "${TMP}/broken" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [missing.md](missing.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/broken" >/dev/null 2>"${TMP}/broken.err"; then bad "link relativo quebrado"; else
  if grep -qi 'quebrado' "${TMP}/broken.err"; then ok "link relativo quebrado"; else bad "link relativo (msg)"; fi
fi

# --- link interno inválido ---
make_base "${TMP}/badfrag"
write_roadmap "${TMP}/badfrag" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [#nao-existe](#nao-existe) | Autorização humana | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/badfrag" >/dev/null 2>"${TMP}/badfrag.err"; then bad "link interno inválido"; else
  if grep -qi 'fragmento' "${TMP}/badfrag.err"; then ok "link interno inválido"; else bad "link interno inválido (msg)"; fi
fi

# --- link interno válido (heading) ---
make_base "${TMP}/head"
{
  skeleton_head
  cat <<'EOF'
## Decisão de governação XYZ

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [#decisão-de-governação-xyz](#decisão-de-governação-xyz) | Autorização humana | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |
EOF
} >"${TMP}/head/ROADMAP.md"
# Extra H2 breaks the required 18-section order — put decision heading as H3 instead.
{
  skeleton_head
  cat <<'EOF'
### Decisão de governação XYZ

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [#decisão-de-governação-xyz](#decisão-de-governação-xyz) | Autorização humana | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |
EOF
} >"${TMP}/head/ROADMAP.md"
if run_verify "${TMP}/head" >/dev/null; then ok "link interno válido (heading)"; else bad "link interno válido (heading)"; fi

# --- placeholders ---
make_base "${TMP}/ph"
write_roadmap "${TMP}/ph" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | links deste PR | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/ph" >/dev/null 2>"${TMP}/ph.err"; then bad "placeholder links deste PR"; else
  if grep -qi 'placeholder' "${TMP}/ph.err"; then ok "placeholder links deste PR"; else bad "placeholder (msg)"; fi
fi

make_base "${TMP}/ph2"
write_roadmap "${TMP}/ph2" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [decisão]([decisão pending]) | Autorização | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/ph2" >/dev/null 2>"${TMP}/ph2.err"; then bad "placeholder decisão pending"; else
  if grep -qi 'placeholder\|pending\|inválid\|quebrado\|URL\|https' "${TMP}/ph2.err"; then ok "placeholder decisão pending"; else bad "placeholder pending (msg)"; fi
fi

# --- prosa consult-dir OK ---
_CONSULT_DIR="loc"$'\x61'"l"
make_base "${TMP}/localok"
write_roadmap "${TMP}/localok" "A pasta \`${_CONSULT_DIR}/\` não é dependência do repositório: não copiar \`${_CONSULT_DIR}/\`.

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |"
if run_verify "${TMP}/localok" >/dev/null; then ok "menção explicativa consult-dir"; else bad "menção explicativa consult-dir"; fi

# --- link consult-dir inválido ---
make_base "${TMP}/localbad"
write_roadmap "${TMP}/localbad" "| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [segredo](${_CONSULT_DIR}/docs/x.pdf) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |"
if run_verify "${TMP}/localbad" >/dev/null 2>"${TMP}/localbad.err"; then bad "link consult-dir inválido"; else
  if grep -qi "${_CONSULT_DIR}/" "${TMP}/localbad.err"; then ok "link consult-dir inválido"; else bad "link consult-dir (msg)"; fi
fi

# --- AGENTS menciona roadmap canónico ---
make_base "${TMP}/agents"
write_roadmap "${TMP}/agents" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/agents" >/dev/null; then ok "AGENTS roadmap canónico"; else bad "AGENTS roadmap canónico"; fi

# --- segundo roadmap canónico ---
make_base "${TMP}/second"
write_roadmap "${TMP}/second" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
cat >"${TMP}/second/docs/other-roadmap.md" <<'EOF'
# Outro

Este documento é a fonte canónica de estado do roadmap canónico do projecto.
EOF
if run_verify "${TMP}/second" >/dev/null 2>"${TMP}/second.err"; then bad "segundo roadmap"; else
  if grep -qi 'segundo\|canónic' "${TMP}/second.err"; then ok "segundo roadmap"; else bad "segundo roadmap (msg)"; fi
fi

# --- apontador incorrecto ---
make_base "${TMP}/badptr"
write_roadmap "${TMP}/badptr" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
cat >"${TMP}/badptr/docs/06-delivery/implementation-roadmap.md" <<'EOF'
# Roadmap antigo

## Fase 0 — Descoberta

Ainda o roadmap detalhado.
EOF
if run_verify "${TMP}/badptr" >/dev/null 2>"${TMP}/badptr.err"; then bad "apontador incorrecto"; else
  if grep -qi 'apontador\|Fase\|ROADMAP' "${TMP}/badptr.err"; then ok "apontador incorrecto"; else bad "apontador incorrecto (msg)"; fi
fi

# --- apontador correcto ---
if run_verify "${TMP}/valid" >/dev/null; then ok "apontador correcto"; else bad "apontador correcto"; fi

# --- HTTPS ok ---
make_base "${TMP}/httpsok"
write_roadmap "${TMP}/httpsok" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [PR](https://github.com/storesace-cv/bwb-modulo-fiscal/pull/28) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/httpsok" >/dev/null; then ok "https URL"; else bad "https URL"; fi

# --- http fail ---
make_base "${TMP}/httpbad"
write_roadmap "${TMP}/httpbad" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [x](http://example.com/evidencia) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/httpbad" >/dev/null 2>"${TMP}/httpbad.err"; then bad "http URL"; else
  if grep -qi 'https' "${TMP}/httpbad.err"; then ok "http URL"; else bad "http URL (msg)"; fi
fi

# --- ftp fail ---
make_base "${TMP}/ftpbad"
write_roadmap "${TMP}/ftpbad" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [x](ftp://files.example.com/a) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/ftpbad" >/dev/null 2>"${TMP}/ftpbad.err"; then bad "ftp URL"; else
  if grep -qi 'https' "${TMP}/ftpbad.err"; then ok "ftp URL"; else bad "ftp URL (msg)"; fi
fi

# --- six cells ---
make_base "${TMP}/six"
write_roadmap "${TMP}/six" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/six" >/dev/null 2>"${TMP}/six.err"; then bad "seis células"; else
  if grep -qi 'malformada' "${TMP}/six.err"; then ok "seis células"; else bad "seis células (msg)"; fi
fi

# --- eight cells ---
make_base "${TMP}/eight"
write_roadmap "${TMP}/eight" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done | EXTRA |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/eight" >/dev/null 2>"${TMP}/eight.err"; then bad "oito células"; else
  if grep -qi 'malformada' "${TMP}/eight.err"; then ok "oito células"; else bad "oito células (msg)"; fi
fi

# --- wrong header with RM-* ---
make_base "${TMP}/badhdr"
write_roadmap "${TMP}/badhdr" '| Check | RM-BAD-HDR | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |'
if run_verify "${TMP}/badhdr" >/dev/null 2>"${TMP}/badhdr.err"; then bad "header incorrecto RM-*"; else
  if grep -qi 'malformada' "${TMP}/badhdr.err"; then ok "header incorrecto RM-*"; else bad "header incorrecto (msg)"; fi
fi

# --- narrative RM-* outside table ---
make_base "${TMP}/narr"
write_roadmap "${TMP}/narr" 'Ver também o item RM-GOV-002 na governação (prosa, fora de tabela).

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/narr" >/dev/null; then ok "prosa RM-* fora de tabela"; else bad "prosa RM-* fora de tabela"; fi

# Mutation tests use --repo-root ROOT so relative evidence links resolve.
# Mutated files must still be named ROADMAP.md (canonical filename check).
run_mut() {
  python3 "${VERIFY}" --repo-root "${ROOT}" --roadmap "$1"
}

# --- remove one section ---
mkdir -p "${TMP}/nosec" && cp "${ROOT}/ROADMAP.md" "${TMP}/nosec/ROADMAP.md"
python3 - <<'PY' "${TMP}/nosec/ROADMAP.md"
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text2 = re.sub(r"^## 9\. Backoffice\n(?:.*\n)*?(?=^## 10\. )", "", text, count=1, flags=re.M)
p.write_text(text2, encoding="utf-8")
PY
if run_mut "${TMP}/nosec/ROADMAP.md" >/dev/null 2>"${TMP}/nosec.err"; then bad "secção removida"; else
  if grep -qi 'secção\|seccao\|Backoffice\|em falta\|ordem' "${TMP}/nosec.err"; then ok "secção removida"; else bad "secção removida (msg)"; fi
fi

# --- swap section order ---
mkdir -p "${TMP}/swap" && cp "${ROOT}/ROADMAP.md" "${TMP}/swap/ROADMAP.md"
python3 - <<'PY' "${TMP}/swap/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("## 8. Faturação electrónica AGT", "## 8. TEMP_FE", 1)
text = text.replace("## 9. Backoffice", "## 8. Faturação electrónica AGT", 1)
text = text.replace("## 8. TEMP_FE", "## 9. Backoffice", 1)
p.write_text(text, encoding="utf-8")
PY
if run_mut "${TMP}/swap/ROADMAP.md" >/dev/null 2>"${TMP}/swap.err"; then bad "ordem secções"; else
  if grep -qi 'ordem\|secção\|seccao' "${TMP}/swap.err"; then ok "ordem secções"; else bad "ordem secções (msg)"; fi
fi

# --- remove homologation distinction ---
mkdir -p "${TMP}/nohom" && cp "${ROOT}/ROADMAP.md" "${TMP}/nohom/ROADMAP.md"
python3 - <<'PY' "${TMP}/nohom/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
for s in (
    "FISCAL_ENV=homologation",
    "sandbox BWB",
    "homologação oficial AGT",
    "homologacao oficial AGT",
):
    text = text.replace(s, "REDACTED")
p.write_text(text, encoding="utf-8")
PY
if run_mut "${TMP}/nohom/ROADMAP.md" >/dev/null 2>"${TMP}/nohom.err"; then bad "distinção homologation"; else
  if grep -qi 'distinção\|homolog' "${TMP}/nohom.err"; then ok "distinção homologation"; else bad "distinção homologation (msg)"; fi
fi

# --- remove SealInTx distinction ---
mkdir -p "${TMP}/noseal" && cp "${ROOT}/ROADMAP.md" "${TMP}/noseal/ROADMAP.md"
python3 - <<'PY' "${TMP}/noseal/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
for s in ("SealInTx", "sealed_locally", "emissão fiscal certificada", "emissao fiscal certificada"):
    text = text.replace(s, "REDACTED")
p.write_text(text, encoding="utf-8")
PY
if run_mut "${TMP}/noseal/ROADMAP.md" >/dev/null 2>"${TMP}/noseal.err"; then bad "distinção SealInTx"; else
  if grep -qi 'distinção\|Seal\|sealed\|emiss' "${TMP}/noseal.err"; then ok "distinção SealInTx"; else bad "distinção SealInTx (msg)"; fi
fi

# --- remove RM-GOV-002 / ruleset limitation ---
mkdir -p "${TMP}/nogov" && cp "${ROOT}/ROADMAP.md" "${TMP}/nogov/ROADMAP.md"
python3 - <<'PY' "${TMP}/nogov/ROADMAP.md"
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = re.sub(
    r"Enquanto RM-GOV-002 não estiver concluído,.*?contornar\.",
    "REDACTED_POLICY.",
    text,
    count=1,
    flags=re.S,
)
text = text.replace("RM-GOV-002", "RM-GOVX002")
text = text.replace("rm-gov-002-ruleset-de-main", "rm-govx002-anchor")
text = text.replace("ruleset", "protecao-ramo")
p.write_text(text, encoding="utf-8")
PY
if run_mut "${TMP}/nogov/ROADMAP.md" >/dev/null 2>"${TMP}/nogov.err"; then bad "RM-GOV-002/ruleset"; else
  if grep -qi 'distinção\|RM-GOV\|ruleset\|contorn\|âncora\|ancora\|ID' "${TMP}/nogov.err"; then ok "RM-GOV-002/ruleset"; else bad "RM-GOV-002/ruleset (msg)"; fi
fi

# --- real repo ---
if python3 "${VERIFY}" --repo-root "${ROOT}" >/dev/null; then
  ok "verificador no repositório real"
else
  bad "verificador no repositório real"
  python3 "${VERIFY}" --repo-root "${ROOT}" || true
fi

# --- 97 items analyzed ---
COUNT="$(python3 - <<'PY' "${ROOT}/ROADMAP.md"
import pathlib, re, sys
text = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8")
n = 0
for line in text.splitlines():
    if not line.strip().startswith("|"):
        continue
    parts = [p.strip() for p in line.strip().strip("|").split("|")]
    if len(parts) == 7 and re.fullmatch(r"RM-[A-Z0-9]+(?:-[A-Z0-9]+)*", parts[1]):
        n += 1
print(n)
PY
)"
if [[ "${COUNT}" == "97" ]]; then ok "97 itens RM-*"; else bad "97 itens RM-* (got ${COUNT})"; fi

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
if [[ "${fail}" -ne 0 ]]; then
  exit 1
fi
