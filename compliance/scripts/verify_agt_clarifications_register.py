#!/usr/bin/env python3
"""Structural checks for docs/01-compliance/agt-clarifications-register.md.

Stdlib only. Does not invent AGT answers. Fail-closed on secrets-like tokens.
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

REGISTER_REL = Path("docs/01-compliance/agt-clarifications-register.md")

ALLOWED_STATES = frozenset({"OPEN", "ANSWERED", "MITIGATED", "NOT_APPLICABLE"})
ALLOWED_CATEGORIES = frozenset(
    {
        "missing_credential_or_info",
        "documentation_conflict",
        "normative_or_process_doubt",
        "network_or_portal_incident",
    }
)

FORBIDDEN_PATTERNS = [
    re.compile(r"BEGIN (RSA )?PRIVATE KEY"),
    re.compile(r"-----BEGIN"),
    re.compile(r"(?i)password\s*[:=]\s*\S+"),
    re.compile(r"(?i)basic\s+[A-Za-z0-9+/=]{16,}"),
    re.compile(r"fingerprint\s*[:=]", re.I),
]

REQUIRED_CONFLICT_LINKS = [
    "C-FE-001-fe-endpoint-path-inconsistency.md",
    "C-FE-JWS-TYP-001-typ-jwt-vs-jose.md",
    "C-FE-JWS-DOC-001-document-totals-sample.md",
    "C-FE-JWS-REQ-001-solicitar-serie-fields.md",
    "C-FE-JWS-REQ-002-listar-series-fields.md",
    "C-FE-JWS-REQ-003-validar-documento-fields.md",
    "C-FE-JWS-REQ-004-listar-facturas-payload-block.md",
]

ID_RE = re.compile(r"^## (AGT-Q-\d{3})\s*$", re.M)
FIELD_RE = re.compile(
    r"^\| (ID|Estado|Categoria|Pronto para email) \| ([^|]+) \|$",
    re.M,
)
QUESTION_RE = re.compile(
    r"\*\*Pergunta exacta a enviar à AGT:\*\*\s*(.+?)(?=\n\n\*\*Resposta|\n---)",
    re.S,
)


def verify(root: Path) -> list[str]:
    errors: list[str] = []
    path = root / REGISTER_REL
    if not path.is_file():
        return [f"ausente: {REGISTER_REL.as_posix()}"]
    text = path.read_text(encoding="utf-8")

    if "Sem chamadas reais" not in text:
        errors.append("registo deve declarar explicitamente «Sem chamadas reais»")

    for pat in FORBIDDEN_PATTERNS:
        if pat.search(text):
            errors.append(f"padrão proibido (segredo/PEM): {pat.pattern}")

    ids = ID_RE.findall(text)
    if len(ids) != len(set(ids)):
        errors.append("IDs AGT-Q duplicados")
    if not ids:
        errors.append("nenhum item AGT-Q-* encontrado")

    # Parse per-item blocks between ## AGT-Q-NNN headings (exclude index).
    parts = re.split(r"(?m)^## (AGT-Q-\d{3})\s*$", text)
    # parts: [preamble, id1, body1, id2, body2, ...]
    items: dict[str, str] = {}
    for i in range(1, len(parts), 2):
        items[parts[i]] = parts[i + 1]

    for qid, body in items.items():
        fields = {m.group(1): m.group(2).strip() for m in FIELD_RE.finditer(body)}
        if fields.get("ID") != qid:
            errors.append(f"{qid}: campo ID em falta ou divergente")
        state = fields.get("Estado")
        if state not in ALLOWED_STATES:
            errors.append(f"{qid}: estado inválido {state!r}")
        cat = fields.get("Categoria")
        if cat not in ALLOWED_CATEGORIES:
            errors.append(f"{qid}: categoria inválida {cat!r}")
        ready = fields.get("Pronto para email")
        if ready not in {"yes", "no"}:
            errors.append(f"{qid}: Pronto para email deve ser yes|no")
        if state == "OPEN":
            qm = QUESTION_RE.search(body)
            if not qm or len(qm.group(1).strip()) < 20:
                errors.append(f"{qid}: OPEN exige pergunta exacta não vazia")

    for rel in REQUIRED_CONFLICT_LINKS:
        if rel not in text:
            errors.append(f"link de conflito ausente: {rel}")
        target = root / "compliance" / "derived" / "conflicts" / rel
        if not target.is_file():
            errors.append(f"ficheiro de conflito inexistente: {rel}")

    # Relative doc links that must resolve.
    for rel in (
        "docs/01-compliance/agt-dependencies.md",
        "docs/01-compliance/regulatory-gaps.md",
        "docs/01-compliance/official-access-plan.md",
        "compliance/derived/conflicts/README.md",
        "ROADMAP.md",
    ):
        if not (root / rel).is_file():
            errors.append(f"ficheiro relacionado ausente: {rel}")

    if "## Perguntas abertas prontas para comunicação à AGT" not in text:
        errors.append("secção de perguntas abertas ausente")

    return errors


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", type=Path, default=Path("."))
    args = ap.parse_args()
    root = args.root.resolve()
    errs = verify(root)
    if errs:
        for e in errs:
            print(f"FAIL: {e}", file=sys.stderr)
        return 1
    print("OK: agt-clarifications-register")
    return 0


if __name__ == "__main__":
    sys.exit(main())
