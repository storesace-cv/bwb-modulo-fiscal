# NOTICE — SAF-T (AO) XSD (produzido pelo projecto BWB)

Este ficheiro **não** faz parte do repositório upstream ASSOFT. Foi criado pelo projecto
`bwb-modulo-fiscal` para documentar proveniência e limites da redistribuição.

## Origem

- Repositório upstream: [ASSOFT SAF-T-AO](https://github.com/assoft-portugal/SAF-T-AO)
- Snapshot/commit registado: `ed86c7d5a5ab21cf5ebb410507a5de04a86663f3`
- Recolha local de referência: `arquivo_fiscal_ao` (2026-07-25)

## Artefactos versionados nesta pasta

| Ficheiro | Natureza |
|---|---|
| `SAFTAO1.01_01.xsd` | Cópia byte-for-byte do XSD ASSOFT |
| `LICENSE` | Cópia byte-for-byte de `SAF-T-AO-master/LICENSE` (MIT, Copyright 2019 ASSOFT) |
| `NOTICE.md` | Este aviso (projecto BWB) |
| `README.md` | Documentação de uso (projecto BWB) |
| `SHA256SUMS.txt` | Manifesto de integridade (projecto BWB) |

## Integridade do XSD

- SHA-256: `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631`
- O mesmo hash aplica-se ao ficheiro directo da recolha e à entrada
  `SAF-T-AO-master/XSD/SAFTAO1.01_01.xsd` no ZIP arquivado.

## Metadados declarados no cabeçalho do XSD

Conforme o próprio `SAFTAO1.01_01.xsd` (não alterado):

- Copyright: OECD (`doc:Copyright`)
- Author: AGT - Administração Geral Tributária (`doc:Author`)
- Versão: `1.01_01` (`doc:Number`)
- Status: `Development` (`doc:Status`)
- Namespace: `urn:OECD:StandardAuditFile-Tax:AO_1.01_01`

## Redistribuição

A redistribuição do XSD e da `LICENSE` baseia-se na licença MIT publicada pelo upstream ASSOFT
(ficheiro `LICENSE` nesta pasta). Conservar o aviso de copyright e a permissão MIT em cópias
substanciais.

## Limites — sem confirmação AGT

Esta cópia **não** constitui confirmação de:

- oficialidade perante a AGT;
- vigência do schema para produção;
- homologação;
- certificação.

O estado no catálogo permanece `pending_validation` até evidência oficial AGT e validação
independente. Em conflito com uma fonte oficial AGT divergente, prevalece a fonte AGT.

Os mecanismos SAF-T são distintos da faturação electrónica (JWS/RS256).
