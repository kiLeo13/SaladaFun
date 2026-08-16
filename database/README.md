# Database migrations

This directory is the ecosystem-wide migration project. It independently owns
the Goose executable, database connection used by migrations, and ordered SQL
history. Applications such as Padinho consume the resulting schema but never
create schemas or apply migrations themselves. SQL files are embedded into the
executable so deployment requires only one binary.

Place migrations in `migrations/` using sequential names such as
`00001_initial.sql`. Each file must contain a `-- +goose Up` section and should
contain a reversible `-- +goose Down` section when the operation can be safely
reversed. The migration adapter enables MySQL time parsing and multi-statement
support internally.

Build the Linux executable locally from this directory:

```powershell
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -trimpath -ldflags="-s -w" -o bin/salada-migrate ./cmd/migrate
```

Upload and run it on the Padinho VM, which is the only host allowed to reach
private MySQL:

```sh
scp ./bin/salada-migrate ubuntu@PADINHO_PUBLIC_IP:/tmp/salada-migrate
ssh ubuntu@PADINHO_PUBLIC_IP
sudo install -o root -g root -m 0750 /tmp/salada-migrate /opt/salada/salada-migrate
sudo sh -c 'set -a; . /etc/salada/salada.env; set +a; /opt/salada/salada-migrate'
```

The executable creates `DB_NAME` when it does not exist and then applies every
pending embedded migration. Run it deliberately before deploying Padinho code
that depends on a new schema; neither GitHub Actions nor Compose runs it.

Padinho requires the non-null Discord token and birthday announcement channel
before it can start:

```sql
INSERT INTO config (name, value) VALUES ('app.token', 'replace-me');
INSERT INTO config (name, value) VALUES ('birthday.channel_id', 'replace-me');
```

Run that statement through a private, trusted MySQL session so the token does
not leak into shell history or logs. Values are stored as plain text; restrict
table access and protect database backups accordingly.

Never put credentials in this directory or upload an environment file with the
binary. The migration executable reads
`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, and `DB_NAME` directly; callers
never assemble or provide a DSN.

Verify this project independently with `go test -race ./...`. Integration tests
run against live MySQL when the corresponding `TEST_DATABASE_*` variables are
set and verify the embedded migration path.

The initial schema also owns `birthdays` and `birthday_announcements`. Birthday
dates use MySQL `DATE`; timezone values are IANA names such as
`America/Sao_Paulo`; Discord snowflakes use `BIGINT UNSIGNED`; and audit times
use Unix milliseconds in unsigned big integers. The announcement table is a
per-user, per-local-date delivery ledger with cascading cleanup.
