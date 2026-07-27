# Modelo de domínio inicial

## Agregados

### Tenant, contribuinte e integrador

- `Tenant` — fronteira de isolamento inequívoca; propriedade e autorização explícitas.
- `Taxpayer` — contribuinte; **único dono do NIF**; pertence a um tenant proprietário (proibida pertença ambígua a vários tenants); **dono da adesão FE** (`DEC-PROD-004`).
- `Establishment` — loja/estabelecimento; **não** assume NIF próprio; códigos fiscais de estabelecimento só conforme regras oficiais; **séries e configuração próprias** (activação grupos/tipos `DEC-PROD-003`).
- `Terminal` — associado a estabelecimento.
- `Integrator` — software house / POS; distinto do contribuinte; vínculo explícito a `Taxpayer` sob autorização.
- `SoftwareVersion` — versão do software / pacote fiscal aplicável.
- `Environment` — `homologation` | `production` (sem partilha de material secreto nem de estado de adesão FE entre ambientes).

`Taxpayer` e `Integrator` não se misturam: o NIF do contribuinte não colapsa no integrador.

### Adesão FE (`DEC-PROD-004`)

- Âmbito: `Taxpayer` + `Environment` (NIF do contribuinte), **não** por estabelecimento.
- Estado canónico (enum, **não** booleano): `not_enrolled` | `pending` | `active` | `suspended`.
- Emissão FE exige `active` no contribuinte **e** séries/config activas no `Establishment` (e restantes regras oficiais) — fail-closed nos outros estados.

### Disponibilidade efectiva (`DEC-PROD-005`)

Conjunção fail-closed (todos verdadeiros):

1. tipo canónico no catálogo (`DEC-PROD-001`/`002`);
2. grupo + tipo activos (`DEC-PROD-003`);
3. regime / NIF elegível;
4. adesão FE ok quando o canal FE é exigido (`DEC-PROD-004`);
5. série AGT válida quando exigida;
6. ambiente correcto;
7. restrição sectorial satisfeita.

Qualquer falha ⇒ tipo **indisponível** para emissão nesse contexto.

### Routing SAF-T vs FE (`DEC-PROD-006`)

| Adesão FE | Canal do tipo | Endpoint FE AGT |
|---|---|---|
| ≠ `active` | SAF-T ou ambos | **proibido** — só fluxos SAF-T aplicáveis |
| `active` | ambos / FE-only | só se elegível no endpoint + autorizado contribuinte/série |
| qualquer | SAF-T-only | **nunca** |
| `active` | FE-only | permitido no canal FE (sem inventar L2/L3 SAF-T) |

### Conceito canónico e adaptadores (`DEC-PROD-007`)

- `CanonicalDocumentType` — identidade estável do catálogo (produto); **não** é código FE/SAF-T cru.
- Adaptadores por canal: canónico → L4 FE e/ou L2/L3 SAF-T conforme `DEC-PROD-006`.
- Pedido POS: **código próprio mapeado** → canónico; **proibido** código fiscal cru sem mapping.
- Mapping ausente / ambíguo / canónico sem adaptador para o canal ⇒ rejeitar ou indisponível.

### Credenciais, referências e autorização

- `ProducerCredential` — Basic Auth AGT do **produtor**; âmbito plataforma BWB + ambiente (não tenant/contribuinte); material só SecAdm/`SecretStore`.
- `AuthorityProfile` — configuração **pública** de preparação AGT por ambiente (`homologation`\|`production`): operações catalogadas, refs lógicas, readiness (`config_ready` / `secrets_ready` / `offline_validated` / `external_verified=false`); UI owner-only `/admin/ui/authority-profiles` (metadados sanitizados); **nunca** segredos (`DEC-BO-004`).
- `ProducerKeyRef` — referência ao par RSA do **produtor**; âmbito plataforma + ambiente.
- `TaxpayerKeyRef` — referência ao par RSA do **contribuinte**; âmbito contribuinte + ambiente; material no `SecretStore` da plataforma só se permitido por DEC-REG-KEY-CUSTODY.
- `TaxpayerKeyAuthorization` — evidência de autorização do contribuinte para custódia/uso (necessária; **não suficiente** sem permissão oficial AGT).

A aplicação persiste metadados das refs (fingerprint a partir de chave pública ou metadados seguros, versão, estado, validade, origem, ambiente, rotação/revogação). **Nunca** o segredo em texto claro, logs ou UI. Ver [backoffice-architecture.md](../02-architecture/backoffice-architecture.md).

### Configuração fiscal

`CountryFiscalProfile`, `TaxRegime`, `TaxCode`, `ExemptionReason`, `CanonicalDocumentType` / `DocumentType` (canónico), mapping código POS → canónico (`DEC-PROD-007`), `Series` (estabelecimento), activação grupos/tipos (`DEC-PROD-003`).

### Emissão

`DocumentIntent`, `FiscalDocument`, `FiscalLine`, `TaxSummary`, `PaymentSummary`, `CorrectionReference`, `FiscalArtifact`.

### Comunicação

`AuthoritySubmission`, `AuthorityAttempt`, `AuthorityResponse`, `WebhookDelivery`.

### Auditoria

`AuditEvent`, `RuleEvaluation`, `PackageVersion`, `ExportJob`, `SaftArtifact`.

Trilho **append-only** (`DEC-PROD-013`): sem apagar/alterar eventos; retenção final = norma consolidada (`pending_norm` até citação oficial).

## Invariantes críticas

- Um `FiscalDocument` possui país e versão do pacote fiscal imutáveis.
- `FiscalDocument` emitido não é atualizado; novas informações são eventos ou artefactos relacionados.
- Um número pertence a uma série e é único; séries pertencem a um `Establishment`.
- Adesão FE é estado do `Taxpayer` (+ ambiente), nunca booleano e nunca por estabelecimento (`DEC-PROD-004`).
- Disponibilidade efectiva = conjunção dos gates `DEC-PROD-005` (não basta activação POS).
- Tipo SAF-T-only nunca chama endpoint FE; FE-only só no canal FE; sem FE `active` só SAF-T aplicável (`DEC-PROD-006`).
- POS nunca envia código fiscal cru sem mapping; um canónico, adaptadores por canal (`DEC-PROD-007`).
- Única autoridade de emissão/numeração = módulo; N POS só via API (`DEC-PROD-008`, ADR-0001).
- Ciclo de vida: `sealed_locally` → `submitted` → `received` → `accepted` \| `rejected`; proibido afirmar aceitação AGT antes de `accepted` (`DEC-PROD-009`).
- Offline: outbox/reenvio/idempotência ok; **não** declarar emissão offline certificada até regra oficial (`DEC-PROD-010`).
- Edge: um único processo fiscal escritor; sem multi-instância na mesma série sem coordenação (`DEC-PROD-011`).
- Chaves: por contribuinte; non-exportable quando possível; nunca no POS; rotação auditada; definitivo AGT (`DEC-PROD-012`).
- Auditoria append-only; retenção final só com norma consolidada (`DEC-PROD-013`).
- Modelo de tipos = união canal SAF-T ∪ FE legalmente aplicável; faseamento ≠ truncar domínio (`DEC-PROD-014`).
- Catálogo: esquema mínimo `DEC-PROD-015` (grupo, canónico, canais, estrutura, elegibilidade, natureza, sector, série, requisitos, rectificação/anulação, estado normativo, activo).
- Uma chave de idempotência resolve sempre para o mesmo resultado ou conflito explícito.
- Totais são derivados/verificados a partir de linhas e impostos segundo regras versionadas.
- Resposta da AGT não substitui o pedido ou artefacto enviados; ambos são preservados.

## Dinheiro

Formato externo decimal em string/JSON number conforme contrato final, com limite de casas explícito. Formato interno decimal exato. Arredondamento definido por regra fiscal, nunca pelo padrão implícito da linguagem.
