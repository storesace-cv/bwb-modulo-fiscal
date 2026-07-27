# Baseline de segurança

## Identidade e acesso

- Autenticação forte para portal e credenciais distintas para máquinas.
- Autorização por tenant, contribuinte, estabelecimento e capacidade.
- Privilégio mínimo e segregação entre suporte, operação e gestão de chaves/credenciais.
- Credenciais de homologação nunca funcionam em produção; ambientes HML e PRD rigorosamente isolados.

### Admin API RBAC (DEC-BO-002 / RM-BO-004)

Contrato OIDC/JWT distinto do POS Bearer. Papéis: `owner` | `admin` | `operator` | `auditor`.
Matriz canónica em código: [`internal/adminauth/rbac.go`](../../internal/adminauth/rbac.go).

| Permissão | owner | admin | operator | auditor |
|---|---|---|---|---|
| `cadastro.write` | sim | sim | não | não |
| `cadastro.read` / `ops.read` / `audit.read` / `secret_meta.read` | sim | sim | sim | sim |
| `secadm.write` (Put/Rotate/Revoke) | **sim** | não | não | não |
| Leitura de plaintext de segredo | **não existe** | — | — | — |

- **OIDC/JWT (RM-BO-006 / DEC-BO-003):** `FISCAL_ADMIN_AUTH_MODE=oidc_jwt` — Bearer + JWKS https; `iss`/`aud` exactos; alg allowlist; mapa de grupos→roles; `owner` só com subject allowlist; fail-closed sem config. Tokens nunca em logs.
- **Sessão UI (RM-UI-005):** cookie opaca `fiscal_admin_session` (HttpOnly, SameSite=Strict, Secure fora de development); mint via `POST /admin/ui/auth/session` com Bearer validado no servidor; logout com CSRF; JWT não persiste no browser.
- **Observabilidade admin (RM-BO-007):** logs `admin_request` e métricas só com `request_id` / `route_class` / `roles` allowlist / `auth_mode` / outcome — **proibido** tokens, cookies, DSN, JWKS, chaves, NIF, subject ou payloads. SecAdm: métricas de classe `secadm` sem plaintext. Ver [`docs/07-operations/admin-observability.md`](../07-operations/admin-observability.md).
- **MFA interactivo / redirect IdP:** exigido em produção com IdP real; redirect authorize ainda não ligado. Sem login local improvisado.
- Operadores ≠ owner SecAdm; metadados sanitizados ≠ material secreto.

## Segredos e chaves

- Abstração `SecretStore` (Secret Manager / KMS / HSM conforme fornecedor ainda por decidir).
- Em produção, provisionamento de segredos na **zona dedicada de administração de integração** (`DEC-BO-001` plano B): TLS autenticado, write-only, gravação direta no `SecretStore`, sem persistência intermédia, sem logs do segredo, sem retorno nem visualização posterior; acesso exclusivo do owner.
- A UI do **backoffice funcional** (plano A) não recebe, armazena nem exibe material secreto; mostra apenas metadados sanitizados (ambiente, estado, fingerprint, validade, algoritmo, key-id, timestamps, última verificação) — `DEC-BO-004` / `AuthorityProfile` em `/admin/ui/authority-profiles` (owner-only).
- Preparação para certificados/auth AGT: config pública no perfil; PEM/PKCS#12/credenciais **só** SecAdm → `SecretStore`; `external_verified` permanece `false` sem probe AGT real.
- **Proibida** cópia automática de chaves privadas cloud→Edge ou Edge→cloud; qualquer provisionamento é explícito, individual, autenticado e auditado.
- Endpoints públicos documentados podem ser configuração técnica; overrides privados (URLs, credenciais) ficam no cofre operacional — nunca no backoffice comum.
- Custódia da chave privada do **contribuinte** no `SecretStore` da plataforma condicionada a autorização do contribuinte **e** a DEC-REG-KEY-CUSTODY (permissão oficial AGT).
- Chaves de teste do vertical slice: par RSA efémero, privada nunca persistida nem no Git.
- Rotação, revogação, expiração e inventário de refs auditados.
- Artefactos e atualizações Edge assinados.

## Criptografia

- TLS em trânsito e cifra em repouso.
- Edge: keystore próprio e política separada do cloud.
- Assinatura interna da API (se aplicável) distinta da assinatura fiscal AGT (esta depende de fontes oficiais).

## Aplicação

- Validação estrita de schema e limites de payload.
- Proteção contra replay e abuso.
- Dependências fixadas, SBOM e análise de vulnerabilidades.
- Segredos fora do repositório e das imagens.
- Logs sem dados fiscais completos ou segredos.

## Disponibilidade

- Backups cifrados, testados e com retenção definida.
- RPO/RTO aprovados antes do piloto.
- Testes de recuperação, perda de rede, disco cheio, relógio incorreto e corrupção.

## Auditoria

Eventos devem incluir ator, ação, recurso, instante, origem, correlação e resultado, evitando dados excessivos. Acesso de suporte a informação fiscal é justificado e auditado; suporte não acede a material secreto nem a dados de outro tenant.
