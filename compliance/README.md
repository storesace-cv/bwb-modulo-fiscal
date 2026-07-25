# Compliance — fontes fiscais angolanas

Governação versionada de fontes fiscais (catálogo, política e estrutura). Os originais da recolha permanecem em `local/` (não sincronizado).

## Princípios

1. **Nunca** depender de `local/` em build, testes, runtime ou CI.
2. Copiar para o Git **apenas** artefactos autorizados (PR B+), com hash e proveniência.
3. O **PDF original** é a única fonte normativa. OCR e Markdown são derivados de consulta — nunca substitutos legais.
4. JWS/RS256 da faturação electrónica é **distinto** de mecanismos SAF-T.
5. Não inventar campos, endpoints, `FE-RNG-*`, QR Code ou regras criptográficas.
6. Não versionar chaves, tokens, Basic Auth reais ou dados pessoais.

## Layout

```text
compliance/
  README.md                 # este ficheiro
  POLICY.md                 # regras jurídicas/técnicas
  catalog/sources.yaml      # catálogo (PR A: só metadados)
  catalog/schema/           # JSON Schema do catálogo
  scripts/                  # validação determinística (sem OCR no CI)
  legislation/ao/           # originais + OCR (PR B, se autorizado)
  saft-ao/                  # XSD versionado (PR B)
  fe/                       # snapshots FE + inventários (PR B)
  derived/                  # requisitos AO-* (PR C)
  superseded/               # ponteiros; nunca apagar versões
```

## Catálogo

- Ficheiro: [`catalog/sources.yaml`](catalog/sources.yaml)
- Schema: [`catalog/schema/sources.schema.json`](catalog/schema/sources.schema.json)
- Recolha indexada: `arquivo_fiscal_ao-2026-07-25` (20 fontes)

Os três PDFs do Diário da República são **image-only** (`text_extractable: false`, `conversion_required: true`) e exigem OCR no PR B (`searchable_pdf` + `markdown_text`). Sem derivados neste PR.

## Validação

```bash
python3 -m venv compliance/scripts/.venv
compliance/scripts/.venv/bin/pip install -r compliance/scripts/requirements.txt
compliance/scripts/.venv/bin/python compliance/scripts/verify_catalog.py
# Desenvolvimento (opcional; exige local/):
compliance/scripts/.venv/bin/python compliance/scripts/verify_catalog.py --with-local
bash compliance/scripts/verify_no_local_deps.sh
```

O CI valida o catálogo **sem** exigir `local/`. A regeneração OCR é operação controlada fora do CI.

## PDFs image-only (facto)

| ID | Páginas | Texto embutido |
|---|---:|---|
| `AO-LEG-DE-74-19-2019` | 12 | não |
| `AO-LEG-RECT-10-19-2019` | 2 | não |
| `AO-LEG-DE-683-25-2025` | 16 | não |

## Relação com `docs/01-compliance/`

A prosa em `docs/01-compliance/` aponta para este catálogo. GAP-012 (manifesto versionado) fica parcialmente fechado com a criação deste índice; GAP-001/002/004/005 permanecem abertos até importação autorizada e validação.

## Auditoria B0 (reutilização SAF-T AO)

Experiência técnica inventariada (sem cópia de código) em
[AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md](../docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md).
Essa auditoria **não** eleva a aplicação privada a fonte normativa; o catálogo e a POLICY continuam a prevalecer.

Sequência: **A** (feito) → **B0** (auditoria) → **B1** (privado+OCR) → **B2** (XSD público) → **C** (requisitos) → **D** (testes).
