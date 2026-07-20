# Plano de acesso e preservação de fontes oficiais

## Resultado da verificação

Temos acesso público direto à documentação técnica online da Facturação Electrónica e a páginas oficiais da AGT/MINFIN. O acesso ao processo do produtor, Modelo 8, XSD oficial e eventuais pacotes de homologação depende do registo da empresa e de credenciais emitidas pela AGT.

Não é possível garantir antecipadamente acesso a uma área restrita sem essas credenciais. É possível garantir que o projeto está preparado para receber, validar, versionar e usar os artefactos oficiais assim que forem disponibilizados.

## Inventário de acesso

| Recurso | Acesso atual | Uso no projeto |
|---|---|---|
| Documentação API FE | Público | Contratos, conector AGT e testes |
| Portal/Quiosque AGT | Público, sujeito a manutenção | Comunicados, FAQs e guias |
| Portal do Parceiro | Público + áreas autenticadas | Guias, modelos, testes e adesão |
| Decreto 74/19 e rectificação | Público, PDF oficial a arquivar | Regras funcionais e criptográficas |
| Modelo 8 | Por confirmar em sessão autenticada | Processo de certificação |
| XSD SAF-T (AO) oficial | Restrito segundo informação do projeto; confirmar | Gerador e validador SAF-T |
| Credenciais API homologação | Pedido formal à AGT | Testes de integração |

## Passos organizacionais

1. Confirmar a entidade legal produtora e respetivo NIF angolano.
2. Criar responsáveis nomeados para portal, compliance e chaves.
3. Registar a empresa como produtora de software.
4. Confirmar no portal autenticado o Modelo 8 e a lista documental atual.
5. Pedir credenciais de homologação à AGT pelo canal oficial publicado.
6. Descarregar XSD, manuais, modelos, catálogos e exemplos oficiais.
7. Registar data, origem, versão, hash SHA-256 e responsável por cada artefacto.
8. Executar validação independente do XSD e criar vetores dourados.
9. Rever fontes antes de cada release fiscal e antes da submissão à AGT.

## Estrutura recomendada para o repositório privado

```text
compliance/
  sources-manifest.yaml
  public/
    legislation/
    api-docs/
    schemas/
  restricted/          # não versionar sem aprovação jurídica e controlo de acesso
  extracted-requirements/
  evidence/
```

O manifesto pode entrar em Git. Ficheiros restritos, credenciais, chaves privadas e dados de teste reais ficam fora do repositório normal.

## Campos do manifesto

- `id`
- `title`
- `issuer`
- `source_url`
- `published_at`
- `retrieved_at`
- `effective_from`
- `version`
- `sha256`
- `access_class`: public/restricted
- `supersedes`
- `status`: current/superseded/pending-verification
- `requirements`

## Gate para começar o motor fiscal

Podemos criar infraestrutura, modelo canónico, idempotência e simulador com documentação pública. Não devemos declarar concluídos assinatura legal, gerador SAF-T ou conformidade de produção até termos:

- PDF oficial do Decreto 74/19 e rectificação;
- XSD SAF-T (AO) oficial aplicável;
- especificação técnica versionada da API;
- credenciais/ambiente de homologação;
- resposta dos testes oficiais da AGT.
