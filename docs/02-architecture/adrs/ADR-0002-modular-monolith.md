# ADR-0002 — Monólito modular no núcleo fiscal

- Estado: Aceite
- Data: 2026-07-20

## Decisão

Implementar o núcleo como monólito modular, separando processos de trabalho assíncrono apenas quando necessário.

## Motivos

- Transações fortes para idempotência, numeração e livro fiscal.
- Menor complexidade operacional durante certificação.
- Auditoria e testes de conformidade mais simples.
- Extração futura possível através de contratos internos claros.

## Limites

Módulos não acedem diretamente às tabelas de outros módulos. Comunicação interna usa interfaces e eventos transacionais.
