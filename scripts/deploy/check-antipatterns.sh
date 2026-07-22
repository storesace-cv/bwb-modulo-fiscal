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
  if grep -nE 'open\.candidate|tls\.open' "${f}" | grep -nE 'install-env|install-release|scp|nginx.*enable|sites-enabled'; then
    echo "error: updater/helper must not activate open Nginx candidate in ${f}" >&2
    failed=1
  fi
done

# Closed public TLS must keep deny-all on documents.
if ! grep -q 'deny all' "${ROOT}/deploy/nginx/bwb-fiscal-sandbox-tls.conf"; then
  echo "error: public TLS config must deny-all /v1/documents" >&2
  failed=1
fi
if ! grep -q 'listen 127.0.0.1:18080' "${ROOT}/deploy/nginx/measure/bwb-fiscal-sandbox-measure-loopback.conf"; then
  echo "error: measure listener must be loopback-only" >&2
  failed=1
fi
if grep -nE 'listen[[:space:]]+18080|listen[[:space:]]+\*:18080|listen[[:space:]]+0\.0\.0\.0:18080' \
  "${ROOT}/deploy/nginx/measure/bwb-fiscal-sandbox-measure-loopback.conf"; then
  echo "error: measure listener must not bind non-loopback" >&2
  failed=1
fi

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi
echo "antipatterns_ok"
