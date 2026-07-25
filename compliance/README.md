# Compliance — fontes fiscais angolanas

Governação versionada de fontes fiscais (catálogo, política e estrutura). Consulta local em `local/` (não sincronizado). Originais B1 sincronizados no repositório privado `storesace-cv/bwb-fiscal-sources-ao` (`storage=private_sync`).

## Princípios

1. **Nunca** depender de `local/` em build, testes, runtime ou CI.
2. Copiar para o Git **apenas** artefactos autorizados, com hash e proveniência.
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
  legislation/ao/           # OCR futuro (se autorizado)
  saft-ao/                  # XSD + LICENSE/NOTICE (SRC-B2)
  fe/                       # snapshots FE + inventários (futuro)
  derived/                  # requisitos AO-* (fase C)
  superseded/               # ponteiros; nunca apagar versões
```

## Catálogo

- Ficheiro: [`catalog/sources.yaml`](catalog/sources.yaml)
- Schema: [`catalog/schema/sources.schema.json`](catalog/schema/sources.schema.json)
- Recolha indexada: `arquivo_fiscal_ao-2026-07-25` (20 fontes)
- Armazenamento privado B1: `storesace-cv/bwb-fiscal-sources-ao` (PDFs + HTML FE + proveniência; sem OCR)
- XSD público B2: [`saft-ao/schemas/`](saft-ao/schemas/) (`AO-SAFT-XSD-1.01_01`, `pending_validation`); ZIP `local_only`

Os três PDFs do Diário da República são **image-only**. OCR: 74/19 e 683/25 v2 `reviewed` no privado (KB auxiliar); Rect. 10/19 sem original integral — `rejected` **não** é KB. RM-SRC-004/RM-M2-C **BLOQUEADOS**. CI valida metadados **sem** regenerar OCR.

## Validação

```bash
python3 -m venv compliance/scripts/.venv
compliance/scripts/.venv/bin/pip install -r compliance/scripts/requirements.txt
compliance/scripts/.venv/bin/python compliance/scripts/verify_catalog.py
bash tests/compliance/run-verify-catalog-tests.sh
# Desenvolvimento (opcional; exige local/):
compliance/scripts/.venv/bin/python compliance/scripts/verify_catalog.py --with-local
bash compliance/scripts/verify_no_local_deps.sh
```

O CI valida o catálogo **sem** exigir `local/`. A regeneração OCR é operação controlada fora do CI.

## PDFs image-only (facto)

| ID | Páginas | Texto embutido |
|---|---:|---|
| `AO-LEG-DE-74-19-2019` | 12 | não |
| `AO-LEG-RECT-10-19-2019` | 2 (incorrecto/incompleto) | não |
| `AO-LEG-DE-683-25-2025` | 66 (original v2 correcto; OCR reviewed) | não |

## Relação com `docs/01-compliance/`

A prosa em `docs/01-compliance/` aponta para este catálogo. GAP-012 (manifesto versionado) fica parcialmente fechado com a criação deste índice; GAP-001/002/004/005 permanecem abertos até importação autorizada e validação.

## Auditoria B0 (reutilização SAF-T AO)

Experiência técnica inventariada (sem cópia de código) em
[AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md](../docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md).
Essa auditoria **não** eleva a aplicação privada a fonte normativa; o catálogo e a POLICY continuam a prevalecer.

Sequência: **A** (feito) → **B0** (feito) → **B1** (feito; OCR separado) → **B2** (XSD público, feito) → **C** (requisitos) → **D** (testes).
