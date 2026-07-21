#!/usr/bin/env bash
# Reject modification/rename/deletion of existing migration files vs PR base.
# Also enforce postgres/sqlite version parity.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PG_DIR="migrations/postgres"
SQ_DIR="migrations/sqlite"

pg_versions="$(find "$PG_DIR" -name '*.up.sql' | sort | while read -r f; do basename "$f" | cut -d_ -f1; done | tr '\n' ' ')"
sq_versions="$(find "$SQ_DIR" -name '*.up.sql' | sort | while read -r f; do basename "$f" | cut -d_ -f1; done | tr '\n' ' ')"

if [[ -z "${pg_versions// }" || -z "${sq_versions// }" ]]; then
  echo "missing migrations in postgres or sqlite" >&2
  exit 1
fi
if [[ "$pg_versions" != "$sq_versions" ]]; then
  echo "migration version mismatch: postgres=[$pg_versions] sqlite=[$sq_versions]" >&2
  exit 1
fi
echo "migration version parity ok: $pg_versions"

BASE_SHA="${MIGRATION_BASE_SHA:-}"
if [[ -z "$BASE_SHA" ]]; then
  echo "skipping immutability check (no MIGRATION_BASE_SHA)"
  exit 0
fi

changed="$(git diff --name-status "$BASE_SHA"...HEAD -- "$PG_DIR" "$SQ_DIR" || true)"
if [[ -z "$changed" ]]; then
  echo "no migration path changes vs $BASE_SHA"
  exit 0
fi

forbidden=0
while IFS= read -r line; do
  [[ -z "$line" ]] && continue
  status="${line%%$'\t'*}"
  path="${line#*$'\t'}"
  case "$status" in
    A)
      echo "OK added migration: $path"
      ;;
    M|D|R*)
      echo "FORBIDDEN migration change ($status): $path" >&2
      forbidden=1
      ;;
    *)
      echo "unexpected status $status for $path" >&2
      forbidden=1
      ;;
  esac
done <<EOF
$changed
EOF

if [[ "$forbidden" -ne 0 ]]; then
  exit 1
fi
echo "migration immutability ok vs $BASE_SHA"
