#!/usr/bin/env bash
# Build linux release: fiscal-api, fiscal-migrate, COMMIT, SHA256SUMS.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
source "${SCRIPT_DIR}/lib/allowlist.sh"

ROOT="$(deploy_repo_root)"
cd "${ROOT}"

GOARCH="${DEPLOY_GOARCH:-amd64}"
GOOS="${GOOS:-linux}"
OUT_DIR="${OUT_DIR:-}"
EXPECTED_COMMIT="${EXPECTED_COMMIT:-}"

deploy_require_cmds go git

HEAD="$(git rev-parse HEAD)"
if [[ -n "${EXPECTED_COMMIT}" && "${HEAD}" != "${EXPECTED_COMMIT}" ]]; then
  echo "error: HEAD does not match EXPECTED_COMMIT" >&2
  echo "error: HEAD=${HEAD}" >&2
  echo "error: EXPECTED_COMMIT=${EXPECTED_COMMIT}" >&2
  exit 1
fi

if [[ -z "${OUT_DIR}" ]]; then
  OUT_DIR="${ROOT}/dist/releases/${HEAD}"
fi

mkdir -p "${OUT_DIR}"
rm -f "${OUT_DIR}/fiscal-api" "${OUT_DIR}/fiscal-migrate" "${OUT_DIR}/COMMIT" "${OUT_DIR}/SHA256SUMS"

echo "building GOOS=${GOOS} GOARCH=${GOARCH} commit=${HEAD}"
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/fiscal-api" ./cmd/fiscal-api
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/fiscal-migrate" ./cmd/fiscal-migrate

printf '%s\n' "${HEAD}" >"${OUT_DIR}/COMMIT"

(
  cd "${OUT_DIR}"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum fiscal-api fiscal-migrate >SHA256SUMS
    sha256sum -c SHA256SUMS >/dev/null
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 fiscal-api fiscal-migrate | awk '{print $1"  "$2}' >SHA256SUMS
    shasum -a 256 -c SHA256SUMS >/dev/null
  else
    echo "error: sha256sum or shasum required" >&2
    exit 1
  fi
)

GOT_COMMIT="$(tr -d '[:space:]' <"${OUT_DIR}/COMMIT")"
if [[ "${GOT_COMMIT}" != "${HEAD}" ]]; then
  echo "error: COMMIT file mismatch" >&2
  exit 1
fi

echo "release_ok dir=${OUT_DIR} commit=${HEAD} arch=${GOARCH}"
