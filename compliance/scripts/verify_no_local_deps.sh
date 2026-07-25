#!/usr/bin/env bash
# Fail if build/test/runtime paths reference local/ as a dependency.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

tmp="$(mktemp)"
trap 'rm -f "${tmp}"' EXIT

# Scan code/CI/deploy/tests for filesystem dependencies on local/...
# Match path-like uses (local/docs, "local/...", `local/...), not prose "no local".
if command -v rg >/dev/null 2>&1; then
  rg -n --glob '!local/**' --glob '!.git/**' --glob '!compliance/**' \
    -g '*.go' -g '*.sh' -g '*.yml' -g '*.yaml' -g 'Dockerfile*' -g 'Makefile' \
    -e '["'\''`]local/' \
    -e '(^|[[:space:]=])local/[A-Za-z0-9_.-]' \
    .github scripts tests cmd internal deploy >"${tmp}" 2>/dev/null || true
else
  grep -RInE '["'\''`]local/|(^|[[:space:]=])local/[A-Za-z0-9_.-]' \
    .github scripts tests cmd internal deploy \
    --include='*.go' --include='*.sh' --include='*.yml' --include='*.yaml' \
    >"${tmp}" 2>/dev/null || true
fi

bad=0
while IFS= read -r line || [[ -n "${line}" ]]; do
  [[ -z "${line}" ]] && continue
  echo "${line}" >&2
  bad=1
done <"${tmp}"

if [[ "${bad}" -ne 0 ]]; then
  echo "ERROR: forbidden local/ references in build/test/runtime paths" >&2
  exit 1
fi

zip_tmp="$(mktemp)"
trap 'rm -f "${tmp}" "${zip_tmp}"' EXIT
if command -v rg >/dev/null 2>&1; then
  rg -n --glob '!local/**' -g '*.go' \
    -e 'SAF-T-AO_repositorio_master\.zip|arquivo_fiscal_ao/02_saft_ao/.*\.zip' \
    cmd internal tests >"${zip_tmp}" 2>/dev/null || true
else
  grep -RInE 'SAF-T-AO_repositorio_master\.zip|arquivo_fiscal_ao/02_saft_ao/.*\.zip' \
    cmd internal tests --include='*.go' >"${zip_tmp}" 2>/dev/null || true
fi

if [[ -s "${zip_tmp}" ]]; then
  cat "${zip_tmp}" >&2
  echo "ERROR: ZIP archive referenced from Go code/tests" >&2
  exit 1
fi

echo "OK: no local/ build-test-runtime dependencies; no ZIP runtime refs"
