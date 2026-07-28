# `migrate`

One-shot command for applying the PostgreSQL schema migrations from the embedded SQL files under `migrations`.

## Behavior

- Loads `.env` unless `SKIP_ENV_AUTO_LOAD=true`.
- Reads `POSTGRES_*` environment variables into `pkg/pg.Config`.
- Opens PostgreSQL with `pkg/pg`.
- Applies all embedded migrations with `pkg/migrate`.
- Exits after reporting success or the first error.

## Environment

Required PostgreSQL variables:

- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DATABASE`
- `POSTGRES_SSLMODE`

## Example

```bash
go run ./cmd/migrate
```
