#!/usr/bin/env bash
# Smoke tests for compliance/scripts/verify_vendor_catalog.py
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PY="${ROOT}/compliance/scripts/.venv/bin/python"
if [[ ! -x "${PY}" ]]; then
  python3 -m venv "${ROOT}/compliance/scripts/.venv"
  "${ROOT}/compliance/scripts/.venv/bin/pip" install -q -r "${ROOT}/compliance/scripts/requirements.txt"
  PY="${ROOT}/compliance/scripts/.venv/bin/python"
fi

ok() { echo "ok $*"; }
bad() { echo "FAIL $*"; exit 1; }

if "${PY}" "${ROOT}/compliance/scripts/verify_vendor_catalog.py"; then
  ok "vendor catalog válido"
else
  bad "vendor catalog deveria passar"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT
mkdir -p "${TMP}/compliance/catalog/schema" "${TMP}/compliance/scripts"
cp "${ROOT}/compliance/catalog/schema/vendor-integrations.schema.json" "${TMP}/compliance/catalog/schema/"
cp "${ROOT}/compliance/scripts/verify_vendor_catalog.py" "${TMP}/compliance/scripts/"
# Patch REPO_ROOT resolution: script uses parents[2] from scripts/ → use same layout
mkdir -p "${TMP}/compliance/scripts"
# Run with broken normative flag
cat >"${TMP}/compliance/catalog/vendor-integrations.yaml" <<'EOF'
catalog_version: 1
collection_id: vendor-bad
retrieved_at: '2026-07-28'
description: bad
normative: true
sources:
- id: VENDOR-NETBO-API-V2
  official_name: x
  vendor: NET-BO
  authority: vendor_technical
  type: vendor_api_doc
  version: V2
  published_at: null
  retrieved_at: '2026-07-28'
  source_url: null
  url_status: not_applicable
  sha256: 09bb09f8064ad49d98bf9cffc0a80eb7a2dbded5f4221c54e5ca19da82cb7c86
  status: pending_validation
  license_redistribution: uncertain
  storage: private_sync
  local_path: consulta/docs/x.pdf
  private_repository: storesace-cv/bwb-fiscal-sources-ao
  private_commit: a889ef623c367e96cb7246ab42a274e54cbb2dc3
  private_repository_path: originals/vendor-integrations/VENDOR-NETBO-API-V2/original/NETBO_API_V2.pdf
  page_count: 52
  text_extractable: true
  affects: []
  notes: x
EOF

# Script hardcodes REPO_ROOT to parents[2]; run from TMP by copying tree shape:
# TMP/compliance/scripts/verify → parents[2]=TMP
if (cd "${TMP}" && "${PY}" compliance/scripts/verify_vendor_catalog.py) >/dev/null 2>&1; then
  bad "normative=true deveria falhar"
else
  ok "normative=true rejeitado"
fi

echo "vendor catalog tests passed"
