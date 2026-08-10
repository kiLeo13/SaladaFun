# Database migrations

This directory is the ecosystem-wide home for ordered SQL migrations. Padinho
uses [Goose](https://github.com/pressly/goose) and applies pending migrations
before opening its Discord gateway, so an incompatible deployment fails closed.

Place migrations in `migrations/` using sequential names such as
`00001_create_guild_settings.sql`. Each file must contain a `-- +goose Up`
section and should contain a reversible `-- +goose Down` section when the
operation can be safely reversed. The Go database adapter enables MySQL time
parsing and multi-statement migration support internally.

Run migrations manually with the Salada image:

```sh
docker compose -f discord/padinho/compose.yaml run --rm --entrypoint /app/migrate bot
```

Never put credentials in this directory. The host-managed environment supplies
`DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_USERNAME`, `DATABASE_PASSWORD`, and
`DATABASE_NAME` separately; callers never assemble or provide a DSN.
