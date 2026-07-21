#!/usr/bin/env bash
# Build linux release: fiscal-api, fiscal-migrate, COMMIT, SHA256SUMS.
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/allowlist.sh
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/allowlist.sh"

ROOT="$(deploy_repo_root)"
cd "${ROOT}"

GOARCH="${DEPLOY_GOARCH:-}"
EXPECTED_COMMIT="${EXPECTED_COMMIT:-}"
OUT_DIR="${OUT_DIR:-}"
# Force Linux targets for staging artefacts.
GOOS=linux

deploy_require_cmds go git

case "${GOARCH}" in
  amd64 | arm64) ;;
  *)
    echo "error: DEPLOY_GOARCH must be amd64 or arm64" >&2
    exit 1
    ;;
esac

if [[ "${DEPLOY_ALLOW_DIRTY_WORKTREE:-0}" != "1" ]]; then
  deploy_assert_clean_worktree
fi

HEAD="$(git rev-parse HEAD)"
if [[ -z "${EXPECTED_COMMIT}" ]]; then
  echo "error: EXPECTED_COMMIT required" >&2
  exit 1
fi
if [[ "${HEAD}" != "${EXPECTED_COMMIT}" ]]; then
  echo "error: HEAD does not match EXPECTED_COMMIT" >&2
  echo "error: HEAD=${HEAD}" >&2
  echo "error: EXPECTED_COMMIT=${EXPECTED_COMMIT}" >&2
  exit 1
fi

if [[ -z "${OUT_DIR}" ]]; then
  OUT_DIR="${ROOT}/dist/releases/${HEAD}"
fi

deploy_validate_out_dir "${ROOT}" "${OUT_DIR}"

if [[ -d "${OUT_DIR}" ]]; then
  if [[ -f "${OUT_DIR}/COMMIT" ]]; then
    existing="$(tr -d '[:space:]' <"${OUT_DIR}/COMMIT")"
    if [[ "${existing}" != "${HEAD}" ]]; then
      echo "error: refusing overwrite of release with different COMMIT" >&2
      exit 1
    fi
  fi
fi

mkdir -p "${OUT_DIR}"
rm -f "${OUT_DIR}/fiscal-api" "${OUT_DIR}/fiscal-migrate" "${OUT_DIR}/COMMIT" "${OUT_DIR}/SHA256SUMS" "${OUT_DIR}/remote-migrate-run.sh"

echo "building GOOS=${GOOS} GOARCH=${GOARCH} commit=${HEAD}"
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/fiscal-api" ./cmd/fiscal-api
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/fiscal-migrate" ./cmd/fiscal-migrate
cp "${SCRIPT_DIR}/remote-migrate-run.sh" "${OUT_DIR}/remote-migrate-run.sh"
chmod 0755 "${OUT_DIR}/remote-migrate-run.sh"
# Bundle allowlist helper next to runner for remote use.
mkdir -p "${OUT_DIR}/lib"
cp "${SCRIPT_DIR}/lib/allowlist.sh" "${OUT_DIR}/lib/allowlist.sh"

printf '%s\n' "${HEAD}" >"${OUT_DIR}/COMMIT"

(
  cd "${OUT_DIR}"
  deploy_sha256_files fiscal-api fiscal-migrate COMMIT >SHA256SUMS
  deploy_sha256_check SHA256SUMS
)

GOT_COMMIT="$(tr -d '[:space:]' <"${OUT_DIR}/COMMIT")"
if [[ "${GOT_COMMIT}" != "${HEAD}" ]]; then
  echo "error: COMMIT file mismatch" >&2
  exit 1
fi

echo "release_ok dir=${OUT_DIR} commit=${HEAD} arch=${GOARCH}"
