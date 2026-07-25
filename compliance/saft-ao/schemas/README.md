# Schemas SAF-T (AO)

## Conteúdo

- `SAFTAO1.01_01.xsd` — schema SAF-T (AO) 1.01_01 (cópia byte-for-byte ASSOFT).
- `LICENSE` — MIT ASSOFT (extraída do snapshot upstream; não inventada).
- `NOTICE.md` — proveniência e limites (ficheiro do projecto BWB).
- `SHA256SUMS.txt` — manifesto determinístico dos quatro artefactos acima (sem se auto-incluir).

## Uso

- Consulta e preparação de validação XML SAF-T (AO) em desenvolvimento.
- Catálogo: `AO-SAFT-XSD-1.01_01` em `compliance/catalog/sources.yaml`
  (`storage=git_public`, `status=pending_validation`).
- Verificar integridade: `sha256sum -c SHA256SUMS.txt` nesta pasta, e
  `compliance/scripts/verify_catalog.py` na raiz do repositório.

## Proveniência

- Upstream: https://github.com/assoft-portugal/SAF-T-AO
- Snapshot registado: `ed86c7d5a5ab21cf5ebb410507a5de04a86663f3`
- SHA-256 do XSD: `e9a938e1f47ac3d84ffbb26d0d95b827fc769a065c9d20533d0262c12f8c2631`

## Estado

`pending_validation`: cópia versionada sob MIT; **não** afirmada como schema oficial aceite,
homologado ou certificado pela AGT. Uma futura fonte oficial AGT divergente tem prioridade.

## SAF-T vs faturação electrónica

Assinatura/cadeia histórica SAF-T **não** se confunde com JWS/RS256 da faturação electrónica AGT.
Ver `compliance/POLICY.md` e a documentação FE catalogada.

## Fora desta pasta

- O ZIP completo `SAF-T-AO_repositorio_master.zip` permanece fora do Git (`local_only`).
- OCR de diplomas PDF e requisitos `AO-*` derivados são incrementos distintos.
