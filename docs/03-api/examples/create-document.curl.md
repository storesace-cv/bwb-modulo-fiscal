# Exemplos curl (sem credenciais)

Substituir `REPLACE_WITH_MODULE_SANDBOX_TOKEN`. Preferir header file:

```bash
printf 'Authorization: Bearer %s\n' "$TOKEN" > /tmp/auth.hdr
chmod 0600 /tmp/auth.hdr
curl -sS -X POST 'https://sandbox.fiscalmod.bwb.pt/v1/documents' \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: 11111111-1111-4111-8111-111111111111' \
  -H @/tmp/auth.hdr \
  --data-binary @body.json
rm -f /tmp/auth.hdr
```

Não usar `curl -H "Authorization: Bearer ${TOKEN}"` (expõe o token no argv).
Não seguir redirects (`-L`) para este contrato.

Health:

```bash
curl -sS 'https://sandbox.fiscalmod.bwb.pt/v1/health'
```
