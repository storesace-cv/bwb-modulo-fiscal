# C-FE-QR-001 — URL do QR Code impresso: DE 683 vs snapshot FE HML

| Campo | Valor |
|---|---|
| Estado | aberto |
| Data | 2026-07-28 |
| Severidade | alta (impressão FE / ENG-007) |

## Factos

1. **DE 683/25** Anexo III (OCR v2 `reviewed`, gazeta **19194–19195**, PDF p.32–33): «Especificações do QR Code…»; nível de correcção **15%**; **33×33** módulos; URL OCR (auxiliar, ruído tipográfico): portal do contribuinte `…/consultar-fe?documentNo`; espaços em `documentNo` → `%20`; logo AGT **&lt;20%** da imagem se incluído.
2. **FE HML** `AO-FE-SNAP-HML-2026-07-25-QRCODE` (`ccade20b…`, `pending_validation`): Model **2**; versão **4** (33×33); ECC **M (15%)**; modo Byte; UTF-8; PNG 350×350; URL `https://quiosqueagt.minfin.gov.ao/facturacao-eletronica/consultar-fe?emissor=nifEmissor&document=documentNo`; logo AGT &lt;20%.
3. As **bases URL** e os **query params** diferem entre diploma (OCR) e HTML HML (`portaldocontribuinte` + `documentNo` vs `quiosqueagt` + `emissor`/`document`).

## Mitigação de engenharia (2026-07-28)

- Pacote `internal/feqr`: `ConflictOpen=true`; recusa construir URL de QR enquanto o conflito estiver aberto; regista parâmetros alinhados (ECC 15%, 33×33) e hosts candidatos **sem** escolher o path «correcto».
- **Não** implementa geração de QR (`RM-ENG-007`); **não** fecha AO-*; **não** inventa FE-RNG QR.

## Não fazer

- Não escolher unilateralmente `quiosqueagt` vs `portaldocontribuinte`.
- Não inventar query params além dos citados.
- Não marcar `RM-ENG-007` CONCLUÍDO.
- Não promover `AO-AGT-*` / FE-RNG por causa deste inventário.

## Resolução candidata

Confirmação AGT (ou FE PRD estável) da URL canónica + parâmetros; confronto visual residual do PDF p.32–33; só então `ConflictOpen=false`.
