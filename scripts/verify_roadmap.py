#!/usr/bin/env python3
"""Structural verifier for the canonical ROADMAP.md (stdlib only, no network)."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path
from urllib.parse import urlparse, unquote

ALLOWED_STATES = frozenset(
    {"CONCLUÍDO", "PENDENTE", "EM_CURSO", "BLOQUEADO", "ADIADO"}
)
FORBIDDEN_PLACEHOLDERS = (
    "links deste PR",
    "[decisão pending]",
    "[decisão GitHub pending]",
    "PR #N",
    "squash SHA",
)
CANONICAL_NAME = "ROADMAP.md"
POINTER_REL = Path("docs/06-delivery/implementation-roadmap.md")
REQUIRED_HEADER = (
    "Check",
    "ID",
    "Entrega",
    "Estado",
    "Evidência",
    "Dependências / gate",
    "Done",
)
MD_LINK_RE = re.compile(r"\[([^\]]*)\]\(([^)]+)\)")
HTML_ANCHOR_RE = re.compile(
    r"<a\s+[^>]*\bid\s*=\s*[\"']([^\"']+)[\"'][^>]*>", re.IGNORECASE
)
HEADING_RE = re.compile(r"^(#{1,6})\s+(.+?)\s*$", re.MULTILINE)
ITEM_ID_RE = re.compile(r"^RM-[A-Z0-9]+(?:-[A-Z0-9]+)*$")
TABLE_ROW_RE = re.compile(r"^\|(.+)\|$")


class Failure(Exception):
    def __init__(self, path: str, item_id: str, cause: str) -> None:
        super().__init__(f"{path} | {item_id} | {cause}")
        self.path = path
        self.item_id = item_id
        self.cause = cause


def fail(path: str, item_id: str, cause: str) -> None:
    raise Failure(path, item_id, cause)


def github_slug(text: str) -> str:
    text = text.strip().lower()
    text = re.sub(r"[^\w\s\-]", "", text, flags=re.UNICODE)
    text = re.sub(r"\s+", "-", text)
    return text


def collect_anchors(text: str) -> set[str]:
    anchors: set[str] = set()
    for match in HTML_ANCHOR_RE.finditer(text):
        anchors.add(match.group(1))
    for match in HEADING_RE.finditer(text):
        raw = match.group(2)
        raw = re.sub(r"`([^`]+)`", r"\1", raw)
        raw = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", raw)
        anchors.add(github_slug(raw))
    return anchors


def split_row(line: str) -> list[str]:
    inner = line.strip()
    if not (inner.startswith("|") and inner.endswith("|")):
        return []
    parts = [p.strip() for p in inner[1:-1].split("|")]
    return parts


def is_separator(cells: list[str]) -> bool:
    if not cells:
        return False
    return all(re.fullmatch(r":?-{3,}:?", c.replace(" ", "")) for c in cells)


def parse_items(text: str, path: str) -> list[dict[str, str]]:
    items: list[dict[str, str]] = []
    lines = text.splitlines()
    i = 0
    while i < len(lines):
        cells = split_row(lines[i])
        if len(cells) == 7 and tuple(cells) == REQUIRED_HEADER:
            i += 1
            if i >= len(lines):
                break
            sep = split_row(lines[i])
            if not is_separator(sep):
                fail(path, "-", "tabela RM-* sem linha separadora")
            i += 1
            while i < len(lines):
                row = split_row(lines[i])
                if len(row) != 7:
                    break
                if is_separator(row):
                    i += 1
                    continue
                if row[0] in ("Check", "---"):
                    break
                items.append(
                    {
                        "check": row[0],
                        "id": row[1],
                        "entrega": row[2],
                        "estado": row[3],
                        "evidencia": row[4],
                        "gate": row[5],
                        "done": row[6],
                    }
                )
                i += 1
            continue
        i += 1
    return items


def extract_links(cell: str) -> list[str]:
    return [m.group(2).strip() for m in MD_LINK_RE.finditer(cell)]


def is_https_url(target: str) -> bool:
    parsed = urlparse(target)
    return parsed.scheme in ("http", "https") and bool(parsed.netloc)


def is_internal_fragment(target: str) -> bool:
    return target.startswith("#") and len(target) > 1


def resolve_relative(repo_root: Path, from_file: Path, target: str) -> Path:
    clean = unquote(target.split("#", 1)[0])
    if not clean:
        return from_file
    base = from_file.parent
    return (base / clean).resolve()


def validate_local_deps(text: str, path: str) -> None:
    for match in MD_LINK_RE.finditer(text):
        target = match.group(2).strip()
        if is_https_url(target) or is_internal_fragment(target):
            continue
        path_part = unquote(target.split("#", 1)[0]).lstrip("./")
        if path_part.startswith("local/") or path_part == "local":
            fail(path, "-", f"link Markdown para local/ proibido: {target}")
    # Build/test style dependencies (not explanatory prose).
    for pattern, label in (
        (r"(?i)(?:^|\s)(?:source|include|require|import)\s+[\"']local/", "include/import local/"),
        (r"(?i)(?:^|\s)(?:cd|cat|cp|mv|ln)\s+local/", "comando com local/"),
        (r"(?i)FISCAL_[A-Z0-9_]*=(?:\./)?local/", "env apontando a local/"),
    ):
        if re.search(pattern, text):
            fail(path, "-", f"dependência de build/test em local/ ({label})")


def validate_pointer(repo_root: Path) -> None:
    pointer = repo_root / POINTER_REL
    rel = str(POINTER_REL)
    if not pointer.is_file():
        fail(rel, "-", "apontador implementation-roadmap.md ausente")
    text = pointer.read_text(encoding="utf-8")
    if "ROADMAP.md" not in text:
        fail(rel, "-", "apontador não referencia ROADMAP.md")
    if not re.search(r"\[ROADMAP\.md\]\(\.\./\.\./ROADMAP\.md\)", text):
        fail(rel, "-", "apontador deve usar [ROADMAP.md](../../ROADMAP.md)")
    # Must not still look like a second detailed roadmap of phases.
    if re.search(r"^## Fase 0", text, re.MULTILINE):
        fail(rel, "-", "apontador ainda contém roadmap de fases (não é só pointer)")


def validate_second_canonical(repo_root: Path) -> None:
    for path in repo_root.rglob("*"):
        if not path.is_file():
            continue
        if "node_modules" in path.parts or ".git" in path.parts:
            continue
        if path.suffix.lower() != ".md":
            continue
        name = path.name
        if "roadmap" not in name.lower():
            continue
        rel = str(path.relative_to(repo_root))
        if path.name == CANONICAL_NAME and path.parent == repo_root:
            continue
        if path.resolve() == (repo_root / POINTER_REL).resolve():
            continue
        text = path.read_text(encoding="utf-8")
        if re.search(r"(?i)roadmap can[oó]nico", text) and re.search(
            r"(?i)fonte can[oó]nica de estado", text
        ):
            fail(rel, "-", "segundo ficheiro roadmap autodeclarado canónico")


def validate_item(
    item: dict[str, str],
    path: str,
    repo_root: Path,
    roadmap_path: Path,
    anchors: set[str],
    seen_ids: set[str],
) -> None:
    item_id = item["id"]
    if not ITEM_ID_RE.fullmatch(item_id):
        fail(path, item_id, "ID RM-* inválido")
    if item_id in seen_ids:
        fail(path, item_id, "ID duplicado")
    seen_ids.add(item_id)

    check = item["check"].strip()
    estado = item["estado"].strip()
    if estado not in ALLOWED_STATES:
        fail(path, item_id, f"estado não permitido: {estado}")
    if check not in ("[x]", "[ ]"):
        fail(path, item_id, f"checkbox inválido: {check}")
    if check == "[x]" and estado != "CONCLUÍDO":
        fail(path, item_id, "[x] exige Estado=CONCLUÍDO")
    if check == "[ ]" and estado == "CONCLUÍDO":
        fail(path, item_id, "CONCLUÍDO exige checkbox [x]")
    if not item["done"].strip() or item["done"].strip() in ("—", "-", "TODO"):
        fail(path, item_id, "Done vazio ou placeholder")
    if not item["entrega"].strip():
        fail(path, item_id, "Entrega vazia")

    low_ev = item["evidencia"].lower()
    low_gate = item["gate"].lower()
    for ph in FORBIDDEN_PLACEHOLDERS:
        if ph.lower() in low_ev or ph.lower() in low_gate:
            fail(path, item_id, f"placeholder proibido: {ph}")

    links_ev = extract_links(item["evidencia"])
    links_gate = extract_links(item["gate"])
    all_links = links_ev + links_gate

    if estado == "CONCLUÍDO":
        if not item["evidencia"].strip() or item["evidencia"].strip() in ("—", "-"):
            fail(path, item_id, "CONCLUÍDO exige Evidência não vazia")
        if not links_ev:
            fail(path, item_id, "CONCLUÍDO exige pelo menos um link na Evidência")

    if estado == "BLOQUEADO":
        gate = item["gate"].strip()
        if not gate or gate in ("—", "-"):
            fail(path, item_id, "BLOQUEADO exige Dependências/gate não vazio")
        if not (links_ev or links_gate):
            fail(
                path,
                item_id,
                "BLOQUEADO exige link para DEC/GAP/INC/auditoria/secção interna",
            )

    if estado == "ADIADO":
        blob = f"{item['gate']} {item['evidencia']} {item['done']} {item['entrega']}"
        if not re.search(r"\bM\d+\b", blob):
            fail(path, item_id, "ADIADO exige marco futuro (ex. M12)")

    for target in all_links:
        if any(ph.lower() in target.lower() for ph in FORBIDDEN_PLACEHOLDERS):
            fail(path, item_id, f"placeholder em link: {target}")
        if is_https_url(target):
            parsed = urlparse(target)
            if parsed.scheme not in ("http", "https") or not parsed.netloc:
                fail(path, item_id, f"URL HTTPS inválida sintaticamente: {target}")
            continue
        if is_internal_fragment(target):
            frag = target[1:]
            if frag not in anchors:
                fail(path, item_id, f"fragmento interno inexistente: {target}")
            continue
        # Relative file link (optional fragment).
        file_part, _, frag = target.partition("#")
        resolved = resolve_relative(repo_root, roadmap_path, file_part or ".")
        try:
            resolved.relative_to(repo_root.resolve())
        except ValueError:
            fail(path, item_id, f"link relativo fora do repositório: {target}")
        if file_part and not resolved.is_file():
            fail(path, item_id, f"link relativo quebrado: {target}")
        if frag:
            target_text = resolved.read_text(encoding="utf-8")
            if frag not in collect_anchors(target_text):
                fail(path, item_id, f"fragmento inexistente em {target}")


def verify(repo_root: Path, roadmap_path: Path) -> None:
    rel = str(roadmap_path.relative_to(repo_root)) if roadmap_path.is_relative_to(repo_root) else str(roadmap_path)
    if not roadmap_path.is_file():
        fail(rel, "-", "ROADMAP.md ausente")
    if roadmap_path.name != CANONICAL_NAME:
        fail(rel, "-", "ficheiro canónico deve chamar-se ROADMAP.md")

    text = roadmap_path.read_text(encoding="utf-8")
    for ph in FORBIDDEN_PLACEHOLDERS:
        # Allow documenting the rule itself in maintenance section only if quoted carefully;
        # still fail if used as evidence placeholders in tables (checked per-item).
        pass

    validate_local_deps(text, rel)
    anchors = collect_anchors(text)
    items = parse_items(text, rel)
    if not items:
        fail(rel, "-", "nenhum item RM-* em tabela canónica encontrado")

    seen: set[str] = set()
    for item in items:
        validate_item(item, rel, repo_root, roadmap_path, anchors, seen)

    if "rm-gov-002-ruleset-de-main" not in anchors:
        fail(rel, "RM-GOV-002", "âncora rm-gov-002-ruleset-de-main ausente")

    validate_pointer(repo_root)
    validate_second_canonical(repo_root)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--repo-root",
        type=Path,
        default=None,
        help="Raiz do repositório (default: pai de scripts/)",
    )
    parser.add_argument(
        "--roadmap",
        type=Path,
        default=None,
        help="Caminho para ROADMAP.md (default: <repo-root>/ROADMAP.md)",
    )
    args = parser.parse_args(argv)
    script_dir = Path(__file__).resolve().parent
    repo_root = (args.repo_root or script_dir.parent).resolve()
    roadmap = (args.roadmap or (repo_root / CANONICAL_NAME)).resolve()
    try:
        verify(repo_root, roadmap)
    except Failure as exc:
        print(str(exc), file=sys.stderr)
        return 1
    print(f"OK: {roadmap.relative_to(repo_root)} ({repo_root})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
