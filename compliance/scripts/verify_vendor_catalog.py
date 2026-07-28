#!/usr/bin/env python3
"""Validate compliance/catalog/vendor-integrations.yaml (non-normative).

Separate from verify_catalog.py / sources.yaml. Does not require local/.
"""

from __future__ import annotations

import hashlib
import json
import os
import sys
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

REPO_ROOT = Path(__file__).resolve().parents[2]
CATALOG_PATH = REPO_ROOT / "compliance" / "catalog" / "vendor-integrations.yaml"
SCHEMA_PATH = (
    REPO_ROOT / "compliance" / "catalog" / "schema" / "vendor-integrations.schema.json"
)
EXPECTED_COUNT = 2


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def main() -> int:
    errors: list[str] = []
    data = yaml.safe_load(CATALOG_PATH.read_text(encoding="utf-8"))
    schema = json.loads(SCHEMA_PATH.read_text(encoding="utf-8"))
    Draft202012Validator(schema).validate(data)

    if data.get("normative") is not False:
        errors.append("normative must be false")
    sources = data.get("sources") or []
    if len(sources) != EXPECTED_COUNT:
        errors.append(f"expected {EXPECTED_COUNT} vendor sources, got {len(sources)}")

    ids: set[str] = set()
    for src in sources:
        sid = src["id"]
        if sid in ids:
            errors.append(f"duplicate id {sid}")
        ids.add(sid)
        if sid.startswith("AO-"):
            errors.append(f"{sid}: must not use AO- prefix")
        if src.get("authority") != "vendor_technical":
            errors.append(f"{sid}: authority must be vendor_technical")
        rel = src.get("private_repository_path") or ""
        if not rel.startswith("originals/vendor-integrations/"):
            errors.append(f"{sid}: private path must be under vendor-integrations/")
        local_root = os.environ.get("COMPLIANCE_LOCAL_ROOT")
        if local_root:
            lp = Path(local_root) / Path(src["local_path"]).relative_to("local")
            # local_path is like local/docs/... — resolve under repo if present
            cand = REPO_ROOT / src["local_path"]
            if cand.is_file():
                digest = sha256_file(cand)
                if digest != src["sha256"]:
                    errors.append(
                        f"{sid}: local sha256 {digest} != catalog {src['sha256']}"
                    )

    if errors:
        for e in errors:
            print(f"ERROR: {e}", file=sys.stderr)
        return 1
    print(f"OK vendor-integrations catalog ({len(sources)} sources)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
