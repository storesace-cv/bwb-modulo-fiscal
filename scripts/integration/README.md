# Kit de integração POS (sandbox)

Script: `pos-sandbox-kit.sh`
Dependências: bash ≥3.2, curl ≥7.55.0, jq ≥1.6, openssl ≥1.1.1.

## Uso

```bash
# Token: ficheiro regular, owner=euid, mode 0600, exactamente 52 bytes ASCII
# (prefixo bwb_sbox_ + 43 Base64URL). Sem CR/LF nem whitespace.
# fiscal-admin --output-file grava exactamente estes 52 bytes (sem newline).
chmod 0600 ./sandbox.token
./pos-sandbox-kit.sh --token-file ./sandbox.token --report-file ./report.json
```

- URL exacta: `https://sandbox.fiscalmod.bwb.pt/v1` (sem override por environment).
- Loopback só em testes: `--allow-loopback-test http://127.0.0.1:PORT/v1`.
- `--token-file` e `--token-stdin` são mutuamente exclusivos.
- `--revoked-token-file` opcional; sem ele, `token_revoked_401=NOT_RUN`.
- Token nunca em argv/logs; header file temporário `0600` em tmpdir `0700`.
- Fixtures sintéticas; IDs CSPRNG por execução; relatório sem NIF/tokens/bodies/document ids.
- `rate_429`: rajada **sincronizada** de exactamente 30 pedidos (workers ready → gate único; sem ondas de cinco), alinhada ao limiter sandbox `10r/s` + `burst=20`. Critério: ≥1×429, 0×5xx/transport/other, 30 resultados. Cooldown no fim; cleanup mata filhos da execução.

## Variáveis de ambiente (apenas harness / loopback)

Rejeitadas salvo com `--allow-loopback-test` (não afectam execução contra sandbox):

| Variável | Função |
|----------|--------|
| `BWB_POS_KIT_CURL` | Binário curl substituto (mock argv / bloqueio) |
| `BWB_POS_KIT_RATE_COOLDOWN` | Segundos de cooldown após rate (default 5) |
| `BWB_POS_KIT_TMP_PATH_FILE` | Ficheiro onde o kit escreve o path do tmpdir |
| `BWB_POS_KIT_READY_FILE` | Touched when the first curl child is about to start (signal tests) |
| `BWB_POS_KIT_RATE_HOLD_BEFORE_GATE` | Path: parent waits for this file after 30 ready, before opening gate |
| `BWB_POS_KIT_RATE_HOLD_AFTER_GATE` | Path: workers wait for this file after gate, before curl |
| `BWB_POS_KIT_RATE_READY_TIMEOUT` | Segundos para readiness dos 30 workers (default 15) |
| `BWB_POS_KIT_RATE_STALL_WORKER` | Índice 1–30 que nunca fica ready (teste de timeout) |

**Validação operacional** contra sandbox real é feita pela BWB/software house após o merge — a CI do repositório usa mocks locais (ThreadingHTTPServer + token bucket `10r/s`/`burst=20`).
