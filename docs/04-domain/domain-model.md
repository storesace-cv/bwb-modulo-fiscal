# Modelo de domínio inicial

## Agregados

### Tenant, contribuinte e integrador

- `Tenant` — fronteira de isolamento inequívoca; propriedade e autorização explícitas.
- `Taxpayer` — contribuinte; **único dono do NIF**; pertence a um tenant proprietário (proibida pertença ambígua a vários tenants).
- `Establishment` — loja/estabelecimento; **não** assume NIF próprio; códigos fiscais de estabelecimento só conforme regras oficiais.
- `Terminal` — associado a estabelecimento.
- `Integrator` — software house / POS; distinto do contribuinte; vínculo explícito a `Taxpayer` sob autorização.
- `SoftwareVersion` — versão do software / pacote fiscal aplicável.
- `Environment` — `homologation` | `production` (sem partilha de material secreto entre ambientes).

`Taxpayer` e `Integrator` não se misturam: o NIF do contribuinte não colapsa no integrador.

### Credenciais, referências e autorização

- `ProducerCredential` — Basic Auth AGT do **produtor**; âmbito plataforma BWB + ambiente (não tenant/contribuinte).
- `ProducerKeyRef` — referência ao par RSA do **produtor**; âmbito plataforma + ambiente.
- `TaxpayerKeyRef` — referência ao par RSA do **contribuinte**; âmbito contribuinte + ambiente; material no `SecretStore` da plataforma só se permitido por DEC-REG-KEY-CUSTODY.
- `TaxpayerKeyAuthorization` — evidência de autorização do contribuinte para custódia/uso (necessária; **não suficiente** sem permissão oficial AGT).

A aplicação persiste metadados das refs (fingerprint a partir de chave pública ou metadados seguros, versão, estado, validade, origem, ambiente, rotação/revogação). **Nunca** o segredo em texto claro, logs ou UI. Ver [backoffice-architecture.md](../02-architecture/backoffice-architecture.md).

### Configuração fiscal

`CountryFiscalProfile`, `TaxRegime`, `TaxCode`, `ExemptionReason`, `DocumentType`, `Series`.

### Emissão

`DocumentIntent`, `FiscalDocument`, `FiscalLine`, `TaxSummary`, `PaymentSummary`, `CorrectionReference`, `FiscalArtifact`.

### Comunicação

`AuthoritySubmission`, `AuthorityAttempt`, `AuthorityResponse`, `WebhookDelivery`.

### Auditoria

`AuditEvent`, `RuleEvaluation`, `PackageVersion`, `ExportJob`, `SaftArtifact`.

## Invariantes críticas

- Um `FiscalDocument` possui país e versão do pacote fiscal imutáveis.
- `FiscalDocument` emitido não é atualizado; novas informações são eventos ou artefactos relacionados.
- Um número pertence a uma série e é único.
- Uma chave de idempotência resolve sempre para o mesmo resultado ou conflito explícito.
- Totais são derivados/verificados a partir de linhas e impostos segundo regras versionadas.
- Resposta da AGT não substitui o pedido ou artefacto enviados; ambos são preservados.

## Dinheiro

Formato externo decimal em string/JSON number conforme contrato final, com limite de casas explícito. Formato interno decimal exato. Arredondamento definido por regra fiscal, nunca pelo padrão implícito da linguagem.
