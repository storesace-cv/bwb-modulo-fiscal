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

write_roadmap() {
  local d="$1"
  local body="$2"
  cat >"${d}/ROADMAP.md" <<EOF
# ROADMAP

<a id="rm-gov-002-ruleset-de-main"></a>

## Governação

${body}
EOF
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

# --- HTML anchor valid (already in fixture) ---
if run_verify "${TMP}/valid" >/dev/null; then ok "âncora HTML válida"; else bad "âncora HTML válida"; fi

# --- duplicate ID ---
make_base "${TMP}/dup"
write_roadmap "${TMP}/dup" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [x] | RM-TEST-001 | B | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/dup" >/dev/null 2>"${TMP}/dup.err"; then bad "ID duplicado"; else
  grep -q 'duplicado' "${TMP}/dup.err" && ok "ID duplicado" || bad "ID duplicado (mensagem)"
fi

# --- CONCLUÍDO sem evidência ---
make_base "${TMP}/noev"
write_roadmap "${TMP}/noev" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | — | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/noev" >/dev/null 2>"${TMP}/noev.err"; then bad "concluído sem evidência"; else
  grep -qi 'evid' "${TMP}/noev.err" && ok "concluído sem evidência" || bad "concluído sem evidência (msg)"
fi

# --- check/estado incompatível ---
make_base "${TMP}/mismatch"
write_roadmap "${TMP}/mismatch" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | PENDENTE | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/mismatch" >/dev/null 2>"${TMP}/mismatch.err"; then bad "check/estado incompatível"; else
  grep -q 'CONCLUÍDO' "${TMP}/mismatch.err" && ok "check/estado incompatível" || bad "check/estado (msg)"
fi

# --- BLOQUEADO sem gate ---
make_base "${TMP}/nogate"
write_roadmap "${TMP}/nogate" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/nogate" >/dev/null 2>"${TMP}/nogate.err"; then bad "bloqueado sem gate"; else
  grep -qi 'gate' "${TMP}/nogate.err" && ok "bloqueado sem gate" || bad "bloqueado sem gate (msg)"
fi

# --- ADIADO sem marco ---
make_base "${TMP}/nomarco"
write_roadmap "${TMP}/nomarco" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | Futuro | ADIADO | [AGENTS.md](AGENTS.md) | depois | Done text |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/nomarco" >/dev/null 2>"${TMP}/nomarco.err"; then bad "adiado sem marco"; else
  grep -qi 'marco' "${TMP}/nomarco.err" && ok "adiado sem marco" || bad "adiado sem marco (msg)"
fi

# --- link relativo quebrado ---
make_base "${TMP}/broken"
write_roadmap "${TMP}/broken" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [missing.md](missing.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/broken" >/dev/null 2>"${TMP}/broken.err"; then bad "link relativo quebrado"; else
  grep -qi 'quebrado' "${TMP}/broken.err" && ok "link relativo quebrado" || bad "link relativo (msg)"
fi

# --- link interno inválido ---
make_base "${TMP}/badfrag"
write_roadmap "${TMP}/badfrag" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [#nao-existe](#nao-existe) | Autorização humana | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/badfrag" >/dev/null 2>"${TMP}/badfrag.err"; then bad "link interno inválido"; else
  grep -qi 'fragmento' "${TMP}/badfrag.err" && ok "link interno inválido" || bad "link interno inválido (msg)"
fi

# --- link interno válido (heading) ---
make_base "${TMP}/head"
cat >"${TMP}/head/ROADMAP.md" <<'EOF'
# ROADMAP

## Decisão de governação XYZ

<a id="rm-gov-002-ruleset-de-main"></a>

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [#decisão-de-governação-xyz](#decisão-de-governação-xyz) | Autorização humana | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |
EOF
if run_verify "${TMP}/head" >/dev/null; then ok "link interno válido (heading)"; else bad "link interno válido (heading)"; fi

# --- placeholders ---
make_base "${TMP}/ph"
write_roadmap "${TMP}/ph" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | links deste PR | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/ph" >/dev/null 2>"${TMP}/ph.err"; then bad "placeholder links deste PR"; else
  grep -qi 'placeholder' "${TMP}/ph.err" && ok "placeholder links deste PR" || bad "placeholder (msg)"
fi

make_base "${TMP}/ph2"
write_roadmap "${TMP}/ph2" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [ ] | RM-TEST-001 | A | BLOQUEADO | [decisão]([decisão pending]) | Autorização | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/ph2" >/dev/null 2>"${TMP}/ph2.err"; then bad "placeholder decisão pending"; else
  grep -qi 'placeholder\|pending\|inválid\|quebrado\|URL' "${TMP}/ph2.err" && ok "placeholder decisão pending" || bad "placeholder pending (msg)"
fi

# --- prosa local/ OK ---
make_base "${TMP}/localok"
write_roadmap "${TMP}/localok" 'A pasta `local/` não é dependência do repositório: não copiar `local/`.

| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [AGENTS.md](AGENTS.md) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/localok" >/dev/null; then ok "menção explicativa local/"; else bad "menção explicativa local/"; fi

# --- link local/ inválido ---
make_base "${TMP}/localbad"
write_roadmap "${TMP}/localbad" '| Check | ID | Entrega | Estado | Evidência | Dependências / gate | Done |
|---|---|---|---|---|---|---|
| [x] | RM-TEST-001 | A | CONCLUÍDO | [segredo](local/docs/x.pdf) | — | Done |
| [ ] | RM-GOV-002 | Ruleset | BLOQUEADO | [#rm-gov-002-ruleset-de-main](#rm-gov-002-ruleset-de-main) | Autorização | Ruleset |'
if run_verify "${TMP}/localbad" >/dev/null 2>"${TMP}/localbad.err"; then bad "link local/ inválido"; else
  grep -qi 'local/' "${TMP}/localbad.err" && ok "link local/ inválido" || bad "link local/ (msg)"
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
  grep -qi 'segundo\|canónic' "${TMP}/second.err" && ok "segundo roadmap" || bad "segundo roadmap (msg)"
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
  grep -qi 'apontador\|Fase\|ROADMAP' "${TMP}/badptr.err" && ok "apontador incorrecto" || bad "apontador incorrecto (msg)"
fi

# --- apontador correcto (já no make_base) ---
if run_verify "${TMP}/valid" >/dev/null; then ok "apontador correcto"; else bad "apontador correcto"; fi

# --- real repo ---
if python3 "${VERIFY}" --repo-root "${ROOT}" >/dev/null; then
  ok "verificador no repositório real"
else
  bad "verificador no repositório real"
  python3 "${VERIFY}" --repo-root "${ROOT}" || true
fi

printf '\n%d passed, %d failed\n' "${pass}" "${fail}"
if [[ "${fail}" -ne 0 ]]; then
  exit 1
fi
