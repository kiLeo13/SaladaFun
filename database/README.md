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

Padinho requires the non-null Discord token, birthday announcement channel, and
voice activity log channel before it can start:

```sql
INSERT INTO config (name, value) VALUES ('app.token', 'replace-me');
INSERT INTO config (name, value) VALUES ('birthday.channel_id', 'replace-me');
INSERT INTO config (name, value) VALUES ('channels.logs.voice', 'replace-me');
INSERT INTO config (name, value) VALUES ('urls.youtube.betaSpirit', 'https://www.youtube.com/watch?v=TI0f3xA5_gg');
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

Migration `00006_quotes.sql` adds `quote_authors` and `quotes` for Padinho's
random quote command. Authors have one canonical name and may optionally map to
one Discord snowflake; this is an identity attribute, not a foreign key to a
shared users table. Quotes retain their original text, an optional Brazilian
Portuguese translation, optional provenance URL, and an enabled flag so a quote
can be withheld without deleting it. Importers must resolve spelling variants
to a canonical author before inserting quotes.

Migration `00007_discord_account_links.sql` adds the sparse
`discord_account_links` hierarchy. Each link has an auto-increment database ID
and a unique `user_id`; `parent_id` is nullable and references the
direct parent snowflake, so a parent may itself have a parent. A foreign key
rejects missing parents, and a check constraint rejects direct self-parenting.
The table deliberately does not represent unlinked singleton accounts. It does
not prevent longer cycles; any future reparenting feature must validate that the
new parent is not in the moved account's descendant subtree before writing.

Migration `00008_voice_activity_logs.sql` adds Padinho's append-only voice
activity delivery ledger. It stores the guild and member snowflakes, nullable
old/new channel snowflakes that encode join, leave, or move, the final `SENT`
or `FAILED` delivery result, and the observed UTC millisecond timestamp. The
schema rejects invalid statuses, channel-less rows, and unchanged channel pairs.

Migration `00009_letterboxd.sql` adds global TMDB title metadata, personal
wishlists, one rated watch per member/title, and append-only review revisions.
Provider identity and membership/watch uniqueness are enforced by MySQL; review
withdrawal is a new NULL-text revision. Apply manually before deploying feature
wiring. See [Padinho architecture](../discord/padinho/ARCHITECTURE.md) for
transaction semantics and isolated live-MySQL verification.


For the Letterboxd release, apply through migration `00009` and provision the
required `tmdb.token` config value before publishing the dependent bot image.
Watchtower follows `latest` automatically. The API Read Access Token is entered
through a trusted private SQL session; do not store it in this directory or in
command history. Padinho still requires only the four existing DML grants.
Application rollback retains these additive tables and the immutable review
ledger rather than executing destructive Down migrations.

Migration `00010_letterboxd_metadata.sql` enriches the Letterboxd title cache with
creators and genres as NOT NULL JSON arrays, a homepage URL, and an unsigned
season count. These columns have no defaults; the application supplies every
value, including empty arrays, an empty homepage, and zero seasons when needed.
JSON array checks reject objects/scalars. This unreleased migration assumes
`letterboxd_titles` is empty; it performs no backfill or cache invalidation.
Before applying it to the disposable development data, clear related records
in foreign-key order: `letterboxd_reviews`, `letterboxd_watches`,
`letterboxd_wishlist_items`, then `letterboxd_titles`. The migration itself
contains no deletion statements. Apply before deploying the rich title card.

Migration `00011_letterboxd_future_metadata.sql` reserves the first batch of
future metadata on **`letterboxd_titles` only**. It adds no tables, indexes, or
JOINs and does not change existing values or expire the metadata cache. The bot
does not fetch or render these new fields yet.

| Column | Storage | Intended meaning |
| --- | --- | --- |
| `content_ratings` | Nullable JSON array | Entries such as `{"country":"BR","certification":"14","release_type":3}`; TV entries omit movie-only release type |
| `runtime_minutes` | Nullable unsigned small integer, positive when present | Movie runtime in minutes |
| `episode_runtime_minutes` | Nullable JSON array | TV runtime alternatives, for example `[45,50]` |
| `original_language` | Nullable `VARCHAR(10)` | Provider language code |
| `origin_countries` | Nullable JSON array | Country codes, for example `["BR","US"]` |
| `status` | Nullable `VARCHAR(64)` | Provider status text without a fixed database enum |
| `episode_count` | Nullable unsigned integer | Total TV episodes; zero is allowed when known |
| `external_ids` | Nullable JSON object | Provider IDs, for example `{"imdb":"tt1234567","tvdb":"123"}` |

All new fields default to SQL `NULL`, meaning unknown or not fetched. Once
ingestion is implemented, empty arrays/objects may represent a successfully
fetched empty result. Classification codes remain strings and retain their
country; missing classification never means unrestricted. JSON checks enforce
outer array/object shapes; future ingestion must validate entry structure,
codes, positive episode runtimes, and external-ID values.

**Migration 00011 alone requires no record deletion.** Existing title metadata,
`letterboxd_wishlist_items`, `letterboxd_watches`, and `letterboxd_reviews` are
preserved. Goose updates its own migration ledger automatically. Apply the
embedded migration executable through the existing manual process above.
The current application can continue inserting titles without these fields.

Live validation: set the `TEST_DATABASE_*` connection variables and run
`go test -race ./...` from `database/`. Tests cover an empty version-9 catalog
upgrade without metadata defaults, a populated version-10 upgrade, preserved
relationships and metadata, SQL NULL defaults,
legacy inserts, valid metadata round trips, empty results, invalid JSON shapes,
numeric constraints, and repeated migration execution.
