#!/usr/bin/env python3
"""Fail-closed checks for the RM-REQ-001 confirmed AO-* matrix.

Stdlib only. Confirmed here means normative confirmation with page citations —
never AGT homologation or production authorization.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

MATRIX_REL = Path("compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md")

ALLOWED_STATUSES = {"confirmed_normative"}
FORBIDDEN_CLAIM = re.compile(
    r"(?i)\b(validated_agt|homologad[oa]|produ[cç][aã]o_autorizada|aprovado_agt)\b"
)
ROW_RE = re.compile(
    r"^###\s+(AO-[A-Z]+-\d+)\s+",
    re.M,
)


def fail(msg: str, errors: list[str]) -> None:
    errors.append(msg)


def verify(root: Path) -> list[str]:
    errors: list[str] = []
    path = root / MATRIX_REL
    if not path.is_file():
        return [f"ausente: {MATRIX_REL.as_posix()}"]
    text = path.read_text(encoding="utf-8")

    if "confirmed_normative" not in text:
        fail("matriz confirmada deve usar estado `confirmed_normative`", errors)
    if "Não é:" not in text:
        fail("matriz deve ter banner 'Não é:' (AGT/homologação/produção)", errors)
    if "aceitação AGT" not in text and "aceitacao AGT" not in text:
        fail("matriz deve declarar que não é aceitação AGT", errors)

    for line in text.splitlines():
        if FORBIDDEN_CLAIM.search(line) and not re.search(r"(?i)não|nao|proibid", line):
            fail(f"afirmação AGT/homologação/produção indevida: {line.strip()[:120]}", errors)
            break

    ids = ROW_RE.findall(text)
    for required in ("AO-DOC-002", "AO-SEQ-001"):
        if required not in ids:
            fail(f"{required} deve constar como secção ### confirmada", errors)

    # AO-DOC-002 evidence package
    section = ""
    m = re.search(
        r"###\s+AO-DOC-002\b.*?(?=\n###\s|\n##\s+Ainda|\n##\s+Verificador|$)",
        text,
        flags=re.S,
    )
    if not m:
        fail("secção AO-DOC-002 ausente ou mal delimitada", errors)
    else:
        section = m.group(0)
        if "`confirmed_normative`" not in section:
            fail("AO-DOC-002: estado deve ser `confirmed_normative`", errors)
        for token in (
            "AO-LEG-DP-71-25-2025",
            "4931fd3c",
            "11904",
            "11907",
            "Art.3",
            "Art.8",
            "elimina",
            "AO-LEG-DE-74-19-2019",
            "5b63c80e",
            "1576",
            "1577",
            "AO-LEG-RECT-10-19-2019",
            "b3db14e2",
            "1948",
            "1949",
            "C-SIGN-001",
            "n.º8",
            "Fail-closed",
            "Homologação",
            "SAF-T",
            "Regime jurídico",
            "JWS",
            "alteração destrutiva",
        ):
            if token not in section:
                fail(f"AO-DOC-002 deve citar `{token}`", errors)

    # AO-SEQ-001 evidence package
    m1 = re.search(
        r"###\s+AO-SEQ-001\b.*?(?=\n###\s|\n##\s+Ainda|\n##\s+Verificador|$)",
        text,
        flags=re.S,
    )
    if not m1:
        fail("secção AO-SEQ-001 ausente ou mal delimitada", errors)
    else:
        s1 = m1.group(0)
        if "`confirmed_normative`" not in s1:
            fail("AO-SEQ-001: estado deve ser `confirmed_normative`", errors)
        for token in (
            "AO-LEG-DP-71-25-2025",
            "11904",
            "11908",
            "Art.10",
            "sequencial",
            "AO-LEG-DE-74-19-2019",
            "1577",
            "contínua",
            "univoc",
            "AO-SEQ-002",
            "concorrência",
            "Fail-closed",
            "Homologação",
            "Regime jurídico",
            "SAF-T",
        ):
            if token not in s1:
                fail(f"AO-SEQ-001 deve citar `{token}`", errors)

    # Must not claim other IDs confirmed without sections
    for rid in ("AO-SEQ-002", "AO-CRYPTO-001", "AO-KEY-001", "AO-DOC-001", "AO-AGT-001"):
        if re.search(rf"(?i)###\s+{rid}\b", text):
            fail(f"{rid}: não promover nesta matriz sem PR dedicado de evidência", errors)

    return errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path, default=None)
    args = parser.parse_args(argv)
    root = (args.repo_root or Path(__file__).resolve().parents[2]).resolve()
    errors = verify(root)
    if errors:
        for e in errors:
            print(f"FAIL: {e}", file=sys.stderr)
        return 1
    print(f"OK: confirmed matrix ({MATRIX_REL.as_posix()})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
