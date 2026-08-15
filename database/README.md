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

Padinho requires the non-null Discord token and birthday announcement channel
before it can start:

```sql
INSERT INTO config (name, value) VALUES ('app.token', 'replace-me');
INSERT INTO config (name, value) VALUES ('birthday.channel_id', 'replace-me');
```

Run that statement through a private, trusted MySQL session so the token does
not leak into shell history or logs. Values are stored as plain text; restrict
table access and protect database backups accordingly.

Never put credentials in this directory. The migration executable reads
`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, and `DB_NAME` directly; callers
never assemble or provide a DSN.

Verify this project independently with `go test -race ./...`. Integration tests
run against MySQL when the corresponding `TEST_DATABASE_*` variables are set.

The initial schema also owns `birthdays` and `birthday_announcements`. Birthday
dates use MySQL `DATE`; timezone values are IANA names such as
`America/Sao_Paulo`; Discord snowflakes use `BIGINT UNSIGNED`; and audit times
use Unix milliseconds in unsigned big integers. The announcement table is a
per-user, per-local-date delivery ledger with cascading cleanup.
