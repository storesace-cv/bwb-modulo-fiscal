# Checklist de aceitação — integração POS

Usar com o kit `scripts/integration/pos-sandbox-kit.sh` e o OpenAPI `0.1.6-draft`.
`sealed_locally` **não** certifica emissão AGT.

## Segurança
- [ ] Token só em ficheiro `0600` (ou canal seguro BWB); nunca git/logs/argv
- [ ] TLS para sandbox HTTPS; sem desligar verificação em produção futura
- [ ] Sem envio de segredos em telemetria

## Idempotência
- [ ] `Idempotency-Key` UUID persistida **antes** da primeira tentativa
- [ ] Timeout → mesma chave; 201 replay estável
- [ ] Tratamento distinto dos dois `409`

## Datas / timezone
- [ ] `issued_at` com offset Angola (`+01:00`); sem `Z` como atalho

## Precisão monetária
- [ ] `Money` / quantidades como strings decimais; sem float binário

## Erros / retries
- [ ] 401/403/422/409/429/5xx mapeados
- [ ] 429: backoff + mesma chave; sem depender de Problem/`Retry-After`
- [ ] `request_id` usado no suporte quando presente

## Token
- [ ] Rotação/revogação alinhada com BWB
- [ ] `token_revoked_401` marcado só com evidência BWB (`--revoked-token-file`)

## Evidências mínimas
- [ ] Relatório sanitizado do kit (sem tokens, bodies, NIF/identificadores fiscais)
- [ ] Exemplos alinhados ao OpenAPI
