#!/usr/bin/env python3
"""Fail-closed checks for DOCUMENT-CATALOG-RM-REQ-001 (DEC-PROD-015 schema).

Does not claim AO-DOC-* confirmed. Stdlib only.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

CATALOG_REL = Path("compliance/derived/requirements/DOCUMENT-CATALOG-RM-REQ-001.md")

REQUIRED_HEADER_FIELDS = [
    "grupo",
    "codigo_canonico",
    "designacao",
    "codigos_canal",
    "estrutura_saft",
    "elegibilidade",
    "natureza_juridica",
    "restricao_sectorial",
    "serie_necessaria",
    "requisitos",
    "regras_rectificacao_anulacao",
    "estado_normativo",
    "activo",
]

REQUIRED_TOKENS = [
    "DEC-PROD-015",
    "não** confirma",
    "bwb.ao.",
    "SalesInvoices",
    "Payments",
    "MovementOfGoods",
    "WorkingDocuments",
    "PurchaseInvoices",
    "pending_dec_reg_003",
    "C-DOC-003",
]


def verify(root: Path) -> list[str]:
    errors: list[str] = []
    path = root / CATALOG_REL
    if not path.is_file():
        return [f"ausente: {CATALOG_REL.as_posix()}"]
    text = path.read_text(encoding="utf-8")

    if "não** confirma" not in text and "não confirma" not in text:
        errors.append("catálogo deve declarar que não confirma AO-DOC-*")

    for token in REQUIRED_TOKENS:
        if token not in text:
            errors.append(f"token obrigatório ausente: {token}")

    # Header row of the seed table must list all schema fields (order flexible).
    header_line = ""
    for line in text.splitlines():
        if line.startswith("| grupo |") or line.startswith("|grupo|"):
            header_line = line
            break
        if re.match(r"^\|\s*grupo\s*\|", line):
            header_line = line
            break
    if not header_line:
        errors.append("tabela seed: cabeçalho com coluna `grupo` ausente")
    else:
        lowered = header_line.lower()
        for field in REQUIRED_HEADER_FIELDS:
            if field not in lowered:
                errors.append(f"cabeçalho seed sem campo obrigatório: {field}")

    # At least one canonical id and FE-only FA row.
    if "bwb.ao.vendas.fa" not in text:
        errors.append("seed deve incluir bwb.ao.vendas.fa (FE-only)")
    if "bwb.ao.pagamentos.rc" not in text:
        errors.append("seed deve incluir bwb.ao.pagamentos.rc (Payments ≠ InvoiceType)")

    for bad in ("AO-DOC-001 confirmado", "catálogo confirmado", "validated_agt"):
        if bad.lower() in text.lower():
            # Allow negation on same line/document
            if "não" not in text.lower() and "nao" not in text.lower():
                errors.append(f"afirmação indevida: {bad}")

    for line in text.splitlines():
        if not re.search(r"(?i)AO-DOC-\d+.*confirmad|cat[aá]logo\s+confirmad", line):
            continue
        lowered = line.lower()
        if "não" in lowered or "nao" in lowered or "nenhum" in lowered:
            continue
        errors.append(f"afirmação indevida de confirmação: {line.strip()[:120]}")
        break

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
    print(f"OK: document catalog ({CATALOG_REL.as_posix()})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
