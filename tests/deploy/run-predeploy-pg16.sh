#!/usr/bin/env bash
# Real PostgreSQL 16 pre-deploy gate test for CI Ubuntu.
# Starts an ephemeral SSL-enabled Postgres on 127.0.0.1:5432 (fails closed if busy).
# Never prints URLs, passwords, or dump contents.
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT}"

if [[ "${BWB_SKIP_PREDEPLOY_PG16:-0}" == "1" ]]; then
  echo "SKIP: BWB_SKIP_PREDEPLOY_PG16=1"
  exit 0
fi

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "SKIP: missing $1"
    exit 0
  }
}
need initdb
need postgres
need psql
need pg_isready
need createdb
need dropdb
need pg_dump
need pg_restore
need openssl

PG_VER="$(postgres -V 2>/dev/null || true)"
if [[ "${PG_VER}" != *" 16."* && "${PG_VER}" != *"16"* && "${BWB_PREDEPLOY_ALLOW_NON16:-0}" != "1" ]]; then
  echo "SKIP: PostgreSQL 16 required (got: ${PG_VER:-unknown})"
  exit 0
fi

if (echo >/dev/tcp/127.0.0.1/5432) 2>/dev/null; then
  echo "SKIP: 127.0.0.1:5432 busy (closed parser requires port 5432)"
  exit 0
fi

TMP="$(mktemp -d)"
PG_PID=""
cleanup() {
  if [[ -n "${PG_PID}" ]]; then
    kill "${PG_PID}" 2>/dev/null || true
    wait "${PG_PID}" 2>/dev/null || true
  fi
  rm -rf "${TMP}"
}
trap cleanup EXIT

PGDATA="${TMP}/pgdata"
CERT_DIR="${TMP}/certs"
SOCK_DIR="${TMP}/sock"
RUN_ROOT="${TMP}/run"
BACKUP_ROOT="${TMP}/backups"
ETC="${TMP}/etc"
LIB="${TMP}/lib"
mkdir -p "${CERT_DIR}" "${SOCK_DIR}" "${RUN_ROOT}" "${BACKUP_ROOT}" "${ETC}" "${LIB}"

cp "${ROOT}/scripts/deploy/lib/allowlist.sh" "${LIB}/allowlist.sh"
cp "${ROOT}/deploy/migrate.env.allowlist" "${LIB}/migrate.env.allowlist"
cp "${ROOT}/deploy/admin.env.allowlist" "${LIB}/admin.env.allowlist"
cp "${ROOT}/scripts/deploy/lib/parse_migrate_dsn.py" "${LIB}/parse_migrate_dsn.py"
cp "${ROOT}/scripts/deploy/lib/predeploy_pg.sh" "${LIB}/predeploy_pg.sh"

openssl req -new -x509 -days 1 -nodes \
  -subj "/CN=127.0.0.1" \
  -out "${CERT_DIR}/server.crt" \
  -keyout "${CERT_DIR}/server.key" >/dev/null 2>&1
chmod 600 "${CERT_DIR}/server.key"

initdb -D "${PGDATA}" -A trust -U postgres --locale=C --encoding=UTF8 >/dev/null
cat >>"${PGDATA}/postgresql.conf" <<EOF
listen_addresses = '127.0.0.1'
port = 5432
ssl = on
ssl_cert_file = '${CERT_DIR}/server.crt'
ssl_key_file = '${CERT_DIR}/server.key'
unix_socket_directories = '${SOCK_DIR}'
EOF
cat >"${PGDATA}/pg_hba.conf" <<EOF
local all all trust
hostssl all all 127.0.0.1/32 scram-sha-256
host all all 127.0.0.1/32 reject
EOF

postgres -D "${PGDATA}" >"${TMP}/pg.log" 2>&1 &
PG_PID=$!
for _ in $(seq 1 80); do
  if pg_isready -h "${SOCK_DIR}" -p 5432 -U postgres >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done
pg_isready -h "${SOCK_DIR}" -p 5432 -U postgres >/dev/null

MIG_PASS='migrate-test-pass'
psql -h "${SOCK_DIR}" -p 5432 -U postgres -v ON_ERROR_STOP=1 <<SQL
CREATE ROLE fiscal_migrate LOGIN PASSWORD '${MIG_PASS}' NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE DATABASE fiscal OWNER fiscal_migrate;
\\c fiscal
CREATE TABLE public.bwb_schema_migrations (
  version bigint PRIMARY KEY,
  dirty boolean NOT NULL
);
INSERT INTO public.bwb_schema_migrations(version, dirty) VALUES (3, false);
GRANT ALL ON TABLE public.bwb_schema_migrations TO fiscal_migrate;
SQL

privs="$(psql -h "${SOCK_DIR}" -p 5432 -U postgres -Atc \
  "SELECT rolsuper::text || '|' || rolcreatedb::text FROM pg_roles WHERE rolname='fiscal_migrate';")"
case "${privs}" in
  false\|false | f\|f) echo "PASS: fiscal_migrate NOSUPERUSER NOCREATEDB" ;;
  *)
    echo "FAIL: fiscal_migrate privileges unexpected" >&2
    exit 1
    ;;
esac

# URL password must not need encoding for this fixed test secret.
cat >"${ETC}/migrate.env" <<EOF
FISCAL_DATABASE_DRIVER=postgres
FISCAL_DATABASE_URL=postgres://fiscal_migrate:${MIG_PASS}@127.0.0.1:5432/fiscal?sslmode=require
EOF
chmod 0600 "${ETC}/migrate.env"

SHA40="$(git rev-parse HEAD)"
BID="$(date -u +%Y%m%dT%H%M%SZ)-${SHA40}"

export BWB_HELPER_LIB="${LIB}"
export BWB_DEPLOY_ETC="${ETC}"
export BWB_DEPLOY_RUN="${RUN_ROOT}"
export BWB_PG_BACKUP_ROOT="${BACKUP_ROOT}"
export BWB_PREDEPLOY_PG_SOCKET="${SOCK_DIR}"
export BWB_PREDEPLOY_PG_PORT=5432
export BWB_PREDEPLOY_PG_USER=postgres
# Prefer trust local socket as postgres role (no OS user switch in CI).
unset BWB_PREDEPLOY_OS_POSTGRES_WRAP

run_helper() {
  bash "${ROOT}/scripts/deploy/remote-deploy-helper.sh" "$@"
}

run_helper deploy-lock-acquire "${BID}" >/dev/null
set +e
out="$(run_helper pre-deploy-pg-backup "${SHA40}" "${BID}" 2>"${TMP}/gate.err")"
st=$?
set -e
printf '%s\n' "${out}"
if [[ "${st}" -ne 0 ]] || ! printf '%s' "${out}" | grep -q 'deploy_allowed=true'; then
  echo "FAIL: real pre-deploy gate refused" >&2
  cat "${TMP}/gate.err" >&2 || true
  exit 1
fi
if ! printf '%s' "${out}" | grep -q 'backup_created=true' \
  || ! printf '%s' "${out}" | grep -q 'restore_verified=true'; then
  echo "FAIL: gate success flags incomplete" >&2
  exit 1
fi
if ! compgen -G "${BACKUP_ROOT}/"*.dump >/dev/null; then
  echo "FAIL: durable dump missing" >&2
  exit 1
fi
if printf '%s' "${out}" | grep -qiE "postgres://|${MIG_PASS}|PGPASSWORD"; then
  echo "FAIL: secret leaked in gate output" >&2
  exit 1
fi
if grep -qiE "postgres://|${MIG_PASS}|PGPASSWORD" "${TMP}/gate.err" 2>/dev/null; then
  echo "FAIL: secret leaked in gate stderr" >&2
  exit 1
fi
run_helper deploy-lock-release "${BID}" >/dev/null

# Static repo proof (defense in depth)
if grep -RInE 'ALTER[[:space:]]+ROLE[[:space:]]+fiscal_migrate[[:space:]].*CREATEDB' \
  scripts/deploy docs/07-operations 2>/dev/null \
  | grep -v 'sem CREATEDB\|without CREATEDB\|não.*CREATEDB\|no CREATEDB'; then
  echo "FAIL: CREATEDB grant found in deploy docs/scripts" >&2
  exit 1
fi

echo "PASS: real pre-deploy pg gate on PostgreSQL 16"
