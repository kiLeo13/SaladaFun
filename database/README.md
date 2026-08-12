# Database migrations

This directory is the ecosystem-wide migration project. It independently owns
the Goose executable, database connection used by migrations, and ordered SQL
history. Applications such as Padinho consume the resulting schema but never
create schemas or apply migrations themselves.

Place migrations in `migrations/` using sequential names such as
`00001_initial.sql`. Each file must contain a `-- +goose Up` section and should
contain a reversible `-- +goose Down` section when the operation can be safely
reversed. The migration adapter enables MySQL time parsing and multi-statement
support internally.

Run migrations through the Salada image:

```sh
docker compose -f discord/padinho/compose.yaml run --rm migrate
```

Padinho requires the non-null `app.token` value before it can connect to
Discord:

```sql
INSERT INTO config (name, value) VALUES ('app.token', 'replace-me');
```

Run that statement through a private, trusted MySQL session so the token does
not leak into shell history or logs. Values are stored as plain text; restrict
table access and protect database backups accordingly.

Never put credentials in this directory. The host-managed environment supplies
`DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_USERNAME`, `DATABASE_PASSWORD`, and
`DATABASE_NAME` separately; callers never assemble or provide a DSN.

Verify this project independently with `go test -race ./...`. Integration tests
run against MySQL when the corresponding `TEST_DATABASE_*` variables are set.
