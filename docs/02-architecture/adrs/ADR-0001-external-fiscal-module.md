# ADR-0001 — Módulo fiscal externo como autoridade de emissão

- Estado: Aceite como premissa do projeto
- Data: 2026-07-20
- Relacionado: ASM-REG-001

## Contexto

Vários POS devem integrar uma única plataforma fiscal sem certificação individual.

## Decisão

O POS envia uma intenção comercial. O módulo valida, calcula/verifica valores, atribui série e número, assina, persiste e comunica. Apenas o resultado devolvido pelo módulo pode ser apresentado como documento fiscal emitido.

## Consequências

- O POS não controla o número fiscal nem altera documentos emitidos.
- O contrato precisa distinguir intenção, receção, emissão e aceitação pela AGT.
- O módulo torna-se infraestrutura crítica e exige elevada disponibilidade.
- A fronteira deve ser demonstrável no dossier de certificação.
- Se a AGT alterar a interpretação, poderá ser necessário um processo de registo/validação por integrador sem mudar o modelo de domínio.
