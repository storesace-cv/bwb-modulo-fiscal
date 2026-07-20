# ADR-0003 — Pacotes fiscais independentes por país

- Estado: Aceite
- Data: 2026-07-20

## Decisão

Separar núcleo técnico, pacote Angola e futuro pacote Cabo Verde. Cada pacote contém schemas, regras, mapeamentos, versões e vetores próprios.

## Consequências

- Alteração angolana não afeta Cabo Verde por defeito.
- A API pode reutilizar conceitos comuns, mas não força equivalência jurídica.
- O país e a versão fiscal aplicável ficam registados em cada documento.
