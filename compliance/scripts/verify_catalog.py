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
import re
import sys
from pathlib import Path

import yaml
from jsonschema import Draft202012Validator

REPO_ROOT = Path(__file__).resolve().parents[2]
CATALOG_PATH = REPO_ROOT / "compliance" / "catalog" / "sources.yaml"
SCHEMA_PATH = REPO_ROOT / "compliance" / "catalog" / "schema" / "sources.schema.json"
SCHEMAS_DIR = REPO_ROOT / "compliance" / "saft-ao" / "schemas"
SCHEMAS_MANIFEST = SCHEMAS_DIR / "SHA256SUMS.txt"
LICENSE_EXPECTED_SHA256 = (
    "b0ae3b4eb33bf63c99fd4c818419f19296cf03dbfde9331e5856fb192cb3ea82"
)
XSD_EXPECTED_SHA256 = (
    "e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631"
)
XSD_CANONICAL_ID = "AO-SAFT-XSD-1.01_01"
XSD_CANONICAL_VERSIONED_PATH = "compliance/saft-ao/schemas/SAFTAO1.01_01.xsd"
MANIFEST_ENTRIES = ("LICENSE", "NOTICE.md", "README.md", "SAFTAO1.01_01.xsd")

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
        for der in src.get("derivatives") or []:
            if der.get("original_sha256") != src.get("sha256"):
                errors.append(
                    f"{sid}: derivative original_sha256 must match source sha256"
                )
            if der.get("page_count") != src.get("page_count"):
                errors.append(
                    f"{sid}: derivative page_count must match source page_count"
                )


def validate_ocr_derivatives(src: dict, errors: list[str]) -> None:
    sid = src["id"]
    pages = src.get("page_count")
    ders = src.get("derivatives") or []
    if not ders:
        errors.append(f"{sid}: derivatives obrigatório após OCR autorizado")
        return
    kinds = [d.get("kind") for d in ders]
    if sorted(kinds) != ["markdown_text", "searchable_pdf"]:
        errors.append(
            f"{sid}: derivatives deve ter searchable_pdf + markdown_text (got {kinds})"
        )
    statuses = {d.get("review_status") for d in ders}
    if "rejected" in statuses:
        errors.append(
            f"{sid}: derivatives rejected não podem ser publicados como KB "
            "(manter só diagnóstico privado; limpar derivatives[] públicos)"
        )
    if not statuses.issubset({"generated_unreviewed", "reviewed", "rejected"}):
        errors.append(f"{sid}: review_status OCR inválido {statuses}")
    if "partially_reviewed" in statuses:
        errors.append(f"{sid}: partially_reviewed não é usado neste workflow")
    if len(statuses) != 1:
        errors.append(f"{sid}: review_status inconsistente entre derivatives")
    status = next(iter(statuses)) if statuses else None
    for der in ders:
        if der.get("original_sha256") != src.get("sha256"):
            errors.append(f"{sid}: original_sha256 diverge")
        if der.get("page_count") != pages:
            errors.append(f"{sid}: page_count derivado diverge")
        if not der.get("ocr_config"):
            errors.append(f"{sid}: ocr_config obrigatório")
        if not der.get("derivative_version"):
            errors.append(f"{sid}: derivative_version obrigatório")
        path = der.get("private_repository_path")
        if not isinstance(path, str) or not path.startswith(
            f"derivatives/legislation/{sid}/"
        ):
            errors.append(f"{sid}: private_repository_path inválido para derivado")
        pr = der.get("pages_reviewed") or []
        if status in {"reviewed", "rejected"}:
            expected = list(range(1, int(pages) + 1))
            if pr != expected:
                errors.append(
                    f"{sid}: pages_reviewed incompleto para {status}"
                )
        elif status == "generated_unreviewed" and pr:
            errors.append(f"{sid}: pages_reviewed deve estar vazio em unreviewed")
        if status != "reviewed" and src.get("derived_requirements"):
            errors.append(
                f"{sid}: derived_requirements só permitido com OCR reviewed"
            )
    tags = set(src.get("tags") or [])
    if status == "reviewed":
        if "ocr_reviewed" not in tags:
            errors.append(f"{sid}: tag ocr_reviewed obrigatória quando reviewed")
        if "ocr_pending" in tags:
            errors.append(f"{sid}: remover ocr_pending quando reviewed")
    elif status in {"generated_unreviewed", "rejected"}:
        if "ocr_pending" not in tags:
            errors.append(f"{sid}: tag ocr_pending obrigatória quando {status}")


def validate_legislation_invariants(sources: list[dict], errors: list[str]) -> None:
    by_id = {s["id"]: s for s in sources}
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
        # OCR authorized: derivatives may be empty only before first private sync of OCR.
        # After RM-SRC-004, legislation must carry searchable_pdf + markdown_text metadata.
        if src.get("derivatives"):
            validate_ocr_derivatives(src, errors)
        else:
            # Allow empty only if still tagged ocr_pending and conversion_required —
            # transitional; prefer populated derivatives.
            tags = set(src.get("tags") or [])
            if "ocr_pending" not in tags:
                errors.append(
                    f"{sid}: sem derivatives exige tag ocr_pending (pré-OCR)"
                )

    # Companion set: if either 74/19 or Rect is reviewed, both must be.
    a = "AO-LEG-DE-74-19-2019"
    b = "AO-LEG-RECT-10-19-2019"
    if a in by_id and b in by_id:
        def status_of(sid: str) -> str | None:
            ders = by_id[sid].get("derivatives") or []
            if not ders:
                return None
            return ders[0].get("review_status")

        sa, sb = status_of(a), status_of(b)
        if sa == "reviewed" and sb == "generated_unreviewed":
            errors.append(f"{a}: reviewed exige companion {b} reviewed ou rejected")
        if sb == "reviewed" and sa == "generated_unreviewed":
            errors.append(f"{b}: reviewed exige companion {a} reviewed ou rejected")


def validate_private_sync(sources: list[dict], errors: list[str]) -> None:
    for src in sources:
        sid = src["id"]
        storage = src.get("storage")
        repo = src.get("private_repository")
        commit = src.get("private_commit")
        path = src.get("private_repository_path")
        if storage == "private_sync":
            if repo != "storesace-cv/bwb-fiscal-sources-ao":
                errors.append(f"{sid}: private_repository inválido para private_sync")
            if not isinstance(commit, str) or len(commit) != 40:
                errors.append(f"{sid}: private_commit obrigatório (40 hex)")
            if not isinstance(path, str) or not path.startswith("originals/"):
                errors.append(f"{sid}: private_repository_path obrigatório sob originals/")
        else:
            if repo is not None or commit is not None or path is not None:
                errors.append(
                    f"{sid}: campos private_* só são permitidos com storage=private_sync"
                )


def resolve_versioned_path(vp: str, sid: str, errors: list[str]) -> Path | None:
    if not isinstance(vp, str) or not vp.strip():
        errors.append(f"{sid}: versioned_path vazio")
        return None
    if vp.startswith("/") or re.match(r"^[A-Za-z]:[\\/]", vp):
        errors.append(f"{sid}: versioned_path absoluto rejeitado: {vp}")
        return None
    if "\\" in vp:
        errors.append(f"{sid}: versioned_path deve usar separadores POSIX: {vp}")
        return None
    parts = Path(vp).parts
    if ".." in parts or "." in parts:
        # Allow no "." components for normalization strictness
        if ".." in parts:
            errors.append(f"{sid}: versioned_path traversal rejeitado: {vp}")
            return None
        if "." in parts:
            errors.append(f"{sid}: versioned_path não normalizado: {vp}")
            return None
    if vp != Path(vp).as_posix():
        errors.append(f"{sid}: versioned_path não normalizado: {vp}")
        return None
    leaf = REPO_ROOT / vp
    try:
        leaf.resolve().relative_to(REPO_ROOT.resolve())
    except ValueError:
        errors.append(f"{sid}: versioned_path fora do repositório: {vp}")
        return None
    return leaf


def validate_storage_paths(sources: list[dict], errors: list[str]) -> None:
    for src in sources:
        sid = src["id"]
        storage = src.get("storage")
        vp = src.get("versioned_path")
        lic = src.get("license_redistribution")

        if storage == "git_public":
            if vp is None:
                errors.append(f"{sid}: git_public exige versioned_path")
                continue
            if lic != "permitted":
                errors.append(
                    f"{sid}: git_public exige license_redistribution=permitted"
                )
            path = resolve_versioned_path(vp, sid, errors)
            if path is None:
                continue
            if path.is_symlink() or (REPO_ROOT / vp).is_symlink():
                errors.append(f"{sid}: versioned_path não pode ser symlink: {vp}")
                continue
            if not path.is_file():
                errors.append(f"{sid}: versioned_path missing: {vp}")
                continue
            digest = sha256_file(path)
            if digest != src.get("sha256"):
                errors.append(
                    f"{sid}: versioned_path sha256 mismatch "
                    f"(catalog={src.get('sha256')} file={digest})"
                )
            continue

        if storage in {"private_sync", "local_only"}:
            if vp is not None:
                errors.append(
                    f"{sid}: {storage} proíbe versioned_path (got {vp})"
                )

        if src.get("type") == "archive" or sid == "AO-SAFT-ZIP-ASSOFT-MASTER":
            if storage == "git_public":
                errors.append(f"{sid}: ZIP/archive nunca git_public")
            if vp is not None:
                errors.append(f"{sid}: ZIP/archive nunca com versioned_path")
            if sid == "AO-SAFT-ZIP-ASSOFT-MASTER" and storage != "local_only":
                errors.append("ZIP archive must remain storage=local_only")


def parse_schemas_manifest(errors: list[str]) -> dict[str, str] | None:
    if not SCHEMAS_MANIFEST.is_file():
        errors.append("compliance/saft-ao/schemas/SHA256SUMS.txt ausente")
        return None
    mapping: dict[str, str] = {}
    for line_no, raw in enumerate(
        SCHEMAS_MANIFEST.read_text(encoding="utf-8").splitlines(), 1
    ):
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split()
        if len(parts) != 2:
            errors.append(f"SHA256SUMS linha {line_no} malformada")
            continue
        digest, name = parts
        if not re.fullmatch(r"[a-f0-9]{64}", digest):
            errors.append(f"SHA256SUMS sha256 inválido linha {line_no}")
            continue
        if name in mapping:
            errors.append(f"SHA256SUMS path duplicado: {name}")
            continue
        mapping[name] = digest
    return mapping


def validate_schemas_manifest(errors: list[str]) -> None:
    mapping = parse_schemas_manifest(errors)
    if mapping is None:
        return

    expected = list(MANIFEST_ENTRIES)
    if list(mapping.keys()) != expected:
        errors.append(
            "SHA256SUMS entradas incorrectas "
            f"(want {expected}, got {list(mapping.keys())})"
        )
        return

    for name in expected:
        path = SCHEMAS_DIR / name
        if path.is_symlink():
            errors.append(f"schemas/{name}: symlink rejeitado")
            continue
        if not path.is_file():
            errors.append(f"schemas/{name}: ficheiro ausente")
            continue
        digest = sha256_file(path)
        if digest != mapping[name]:
            errors.append(f"schemas/{name}: sha256 diverge do manifesto")
        if name == "LICENSE" and digest != LICENSE_EXPECTED_SHA256:
            errors.append(
                f"schemas/LICENSE: sha256 diverge do valor fixo esperado "
                f"({LICENSE_EXPECTED_SHA256})"
            )


def validate_canonical_xsd(sources: list[dict], errors: list[str]) -> None:
    """Integrity pin for the immutable ASSOFT SAF-T AO XSD (does not replace
    generic git_public / versioned_path invariants).
    """
    matches = [s for s in sources if s.get("id") == XSD_CANONICAL_ID]
    if len(matches) != 1:
        errors.append(
            f"{XSD_CANONICAL_ID}: exactamente uma entrada exigida "
            f"(got {len(matches)})"
        )
        return
    src = matches[0]
    if src.get("storage") != "git_public":
        errors.append(f"{XSD_CANONICAL_ID}: storage deve ser git_public")
    vp = src.get("versioned_path")
    if vp != XSD_CANONICAL_VERSIONED_PATH:
        errors.append(
            f"{XSD_CANONICAL_ID}: versioned_path canónico exacto exigido "
            f"({XSD_CANONICAL_VERSIONED_PATH}), got {vp}"
        )
    catalog_sha = src.get("sha256")
    if catalog_sha != XSD_EXPECTED_SHA256:
        errors.append(
            f"{XSD_CANONICAL_ID}: sha256 do catálogo diverge do hash canónico "
            f"fixo ({XSD_EXPECTED_SHA256})"
        )

    xsd_path = REPO_ROOT / XSD_CANONICAL_VERSIONED_PATH
    if xsd_path.is_symlink():
        errors.append(
            f"{XSD_CANONICAL_ID}: ficheiro canónico não pode ser symlink"
        )
    elif not xsd_path.is_file():
        errors.append(
            f"{XSD_CANONICAL_ID}: ficheiro canónico ausente "
            f"({XSD_CANONICAL_VERSIONED_PATH})"
        )
    else:
        file_sha = sha256_file(xsd_path)
        if file_sha != XSD_EXPECTED_SHA256:
            errors.append(
                f"{XSD_CANONICAL_ID}: SHA-256 do ficheiro diverge do hash "
                f"canónico fixo ({XSD_EXPECTED_SHA256})"
            )

    # Read only the XSD manifest line (avoid re-emitting SHA256SUMS parse errors).
    manifest_sha: str | None = None
    if SCHEMAS_MANIFEST.is_file():
        for raw in SCHEMAS_MANIFEST.read_text(encoding="utf-8").splitlines():
            line = raw.strip()
            if not line or line.startswith("#"):
                continue
            parts = line.split()
            if len(parts) == 2 and parts[1] == "SAFTAO1.01_01.xsd":
                manifest_sha = parts[0]
                break
    if manifest_sha != XSD_EXPECTED_SHA256:
        errors.append(
            f"{XSD_CANONICAL_ID}: entrada SHA256SUMS diverge do hash canónico "
            f"fixo ({XSD_EXPECTED_SHA256})"
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

    typed = [s for s in sources if isinstance(s, dict)]
    validate_relations(typed, errors)
    validate_legislation_invariants(typed, errors)
    validate_private_sync(typed, errors)
    validate_storage_paths(typed, errors)
    validate_schemas_manifest(errors)
    validate_canonical_xsd(typed, errors)

    if args.with_local or os.environ.get("COMPLIANCE_VERIFY_LOCAL") == "1":
        local_root = args.local_root or (REPO_ROOT / "local")
        if not local_root.is_dir():
            errors.append(f"--with-local requested but local root missing: {local_root}")
        else:
            validate_optional_local(typed, local_root, errors)

    if errors:
        return fail(errors)

    print(f"OK: catalog valid ({len(sources)} sources)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
