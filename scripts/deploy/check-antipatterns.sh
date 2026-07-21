#!/usr/bin/env bash
# Fail if forbidden SSH / ignore-error antipatterns appear in deploy scripts.
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

failed=0
scan_files=()
while IFS= read -r f; do
  scan_files+=("${f}")
done < <(find scripts/deploy -type f -name '*.sh' ! -name 'check-antipatterns.sh')

for f in "${scan_files[@]}"; do
  if grep -nE 'StrictHostKeyChecking[[:space:]]*=[[:space:]]*no' "${f}"; then
    echo "error: forbidden StrictHostKeyChecking disabled in ${f}" >&2
    failed=1
  fi
  if grep -nE 'UserKnownHostsFile[[:space:]]*=[[:space:]]*(/dev/null|NUL)' "${f}"; then
    echo "error: forbidden UserKnownHostsFile null sink in ${f}" >&2
    failed=1
  fi
  if grep -nE '\|\|[[:space:]]*true' "${f}"; then
    echo "error: '|| true' found in ${f}" >&2
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi
echo "antipatterns_ok"
