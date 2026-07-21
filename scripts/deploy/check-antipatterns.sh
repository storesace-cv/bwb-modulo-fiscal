#!/usr/bin/env bash
# Fail if forbidden SSH / privileged-shell / unsafe env antipatterns appear.
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
  if grep -nE 'source[[:space:]].*migrate\.env' "${f}"; then
    echo "error: sourcing migrate.env is forbidden in ${f}" >&2
    failed=1
  fi
  if grep -nE '^[^#]*\beval\b' "${f}"; then
    echo "error: eval is forbidden in ${f}" >&2
    failed=1
  fi
  if grep -nE 'current/fiscal-migrate' "${f}"; then
    echo "error: migrate must use new release binary, not current/, in ${f}" >&2
    failed=1
  fi
  if grep -nE 'deploy_require_cmds[[:space:]]+.*\b(sudo|systemctl)\b' "${f}"; then
    echo "error: sudo/systemctl must not be required on the operator host in ${f}" >&2
    failed=1
  fi
  if grep -nE '^[^#]*sudo[[:space:]]+-n[[:space:]]+bash|^[^#]*sudo[[:space:]]+bash|^[^#]*sudo[[:space:]]+-n[[:space:]]+sh|^[^#]*sudo[[:space:]]+sh\b' "${f}"; then
    echo "error: privileged shell via sudo is forbidden in ${f}" >&2
    failed=1
  fi
  if grep -nE '^[^#]*sudo[[:space:]]+-n[[:space:]]+(rm|cp|install|systemctl|mv|chmod|chown)\b' "${f}"; then
    echo "error: generic privileged command via sudo is forbidden in ${f} (use deploy helper)" >&2
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi
echo "antipatterns_ok"
