#!/usr/bin/env python3
"""Fail-closed checks for the RM-REQ-001 provisional AO-* matrix.

Stdlib only. Does not claim requirements are confirmed.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

MATRIX_REL = Path("compliance/derived/requirements/PROVISIONAL-MATRIX-RM-REQ-001.md")

REQUIRED_IDS = [
    "ASM-REG-001",
    "AO-ID-001",
    "AO-DOC-001",
    "AO-DOC-002",
    "AO-SEQ-001",
    "AO-SEQ-002",
    "AO-IDEM-001",
    "AO-TAX-001",
    "AO-CRYPTO-001",
    "AO-KEY-001",
    "AO-AGT-001",
    "AO-AGT-002",
    "AO-OFF-001",
    "AO-OFF-002",
    "AO-AUD-001",
    "AO-SAF-001",
    "AO-SAF-002",
    "AO-OPS-001",
    "AO-UPD-001",
]

# Must stay blocked while Rect. 10/19 original is incorrect / absent.
RECT_DEPENDENT_BLOCKED = [
    "AO-DOC-001",
    "AO-DOC-002",
    "AO-SEQ-001",
    "AO-TAX-001",
]

ALLOWED_STATUSES = {"scaffold", "partial", "blocked", "pending_validation"}
FORBIDDEN_STATUS_WORDS = re.compile(
    r"(?i)\b(confirmed|confirmado|validated_agt|aprovado_agt)\b"
)
ROW_RE = re.compile(
    r"^\|\s*(ASM-REG-001|AO-[A-Z]+-\d+)\s*\|\s*`([^`]+)`\s*\|",
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

    if "não** é matriz AO-* confirmada" not in text and "não é matriz AO-* confirmada" not in text:
        fail("matriz deve declarar explicitamente que não é confirmada", errors)
    if "AO-LEG-RECT-10-19-2019" not in text:
        fail("matriz deve mencionar AO-LEG-RECT-10-19-2019 como excluída", errors)
    if "GAP-002" not in text:
        fail("matriz deve referir GAP-002", errors)
    if "RM-SRC-004" not in text or "BLOQUEADO" not in text:
        fail("matriz deve manter RM-SRC-004 BLOQUEADO", errors)
    if "19164" not in text or "19227" not in text:
        fail("matriz deve limitar DE 683/25 a 19164–19227", errors)
    if "19228" not in text and "Aviso" not in text:
        fail("matriz deve alertar p.66 / Aviso 4/25", errors)

    rows = {m.group(1): m.group(2).strip() for m in ROW_RE.finditer(text)}
    missing = [i for i in REQUIRED_IDS if i not in rows]
    if missing:
        fail(f"IDs em falta na tabela: {', '.join(missing)}", errors)

    for rid, status in rows.items():
        if status not in ALLOWED_STATUSES:
            fail(f"{rid}: estado inválido `{status}`", errors)
        if FORBIDDEN_STATUS_WORDS.search(status):
            fail(f"{rid}: estado proibido `{status}`", errors)

    # Body must not claim confirmed requirements.
    if re.search(r"(?i)requisitos?\s+AO-\*\s+confirmad", text) and "não** é matriz" not in text:
        # Allow negations; flag affirmative claims outside the banner.
        for line in text.splitlines():
            if re.search(r"(?i)AO-\*.*confirmad", line) and "não" not in line.lower():
                fail(f"afirmação indevida de confirmação: {line.strip()[:120]}", errors)
                break

    for rid in RECT_DEPENDENT_BLOCKED:
        st = rows.get(rid)
        if st != "blocked":
            fail(f"{rid}: deve estar `blocked` (dependência Rect. 10/19 / fontes incompletas)", errors)

    if rows.get("AO-SAF-001") != "pending_validation":
        fail("AO-SAF-001: deve estar `pending_validation`", errors)

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
    print(f"OK: provisional matrix ({MATRIX_REL.as_posix()})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
