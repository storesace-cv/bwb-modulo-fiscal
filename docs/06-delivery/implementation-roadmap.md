# Roadmap de implementação

## Fase 0 — Descoberta (4–8 semanas)

- Completar matriz legal Angola.
- Validar `ASM-REG-001` no processo de relacionamento com a AGT.
- Entrevistar 2–3 software houses.
- Fechar tipos documentais, impostos e fluxo de contingência.
- Aprovar arquitetura, ameaça e stack.

### Fontes fiscais e SAF-T AO (trilho paralelo)

| Fase | Entrega |
|---|---|
| A | Catálogo versionado + governação (`compliance/`) — concluído |
| B0 | Auditoria de reutilização SAF-T AO cross-project — [AUD-B0](../01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md) |
| B1 | Repo privado `bwb-fiscal-sources-ao` + PDFs DR + OCR + HTML FE |
| B2 | Importação pública XSD (MIT ASSOFT) + inventários FE |
| C | Requisitos `AO-*` a partir de fontes oficiais |
| D | Implementação/testes SAF-T com vetores aprovados |

A experiência em `bwb-efatura-docs` é pista técnica, **não** fonte normativa.

**Gate:** catálogo crítico validado e contrato API rascunhado.

## Fase 1 — Contrato e vertical (8–12 semanas)

- OpenAPI, schemas e mock sandbox.
- POS demo.
- Primeira fatura ponta a ponta.
- Persistência, idempotência, série e assinatura.
- Conector sandbox AGT.
- Primeira distribuição Edge Linux.

**Gate:** demonstração repetível com evidências.

## Fase 2 — MVP certificável (12–20 semanas)

- Tipos documentais aprovados.
- Retificações/anulações.
- Contingência e reconciliação.
- SAF-T (AO).
- Portal / backoffice operacional (fora do primeiro vertical slice); ver [backoffice-architecture.md](../02-architecture/backoffice-architecture.md).
- Hardening, carga, backup/restauro e documentação de integradores.

**Gate:** readiness review de certificação.

## Fase 3 — Certificação e piloto

- Submeter dossier.
- Responder a não conformidades.
- Pilotar com 1–2 software houses e poucos contribuintes.
- Operação assistida e relatório final.

## Fase 4 — Escala Angola

- SLA, suporte, partner program, SDKs e rollout progressivo.

## Fase 5 — Cabo Verde

- Descoberta legal própria, pacote CV, SAF-T (CV), certificação e piloto.
