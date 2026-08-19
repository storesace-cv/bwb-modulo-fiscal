# AO-SEQ-001 — motor de numeração (RM-ENG-002)

**Estado:** residual de engenharia fechado por testes (RM-ENG-002).

**Norma:** [`CONFIRMED-MATRIX-RM-REQ-001.md`](../../compliance/derived/requirements/CONFIRMED-MATRIX-RM-REQ-001.md) — `confirmed_normative` (DP 71 + DE 74).

**Não é:** `AO-SEQ-002` (séries atribuídas pela AGT / `solicitarSerie`); homologação AGT; produção FE.

## Invariantes implementados

| Invariante | Mecanismo |
|---|---|
| Sequência progressiva e contínua por `(scope_id, series_code)` | `series_counters.last_seq` + `nextFiscalSeq` em `SealInTx` |
| Identificação unívoca | `UNIQUE (scope_id, series_code, fiscal_seq)` + lock (`FOR UPDATE` / `BEGIN IMMEDIATE`) |
| Sem buracos em selagens commitadas | Contador e documento na mesma transacção; rollback restaura |
| POS não atribui o número fiscal | HTTP usa `SeriesEffectiveCode` do binding; `requested_series` ignorado |

## Evidência de teste

- `AO-SEQ-001_concurrent_continuous_sequence` — N goroutines → conjunto exacto `{1..N}`
- `AO-SEQ-001_sequential_no_gaps` — selagens seriadas 1..5
- VS-T06 / VS-T07 — unicidade e rollback do contador (suite Seal)

## Fora de âmbito deste item

- `AO-SEQ-002` / FE `solicitarSerie` / FE-RNG (fontes `pending_validation` / C-FE-*)
- Ligação obrigatória SealInTx ↔ `establishment_series` (RM-BO-014 = metadados ≠ numeração)
- Aceitação ou certificado AGT
