#!/usr/bin/env python3
# Fixed migrate DSN parser for pre-deploy pg gate (B1).
# Reads URL from stdin (never argv). Writes pg_service.conf + PGPASSFILE under --outdir.
# No operator-supplied code. No secrets on stdout/stderr.
from __future__ import annotations

import argparse
import os
import sys
from urllib.parse import unquote, urlparse, parse_qsl


ALLOWED_DB = frozenset(
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
)


def fail(msg: str) -> None:
    print(f"error: {msg}", file=sys.stderr)
    raise SystemExit(1)


def reject_inject(label: str, value: str) -> None:
    if any(c in value for c in ("\n", "\r", "\0")):
        fail(f"{label}_injection")


def escape_pgpass_field(value: str) -> str:
    # libpq: backslash-escape backslash and colon
    out: list[str] = []
    for ch in value:
        if ch in ("\\", ":"):
            out.append("\\" + ch)
        else:
            out.append(ch)
    return "".join(out)


def parse_url(raw: str) -> tuple[str, str]:
    raw = raw.strip()
    if not raw:
        fail("url_empty")
    if any(c in raw for c in ("\n", "\r", "\0")):
        fail("url_control_chars")

    try:
        u = urlparse(raw)
    except Exception:
        fail("url_parse")

    if u.scheme not in ("postgres", "postgresql"):
        fail("url_scheme")
    if u.hostname != "127.0.0.1":
        fail("url_host")
    if u.port != 5432:
        fail("url_port")
    if u.fragment:
        fail("url_fragment")
    # Reject multi-host (netloc with commas) and userinfo anomalies
    if "," in (u.netloc or ""):
        fail("url_multi_host")

    user = unquote(u.username or "")
    password = unquote(u.password if u.password is not None else "")
    if user != "fiscal_migrate":
        fail("url_user")
    if not password:
        fail("url_password_empty")
    reject_inject("user", user)
    reject_inject("password", password)

    path = u.path or ""
    if not path.startswith("/") or path.count("/") != 1:
        fail("url_dbname_path")
    dbname = unquote(path[1:])
    if not dbname or dbname.startswith("/"):
        fail("url_dbname")
    if any(c not in ALLOWED_DB for c in dbname):
        fail("url_dbname_charset")
    if not (dbname[0].isalpha() or dbname[0] == "_"):
        fail("url_dbname_charset")
    reject_inject("dbname", dbname)

    # Query: exactly sslmode=require once; reject duplicates/unknown
    q = u.query or ""
    pairs = parse_qsl(q, keep_blank_values=True)
    if len(pairs) != 1:
        fail("url_query")
    key, val = pairs[0]
    if key != "sslmode" or val != "require":
        fail("url_sslmode")
    # Detect duplicate keys in raw query
    keys = [k for k, _ in parse_qsl(q, keep_blank_values=True)]
    if keys.count("sslmode") != 1:
        fail("url_query_duplicate")

    # Percent-decoding invalid: urllib usually substitutes; reject % not followed by hex
    i = 0
    while i < len(raw):
        if raw[i] == "%":
            hx = raw[i + 1 : i + 3]
            if len(hx) < 2 or any(c not in "0123456789abcdefABCDEF" for c in hx):
                fail("url_percent_decoding")
            i += 3
        else:
            i += 1

    return dbname, password


def main() -> None:
    ap = argparse.ArgumentParser(add_help=False)
    ap.add_argument("--outdir", required=True)
    ap.add_argument("--temp-db", required=True)
    args = ap.parse_args()

    outdir = args.outdir
    temp_db = args.temp_db
    if any(c not in ALLOWED_DB and c != "" for c in temp_db):
        # temp_db validated by caller regex; double-check charset
        if not all(c.isalnum() or c == "_" for c in temp_db):
            fail("temp_db_invalid")
    reject_inject("temp_db", temp_db)

    if not os.path.isdir(outdir):
        fail("outdir_missing")

    raw = sys.stdin.read()
    dbname, password = parse_url(raw)

    service_path = os.path.join(outdir, "pg_service.conf")
    pgpass_path = os.path.join(outdir, "pgpass")

    service = (
        "[bwb_predeploy]\n"
        "host=127.0.0.1\n"
        "port=5432\n"
        "user=fiscal_migrate\n"
        f"dbname={dbname}\n"
        "sslmode=require\n"
        "\n"
        "[bwb_predeploy_temp]\n"
        "host=127.0.0.1\n"
        "port=5432\n"
        "user=fiscal_migrate\n"
        f"dbname={temp_db}\n"
        "sslmode=require\n"
    )
    with open(service_path, "w", encoding="ascii", newline="\n") as f:
        f.write(service)
        f.flush()
        os.fsync(f.fileno())
    os.chmod(service_path, 0o600)

    # Cover both dbnames with wildcard database field
    line = (
        "127.0.0.1:5432:*:fiscal_migrate:"
        + escape_pgpass_field(password)
        + "\n"
    )
    with open(pgpass_path, "w", encoding="utf-8", newline="\n") as f:
        f.write(line)
        f.flush()
        os.fsync(f.fileno())
    os.chmod(pgpass_path, 0o600)

    # Success marker only (no secrets)
    print("parse_ok")


if __name__ == "__main__":
    main()
