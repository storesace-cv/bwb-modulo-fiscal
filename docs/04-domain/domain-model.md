# Modelo de domínio inicial

## Agregados

### Tenant e empresa

`Tenant`, `Taxpayer`, `Establishment`, `Terminal`, `Integrator`, `SoftwareVersion`.

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
