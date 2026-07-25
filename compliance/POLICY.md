# Política de fontes fiscais (Angola)

## Precedência normativa

1. **Decreto Executivo n.º 74/19** e **Rectificação n.º 10/19** formam um conjunto normativo; requisitos derivados devem citar ambos quando aplicável.
2. **Legislação posterior**, incluindo o **Decreto Executivo n.º 683/25**, tem precedência quando o diploma o estabelecer.
3. Conflitos entre fontes registam-se em `compliance/derived/conflicts/` — não se resolvem por omissão nem por invenção.
4. Versões anteriores **não se apagam**; alteram-se `status` e ligações `supersedes` / `superseded_by`.

## Representações de diplomas PDF

| Representação | Papel |
|---|---|
| PDF original | Única fonte normativa imutável |
| PDF pesquisável (OCR) | Derivado de consulta |
| Markdown/TXT com marcadores de página | Derivado para pesquisa, Cursor e citações |

Nenhum derivado OCR/textual pode ser apresentado como substituto legal do PDF original. OCR e transcrição seguem a **mesma** decisão de redistribuição do PDF.

## Estados de derivados OCR

- `generated_unreviewed` — só pesquisa e localização preliminar
- `partially_reviewed` — pesquisa; não sustenta requisitos nas páginas não revistas
- `reviewed` — único estado que pode sustentar requisitos fiscais **confirmados**
- `rejected` — não usar; manter para auditoria

Mudança do SHA-256 do PDF original invalida todos os derivados anteriores.

## Faturação electrónica vs SAF-T

- Assinaturas **JWS/RS256** da FE são mecanismos distintos de qualquer cadeia/assinatura histórica SAF-T.
- Endpoints de **homologação** (`sifphml.minfin.gov.ao`) e **produção** (`sifp.minfin.gov.ao`) inventariam-se em separado.
- Campos assinados, `FE-RNG-*`, QR Code e regras criptográficas só a partir de fontes catalogadas — sem inventar.

## `local/` e Git

- `local/` é consulta não versionada (ver `.gitignore` e `AGENTS.md`).
- Build, testes, runtime e CI **nunca** apontam para `local/`.
- Originais B1 (PDFs DR + HTML FE + proveniência) sincronizados em `storesace-cv/bwb-fiscal-sources-ao` (privado; `storage=private_sync` no catálogo). Acesso privado ≠ redistribuição pública.
- Se a redistribuição pública do Diário da República não for autorizada, originais e derivados OCR futuros permanecem em armazenamento privado sincronizado; o Git público mantém catálogo, hashes, proveniência e referências.

## Experiência cross-project

Conhecimento técnico proveniente de outras aplicações internas (ex. inventário B0) pode informar testes e desenho, mas **não** constitui fonte normativa. Hipóteses operacionais (incluindo compatibilidade de namespaces com validadores AGT) não geram requisitos `AO-*` nem autofix até evidência oficial. Ver [AUD-B0](../docs/01-compliance/audits/AUD-B0-SAFTAO-CROSS-PROJECT-REUSE.md).

## Segurança

Proibido versionar ou registar no catálogo: chaves privadas, certificados secretos, tokens, palavras-passe, Basic Auth reais ou dados pessoais/fiscais desnecessários.
