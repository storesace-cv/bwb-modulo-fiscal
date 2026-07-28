#!/usr/bin/env python3
"""Fail-closed checks for the provisional FE services / FE-RNG matrix.

Does not claim AO-AGT-* confirmed. Stdlib only. Does not invent FE-RNG codes.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

MATRIX_REL = Path("compliance/derived/requirements/FE-SERVICES-MATRIX-RM-REQ-001.md")

REQUIRED_TOKENS = [
    "não** confirma",
    "pending_validation",
    "AO-FE-SNAP-HML-2026-07-25-REGISTAR",
    "AO-FE-SNAP-HML-2026-07-25-SOLICITAR",
    "AO-FE-SNAP-HML-2026-07-25-LISTAR",
    "AO-FE-SNAP-HML-2026-07-25-LISTAR-FATURAS",
    "AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA",
    "AO-FE-SNAP-HML-2026-07-25-VALIDAR",
    "AO-FE-SNAP-HML-2026-07-25-GESTAO",
    "AO-FE-SNAP-HML-2026-07-25-QRCODE",
    "AO-FE-SNAP-HML-2026-07-25-API",
    "AO-FE-SNAP-HML-2026-07-25-MODELO",
    "eb430954",
    "f8fb22e7",
    "5729f02c",
    "c748caca",
    "6d5cc1a0",
    "7ab70629",
    "de423e66",
    "ccade20b",
    "06a9dbdf",
    "f851f512",
    "registarFactura",
    "solicitarSerie",
    "listarSeries",
    "listarFacturas",
    "obterEstado",
    "consultarFactura",
    "validarDocumento",
    "requestID",
    "C-FE-001",
    "GAP-006",
    "GAP-013",
    "FE-RNG-002",
    "FE-RNG-051",
    "FE-RNG-080",
    "FE-RNG-010",
    "FE-RNG-033",
    "FE-RNG-034",
    "FE-RNG-077",
    "sifphml.minfin.gov.ao",
    "sifp.minfin.gov.ao",
]

# Codes that must appear as inventory rows (extracted from REGISTAR/SOLICITAR/VALIDAR).
REQUIRED_RNG = [
    "FE-RNG-002",
    "FE-RNG-010",
    "FE-RNG-029",
    "FE-RNG-033",
    "FE-RNG-034",
    "FE-RNG-051",
    "FE-RNG-053",
    "FE-RNG-059",
    "FE-RNG-077",
    "FE-RNG-080",
    "FE-RNG-081",
]


def verify(root: Path) -> list[str]:
    errors: list[str] = []
    path = root / MATRIX_REL
    if not path.is_file():
        return [f"ausente: {MATRIX_REL.as_posix()}"]
    text = path.read_text(encoding="utf-8")

    if "não** confirma" not in text and "não confirma" not in text:
        errors.append("matriz FE deve declarar que não confirma AO-AGT-*")

    for token in REQUIRED_TOKENS:
        if token not in text:
            errors.append(f"token obrigatório ausente: {token}")

    for code in REQUIRED_RNG:
        if not re.search(rf"\b{re.escape(code)}\b", text):
            errors.append(f"código FE-RNG ausente: {code}")

    # Must not invent a standalone confirmed description for FE-RNG-001.
    for line in text.splitlines():
        if re.search(r"FE-RNG-001\s*\|\s*E0", line) and "não" not in line.lower():
            errors.append("FE-RNG-001 não deve ter E-code inventado na linha")
            break

    for line in text.splitlines():
        if not re.search(r"(?i)AO-AGT-\d+.*confirmad|FE-RNG.*confirmad", line):
            continue
        lowered = line.lower()
        if "não" in lowered or "nao" in lowered or "nenhum" in lowered:
            continue
        errors.append(f"afirmação indevida de confirmação: {line.strip()[:120]}")
        break

    if "/fe/ws/v1" not in text:
        errors.append("matriz deve registar path HML /fe/ws/v1 (C-FE-001)")

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
    print(f"OK: FE services matrix ({MATRIX_REL.as_posix()})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
