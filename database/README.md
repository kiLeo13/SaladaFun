# Database migrations

This directory is the ecosystem-wide home for ordered SQL migrations. Padinho
uses [Goose](https://github.com/pressly/goose) and applies pending migrations
before opening its Discord gateway, so an incompatible deployment fails closed.

Place migrations in `migrations/` using sequential names such as
`00001_create_guild_settings.sql`. Each file must contain a `-- +goose Up`
section and should contain a reversible `-- +goose Down` section when the
operation can be safely reversed. MySQL migration DSNs must include
`parseTime=true&multiStatements=true`.

Run migrations manually with the same Padinho image:

```sh
docker compose -f padinho/compose.yaml run --rm --entrypoint /app/migrate bot
```

Never put credentials in this directory. The runtime reads `DATABASE_DSN` from
the host-managed environment file.
