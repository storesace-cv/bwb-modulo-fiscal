#!/usr/bin/env bash
# Build linux release: binaries, helpers, COMMIT, EXPECTED_SCHEMA_VERSION, SHA256SUMS.
# Never ships remote-migrate-run.sh — migrate is executed by the closed helper after drop-priv.
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
EXPECTED_SCHEMA_VERSION="${EXPECTED_SCHEMA_VERSION:-${DEPLOY_EXPECTED_SCHEMA_VERSION_DEFAULT}}"
GOOS=linux

deploy_require_cmds go git

case "${GOARCH}" in
  amd64 | arm64) ;;
  *)
    echo "error: DEPLOY_GOARCH must be amd64 or arm64" >&2
    exit 1
    ;;
esac

deploy_assert_sha1 "EXPECTED_COMMIT" "${EXPECTED_COMMIT}"

# Real releases must always be from a clean tree. Builds under DEPLOY_TEST_OUT_ROOT
# are isolated test artefacts only (never used for live deploy without that gate).
if [[ -z "${DEPLOY_TEST_OUT_ROOT:-}" ]]; then
  deploy_assert_clean_worktree
fi

HEAD="$(git rev-parse HEAD)"
deploy_assert_sha1 "HEAD" "${HEAD}"
if [[ "${HEAD}" != "${EXPECTED_COMMIT}" ]]; then
  echo "error: HEAD does not match EXPECTED_COMMIT" >&2
  echo "error: HEAD=${HEAD}" >&2
  echo "error: EXPECTED_COMMIT=${EXPECTED_COMMIT}" >&2
  exit 1
fi

if [[ ! "${EXPECTED_SCHEMA_VERSION}" =~ ^[0-9]+$ ]]; then
  echo "error: EXPECTED_SCHEMA_VERSION must be a non-negative integer" >&2
  exit 1
fi

if [[ -z "${OUT_DIR}" ]]; then
  OUT_DIR="${ROOT}/dist/releases/${HEAD}"
fi

deploy_validate_out_dir "${ROOT}" "${OUT_DIR}"

if [[ -d "${OUT_DIR}" ]]; then
  if [[ -f "${OUT_DIR}/COMMIT" ]]; then
    existing="$(tr -d '[:space:]' <"${OUT_DIR}/COMMIT")"
    if [[ -n "${existing}" && "${existing}" != "${HEAD}" ]]; then
      echo "error: refusing overwrite of release with different COMMIT" >&2
      exit 1
    fi
  fi
  # Wipe fully so stale helpers cannot survive into the new artefact.
  rm -rf "${OUT_DIR}"
fi

mkdir -p "${OUT_DIR}/lib"

echo "building GOOS=${GOOS} GOARCH=${GOARCH} commit=${HEAD} schema=${EXPECTED_SCHEMA_VERSION}"
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/fiscal-api" ./cmd/fiscal-api
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/fiscal-migrate" ./cmd/fiscal-migrate
cp "${SCRIPT_DIR}/lib/allowlist.sh" "${OUT_DIR}/lib/allowlist.sh"
cp "${ROOT}/deploy/migrate.env.allowlist" "${OUT_DIR}/lib/migrate.env.allowlist"
chmod 0755 "${OUT_DIR}/fiscal-api" "${OUT_DIR}/fiscal-migrate"
chmod 0644 "${OUT_DIR}/lib/allowlist.sh" "${OUT_DIR}/lib/migrate.env.allowlist"

printf '%s\n' "${HEAD}" >"${OUT_DIR}/COMMIT"
printf '%s\n' "${EXPECTED_SCHEMA_VERSION}" >"${OUT_DIR}/EXPECTED_SCHEMA_VERSION"

(
  cd "${OUT_DIR}"
  deploy_sha256_files \
    fiscal-api \
    fiscal-migrate \
    lib/allowlist.sh \
    lib/migrate.env.allowlist \
    COMMIT \
    EXPECTED_SCHEMA_VERSION \
    >SHA256SUMS
  deploy_verify_release_manifest "${OUT_DIR}" "${HEAD}"
)

echo "release_ok dir=${OUT_DIR} commit=${HEAD} arch=${GOARCH} schema=${EXPECTED_SCHEMA_VERSION}"
