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

# Still without closed validator/MVP mapping (tax fields cited → AO-TAX-001 partial).
CITATION_PENDING_SCAFFOLD = [
    "AO-DOC-001",
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
    if "AO-LEG-RECT-10-19-2019" not in text or "b3db14e2" not in text:
        fail("matriz deve admitir AO-LEG-RECT-10-19-2019 reviewed v2 (b3db14e2…)", errors)
    if "AO-LEG-DP-71-25-2025" not in text or "4931fd3c" not in text:
        fail("matriz deve admitir AO-LEG-DP-71-25-2025 reviewed (4931fd3c…)", errors)
    if "11902" not in text or "11920" not in text:
        fail("matriz deve limitar DP 71/25 a 11902–11920", errors)
    if "11921" not in text and "372/25" not in text:
        fail("matriz deve alertar p.21 / DE 372/25 fora do intervalo DP 71/25", errors)
    if "11908" not in text or "11909" not in text:
        fail("matriz deve citar DP 71/25 Art.10 @11908–11909", errors)
    if "4931fd3c" not in text:
        fail("matriz deve referir sha256 do original DP 71/25 (4931fd3c…)", errors)
    if "5b63c80e" not in text or "b3db14e2" not in text:
        fail("matriz deve referir sha256 DE 74/19 (5b63c80e…) e Rect. v2 (b3db14e2…)", errors)
    if "1576" not in text or "1582" not in text:
        fail("matriz deve citar DE 74/19 gazetas 1576+ e n.º34 @1582+", errors)
    if "1948" not in text or "1949" not in text:
        fail("matriz deve citar Rect. 10/19 gazeta 1948–1949", errors)
    if "C-SIGN-001" not in text:
        fail("matriz deve referir C-SIGN-001 (SAF-T RSA ≠ JWS FE)", errors)
    if "GAP-002" not in text:
        fail("matriz deve referir GAP-002", errors)
    if "RM-SRC-004" not in text:
        fail("matriz deve referir RM-SRC-004", errors)
    if "DOCUMENT-TYPES-MATRIX-RM-REQ-001" not in text:
        fail("matriz deve referir DOCUMENT-TYPES-MATRIX-RM-REQ-001 (inventário tipos)", errors)
    if "19164" not in text or "19227" not in text:
        fail("matriz deve limitar DE 683/25 a 19164–19227", errors)
    if "19228" not in text and "Aviso" not in text:
        fail("matriz deve alertar p.66 / Aviso 4/25", errors)
    if "Citação G" not in text and "### Citação G" not in text:
        fail("matriz deve incluir Citação G (DE 683/25 Anexos)", errors)
    for token in (
        "Anexo I",
        "Anexo II",
        "Anexo III",
        "19166",
        "19193",
        "19194",
        "19212",
        "19223",
        "solicitarSerie",
        "registarFactura",
        "taxType",
    ):
        if token not in text:
            fail(f"Citação G / DE 683 deve incluir `{token}`", errors)

    rows = {m.group(1): m.group(2).strip() for m in ROW_RE.finditer(text)}
    missing = [i for i in REQUIRED_IDS if i not in rows]
    if missing:
        fail(f"IDs em falta na tabela: {', '.join(missing)}", errors)

    for rid, status in rows.items():
        if status not in ALLOWED_STATUSES:
            fail(f"{rid}: estado inválido `{status}`", errors)
        if FORBIDDEN_STATUS_WORDS.search(status):
            fail(f"{rid}: estado proibido `{status}`", errors)

    # Body must not claim confirmed requirements (scan every line; banner negation
    # must not disable detection of later affirmative claims).
    for line in text.splitlines():
        if not re.search(r"(?i)AO-\*.*confirmad|requisitos?\s+AO-\*\s+confirmad", line):
            continue
        lowered = line.lower()
        if "não" in lowered or "nao" in lowered or "nenhum" in lowered:
            continue
        fail(f"afirmação indevida de confirmação: {line.strip()[:120]}", errors)
        break

    for rid in CITATION_PENDING_SCAFFOLD:
        st = rows.get(rid)
        if st != "scaffold":
            fail(
                f"{rid}: deve estar `scaffold` até citação página a página "
                "(fontes OCR reviewed ≠ AO-* confirmados)",
                errors,
            )

    if rows.get("AO-SAF-001") != "pending_validation":
        fail("AO-SAF-001: deve estar `pending_validation`", errors)
    for token in (
        "AO-SAFT-XSD-1.01_01",
        "e9a938e1",
        "AuditFile",
        "urn:OECD:StandardAuditFile-Tax:AO_1.01_01",
        "SAFTAO1.01_01.xsd",
    ):
        if token not in text:
            fail(f"citação AO-SAF-001 deve incluir `{token}`", errors)
    saf_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-SAF-001\s*\|", ln)]
    if saf_rows:
        sl = saf_rows[0].lower()
        if "`pending_validation`" not in saf_rows[0]:
            fail("AO-SAF-001: célula de estado deve ser `pending_validation`", errors)
        if re.search(r"(?i)\b(confirmed|confirmado|validated_agt)\b", sl) and "não" not in sl and "nao" not in sl:
            fail("AO-SAF-001: afirmação de confirmação sem negação na linha", errors)
        if "não" not in sl or "satisfeito" not in sl:
            fail("AO-SAF-001: a linha deve declarar que o critério não fica satisfeito", errors)

    # AO-SEQ-002: partial citation to DE 683 ART.4 / gazeta 19164 — never confirmed.
    if rows.get("AO-SEQ-002") != "partial":
        fail("AO-SEQ-002: deve estar `partial` (citação preliminar DE 683/25 ART.4)", errors)
    if "AO-LEG-DE-683-25-2025" not in text:
        fail("citação AO-SEQ-002 deve referir source_id AO-LEG-DE-683-25-2025", errors)
    if "b01e4581" not in text:
        fail("citação AO-SEQ-002 deve referir sha256 do original DE 683/25 (b01e4581…)", errors)
    if "ARTIGO 4" not in text and "ART. 4" not in text:
        fail("matriz deve citar ARTIGO 4.º / ART. 4 para AO-SEQ-002", errors)
    if "19164" not in text:
        fail("citação AO-SEQ-002 deve incluir gazeta 19164", errors)
    if "não** fica satisfeito" not in text and "não fica satisfeito" not in text:
        fail("matriz deve declarar que critérios de catálogo não ficam satisfeitos só com as citações parciais", errors)

    def check_partial_row(rid: str) -> None:
        rows_ln = [ln for ln in text.splitlines() if re.match(rf"^\|\s*{re.escape(rid)}\s*\|", ln)]
        if not rows_ln:
            fail(f"linha de tabela {rid} ausente", errors)
            return
        line = rows_ln[0]
        if "`partial`" not in line:
            fail(f"{rid}: célula de estado deve ser `partial`", errors)
        if re.search(r"(?i)\b(confirmed|confirmado|validated_agt)\b", line) and not re.search(
            r"(?i)não|nao", line
        ):
            fail(f"{rid}: afirmação de confirmação sem negação na linha", errors)
        if not re.search(r"(?i)não\*\*\s*confirmado|não\s+confirmado|\*\*não\*\*\s+confirmado", line):
            fail(f"{rid}: linha deve declarar explicitamente que não está confirmado", errors)

    check_partial_row("AO-SEQ-002")

    # AO-ID-001: partial citation — contribuinte/software fields; never full catalog criterion.
    if rows.get("AO-ID-001") != "partial":
        fail("AO-ID-001: deve estar `partial` (citação preliminar DE 683/25 campos FE)", errors)
    for token in ("taxRegistrationNumber", "softwareValidationNumber", "productVersion", "19166", "19167"):
        if token not in text:
            fail(f"citação AO-ID-001 deve incluir `{token}`", errors)
    check_partial_row("AO-ID-001")
    id_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-ID-001\s*\|", ln)]
    if id_rows:
        id_line = id_rows[0].lower()
        if "estabelecimento" not in id_line or "terminal" not in id_line:
            fail("AO-ID-001: a linha da tabela deve declarar lacuna estabelecimento/terminal", errors)
        if "não" not in id_line or "satisfeito" not in id_line:
            fail("AO-ID-001: a linha da tabela deve declarar que o critério não fica satisfeito", errors)

    # AO-CRYPTO-001: partial — JWS document signature + RS256 FE; never confirmed; ≠ SAF-T.
    if rows.get("AO-CRYPTO-001") != "partial":
        fail("AO-CRYPTO-001: deve estar `partial` (jwsDocumentSignature + RS256 FE)", errors)
    for token in (
        "jwsDocumentSignature",
        "19168",
        "RS256",
        "AO-FE-SNAP-HML-2026-07-25-ESTRUTURA",
        "pending_validation",
    ):
        if token not in text:
            fail(f"citação AO-CRYPTO-001 deve incluir `{token}`", errors)
    if "SAF-T" not in text and "SAFT" not in text:
        fail("AO-CRYPTO-001 deve distinguir JWS FE de mecanismos SAF-T", errors)
    check_partial_row("AO-CRYPTO-001")
    crypto_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-CRYPTO-001\s*\|", ln)]
    if crypto_rows:
        cl = crypto_rows[0].lower()
        if "encadeamento" not in cl and "encadear" not in cl:
            fail("AO-CRYPTO-001: a linha deve mencionar encadeamento (lacuna/não citado)", errors)
        if "não" not in cl or "satisfeito" not in cl:
            fail("AO-CRYPTO-001: a linha deve declarar que o critério não fica satisfeito", errors)

    # AO-SEQ-001: partial — DP 71 Art.10 b numbering by type/series.
    if rows.get("AO-SEQ-001") != "partial":
        fail("AO-SEQ-001: deve estar `partial` (DP 71/25 Art.10 n.º1 b)", errors)
    for token in ("Art.10", "11908"):
        if token not in text:
            fail(f"citação AO-SEQ-001 deve incluir `{token}`", errors)
    check_partial_row("AO-SEQ-001")
    seq1_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-SEQ-001\s*\|", ln)]
    if seq1_rows:
        s1 = seq1_rows[0].lower()
        if "não" not in s1 or "satisfeito" not in s1:
            fail("AO-SEQ-001: a linha deve declarar que o critério não fica satisfeito", errors)

    # AO-DOC-002: partial — no deletion after issue + NC rectification path.
    if rows.get("AO-DOC-002") != "partial":
        fail("AO-DOC-002: deve estar `partial` (DP 71/25 Art.3 n / Art.8)", errors)
    if "11904" not in text or "11907" not in text:
        fail("citação AO-DOC-002 deve incluir gazetas 11904 e 11907", errors)
    if "elimina" not in text.lower():
        fail("citação AO-DOC-002 deve mencionar eliminação pós-emissão", errors)
    check_partial_row("AO-DOC-002")
    doc2_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-DOC-002\s*\|", ln)]
    if doc2_rows:
        d2 = doc2_rows[0].lower()
        if "não" not in d2 or "satisfeito" not in d2:
            fail("AO-DOC-002: a linha deve declarar que o critério não fica satisfeito", errors)

    # AO-OFF-001: partial — DP 71 Art.18 contingency.
    if rows.get("AO-OFF-001") != "partial":
        fail("AO-OFF-001: deve estar `partial` (DP 71/25 Art.18)", errors)
    for token in ("Art.18", "11911", "11912"):
        if token not in text:
            fail(f"citação AO-OFF-001 deve incluir `{token}`", errors)
    check_partial_row("AO-OFF-001")
    off_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-OFF-001\s*\|", ln)]
    if off_rows:
        ol = off_rows[0].lower()
        if "não" not in ol or "satisfeito" not in ol:
            fail("AO-OFF-001: a linha deve declarar que o critério não fica satisfeito", errors)

    # AO-OFF-002: partial — DE 74 integration / recovery series.
    if rows.get("AO-OFF-002") != "partial":
        fail("AO-OFF-002: deve estar `partial` (DE 74/19 Anexo I n.º7–9)", errors)
    check_partial_row("AO-OFF-002")
    off2_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-OFF-002\s*\|", ln)]
    if off2_rows:
        o2 = off2_rows[0]
        o2l = o2.lower()
        if "1579" not in o2 and "1580" not in o2:
            fail("AO-OFF-002: a linha da tabela deve citar gazeta 1579 ou 1580", errors)
        if "não" not in o2l or "satisfeito" not in o2l:
            fail("AO-OFF-002: a linha deve declarar que o critério não fica satisfeito", errors)
    else:
        fail("linha de tabela AO-OFF-002 ausente", errors)

    # DOC-002 / SEQ-001 rows must cite DE 74 gazeta 1577 (not only Citação F).
    for rid in ("AO-DOC-002", "AO-SEQ-001"):
        row_ln = [ln for ln in text.splitlines() if re.match(rf"^\|\s*{rid}\s*\|", ln)]
        if not row_ln:
            fail(f"linha de tabela {rid} ausente", errors)
            continue
        if "1577" not in row_ln[0]:
            fail(f"{rid}: a linha da tabela deve citar DE 74/19 @1577", errors)

    # AO-TAX-001: partial — FE tax fields + Tabelas 2–6; never confirmed / no full calc.
    if rows.get("AO-TAX-001") != "partial":
        fail("AO-TAX-001: deve estar `partial` (DE 683 taxType + Tabelas 2–6)", errors)
    check_partial_row("AO-TAX-001")
    for token in ("taxType", "19171", "19212", "19227", "Tabela"):
        if token not in text:
            fail(f"citação AO-TAX-001 deve incluir `{token}`", errors)
    tax_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-TAX-001\s*\|", ln)]
    if tax_rows:
        tl = tax_rows[0].lower()
        if "não" not in tl or "satisfeito" not in tl:
            fail("AO-TAX-001: a linha deve declarar que o critério não fica satisfeito", errors)
        if "19212" not in tax_rows[0] and "19227" not in tax_rows[0] and "19171" not in tax_rows[0]:
            fail("AO-TAX-001: a linha da tabela deve citar gazeta FE impostos/tabelas", errors)
    else:
        fail("linha de tabela AO-TAX-001 ausente", errors)

    # AO-SEQ-002 row should cite solicitarSerie / series service pages when expanded.
    seq2_rows = [ln for ln in text.splitlines() if re.match(r"^\|\s*AO-SEQ-002\s*\|", ln)]
    if seq2_rows:
        if "19183" not in seq2_rows[0] and "solicitarSerie" not in seq2_rows[0]:
            fail("AO-SEQ-002: a linha deve citar solicitarSerie ou gazeta 19183", errors)

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
