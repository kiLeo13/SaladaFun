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
sudo -i
read -r -p 'MySQL host: ' DB_HOST
read -r -p 'MySQL administrator: ' DB_USER
read -r -s -p 'MySQL administrator password: ' DB_PASSWORD
export DB_HOST DB_USER DB_PASSWORD DB_PORT=3306 DB_NAME=salada
/opt/salada/salada-migrate
unset DB_HOST DB_USER DB_PASSWORD DB_PORT DB_NAME
```

The executable creates `DB_NAME` when it does not exist and then applies every
pending embedded migration. Run it deliberately before deploying Padinho code
that depends on a new schema; neither GitHub Actions nor Compose runs it.
The administrator credential is entered interactively for this maintenance
session and is never written to Padinho's environment.

After migrating, create Padinho's separate runtime account through the same
trusted private MySQL session. Restrict it to DML from the VM subnet and use the
same password stored in the encrypted Ansible variables:

```sql
CREATE USER IF NOT EXISTS 'padinho'@'10.42.10.%' IDENTIFIED BY 'replace-me';
ALTER USER 'padinho'@'10.42.10.%' IDENTIFIED BY 'replace-me';
GRANT SELECT, INSERT, UPDATE, DELETE ON salada.* TO 'padinho'@'10.42.10.%';
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

Migration `00002_user_preferences.sql` adds the shared `users_preferences`
table. Each Discord user has at most one row. Feature-owned Boolean columns such
as `auto_mudae_oc` are nullable so `NULL` can retain that feature's application
default without forcing unrelated modules to populate every preference. Its
audit timestamps use the same Unix-millisecond convention as the initial
schema.

Migration `00003_user_preferences_ouroquest.sql` adds nullable
`auto_mudae_oq`; `NULL` preserves the Ouroquest module's enabled-by-default
automatic-assistance behavior.

Migration `00005_remove_ouroharvest_preference.sql` removes the abandoned
`auto_mudae_oh` column. Migration `00004` remains in the ordered history so
databases that previously applied it can safely advance to the final schema.
