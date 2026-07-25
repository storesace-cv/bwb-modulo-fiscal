#!/usr/bin/env python3
"""Validate compliance/catalog/sources.yaml against the JSON Schema and invariants.

Uses PyYAML + jsonschema (pinned in requirements.txt). Does not require local/.
Optional: COMPLIANCE_LOCAL_ROOT — if set, verify sha256 of local_path files when present.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

REPO_ROOT = Path(__file__).resolve().parents[2]
CATALOG_PATH = REPO_ROOT / "compliance" / "catalog" / "sources.yaml"
SCHEMA_PATH = REPO_ROOT / "compliance" / "catalog" / "schema" / "sources.schema.json"

EXPECTED_COLLECTION_COUNT = 20
LEGISLATION_PAGES = {
    "AO-LEG-DE-74-19-2019": 12,
    "AO-LEG-RECT-10-19-2019": 2,
    "AO-LEG-DE-683-25-2025": 16,
}


def load_yaml(path: Path) -> object:
    with path.open("r", encoding="utf-8") as fh:
        return yaml.safe_load(fh)


def load_schema(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as fh:
        return json.load(fh)


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def fail(errors: list[str]) -> int:
    for err in errors:
        print(f"ERROR: {err}", file=sys.stderr)
    return 1


def validate_relations(sources: list[dict], errors: list[str]) -> None:
    by_id = {s["id"]: s for s in sources}
    for src in sources:
        sid = src["id"]
        for field in ("supersedes", "superseded_by", "related_to"):
            for ref in src.get(field) or []:
                if ref not in by_id:
                    errors.append(f"{sid}: {field} references unknown id {ref}")
                elif ref == sid:
                    errors.append(f"{sid}: {field} must not self-reference")
        for older in src.get("supersedes") or []:
            if older in by_id and sid not in (by_id[older].get("superseded_by") or []):
                # Soft check: recommend bidirectional link when both listed.
                pass
        for der in src.get("derivatives") or []:
            if der.get("original_sha256") != src.get("sha256"):
                errors.append(
                    f"{sid}: derivative original_sha256 must match source sha256"
                )
            if der.get("page_count") != src.get("page_count"):
                errors.append(
                    f"{sid}: derivative page_count must match source page_count"
                )


def validate_legislation_invariants(sources: list[dict], errors: list[str]) -> None:
    for sid, pages in LEGISLATION_PAGES.items():
        matches = [s for s in sources if s["id"] == sid]
        if len(matches) != 1:
            errors.append(f"missing legislation source {sid}")
            continue
        src = matches[0]
        if src.get("page_count") != pages:
            errors.append(f"{sid}: page_count want {pages} got {src.get('page_count')}")
        if src.get("text_extractable") is not False:
            errors.append(f"{sid}: text_extractable must be false")
        if src.get("conversion_required") is not True:
            errors.append(f"{sid}: conversion_required must be true")
        if src.get("normative_role") != "normative_original":
            errors.append(f"{sid}: normative_role must be normative_original")
        req = set(src.get("required_future_derivatives") or [])
        if req != {"searchable_pdf", "markdown_text"}:
            errors.append(
                f"{sid}: required_future_derivatives must be searchable_pdf and markdown_text"
            )
        if src.get("derivatives"):
            errors.append(f"{sid}: derivatives must be empty until authorized OCR (PR B)")
        if src.get("versioned_path") is not None:
            errors.append(f"{sid}: versioned_path must be null in catalog-only phase")


def validate_no_versioned_binaries_yet(sources: list[dict], errors: list[str]) -> None:
    for src in sources:
        vp = src.get("versioned_path")
        if vp is not None:
            path = REPO_ROOT / vp
            if not path.is_file():
                errors.append(f"{src['id']}: versioned_path missing: {vp}")
            else:
                digest = sha256_file(path)
                if digest != src["sha256"]:
                    errors.append(
                        f"{src['id']}: versioned_path sha256 mismatch "
                        f"(catalog={src['sha256']} file={digest})"
                    )


def validate_optional_local(sources: list[dict], local_root: Path, errors: list[str]) -> None:
    for src in sources:
        rel = src["local_path"]
        if rel.startswith("local/"):
            rel = rel[len("local/") :]
        path = local_root / rel
        if not path.is_file():
            repo_local = REPO_ROOT / src["local_path"]
            if repo_local.is_file():
                path = repo_local
            else:
                errors.append(
                    f"{src['id']}: local file not found for hash check: {src['local_path']}"
                )
                continue
        digest = sha256_file(path)
        if digest != src["sha256"]:
            errors.append(
                f"{src['id']}: local sha256 mismatch "
                f"(catalog={src['sha256']} file={digest})"
            )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--with-local",
        action="store_true",
        help="Also verify sha256 against local/ files (development only).",
    )
    parser.add_argument(
        "--local-root",
        type=Path,
        default=None,
        help="Root containing docs/... (default: <repo>/local).",
    )
    args = parser.parse_args()

    errors: list[str] = []
    if not CATALOG_PATH.is_file():
        return fail([f"catalog missing: {CATALOG_PATH}"])
    if not SCHEMA_PATH.is_file():
        return fail([f"schema missing: {SCHEMA_PATH}"])

    catalog = load_yaml(CATALOG_PATH)
    schema = load_schema(SCHEMA_PATH)
    validator = Draft202012Validator(schema)
    for err in sorted(validator.iter_errors(catalog), key=lambda e: list(e.path)):
        loc = "/".join(str(p) for p in err.path) or "(root)"
        errors.append(f"schema {loc}: {err.message}")

    if not isinstance(catalog, dict):
        return fail(errors + ["catalog root must be a mapping"])

    sources = catalog.get("sources") or []
    if not isinstance(sources, list):
        return fail(errors + ["sources must be a list"])

    ids = [s.get("id") for s in sources if isinstance(s, dict)]
    if len(ids) != len(set(ids)):
        errors.append("duplicate source ids")
    if len(sources) != EXPECTED_COLLECTION_COUNT:
        errors.append(
            f"expected {EXPECTED_COLLECTION_COUNT} sources in collection, got {len(sources)}"
        )

    validate_relations([s for s in sources if isinstance(s, dict)], errors)
    validate_legislation_invariants([s for s in sources if isinstance(s, dict)], errors)
    validate_no_versioned_binaries_yet([s for s in sources if isinstance(s, dict)], errors)

    # ZIP must never be marked as git_public runtime dependency.
    for src in sources:
        if not isinstance(src, dict):
            continue
        if src.get("type") == "archive" and src.get("id") == "AO-SAFT-ZIP-ASSOFT-MASTER":
            if src.get("storage") != "local_only":
                errors.append("ZIP archive must remain storage=local_only")
            if src.get("versioned_path") is not None:
                errors.append("ZIP must not have versioned_path")

    if args.with_local or os.environ.get("COMPLIANCE_VERIFY_LOCAL") == "1":
        local_root = args.local_root or (REPO_ROOT / "local")
        if not local_root.is_dir():
            errors.append(f"--with-local requested but local root missing: {local_root}")
        else:
            validate_optional_local(
                [s for s in sources if isinstance(s, dict)], local_root, errors
            )

    if errors:
        return fail(errors)

    print(f"OK: catalog valid ({len(sources)} sources)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
