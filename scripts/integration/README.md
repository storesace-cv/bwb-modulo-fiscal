# Kit de integração POS (sandbox)

Script: `pos-sandbox-kit.sh`  
Dependências: bash ≥3.2, curl ≥7.55.0, jq ≥1.6, openssl ≥1.1.1.

## Uso

```bash
chmod 0600 ./sandbox.token   # regular file, owner=you, mode 0600
./pos-sandbox-kit.sh --token-file ./sandbox.token --report-file ./report.json
```

- URL exacta: `https://sandbox.fiscalmod.bwb.pt/v1` (sem `BASE_URL` por environment).
- Loopback só em testes: `--allow-loopback-test http://127.0.0.1:PORT/v1`.
- `--token-file` e `--token-stdin` são mutuamente exclusivos.
- `--revoked-token-file` opcional; sem ele, `token_revoked_401=NOT_RUN`.
- Token nunca em argv/logs; header file temporário `0600` em tmpdir `0700`.
- Fixtures sintéticas; IDs CSPRNG por execução; relatório sem NIF/tokens/bodies.
- `rate_429` corre no fim (máx. 30 pedidos).

**Validação operacional** contra sandbox real é feita pela BWB/software house após o merge documental — a CI do repositório usa mocks locais.
