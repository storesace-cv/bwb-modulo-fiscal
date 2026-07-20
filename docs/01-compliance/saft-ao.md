# SAF-T (AO)

## Objetivo

Gerar um ficheiro SAF-T (AO) determinístico, completo e validável a partir do livro fiscal, nunca a partir de reconstruções parciais do POS.

## Decisões iniciais

- Persistir desde a emissão todos os campos necessários ao SAF-T.
- Versionar schema, mapeamentos e regras de exportação.
- Gerar por empresa e período fiscal, com parâmetros auditáveis.
- Guardar hash, utilizador, instante, versão do gerador e resultado da validação.
- Não corrigir silenciosamente dados durante a exportação.
- Separar erro de dados, erro de configuração e erro de schema.

## Pipeline

1. Fechar o conjunto lógico do período.
2. Ler snapshot consistente do livro fiscal.
3. Mapear entidades e documentos.
4. Gerar XML determinístico.
5. Validar contra XSD e regras semânticas.
6. Produzir relatório e hash do ficheiro.
7. Disponibilizar exportação com autorização e auditoria.

## Testes obrigatórios

- documento normal, anulado e retificativo;
- cliente sem NIF nos casos permitidos;
- impostos, isenções e múltiplas taxas;
- sequências e séries anuais;
- períodos grandes e limites de memória;
- caracteres Unicode e escaping XML;
- ficheiro inválido por ausência de dados;
- comparação com vetores aprovados.
