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
# Titles as listed in the executive structure (optional leading "N. " in the file).
REQUIRED_SECTION_TITLES = (
    "Visão executiva",
    "Estado atual",
    "O que já foi construído",
    "Caminho crítico para Angola",
    "Roadmap detalhado por área",
    "Fontes fiscais e SAF-T AO",
    "Motor fiscal Angola",
    "Faturação electrónica AGT",
    "Backoffice",
    "Edge/offline",
    "Integradores e software houses",
    "Operações de produção",
    "Certificação AGT",
    "Cabo Verde",
    "Bloqueios, decisões e incidentes",
    "Critérios de conclusão",
    "Evidências e documentos relacionados",
    "Regras de manutenção do roadmap",
)
MD_LINK_RE = re.compile(r"\[([^\]]*)\]\(([^)]+)\)")
HTML_ANCHOR_RE = re.compile(
    r"<a\s+[^>]*\bid\s*=\s*[\"']([^\"']+)[\"'][^>]*>", re.IGNORECASE
)
HEADING_RE = re.compile(r"^(#{1,6})\s+(.+?)\s*$", re.MULTILINE)
H2_RE = re.compile(r"^##\s+(.+?)\s*$", re.MULTILINE)
ITEM_ID_RE = re.compile(r"^RM-[A-Z0-9]+(?:-[A-Z0-9]+)*$")
RM_TOKEN_RE = re.compile(r"\bRM-[A-Z0-9]+(?:-[A-Z0-9]+)*\b")
SECTION_NUM_RE = re.compile(r"^\d+\.\s+")


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


def normalize_section_title(title: str) -> str:
    title = title.strip()
    title = re.sub(r"`([^`]+)`", r"\1", title)
    title = re.sub(r"\[([^\]]+)\]\([^)]+\)", r"\1", title)
    title = SECTION_NUM_RE.sub("", title)
    return title.strip()


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
    return [p.strip() for p in inner[1:-1].split("|")]


def is_separator(cells: list[str]) -> bool:
    if not cells:
        return False
    return all(re.fullmatch(r":?-{3,}:?", c.replace(" ", "")) for c in cells)


def is_table_line(line: str) -> bool:
    stripped = line.strip()
    return stripped.startswith("|") and stripped.endswith("|") and stripped.count("|") >= 2


def parse_items(text: str, path: str) -> tuple[list[dict[str, str]], set[int]]:
    """Parse canonical RM-* tables. Returns (items, consumed_line_indexes)."""
    items: list[dict[str, str]] = []
    consumed: set[int] = set()
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
                        "_line": str(i),
                    }
                )
                consumed.add(i)
                i += 1
            continue
        i += 1
    return items, consumed


def extract_links(cell: str) -> list[str]:
    return [m.group(2).strip() for m in MD_LINK_RE.finditer(cell)]


def link_kind(target: str) -> str:
    """Classify a Markdown link target without network I/O.

    Returns: https | fragment | relative | bad_external
    """
    if target.startswith("#"):
        return "fragment"
    parsed = urlparse(target)
    scheme = (parsed.scheme or "").lower()
    if scheme:
        if scheme == "https" and parsed.netloc:
            return "https"
        return "bad_external"
    return "relative"


def is_https_url(target: str) -> bool:
    """True only for syntactically valid https:// URLs (never http)."""
    return link_kind(target) == "https"


def resolve_relative(repo_root: Path, from_file: Path, target: str) -> Path:
    clean = unquote(target.split("#", 1)[0])
    if not clean:
        return from_file
    base = from_file.parent
    return (base / clean).resolve()


def validate_local_deps(text: str, path: str) -> None:
    for match in MD_LINK_RE.finditer(text):
        target = match.group(2).strip()
        kind = link_kind(target)
        if kind in ("https", "fragment", "bad_external"):
            continue
        path_part = unquote(target.split("#", 1)[0]).lstrip("./")
        if path_part.startswith("local/") or path_part == "local":
            fail(path, "-", f"link Markdown para local/ proibido: {target}")
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
        if "roadmap" not in path.name.lower():
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


def normalize_text(text: str) -> str:
    text = text.lower()
    text = (
        text.replace("á", "a")
        .replace("à", "a")
        .replace("ã", "a")
        .replace("â", "a")
        .replace("é", "e")
        .replace("ê", "e")
        .replace("í", "i")
        .replace("ó", "o")
        .replace("õ", "o")
        .replace("ô", "o")
        .replace("ú", "u")
        .replace("ç", "c")
    )
    text = re.sub(r"\s+", " ", text)
    return text


def validate_sections(text: str, path: str) -> None:
    found: list[str] = []
    for match in H2_RE.finditer(text):
        found.append(normalize_section_title(match.group(1)))
    expected = list(REQUIRED_SECTION_TITLES)
    if found != expected:
        # Missing / extra / wrong order
        if set(found) != set(expected):
            missing = [t for t in expected if t not in found]
            if missing:
                fail(path, "-", f"secção em falta: {missing[0]}")
            extra = [t for t in found if t not in expected]
            if extra:
                fail(path, "-", f"secção H2 inesperada: {extra[0]}")
        fail(path, "-", "ordem das 18 secções H2 incorrecta")


def validate_distinctions(text: str, path: str) -> None:
    norm = normalize_text(text)
    checks = [
        ("FISCAL_ENV=homologation", "FISCAL_ENV=homologation" in text or "fiscal_env=homologation" in norm),
        ("sandbox BWB", "sandbox bwb" in norm),
        (
            "não é homologação oficial AGT",
            (
                "nao e homologacao oficial agt" in norm
                or "nao significa acesso ao ambiente oficial de homologacao da agt" in norm
                or ("nao" in norm and "homologacao oficial agt" in norm)
            ),
        ),
        ("sealed_locally/SealInTx", "sealed_locally" in norm or "sealintx" in norm),
        (
            "não equivale a emissão/certificação AGT",
            (
                "nao constituem emissao fiscal certificada" in norm
                or "emissao fiscal certificada" in norm
                or ("sealed_locally" in norm and "certificad" in norm)
                or ("sealintx" in norm and "certificad" in norm)
            ),
        ),
    ]
    for label, ok in checks:
        if not ok:
            fail(path, "-", f"distinção essencial ausente: {label}")


def has_active_ruleset_evidence(norm: str) -> bool:
    if "protect main and require project checks" in norm:
        return True
    # Ignorar negações do tipo «sem ruleset activo».
    cleaned = re.sub(r"\bsem ruleset activ[oa]\b", " ", norm)
    return bool(re.search(r"\bruleset activ[oa]\b", cleaned))


def has_nonempty_bypass_claim(norm: str) -> bool:
    if re.search(r"\bbypass\s+permitido\b", norm):
        return True
    if re.search(r"\bbypass\s+configurado\b", norm):
        return True
    if re.search(
        r"\bbypass(?:[_\s]+actors?)?\s*[:=]\s*(?:\*\*)?"
        r"(?:admin|always|true|enabled|[1-9]\d*|"
        r"\[(?!\s*\])[^\]]+\])",
        norm,
    ):
        return True
    return False


def has_empty_bypass_evidence(norm: str) -> bool:
    """Exige ausência explícita de bypass; não aceita só a palavra «bypass»."""
    if has_nonempty_bypass_claim(norm):
        return False
    if re.search(r"\bsem bypass\b", norm):
        return True
    if re.search(r"\bbypass(?:[_\s]+actors?)?\s*[:=]\s*(?:\*\*)?vazio\b", norm):
        return True
    if re.search(r"\bbypass(?:[_\s]+actors?)?\s*[:=]\s*\[\s*\]", norm):
        return True
    return False


def has_never_bypass_evidence(norm: str) -> bool:
    if re.search(r"current_user_can_bypass\s*=\s*always\b", norm):
        return False
    if re.search(r"current_user_can_bypass\s*=\s*never\b", norm):
        return True
    if re.search(r"current_user_can_bypass\s*:\s*never\b", norm):
        return True
    return False


def has_legacy_contornavel_policy(norm: str) -> bool:
    return "nao tecnicamente impossivel de contornar" in norm or (
        "contornar" in norm and "politica assistida" in norm
    )


def validate_rm_gov_002(text: str, path: str, items: list[dict[str, str]]) -> None:
    """Valida RM-GOV-002 com base no Estado canónico do item analisado."""
    gov = next((it for it in items if it["id"] == "RM-GOV-002"), None)
    if gov is None:
        fail(path, "RM-GOV-002", "RM-GOV-002 ausente da tabela canónica")

    estado = gov["estado"].strip()
    norm = normalize_text(text)
    active = has_active_ruleset_evidence(norm)
    empty_bypass = has_empty_bypass_evidence(norm)
    never_bypass = has_never_bypass_evidence(norm)
    legacy = has_legacy_contornavel_policy(norm)

    if estado == "CONCLUÍDO":
        if legacy and not active:
            fail(
                path,
                "RM-GOV-002",
                "fallback legado só é permitido enquanto RM-GOV-002 não estiver concluído",
            )
        if not active:
            fail(path, "RM-GOV-002", "RM-GOV-002 concluído exige ruleset ativo")
        if not empty_bypass:
            fail(path, "RM-GOV-002", "RM-GOV-002 concluído exige bypass vazio")
        if not never_bypass:
            fail(
                path,
                "RM-GOV-002",
                "RM-GOV-002 concluído exige current_user_can_bypass=never",
            )
        return

    if estado in {"BLOQUEADO", "PENDENTE", "EM_CURSO"}:
        if legacy or (active and empty_bypass and never_bypass):
            return
        fail(
            path,
            "RM-GOV-002",
            "RM-GOV-002 não concluído exige formulação legada ou ruleset ativo completo",
        )

    fail(path, "RM-GOV-002", f"estado RM-GOV-002 inesperado para governação: {estado}")

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
        kind = link_kind(target)
        if kind == "https":
            continue
        if kind == "bad_external":
            fail(path, item_id, f"URL externa deve ser https://: {target}")
        if kind == "fragment":
            frag = target[1:]
            if frag not in anchors:
                fail(path, item_id, f"fragmento interno inexistente: {target}")
            continue
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


def validate_rm_candidates(
    text: str, path: str, items: list[dict[str, str]], consumed: set[int]
) -> None:
    lines = text.splitlines()
    candidate_idxs: set[int] = set()
    for i, line in enumerate(lines):
        if not is_table_line(line):
            continue
        cells = split_row(line)
        if is_separator(cells):
            continue
        if len(cells) == 7 and tuple(cells) == REQUIRED_HEADER:
            continue
        if not RM_TOKEN_RE.search(line):
            continue
        candidate_idxs.add(i)

    if len(items) != len(consumed):
        fail(path, "-", f"invariante items≠consumed ({len(items)}≠{len(consumed)})")

    unconsumed = sorted(candidate_idxs - consumed)
    if unconsumed:
        idx = unconsumed[0]
        tokens = RM_TOKEN_RE.findall(lines[idx])
        rid = tokens[0] if tokens else "-"
        fail(path, rid, "linha RM-* malformada/não analisada")

    if len(items) != len(candidate_idxs):
        fail(
            path,
            "-",
            f"contagem RM-* divergente (analisados={len(items)} candidatos={len(candidate_idxs)})",
        )


def verify(repo_root: Path, roadmap_path: Path) -> None:
    if roadmap_path.is_relative_to(repo_root):
        rel = str(roadmap_path.relative_to(repo_root))
    else:
        rel = str(roadmap_path)
    if not roadmap_path.is_file():
        fail(rel, "-", "ROADMAP.md ausente")
    if roadmap_path.name != CANONICAL_NAME:
        fail(rel, "-", "ficheiro canónico deve chamar-se ROADMAP.md")

    text = roadmap_path.read_text(encoding="utf-8")

    validate_sections(text, rel)
    validate_distinctions(text, rel)
    validate_local_deps(text, rel)
    anchors = collect_anchors(text)
    items, consumed = parse_items(text, rel)
    validate_rm_candidates(text, rel, items, consumed)
    if not items:
        fail(rel, "-", "nenhum item RM-* em tabela canónica encontrado")

    if "rm-gov-002-ruleset-de-main" not in anchors:
        fail(rel, "RM-GOV-002", "âncora rm-gov-002-ruleset-de-main ausente")

    validate_rm_gov_002(text, rel, items)

    seen: set[str] = set()
    for item in items:
        validate_item(item, rel, repo_root, roadmap_path, anchors, seen)

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
