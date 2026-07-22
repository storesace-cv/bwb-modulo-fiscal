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

mkdir -p "${OUT_DIR}/lib" "${OUT_DIR}/fixtures/sandbox"

echo "building GOOS=${GOOS} GOARCH=${GOARCH} commit=${HEAD} schema=${EXPECTED_SCHEMA_VERSION}"
REVISION_PKG="github.com/storesace-cv/bwb-modulo-fiscal/internal/buildinfo.Revision"
LDFLAGS="-s -w -X ${REVISION_PKG}=${HEAD}"
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="${LDFLAGS}" -o "${OUT_DIR}/fiscal-api" ./cmd/fiscal-api
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="${LDFLAGS}" -o "${OUT_DIR}/fiscal-migrate" ./cmd/fiscal-migrate
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="${LDFLAGS}" -o "${OUT_DIR}/fiscal-admin" ./cmd/fiscal-admin
CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build -trimpath -ldflags="${LDFLAGS}" -o "${OUT_DIR}/fiscal-sandbox-measure" ./cmd/fiscal-sandbox-measure
cp "${SCRIPT_DIR}/lib/allowlist.sh" "${OUT_DIR}/lib/allowlist.sh"
cp "${ROOT}/deploy/migrate.env.allowlist" "${OUT_DIR}/lib/migrate.env.allowlist"
cp "${ROOT}/deploy/admin.env.allowlist" "${OUT_DIR}/lib/admin.env.allowlist"
install -m 0755 "${SCRIPT_DIR}/fiscal-sandbox-e2e.sh" "${OUT_DIR}/fiscal-sandbox-e2e"
cp "${ROOT}/deploy/fixtures/sandbox/"*.json "${OUT_DIR}/fixtures/sandbox/"
chmod 0755 "${OUT_DIR}/fiscal-api" "${OUT_DIR}/fiscal-migrate" "${OUT_DIR}/fiscal-admin" \
  "${OUT_DIR}/fiscal-sandbox-e2e" "${OUT_DIR}/fiscal-sandbox-measure"
chmod 0644 "${OUT_DIR}/lib/allowlist.sh" "${OUT_DIR}/lib/migrate.env.allowlist" \
  "${OUT_DIR}/lib/admin.env.allowlist" "${OUT_DIR}/fixtures/sandbox/"*.json

printf '%s\n' "${HEAD}" >"${OUT_DIR}/COMMIT"
printf '%s\n' "${EXPECTED_SCHEMA_VERSION}" >"${OUT_DIR}/EXPECTED_SCHEMA_VERSION"

# Fail closed: injected revision must equal COMMIT/HEAD (SHA40).
# On a foreign host OS, do not execute the cross-compiled linux binary; verify via
# host-native `go run` with the same -X Revision ldflags. On linux CI, execute the artefact.
set +e
HOST_GOOS="$(go env GOOS)"
if [[ "${HOST_GOOS}" == "${GOOS}" ]]; then
  ver_out="$("${OUT_DIR}/fiscal-api" version 2>"${OUT_DIR}/.version.err")"
  ver_st=$?
else
  ver_out="$(CGO_ENABLED=0 go run -trimpath -ldflags="${LDFLAGS}" ./cmd/fiscal-api version 2>"${OUT_DIR}/.version.err")"
  ver_st=$?
fi
set -e
if [[ "${ver_st}" -ne 0 ]]; then
  echo "error: fiscal-api version failed" >&2
  if [[ -s "${OUT_DIR}/.version.err" ]]; then
    cat "${OUT_DIR}/.version.err" >&2
  fi
  rm -f "${OUT_DIR}/.version.err"
  exit 1
fi
rm -f "${OUT_DIR}/.version.err"
rev="$(printf '%s\n' "${ver_out}" | sed -n 's/^version=[^ ]* revision=\([0-9a-f]\{40\}\)$/\1/p' | head -1)"
if [[ -z "${rev}" ]]; then
  echo "error: fiscal-api version did not report lowercase sha40 revision" >&2
  echo "error: output=${ver_out}" >&2
  exit 1
fi
if [[ "${rev}" != "${HEAD}" ]]; then
  echo "error: fiscal-api revision does not match HEAD/COMMIT" >&2
  echo "error: revision=${rev}" >&2
  echo "error: HEAD=${HEAD}" >&2
  exit 1
fi
committed="$(tr -d '[:space:]' <"${OUT_DIR}/COMMIT")"
if [[ "${rev}" != "${committed}" ]]; then
  echo "error: fiscal-api revision does not match COMMIT file" >&2
  exit 1
fi

(
  cd "${OUT_DIR}"
  deploy_sha256_files \
    fiscal-api \
    fiscal-migrate \
    fiscal-admin \
    fiscal-sandbox-e2e \
    fiscal-sandbox-measure \
    lib/allowlist.sh \
    lib/migrate.env.allowlist \
    lib/admin.env.allowlist \
    fixtures/sandbox/create-document.min.json \
    fixtures/sandbox/create-document.b.json \
    fixtures/sandbox/create-document.nif-mismatch.json \
    fixtures/sandbox/create-document.invalid.json \
    COMMIT \
    EXPECTED_SCHEMA_VERSION \
    >SHA256SUMS
  deploy_verify_release_manifest "${OUT_DIR}" "${HEAD}"
)

echo "release_ok dir=${OUT_DIR} commit=${HEAD} arch=${GOARCH} schema=${EXPECTED_SCHEMA_VERSION}"
