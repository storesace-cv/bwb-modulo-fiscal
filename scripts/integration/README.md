# Kit de integração POS (sandbox)

Script: `pos-sandbox-kit.sh`
Dependências: bash ≥3.2, curl ≥7.55.0, jq ≥1.6, openssl ≥1.1.1.

## Uso

```bash
# Token: ficheiro regular, owner=euid, mode 0600, exactamente 52 bytes ASCII
# (prefixo bwb_sbox_ + 43 Base64URL). Sem CR/LF nem whitespace.
chmod 0600 ./sandbox.token
./pos-sandbox-kit.sh --token-file ./sandbox.token --report-file ./report.json
```

- URL exacta: `https://sandbox.fiscalmod.bwb.pt/v1` (sem override por environment).
- Loopback só em testes: `--allow-loopback-test http://127.0.0.1:PORT/v1`.
- `--token-file` e `--token-stdin` são mutuamente exclusivos.
- `--revoked-token-file` opcional; sem ele, `token_revoked_401=NOT_RUN`.
- Token nunca em argv/logs; header file temporário `0600` em tmpdir `0700`.
- Fixtures sintéticas; IDs CSPRNG por execução; relatório sem NIF/tokens/bodies/document ids.
- `rate_429` corre no fim (máx. 30 pedidos); cleanup mata filhos da execução.

## Variáveis de ambiente (apenas harness / loopback)

Rejeitadas salvo com `--allow-loopback-test` (não afectam execução contra sandbox):

| Variável | Função |
|----------|--------|
| `BWB_POS_KIT_CURL` | Binário curl substituto (mock argv / bloqueio) |
| `BWB_POS_KIT_RATE_COOLDOWN` | Segundos de cooldown após rate (default 5) |
| `BWB_POS_KIT_TMP_PATH_FILE` | Ficheiro onde o kit escreve o path do tmpdir |
| `BWB_POS_KIT_READY_FILE` | Touched when the first curl child is about to start (signal tests) |

**Validação operacional** contra sandbox real é feita pela BWB/software house após o merge documental — a CI do repositório usa mocks locais.
