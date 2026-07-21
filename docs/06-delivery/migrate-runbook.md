# Runbook de migrations (cloud)

## Produção

Comando suportado: `fiscal-migrate` com **apenas**:

- `up` — aplica migrations forward embutidas
- `version` — mostra versão e dirty flag

```bash
export FISCAL_DATABASE_DRIVER=postgres
export FISCAL_DATABASE_URL='postgres://…/fiscal?sslmode=require'
fiscal-migrate up
fiscal-migrate version
```

A API **não** corre migrate no arranque.

Controlo: tabela `public.bwb_schema_migrations`. Schema da aplicação: `fiscal`.

## Dirty state (manual)

Se `version` reportar `dirty=true`:

1. Parar deploys que escrevam schema.
2. Investigar a migration falhada e o estado da BD (backup primeiro).
3. Restaurar backup **ou** corrigir manualmente sob revisão.
4. Usar recuperação `force` **apenas** via procedimento manual fora da CLI de produção, com confirmação explícita de dois operadores.
5. Aplicar migration corretiva nova (forward-only). Não editar migrations já publicadas.

## Edge

Migrate no processo único Edge fica para incremento posterior.
