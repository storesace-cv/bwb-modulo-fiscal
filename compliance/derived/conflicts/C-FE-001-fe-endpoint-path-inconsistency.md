# C-FE-001 — inconsistência de paths FE HML vs PRD nos snapshots

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-07-26 |
| Severidade | alta (integração AGT / dual HML-PRD) |

## Factos

1. Em `AO-FE-SNAP-HML-2026-07-25-REGISTAR` (`eb430954…`), HML e PRD usam `/sigt/fe/v1/registarFactura`.
2. Em `AO-FE-SNAP-HML-2026-07-25-SOLICITAR` (`f8fb22e7…`), a URL rotulada HML é `…/sigt/fe/ws/v1/registarFactura` (serviço **registar**, path `/ws/`), enquanto PRD é `…/sigt/fe/v1/solicitarSerie`.
3. Em `AO-FE-SNAP-HML-2026-07-25-LISTAR-FATURAS` (`c748caca…`), HML usa `/sigt/fe/ws/v1/listarFacturas` e PRD `/sigt/fe/v1/listarFacturas`.
4. Em `AO-FE-SNAP-HML-2026-07-25-LISTAR` (`5729f02c…`), HML/PRD alinham em `/sigt/fe/v1/listarSeries`.

## Não fazer

- Não inventar o path «correcto» (`/fe/v1` vs `/fe/ws/v1`) sem confirmação AGT ou documentação PRD estável.
- Não assumir que a URL HML na página `solicitarSerie` é `solicitarSerie` — o HTML cita `registarFactura`.
- Não promover `AO-AGT-001` a confirmado enquanto C-FE-001 + GAP-006 estiverem abertos.
