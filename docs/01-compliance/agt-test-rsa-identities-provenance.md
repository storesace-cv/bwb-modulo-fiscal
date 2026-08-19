# Proveniência — identidades RSA de teste AGT (workbook não versionado)

**Item:** `RM-FEFIX-001`
**Data (UTC):** 2026-08-16
**Estado:** inventário/validação local apenas — **≠** certificados X.509, **≠** Basic Auth, **≠** `softwareValidationNo`, **≠** registo/autorização produtiva da BWB.

## Material

| Campo | Valor |
|---|---|
| Armazenamento | Workbook AGT recebido em 2026-08, guardado em área de consulta **não versionada** (fora do Git; ver `.gitignore`) |
| SHA-256 do ficheiro | `9b49d5104dd284f374d6f5638fb4be9e7751b959e92024db64dffa4c6be7b179` |
| Tamanho (bytes) | `17567` |
| Classificação | Fixtures / identidades fiscais de **teste** fornecidas pela AGT para desenvolvimento e futura homologação |
| Conteúdo (sanitizado) | 5 pares RSA PEM (`PRIVATE KEY` / `PUBLIC KEY`); cabeçalhos `NIF` / `NOME` / `CHAVE PRÍVADA` / `CHAVE PÚBLICA` |
| Tratamento | Read-only; loader lê o workbook em memória a partir de um path fornecido pelo operador; **sem** extracção para ficheiros PEM permanentes; **sem** cópia para Git (público ou privado de fontes) |

## O que este material **não** é

- Certificados X.509;
- Credenciais Basic Auth de produtor;
- `softwareValidationNo` / número de validação de software certificado;
- Identidade definitiva da BWB;
- Prova de autorização para produção AGT ou de registo de software.

## Fontes FE citadas (`pending_validation`)

Não fecham conflitos normativos nem promovem `AO-*` confirmados. JWS/RS256 FE ≠ SAF-T.

| `source_id` | Papel |
|---|---|
| `AO-FE-SNAP-HML-2026-07-25-API` | Autenticação/endereços (snapshot HML) |
| `AO-FE-SNAP-HML-2026-07-25-ESTRUTURA` | JWS RS256 / campos assinados (snapshot) |
| `AO-FE-SNAP-HML-2026-07-25-GESTAO` | Regras públicas de chaves (contribuinte vs produtor) |
| `AO-FE-SNAP-HML-2026-07-25-REGISTAR` | Registo + catálogo FE-RNG (snapshot) |

Catálogo: [`compliance/catalog/sources.yaml`](../../compliance/catalog/sources.yaml). Política: [`compliance/POLICY.md`](../../compliance/POLICY.md).

## Inventário público permitido

Apenas: quantidade de identidades; algoritmo; tamanho RSA (bits); resultado pub↔priv; identificadores opacos (`agt-test-…`); fingerprints SHA-256 das chaves **públicas** (calculáveis localmente; **não** publicadas neste documento).
**Proibido** em Git, logs, PR e relatórios: PEM, NIF em claro, nomes do workbook, células, screenshots, workbook real, fixtures derivadas das linhas reais, nome/UUID do ficheiro real.

## Ferramenta

Pacote [`internal/agttestkit`](../../internal/agttestkit/doc.go): `LoadAndValidate(path)`; CI usa workbook **sintético** efémero (`WriteSyntheticWorkbook`). Fila persistente mock: [`agt-fe-fixture-queue.md`](agt-fe-fixture-queue.md) (`RM-FEFIX-007`). A autoridade contra dependências de árvores não versionadas permanece `compliance/scripts/verify_no_local_deps.sh`. Validação do workbook real AGT é operação manual do operador (fora da suite CI).
