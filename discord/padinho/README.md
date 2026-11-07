# Padinho

Padinho is SaladaFun's Go 1.26 Discord bot for a small private guild.

## Commands

| Command | Behavior |
| --- | --- |
| `/birthdays [month] [full-date]` | Monthly birthday calendar; defaults to the process's current month and hides birth years unless requested. Navigation preserves that choice. |
| `/move-all destination: [origin:]` | Moves everyone in the origin voice channel; defaults to the caller's channel. Discord evaluates each move, including destination capacity. |
| `!quote [id]` | A positive decimal ID returns that exact quote, including disabled entries; otherwise picks an enabled random quote. Unknown IDs report not found. |
| `!childrentree [@user-or-id]` | Shows the complete linked-account tree above the requested account, defaulting to the caller. Only the first argument is used. |
| `!betaspirit` | Returns the configured BetaSpirit YouTube URL. |
| `!toggleochelper`, `!toggleoqhelper` | Toggle automatic Mudae assistance independently for the caller; both default to enabled. |
| `!ochelper`, `!oqhelper` | Start manual assistance by replying to a current Mudae board, regardless of the automatic preference. |

Movie tracking commands are described [below](#movie-and-tv-tracker).
Literal message commands match the first token case-insensitively and accept
repeated whitespace between arguments. Entity arguments accept raw IDs or
Discord mention markup.

The birthday calendar includes self-inspection and the next upcoming birthday
in each member's local timezone without notifying them. Server managers can add
a registration for a selected member; administrators can edit an existing
registration's name, date, timezone, or custom message through prefilled modals.
The Discord user ID is immutable. Dates use `DD/MM/AAAA`; timezone choices are
Brasilia, Amazonas, and UTC. Announcements run at UTC hour boundaries and deliver
once per local date as plain text with Discord's normal mention behavior.

Mudae helpers correlate exact `$oc`/`$ourochest` and `$oq`/`$ouroquest` requests,
solve locally, and update a reply as reveals arrive. Helper replies disappear on
completion, source deletion, or three idle minutes. Automatic assistance and
manual activation share the same solver. `$oh`/`$ouroharvest` boards are ignored.

Quotes prefer Portuguese translations and render two lines: a block quote and
author attribution. Linked authors permit only user mentions; role/everyone and
reply-author mentions are suppressed. Account trees do not ping members.
Voice joins, moves, and leaves produce log embeds and a `SENT`/`FAILED` database
record for each delivery attempt, with no application retry.

## Gateway lifecycle

DiscordGo's internal gateway reconnect is deliberately disabled. A gateway
disconnect is treated as a process-fatal condition rather than attempting to
reuse a session whose WebSocket state, goroutines, or locks may be unhealthy.
`Gateway.Run` also limits the initial open attempt to 60 seconds. Either failure
returns to `cmd/padinho`, which logs the error and exits non-zero. The bot
service's Compose `restart: unless-stopped` policy then creates a new process,
DiscordGo session, and WebSocket connection. SIGINT and SIGTERM remain graceful
shutdown paths and do not report a gateway failure.

## Database configuration

`internal/database.Open()` reads `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`,
and `DB_NAME` directly and returns `*gorm.DB`. Optional pool limits use
`DB_MAX_OPEN`, `DB_MAX_IDLE`, and `DB_MAX_LIFETIME`. There is no environment
configuration struct and callers never provide a DSN.

`cmd/padinho` passes GORM directly to `internal/config`, loads `app.token`,
`birthday.channel_id`, `channels.logs.voice`, and the Mudae settings below, and then constructs Discord. Padinho derives its
application ID after connecting and always synchronizes global commands; there
are no Discord environment switches. When an announced birthday has no custom
message, Padinho reads `birthday.defaultMessage` from `config` and applies
`{age}`, `{name}`, and `{mention}` before sending it.

Schema creation and migration belong exclusively to the root
[`database`](../../database/README.md) project. Build its self-contained Linux
executable locally, upload it to the Padinho VM, and run it there before
deploying code that requires a new schema. Compose never applies migrations.
Apply all pending migrations before deploying dependent bot code.
Insert the following values through a trusted private database session. Mudae's
emoji values are custom emoji IDs only, without names or Discord `<:...:...>`
markup:

```text
app.token
birthday.channel_id
channels.logs.voice
birthday.defaultMessage
urls.youtube.betaSpirit
tmdb.token
bots.mudae.id
bots.mudae.oc.emoji.blue
bots.mudae.oc.emoji.teal
bots.mudae.oc.emoji.green
bots.mudae.oc.emoji.yellow
bots.mudae.oc.emoji.orange
bots.mudae.oc.emoji.red
bots.mudae.oq.emoji.purple
```

Enable the privileged **Message Content Intent** for Padinho in Discord's
Developer Portal. The gateway also requests `GUILD_MESSAGES`; both are needed
to receive user commands and Mudae's message components.

Padinho never infers `$oc` from an arbitrary 5-by-5 button grid and never adds
a reaction as a confirmation step. Automatic assistance requires correlation
with an exact recent `$oc`/`$ourochest` command and therefore has an invoking
user whose preference can be respected. Exact `$oh` and `$ouroharvest`
commands are tracked separately and ignored. A numeric suffix from 1 through
10 correlates that many consecutive boards. Correlations expire after five
seconds. Mudae responses reporting that the user has no `$oc` available or is
waiting for its recharge cancel the failed correlation before a subsequent
`$oh` board can consume it.

Message-command triggers include their prefix in the registered literal. They
are case-insensitive, must be the first complete whitespace-separated token,
and currently expose `!toggleochelper`, `!ochelper`, `!toggleoqhelper`,
`!oqhelper`, `!quote`, and `!betaspirit`. `!betaspirit` replies with the URL
stored in `urls.youtube.betaSpirit`. The first `!quote` argument is interpreted
only when it is a positive decimal quote ID; otherwise it is ignored. Arguments use Go's
`strings.Fields` behavior, so repeated spaces and tabs do not create empty
values.

When a message command accepts a Discord entity argument, it extracts the
numeric snowflake and resolves the expected entity by ID. This accepts both
mentions and raw IDs without relying on the event's mention collection.

## Verification and container

Run the Go checks from `discord/padinho`. Automated Discord/TMDB tests use
local fixtures; persistence checks require a migrated, isolated live MySQL
database and the `TEST_DATABASE_*` environment variables listed below.

```sh
go test -race ./...
go vet ./...
go build ./...
```

From the repository root, build the image:

```sh
docker build -f discord/padinho/Dockerfile -t salada:local .
```

The image contains only Padinho;
migrations remain an independently built manual tool. See
[ARCHITECTURE.md](ARCHITECTURE.md) for the detailed boundaries.

The `Padinho` GitHub Actions workflow tests this Go module and builds the image
for pull requests. Pushes to `master` publish only
`ghcr.io/kileo13/salada:latest`. The workflow authenticates with its built-in
`GITHUB_TOKEN`, so it requires no repository secret. GHCR creates the package as
private on its first publication; make it public in the package settings before
deploying, because the OCI host intentionally pulls it without registry
credentials.

## Movie and TV tracker

Commands: `/letterboxd show title:`, `/letterboxd wishlist add title:`,
`/letterboxd wishlist remove title:`, `/letterboxd wishlist list`,
`/letterboxd about`, `/letterboxd pick-random`, and
`/letterboxd history [member] [sort] [order]`. Show supports TMDB autocomplete and free-text result lists;
selected titles display artwork, community ratings, and a reviews button.
`/letterboxd about` explains the feature and local fallback in a Components V2
page, followed by TMDB attribution in pt-BR beside the provider logo.
Search pages show `Resultados (x)` using TMDB's absolute total, including
people counted by multi-search even though the menu lists only movies/TV.
During provider failures, `⚠️ Resultados (x)` shows the count of all matching
cached titles instead. The footer displays `Consulta: <query>` as plain text.
Wishlist additions exclude existing interests. Removal/listing use local data
and work during TMDB outages. Data is global across guilds; interactions require
a server. Show replies, title selections, and review-button replies are public.
All Letterboxd pagination updates the current message rather than posting
another reply, retaining that message's visibility. Personal wishlist actions
reply privately.

Search text supports up to 200 Unicode characters. Click the green **Assisti**
button to open a rating/review modal. A rating of
1-5 is required; review text is optional and must contain 2-500 characters when
supplied. Existing opinions are prefilled; leaving the review empty withdraws
its text while preserving the rating. Cancelling the modal saves nothing.
Review submissions collapse whitespace (including CR/LF) to single spaces and
trim the edges before validation and storage. Existing reviews are normalized
again for display and modal prefilling.

**Ver avaliações** shows `Avaliações de <título>`, three members per page, and
the total number of member ratings across all pages. Ratings remain prominent;
each written review appears on one quoted line, with a blank line between
members. Ratings without text show `> *Sem comentário*`. Dates remain in
personal history rather than this quick view.

The **Assisti** button appears first and opens the member's prefilled rating/review
modal on every click, including later edits. Cards show release year, creators
(TV) or directors (movies), and genres as embed fields. For TV, Lançamento
combines the first-air year and known season count: `2018 - 8 temporadas`
(`2018 - 1 temporada` for one). Movies and TV with unknown/zero seasons show
only the year. Season counts follow cached TMDB metadata as new seasons appear.
The author links to the official homepage when available, otherwise to the
TMDB title page. Community ratings come from the database and display as
“Nota geral: 4,5/5 · avaliado 2 vezes”. Whole-number averages omit the decimal
(`5/5`), and a single rating uses `avaliado 1 vez`. Unrated titles show
“Sem avaliações.” Artwork keeps its existing dominant-color extraction and
fallback behavior. Age classifications are not displayed in cards; existing
TMDB fetching, caching, and database storage remain in place.
Result menus filter repeated title values and show at most 20 options. Titles,
summaries, and autocomplete labels respect Discord limits, including emoji.
Apply migration `00010_letterboxd_metadata.sql` before deploying this version.
This unreleased migration requires an empty title catalog; see the
[database instructions](../../database/README.md) for clearing disposable test
data in foreign-key order. Creators/genres use JSON arrays on the
title row, with no new tables or JOINs.
Payload regression checks run with
`go test ./internal/discord/letterboxd`.

Random selection uses only the caller's unwatched wishlist. History defaults to
the caller, watched date, and descending order. The `sort` choices are `Date`
and `Rating`; `order` choices are `Ascending` and `Descending`. Member, sort,
and order persist through pagination. History and
its pagination are public and use one embed with up to three entries. Title
names omit the release year; each rating has a compact Discord-localized date
and a quoted review or `Sem comentário`. The author uses the member's global
name, then username, then snowflake, with their avatar when available. The footer
counts all of that member's ratings across pages. Review edits preserve the
original watched date and retain old revisions
in the database; ordinary views show only the latest text.

### Manual release order

1. Before publishing a new bot image, build and apply the independent migration
   executable using [database instructions](../../database/README.md). Confirm
   `00009_letterboxd.sql` has been applied. Provision the required `tmdb.token`
   API Read Access Token through a trusted private database session. Neither
   migrations nor tokens belong in the application image or committed files.
2. Retain the existing DML-only runtime account; the feature needs only SELECT,
   INSERT, UPDATE, and DELETE. Validate against isolated MySQL using the
   `TEST_DATABASE_HOST`, `TEST_DATABASE_PORT`, `TEST_DATABASE_USERNAME`,
   `TEST_DATABASE_PASSWORD`, and `TEST_DATABASE_NAME` environment contract.
   Skipped database tests are not successful live validation. Run both Go
   modules' race tests, vet, and builds, then the root container build.
3. After separate authorization to publish/deploy, release the reviewed image
   using the existing workflow. Complete schema/configuration first: Watchtower
   automatically follows `latest`, so publishing can start rollout without a
   later manual Compose command. No workflow publishes migrations.
4. If an explicit manual rollout is chosen, run `docker compose pull bot` and
   `docker compose up -d bot` from the deployed Compose directory. Check process
   startup and logs for configuration/database/gateway errors without displaying
   token values. Missing TMDB configuration fails before Discord connects.

To roll back application code, select the previously recorded image digest with
`SALADA_IMAGE` through the existing deployment configuration. Keep the additive
schema and review ledger; do not run the migration Down section as an application
rollback.
