# SAF-T (AO)

## Objetivo

Gerar um ficheiro SAF-T (AO) determinístico, completo e validável a partir do livro fiscal, nunca a partir de reconstruções parciais do POS.

## Roadmap de fontes e implementação (A → D)

| Fase | Conteúdo | Notas |
|---|---|---|
| A | Catálogo/governação | Concluído |
| B0 | Auditoria de reutilização cross-project | [AUD-B0](audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md) — experiência da app **não** é fonte normativa |
| B1 | Storage privado de originais B1 (PDFs + HTML FE) | Repo `bwb-fiscal-sources-ao` (SRC-B1); OCR legislação v1 em RM-SRC-004 |
| B2 | XSD + LICENSE/NOTICE + inventário no Git público | MIT ASSOFT; `pending_validation` AGT (SRC-B2) |
| C | Requisitos `AO-*` + rastreabilidade | Só fontes oficiais + páginas OCR `reviewed` |
| D | Implementação/testes | Vetores aprovados da matriz B0; sem autofix sem requisito |

## Fundação estrutural (RM-SAFT-001 … RM-SAFT-005)

- Pacote Go [`internal/saftao`](../../internal/saftao/): skeleton `AuditFile` + inventário de elementos do XSD embutido.
- RM-SAFT-002: inventário dos filhos obrigatórios de `Header`; stubs tipados `MasterFiles` / `SourceDocuments`; `NewSalesSkeleton()` para totais vazios.
- RM-SAFT-003: tipagem `SalesInvoices` / `Invoice` / `Line` / `Tax` / `DocumentTotals`; `Money2` e datas sem float; cinco grupos L3 (`DEC-PROD-001`, inclui `MovementOfGoods`); serialização XML determinística; validação estrutural + fixture sintética contra o XSD (`xmllint`) quando disponível.
- RM-SAFT-004: `BuildIncrementalExport` — filtro por período (`Header.StartDate`/`EndDate`), allowlist de `InvoiceType` (adesão/config fail-closed), agregação de totais, SHA-256 do XML (integridade do artefacto ≠ `Invoice.Hash`), validação XSD opcional. **Ainda sem** leitura do livro fiscal persistido.
- RM-SAFT-005: tipagem `MovementOfGoods` / `StockMovement` / `Line` / `MovementTax`; enums `MovementType`/`MovementStatus`; `CustomerID` XOR `SupplierID`; fixture XSD sintética.
- Distinção: **estrutura XSD** (forma/enums/padrões do schema) ≠ **conformidade legal** / aceitação AGT / requisitos `AO-*` confirmados.
- Pendências regulatórias no modelo (não bloqueiam o slice estrutural): algoritmo de `Hash`/`HashControl`; semântica de activação de `InvoiceType`/`MovementType` por adesão (`DEC-PROD-002`/`004`).
- XSD: `source_id` **AO-SAFT-XSD-1.01_01** (`compliance/catalog/sources.yaml`), status **`pending_validation`** — **não** afirmado como XSD validado/certificado pela AGT.
- Integridade: SHA-256 `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631` (NOTICE + catálogo).
- Namespace: `urn:OECD:StandardAuditFile-Tax:AO_1.01_01` · versão `1.01_01`.
- Mecanismo SAF-T ≠ JWS/RS256 da faturação electrónica.
- Esta fundação **não** gera ficheiro de produção, **não** valida semanticamente contra AGT e **não** fecha requisitos `AO-*`.

## Decisões iniciais

- Persistir desde a emissão todos os campos necessários ao SAF-T.
- Versionar schema, mapeamentos e regras de exportação.
- Gerar por empresa e período fiscal, com parâmetros auditáveis.
- Guardar hash, utilizador, instante, versão do gerador e resultado da validação.
- Não corrigir silenciosamente dados durante a exportação.
- Separar erro de dados, erro de configuração e erro de schema.
- Não promover hipóteses operacionais (ex. namespaces) a requisitos `AO-*` sem evidência oficial.

## Pipeline

1. Fechar o conjunto lógico do período.
2. Ler snapshot consistente do livro fiscal.
3. Mapear entidades e documentos.
4. Gerar XML determinístico.
5. Validar contra XSD e regras semânticas.
6. Produzir relatório e hash do ficheiro.
7. Disponibilizar exportação com autorização e auditoria.

## Testes obrigatórios

- documento normal, anulado e retificativo;
- cliente sem NIF nos casos permitidos;
- impostos, isenções e múltiplas taxas;
- sequências e séries anuais;
- períodos grandes e limites de memória;
- caracteres Unicode e escaping XML;
- ficheiro inválido por ausência de dados;
- comparação com vetores aprovados.
