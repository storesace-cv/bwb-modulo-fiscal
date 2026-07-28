# C-FE-001 — inconsistência de paths FE HML vs PRD nos snapshots

| Campo | Valor |
|---|---|
| Estado | aberto (mitigação engenharia fail-closed 2026-07-28; residual AGT / GAP-006) |
| Data | 2026-07-28 (inventário alargado; mitigação `fepath`) |
| Severidade | alta (integração AGT / dual HML-PRD) |

## Factos

1. Em `AO-FE-SNAP-HML-2026-07-25-REGISTAR` (`eb430954…`), HML e PRD usam `/sigt/fe/v1/registarFactura`.
2. Em `AO-FE-SNAP-HML-2026-07-25-SOLICITAR` (`f8fb22e7…`), a URL rotulada HML é `…/sigt/fe/ws/v1/registarFactura` (serviço **registar**, path `/ws/`), enquanto PRD é `…/sigt/fe/v1/solicitarSerie`.
3. Em `AO-FE-SNAP-HML-2026-07-25-LISTAR-FATURAS` (`c748caca…`), HML usa `/sigt/fe/ws/v1/listarFacturas` e PRD `/sigt/fe/v1/listarFacturas`.
4. Em `AO-FE-SNAP-HML-2026-07-25-LISTAR` (`5729f02c…`), HML/PRD alinham em `/sigt/fe/v1/listarSeries`.
5. Em `AO-FE-SNAP-HML-2026-07-25-CONSULTAR-FATURA` (`6d5cc1a0…`), HML/PRD alinham em `/sigt/fe/v1/consultarFactura`.
6. Em `AO-FE-SNAP-HML-2026-07-25-VALIDAR` (`7ab70629…`), HML/PRD alinham em `/sigt/fe/v1/validarDocumento`.

## Mitigação de engenharia (2026-07-28)

- Pacote `internal/fepath`: prefixos `/v1` ≠ `/ws/v1`; `ConflictOpen=true`; recusa construir URL para `solicitarSerie` / `listarFacturas` enquanto o conflito estiver aberto; só permite URLs `PrefixV1` para serviços alinhados no inventário (`registarFactura`, `listarSeries`, `obterEstado`, `consultarFactura`, `validarDocumento`).
- Testes: invariantes + recusa de serviços conflituosos + hosts limitados aos citados no snapshot.
- **Não** inventa o path «correcto»; **não** fecha `AO-AGT-001` nem GAP-006; **não** marca C-FE-001 resolvido.

## Não fazer

- Não inventar o path «correcto» (`/fe/v1` vs `/fe/ws/v1`) sem confirmação AGT ou documentação PRD estável.
- Não assumir que a URL HML na página `solicitarSerie` é `solicitarSerie` — o HTML cita `registarFactura`.
- Não promover `AO-AGT-001` a confirmado enquanto C-FE-001 + GAP-006 estiverem abertos.
- Não tratar o alinhamento de `consultarFactura` / `validarDocumento` como fecho deste conflito.
- Não marcar C-FE-001 como resolvido só porque os guards `fepath` existem.

## Resolução candidata

Confirmação AGT (ou documentação PRD estável) dos paths HML/PRD por serviço + credenciais (GAP-006); só então `ConflictOpen=false` e fecho documentado.
