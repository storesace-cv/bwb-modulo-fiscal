# PT-CERT REST API — notas POS_API (vendor)

| Campo | Valor |
|---|---|
| source_id | `VENDOR-PTCERT-REST-API` |
| SHA-256 | `52e3408058cf4ca707b00daaa2cdacafac94d9874a89e2268517f41370590465` |
| Versão doc | V0.1 — 2021-12-01 (Opala Consult) |
| Páginas | 19 |
| Privado | `originals/vendor-integrations/VENDOR-PTCERT-REST-API/original/PTCERT_REST_API.pdf` @ `a889ef6…` |
| Papel | técnico de fornecedor POS PT — **≠** AGT Angola |

## Contrato local HTTP `:3041`

- **Pré-requisito de licença:** módulo `POS_API` (campo em `license_info`); sem ele a API não arranca (texto do PDF).
- **Base:** `http://<ip_pos>:3041/api/…` — **HTTP em claro**, porto **3041**, tipicamente LAN do POS.
- **Implicação edge-connector:** qualquer cliente tem de correr **junto da rede do POS** (Edge/appliance), não como chamadas cloud→AGT. Não expor `:3041` à Internet.

## Inventário de rotas (PDF; sanitizado)

| Método | Rota | Notas |
|---|---|---|
| GET | `/api/sync_md5` | hash de sync |
| GET | `/api/license_info` | string de licença (incl. flags de módulos) |
| GET | `/api/areas` | **array posicional** por área |
| GET | `/api/areas_friendly` | objectos nomeados (preferível) |
| GET | `/api/checks?area=` | área = índice 0-based |
| GET | `/api/check_get` | detalhe de conta; **linhas posicionais** |
| GET | `/api/customer_by_nif` | lookup cliente |
| GET | `/api/check_update` | actualiza conta (query) |
| POST | `/api/clerks` | empregados |
| POST | `/api/customers` | clientes |
| POST | `/api/products_tree` | árvore produtos |
| POST | `/api/products_soldout` | ids esgotados |
| POST | `/api/products` | produtos (filtros JSON) |
| GET | `/api/products_extra_data` | extras |
| GET | `/api/groups` / `/api/departments` | agrupamentos |
| GET | `/api/image` | ficheiros de imagem |
| POST | `/api/stock_in` / `/api/stock_out` | stock |
| GET | `/api/stock_current` | stock actual |
| POST | `/api/daily_special_qty*` | especiais do dia |
| GET | `/api/db_table?table=` | **leitura genérica de tabela** |
| POST | `/api/check_extra_data_update` | extras da conta |
| POST | `/api/check_create` | **cria conta/pedido no POS** (+ print / save ficheiro) |
| GET | `/api/file_get` / `/api/file_delete` | ficheiros no servidor API |

Exemplos do PDF usam NIFs/IPs ilustrativos — **omitidos** aqui; não tratar como dados reais.

## Fragilidade de payload posicional

- `areas` devolve listas em que a semântica depende da **posição** do elemento (nome, nº mesas, `sell_price*`, `sell_vat*`, flags…), não de chaves JSON estáveis.
- `check_get` → `details` como arrays aninhados de strings/números sem schema nomeado no extracto.
- `areas_friendly` mitiga parcialmente para áreas; contas/linhas continuam frágeis.
- **Fail-closed:** não mapear índices «por convenção» sem contrato versionado do fornecedor + testes; mudanças de firmware partem integrações.

## Allowlist e riscos de segurança

| Risco | Evidência no PDF | Mitigação exigida (futuro conector) |
|---|---|---|
| HTTP sem TLS | base `http://…:3041` | só LAN/VPN; nunca Internet; preferir túnel |
| Sem auth documentada | todas as rotas sem token/Bearer | allowlist de origem; firewall host; mTLS edge se existir |
| `db_table` | `?table=products` (genérico) | **deny por omissão**; allowlist de tabelas |
| `file_get` / `file_delete` | nome de ficheiro em query | path allowlist; sem `..`; sem dados fiscais em logs |
| `check_create` | cria operação no POS + ficheiros `.order` | **não** usar como emissão AGT |
| Superfície ampla | 28 rotas | allowlist de endpoints no edge-connector |

## Autoridade de emissão / dupla emissão

**Decisão (produto):**

1. Para Angola / AGT, o **módulo fiscal BWB** é a autoridade de numeração e selagem.
2. PT-CERT é software POS certificado no contexto **português** do fornecedor; a API `POS_API` opera **sobre o POS** (contas, stock, impressão).
3. Usar `check_create` / fluxos de documento do PT-CERT **em paralelo** com emissão no módulo BWB cria risco de **dupla emissão** / números fiscais incoerentes.
4. Integração futura permitida apenas como **edge sync operacional** (leitura de catálogo/estado, nunca atribuição do número fiscal AO no POS).
5. **Não** implementar conector live neste incremento.

## Relação com `AO-*`

Nenhuma. Documentação auxiliar de integração Edge; não cita legislação angolana.
