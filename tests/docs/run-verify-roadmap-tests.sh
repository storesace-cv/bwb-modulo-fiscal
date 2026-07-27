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

**Estado revisto em:** 2020-01-01

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
  # Fixtures sem histórico Git: validar estrutura; frescura coberta em testes dedicados.
  python3 "${d}/scripts/verify_roadmap.py" --repo-root "${d}" --roadmap "${d}/ROADMAP.md" \
    --skip-reviewed-freshness
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

# --- remove RM-GOV-002 / ruleset protection facts (mantém ID canónico) ---
mkdir -p "${TMP}/nogov" && cp "${ROOT}/ROADMAP.md" "${TMP}/nogov/ROADMAP.md"
python3 - <<'PY' "${TMP}/nogov/ROADMAP.md"
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = re.sub(
    r"#### RM-GOV-002 — ruleset de main\n\n.*?(?=\n### |\n## |\Z)",
    "#### RM-GOV-002 — ruleset de main\n\nREDACTED_RULESET_SECTION.\n\n",
    text,
    count=1,
    flags=re.S,
)
text = text.replace(
    "Ruleset activo em main com PR e checks obrigatórios",
    "Governação documental de main",
)
text = text.replace(
    "Ruleset activo em main; PR obrigatório; checks obrigatórios; sem bypass",
    "Documentação actualizada",
)
text = text.replace("Protect main and require project checks", "REDACTED_NAME")
text = text.replace("ruleset activo", "protecao activa")
text = text.replace("Ruleset activo", "Protecao activa")
# Frase legada sem ruleset activo — inválida para CONCLUÍDO.
text = text.replace(
    "`FISCAL_ENV=homologation`",
    "RM-GOV-002: a política assistida não é tecnicamente impossível de contornar.\n\n`FISCAL_ENV=homologation`",
    1,
)
p.write_text(text, encoding="utf-8")
PY
if run_mut "${TMP}/nogov/ROADMAP.md" >/dev/null 2>"${TMP}/nogov.err"; then bad "RM-GOV-002/ruleset"; else
  if grep -qi 'RM-GOV-002\|ruleset ativo\|legado\|bypass vazio\|current_user_can_bypass=never' "${TMP}/nogov.err"; then ok "RM-GOV-002/ruleset"; else bad "RM-GOV-002/ruleset (msg)"; fi
fi

# --- RM-GOV-002 CONCLUÍDO + bypass vazio + never → PASS ---
make_base "${TMP}/govok"
write_roadmap "${TMP}/govok" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-GOV-002 | Ruleset activo em main | CONCLUÍDO | [AGENTS.md](AGENTS.md) · [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | — | Ruleset activo; sem bypass |

#### RM-GOV-002 — ruleset de main
- Ruleset activo: Protect main and require project checks
- Bypass actors: vazio
- current_user_can_bypass=never
'
if run_verify "${TMP}/govok" >/dev/null; then ok "RM-GOV-002 CONCLUÍDO bypass vazio"; else bad "RM-GOV-002 CONCLUÍDO bypass vazio"; fi

# --- CONCLUÍDO: vazio→admin e never→always → FAIL ---
make_base "${TMP}/govadmin"
cp "${TMP}/govok/ROADMAP.md" "${TMP}/govadmin/ROADMAP.md"
python3 - <<'PY' "${TMP}/govadmin/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("Bypass actors: vazio", "Bypass actors: admin")
text = text.replace("current_user_can_bypass=never", "current_user_can_bypass=always")
text = text.replace("sem bypass", "bypass configurado")
p.write_text(text, encoding="utf-8")
PY
if run_verify "${TMP}/govadmin" >/dev/null 2>"${TMP}/govadmin.err"; then bad "RM-GOV-002 bypass admin"; else
  if grep -qi 'bypass vazio\|current_user_can_bypass=never' "${TMP}/govadmin.err"; then ok "RM-GOV-002 bypass admin"; else bad "RM-GOV-002 bypass admin (msg)"; fi
fi

# --- CONCLUÍDO: apenas a palavra bypass → FAIL ---
make_base "${TMP}/govword"
cp "${TMP}/govok/ROADMAP.md" "${TMP}/govword/ROADMAP.md"
python3 - <<'PY' "${TMP}/govword/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("Bypass actors: vazio", "bypass")
text = text.replace("current_user_can_bypass=never", "")
text = text.replace("sem bypass", "bypass")
p.write_text(text, encoding="utf-8")
PY
if run_verify "${TMP}/govword" >/dev/null 2>"${TMP}/govword.err"; then bad "RM-GOV-002 só bypass"; else
  if grep -qi 'bypass vazio\|current_user_can_bypass=never' "${TMP}/govword.err"; then ok "RM-GOV-002 só bypass"; else bad "RM-GOV-002 só bypass (msg)"; fi
fi

# --- CONCLUÍDO: remover never e equivalentes → FAIL ---
make_base "${TMP}/govnonever"
cp "${TMP}/govok/ROADMAP.md" "${TMP}/govnonever/ROADMAP.md"
python3 - <<'PY' "${TMP}/govnonever/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("current_user_can_bypass=never", "")
p.write_text(text, encoding="utf-8")
PY
if run_verify "${TMP}/govnonever" >/dev/null 2>"${TMP}/govnonever.err"; then bad "RM-GOV-002 sem never"; else
  if grep -qi 'current_user_can_bypass=never' "${TMP}/govnonever.err"; then ok "RM-GOV-002 sem never"; else bad "RM-GOV-002 sem never (msg)"; fi
fi

# --- CONCLUÍDO: remover ruleset activo e manter/adicionar legado → FAIL ---
make_base "${TMP}/govlegacy"
cp "${TMP}/govok/ROADMAP.md" "${TMP}/govlegacy/ROADMAP.md"
python3 - <<'PY' "${TMP}/govlegacy/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("Ruleset activo em main", "Governação documental")
text = text.replace("Ruleset activo: Protect main and require project checks", "Sem ruleset activo")
text = text.replace("Ruleset activo; sem bypass", "sem bypass")
text = text.replace("Protect main and require project checks", "")
# Ensure legacy phrase remains (skeleton already has it); reinforce.
if "tecnicamente impossível de contornar" not in text and "tecnicamente impossivel de contornar" not in text.lower():
    text = text.replace(
        "RM-GOV-002: enquanto não houver ruleset, a política assistida não é tecnicamente impossível de contornar.",
        "RM-GOV-002: enquanto não houver ruleset, a política assistida não é tecnicamente impossível de contornar.",
    )
p.write_text(text, encoding="utf-8")
PY
if run_verify "${TMP}/govlegacy" >/dev/null 2>"${TMP}/govlegacy.err"; then bad "RM-GOV-002 legado em CONCLUÍDO"; else
  if grep -qi 'legado\|ruleset ativo' "${TMP}/govlegacy.err"; then ok "RM-GOV-002 legado em CONCLUÍDO"; else bad "RM-GOV-002 legado em CONCLUÍDO (msg)"; fi
fi

# --- CONCLUÍDO: Bypass actors: @admins + «sem bypass» noutro sítio → FAIL ---
make_base "${TMP}/govat"
cp "${TMP}/govok/ROADMAP.md" "${TMP}/govat/ROADMAP.md"
python3 - <<'PY' "${TMP}/govat/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("Bypass actors: vazio", "Bypass actors: @admins")
p.write_text(text, encoding="utf-8")
PY
if run_verify "${TMP}/govat" >/dev/null 2>"${TMP}/govat.err"; then bad "RM-GOV-002 bypass @admins"; else
  if grep -qi 'bypass vazio' "${TMP}/govat.err"; then ok "RM-GOV-002 bypass @admins"; else bad "RM-GOV-002 bypass @admins (msg)"; fi
fi

# --- CONCLUÍDO: só o nome do ruleset, sem activo/enforcement → FAIL ---
make_base "${TMP}/govname"
cp "${TMP}/govok/ROADMAP.md" "${TMP}/govname/ROADMAP.md"
python3 - <<'PY' "${TMP}/govname/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("Ruleset activo em main", "Governação de main")
text = text.replace("Ruleset activo: Protect main and require project checks", "Nome: Protect main and require project checks")
text = text.replace("Ruleset activo; sem bypass", "sem bypass")
p.write_text(text, encoding="utf-8")
PY
if run_verify "${TMP}/govname" >/dev/null 2>"${TMP}/govname.err"; then bad "RM-GOV-002 só nome ruleset"; else
  if grep -qi 'legado\|ruleset ativo' "${TMP}/govname.err"; then ok "RM-GOV-002 só nome ruleset"; else bad "RM-GOV-002 só nome ruleset (msg)"; fi
fi

# --- BLOQUEADO com frase legada e gate válido → PASS (fixture histórica) ---
if run_verify "${TMP}/valid" >/dev/null; then ok "RM-GOV-002 BLOQUEADO legado"; else bad "RM-GOV-002 BLOQUEADO legado"; fi

# --- real repo ---
if python3 "${VERIFY}" --repo-root "${ROOT}" >/dev/null; then
  ok "verificador no repositório real"
else
  bad "verificador no repositório real"
  python3 "${VERIFY}" --repo-root "${ROOT}" || true
fi

# --- Estado revisto em obrigatório ---
make_base "${TMP}/norev"
write_roadmap "${TMP}/norev" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | Item válido | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done text |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização humana settings GitHub | Ruleset activo |'
# remove reviewed date line from fixture
python3 - <<'PY' "${TMP}/norev/ROADMAP.md"
import pathlib, re, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = re.sub(r"^\*\*Estado revisto em:\*\*.*\n", "", text, count=1, flags=re.M)
p.write_text(text, encoding="utf-8")
PY
if run_verify "${TMP}/norev" >/dev/null 2>"${TMP}/norev.err"; then bad "Estado revisto ausente"; else
  if grep -qi 'Estado revisto' "${TMP}/norev.err"; then ok "Estado revisto ausente"; else bad "Estado revisto ausente (msg)"; fi
fi

# --- alteração material exige data fresca (Git local) ---
REV_REPO="${TMP}/revgit"
rm -rf "${REV_REPO}"
mkdir -p "${REV_REPO}"
make_base "${REV_REPO}"
write_roadmap "${REV_REPO}" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | Item válido | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done text |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização humana settings GitHub | Ruleset activo |'
(
  cd "${REV_REPO}"
  git init -q
  git config user.email "test@example.com"
  git config user.name "Test"
  git add .
  git commit -q -m "base"
  git branch -M main
)
# material change + stale date
python3 - <<'PY' "${REV_REPO}/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("**Estado revisto em:** 2020-01-01", "**Estado revisto em:** 2020-01-01")
text = text.replace("Item válido", "Item válido alterado")
p.write_text(text, encoding="utf-8")
PY
if python3 "${REV_REPO}/scripts/verify_roadmap.py" --repo-root "${REV_REPO}" --base-ref main \
  --today 2026-07-26 >/dev/null 2>"${TMP}/revstale.err"; then
  bad "data desactualizada com alteração material"
else
  if grep -qi 'alteração material' "${TMP}/revstale.err"; then ok "data desactualizada com alteração material"; else bad "data desactualizada (msg)"; fi
fi
# same material change with fresh date → pass
python3 - <<'PY' "${REV_REPO}/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("**Estado revisto em:** 2020-01-01", "**Estado revisto em:** 2026-07-26")
p.write_text(text, encoding="utf-8")
PY
if python3 "${REV_REPO}/scripts/verify_roadmap.py" --repo-root "${REV_REPO}" --base-ref main \
  --today 2026-07-26 >/dev/null; then
  ok "data fresca com alteração material"
else
  bad "data fresca com alteração material"
  python3 "${REV_REPO}/scripts/verify_roadmap.py" --repo-root "${REV_REPO}" --base-ref main --today 2026-07-26 || true
fi
# only date bump (no other body change) must not require matching --today against stale... wait:
# only date change: strip_reviewed ignores date → no material → pass with old... we're changing TO fresh.
# Test: only bump date from 2020 to 2026 without other changes — not material, pass even if --today differs
(
  cd "${REV_REPO}"
  git add ROADMAP.md
  git commit -q -m "fresh"
)
python3 - <<'PY' "${REV_REPO}/ROADMAP.md"
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
text = text.replace("**Estado revisto em:** 2026-07-26", "**Estado revisto em:** 2026-07-27")
p.write_text(text, encoding="utf-8")
PY
if python3 "${REV_REPO}/scripts/verify_roadmap.py" --repo-root "${REV_REPO}" --base-ref main \
  --today 2099-01-01 >/dev/null; then
  ok "só data sem alteração material"
else
  bad "só data sem alteração material"
fi

# --- 99 items analyzed ---
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
if [[ "${COUNT}" == "124" ]]; then ok "124 itens RM-*"; else bad "124 itens RM-* (got ${COUNT})"; fi

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
if [[ "${fail}" -ne 0 ]]; then
  exit 1
fi
