# Padinho architecture

## Startup and shutdown

`cmd/padinho` is the visible composition root. It opens a bounded `*gorm.DB`,
retrieves the Discord/TMDB tokens, birthday/voice-log channels, BetaSpirit URL,
and Mudae identifiers through the `config` repository, constructs the application
services and Discord listeners, registers and freezes all Discord routes, and
starts the gateway plus recurring-job scheduler. Missing, empty, non-numeric,
or duplicate Mudae identifiers fail before Discord connects.

One DiscordGo session exists for the lifetime of one process. Its internal
reconnect option is disabled: an unexpected Disconnect event makes
`Gateway.Run` fail, and an initial Open attempt that exceeds 60 seconds fails in
the same way. `cmd/padinho` exits non-zero and the Compose
`restart: unless-stopped` policy creates a fresh process, session, state cache,
handler set, and WebSocket. The application never calls Open again on a
disconnected session. Context cancellation from SIGINT or SIGTERM remains a
normal shutdown and closes the connected session without reporting a failure.

## Command boundary

The unique `internal/command.Registry` exposes typed options, top-level slash
commands, direct subcommands, and Discord's one supported subcommand-group
level. It intentionally has no recursive groups that Discord cannot represent.

Metadata belongs in feature registration functions. Handlers receive a typed
`CommandRequest`; `context.Context` is reserved for cancellation and deadlines.
`Freeze` first validates the declaration tree by command level, then compiles
each root according to its route shape. Focused compilation helpers snapshot
definitions and compose the matching dispatch entry without obscuring the
top-level lifecycle:

Registration methods acquire the declaration lock directly, making their
state changes explicit while rejecting every post-freeze mutation.

```text
registry -> command group -> subcommand group -> route -> handler
```

`internal/discord.Routes` owns that command registry plus stable component and
modal route maps. It also owns a separate `messagecommand.Registry`, exposed as
`routes.Messages()`, because literal message triggers do not share Discord
application-command metadata or synchronization. A complete trigger such as
`!ochelper` includes its prefix; exact first-token matching is case-insensitive,
and arguments are normalized with `strings.Fields`. Routes freezes both
registries as one composition and dispatches application
commands, message components, and modal submissions from the same gateway.
Component `custom_id` values use `route:param...`; handlers validate every
parameter because client input is untrusted.

## Quote feature

```text
discord/quote message handler -> application/quote service
                                      |
                                      v
                         persistence/mysql repository -> GORM
```

`!quote [id]` treats a positive decimal first argument as an exact quote ID.
That path does not filter `enabled`, so an operator can retrieve a disabled
quote by ID; an unknown valid ID becomes a typed not-found error. Any missing,
zero, negative, or malformed first argument uses `WHERE enabled = TRUE ORDER BY
RAND() LIMIT 1`, which is uniform and appropriate for the current catalog size.
It joins the canonical author through GORM preloading and returns an empty
result when no enabled rows exist; the application service turns that into a
typed empty-catalog error.

The handler prefers `translated_quote` and otherwise sends `original_quote` in
exactly two lines: a Markdown block quote and an em-dash attribution. A normal
author uses its canonical name. An author with `discord_user_id` is rendered as
`<@snowflake>` and sends through the dedicated user-only mention responder.
Its Discord `allowed_mentions` payload has only `users` in `parse`; role,
`@everyone`, and reply-author mentions remain disabled. `source_url` is
provenance for imports and is intentionally excluded from that fixed format.

## BetaSpirit command

`!betaspirit` is a literal message command registered through the shared route
composition. Its handler replies with the required `urls.youtube.betaSpirit`
configuration value, loaded and validated during process startup.

## Discord account hierarchy

```text
discord/accounttree message command -> application/accounttree service
                                               |
                                               v
                             persistence/mysql recursive GORM query -> MySQL
```

`!childrentree [@user-or-id]` uses the invoking member when no argument is
given. When present, only its first whitespace-separated argument matters; its
first positive decimal snowflake is extracted from raw-ID or Discord-mention
markup, then resolved as a guild member by ID. An invalid identifier and an
unresolved member receive separate localized replies. Every entity-like message
command argument follows this numeric-ID resolution rule rather than trusting a
Discord mention collection.

The MySQL repository executes one inline `WITH RECURSIVE` statement. It walks
the requested row's parents to the root, then walks all descendants from that
root, so asking for a leaf still renders the complete hierarchy. The statement
is issued through `gorm.DB.Raw` and creates no database object. Application code
validates the resulting flat rows into one rooted tree and rejects malformed or
cyclic selections. An absent table row becomes a one-account tree.

The reply is one non-pinging Components V2 container with no accent color and
exactly two Text Displays: a title mentioning the resolved root and a plain
Markdown code block using compact `|__` indentation followed by the localized
total-account footer. Discord usernames are resolved by ID; a stale descendant
falls back to its snowflake rather than disappearing. Names are sanitized before
being inserted into the code block. Tree rendering sorts siblings by display
name and enforces a conservative 4,000-rune message-text budget; it keeps whole
lines and appends `...` while the footer still reports the complete count.

String command options may declare fixed `OptionChoice` values in the
framework-neutral definition. The Discord adapter compiles those values into
Discord's native dropdown choices, keeping command metadata separate from
DiscordGo types.

`command.CommandRequest` is the typed boundary for slash commands: `Path`
identifies the registered command route, `Actor` identifies the invoking user
and effective guild permission bitmask, `GuildID` and `ChannelID` identify the
Discord context, `Options` contains normalized command arguments, `Responder`
owns the single initial response, and `RequestID`/`ReceivedAt` support
correlation and timing. `discord.InteractionRequest` is the equivalent boundary
for buttons and modals. Its `Parameters` are the colon-separated values after
the registered custom-ID route; for example,
`birthdays.page:next:1` becomes `[]string{"next", "1"}`. They are untrusted
input and must be validated by the receiving handler. `Interaction` remains
available only when a handler needs native Discord data, such as modal fields.

## Voice moves

The cohesive `internal/discord/move` feature owns `/move-all`. It receives a
small Discord voice capability from the gateway, rather than reaching into the
session directly. `origin` is optional: the gateway's voice-state cache resolves
the caller's current channel when omitted, otherwise the command rejects the
request. The slash-command options restrict both origin and destination to
guild voice channels; the handler also validates the selected channels in the
same guild (including stage channels for direct, already-validated requests).
The gateway requests `GUILD_VOICE_STATES` and snapshots the origin's
voice states before requesting an individual Discord move for every member.
It deliberately does not inspect channel member limits; Discord evaluates each
move request.

## Voice activity logging

`internal/discord/voiceactivity` is a gateway subscriber rather than a command.
It uses DiscordGo's cached `BeforeUpdate` voice state to classify only channel
changes as joins, moves, or leaves, then renders the configured log-channel
embed. The post-update guild cache supplies the connected-member count for join
and leave embeds. The feature uses the guild name verbatim in its footer, so a
guild-owned emoji is not duplicated by the renderer.

After one send attempt, the listener asks `application/voiceactivity` to persist
the final `SENT` or `FAILED` outcome through
`persistence/mysql/VoiceActivityLogRepository`. `voice_activity_logs` is an
append-only ledger: old and new channel IDs encode the transition, `guild_id`
preserves its server context, and `created_at` is the observed UTC millisecond
time. There is no retry or mutable status yet; a future retry feature must add
its own attempt metadata and update semantics.

Response payloads intentionally use native DiscordGo types. The small bound
responder owns the session and source interaction, sends exactly one initial
response, and reports whether the gateway may still send an error. This is a
documented exception to library-neutral command requests: mirroring Discord's
Components V2 model would add maintenance and conversion failures without
protecting the application layer. Domain, application, persistence, and job
packages remain DiscordGo-free.

## Birthday feature

```text
discord/birthday handlers -> application/birthday service
                                      |
                                      v
                         persistence/mysql repository -> GORM

job/birthday -> application/birthday service -> discord/birthday sender
```

`internal/domain/entity` owns the GORM birthday and delivery-ledger structures.
The application package owns the repository interface it consumes, validation,
IANA timezone loading, age calculation, and the February 29 rule (February 28
in non-leap years). `internal/persistence/mysql` provides the concrete GORM
implementation. Discord handlers consume a smaller service interface local to
their feature package.

The `/birthdays` command accepts an optional `month` string choice with English
labels from January through December and lowercase values internally, plus an
optional `full-date` Boolean. When `month` is omitted, it uses the bot process's
current calendar month. Birthday rows render day and month by default; enabling
`full-date` also renders the stored year. Each of the twelve pages queries and
renders one month in day/name order with Components V2. Arrow-only buttons carry
direction, current month, and the date-format choice in a stateless custom ID,
so any member can browse the list without losing the requested format. The page
parser continues accepting older two-parameter IDs as abbreviated dates. A
magnifier accessory in the title section looks up
the clicking member from the interaction actor and returns their full stored
registration ephemerally; the route never accepts a target-user parameter. The
private legacy embed reads the cached guild through the gateway, renders its name
and icon in the footer, and falls back to localized text when the guild is not
available in state. The
`Adicionar` button and modal submission require
Discord's `Manage Server` (`PermissionManageGuild`) permission. The birthday
saves the member selected in the modal's required User Select, allowing a
manager to register a birthday for another server member. All visible copy
except command names lives in the typed `internal/locale` packages. Allowed
mentions are explicitly restricted, and usernames are Markdown-escaped.

The public `Editar` button opens one ephemeral Components V2 dashboard, but the
button, target selector, field buttons, and modal submissions each independently
require Discord's exact `Administrator` permission. A User Select updates the
same dashboard with either a not-found state or the chosen registration. User ID
is rendered as immutable text whose label starts with the custom snowflake
emoji; name, full date, raw IANA timezone, and custom message use Sections with
pencil-button accessories. Each pencil opens one
prefilled single-field modal whose stateless custom ID carries the validated
field and target user. Creation and editing build those modal controls from the
same field definitions; timezone is a required select in both flows. The final
dashboard divider, `Usuário` Text Display, and attached User Select all live
inside the same Container. Successful submission performs a validated atomic
update of only that column in the existing `birthdays` table, reloads the row, and updates
the source dashboard message. No edit table, schema migration, or expiring
server-side interaction state is used.

Birthday pages use Components V2 separators between the capitalized month
heading, the dated member-mention list, and a compact footer. The application
service calculates the next strictly future birthday across all stored members
in each member's timezone, applying the February 29 rule, and the footer renders
that member plus the local-date start with Discord's relative timestamp syntax.
Allowed mentions remain disabled for page browsing, so these are visible member
references rather than notifications.

The cron-backed scheduler runs the birthday job at every UTC hour boundary with
the standard expression `0 * * * *`; it is therefore aligned to wall-clock
time rather than the process start time. The birthday UI offers only the
hour-aligned Brasília, Amazonas, and UTC timezones, so each local midnight is
checked exactly at its corresponding UTC hour. A due announcement is sent as
plain Discord message content, using Discord's default mention behavior, and
recorded in `birthday_announcements` only after successful Discord delivery.
Cron skips an invocation when that same job is still running, and the ledger
prevents later checks from sending the same local-date birthday again.

Birthday messages are trimmed before storage. An empty stored `message` remains
empty rather than receiving a copied default. When the announcement selector
encounters that empty value, it fetches `birthday.defaultMessage` from `config`
once for that scheduler run and supplies its `{age}`, `{name}`, and `{mention}`
template to the Discord sender. A birthday with a non-empty stored message never
reads that configuration value.

The add- and edit-birthday modals use Discord's current modal component contract:
text inputs and selects are wrapped in `Label` components. Shared field
definitions keep labels, placeholders, required state, limits, and control type
consistent between both flows. The add modal's User Select supplies the target
member's snowflake, while the timezone select offers localized Brazilian Portuguese
labels for Brasília (`America/Sao_Paulo`), Amazonas (`America/Manaus`), and
UTC. Padinho pins the official `github.com/bwmarrin/discordgo` upstream
revision `v0.29.1-0.20260214123928-f43dd94faaac`, which exposes the required
Label and submitted-select value types without a module replacement. That
revision does not include DAVE voice support; a future voice playback feature
must reassess upstream support or introduce an explicitly reviewed voice
implementation.

## Mudae Ourochest helper

```text
Discord create/update events -> discord/ourochest session actor
                                      |
                                      v
                         application/ourochest solver
                                      |
                                      v
                            gateway reply lifecycle
```

The listener accepts messages only from the configured `bots.mudae.id` and
parses exactly five action rows containing five buttons each. The six semantic
colors are mapped from custom emoji IDs stored under
`bots.mudae.oc.emoji.{blue,teal,green,yellow,orange,red}`. Emoji names are not
configuration or identity. Padinho requests `GUILD_MESSAGES` and the privileged
`MESSAGE_CONTENT` intent because Discord otherwise omits the required message
content and components.

Game classification is conservative. Automatic help requires an exact `$oc` or
`$ourochest` user command correlated to the Mudae board in the same channel
within five seconds. Message references win when Mudae supplies one; otherwise
the most recent compatible command handles bursts such as a failed `$oc`
immediately followed by `$oh`. The known Portuguese no-uses and recharge
responses cancel the failed `$oc` correlation. Strong `$oc`/`$oh` signatures
that conflict with pending state cause the board to be ignored rather than
guessed. `$oh`/`$ouroharvest` boards are recorded but never solved. Optional
play counts from 1 through 10 reserve that many responses. Padinho does not
inspect an arbitrary history window, guess from the grid shape alone, or use
reactions as a human-in-the-loop mode switch.

The correlated command stores its invoking user. `preferences.Service` treats a
missing row or nullable `auto_mudae_oc` as the application default `true` and
queries that effective setting before automatic startup. `!toggleochelper`
persists its inverse. With automation disabled, the listener still records the
verified board; replying with `!ochelper` fetches its current Discord payload,
revalidates the configured Mudae author and exact 5×5 shape, rejects known `$oh`
or completed games, and starts the same actor from the mid-game snapshot.

The application solver models the fixed Ourochest geometry exactly: two orange
spheres are orthogonally adjacent to red, three yellow spheres are diagonal,
four green spheres share red's row or column, teal is otherwise aligned by row,
column, or diagonal, and blue is unaligned. Red cannot occupy the center. For
each possible red position, the solver counts compatible unseen placements
with binomial coefficients and divides by the total placements for that red
position. This preserves a uniform prior over red locations instead of
incorrectly favoring locations merely because they have more possible color
layouts.

Every enabled unknown button is evaluated over all possible next colors. The
solver derives its exact immediate red probability, expected sphere value,
Shannon information gain, and expected remaining red candidates. It emits the
distinct winners for a normalized equal-regret balanced objective,
information-first play, reward-first play, and direct-red probability. This is
a one-reveal lookahead recalculated after every update; it is not an unbounded
multi-turn search. Initial symmetry uses the same position preference as the
reference helper.

The Discord presentation exposes only the first, balanced winner. It is a
Components V2 container using the shared birthday accent color, with the button
number as its heading, the row, column, probability, information, and expected
value as secondary text, followed by a divider and the automatic-helper opt-out
command. The additional objective winners remain internal solver output and do
not add competing instructions to the chat.

Each Mudae message owns a small actor that serializes solving and REST edits,
so updates for separate boards can proceed independently without racing the
same helper message. Repeated, stale, and out-of-order snapshots are ignored.
The actor deletes its reply after five reveals, an all-disabled board, source
deletion, context shutdown, or three idle minutes. All replies suppress allowed
mentions and use a soft Discord message reference. No external solver endpoint
or website scraping participates in this flow.

## Mudae Ouroquest helper

`discord/ouroquest` independently correlates exact `$oq` or `$ouroquest`
commands for two seconds, so adjacent `$oc` and `$oh` traffic cannot claim its
anonymous 5×5 grid. The nullable `auto_mudae_oq` preference defaults to enabled;
`!toggleoqhelper` changes automatic activation and `!oqhelper` manually starts
a verified Mudae board.

`application/ouroquest` enumerates all `C(25,4) = 12,650` equally likely target
layouts. Blue through orange encode zero through four purple neighbors in the
eight-cell neighborhood. Exact filtering feeds a bounded expectimax policy
that maximizes sphere payout, then completion probability and information. It
searches two plies for large early posteriors and deepens as filtering makes
that safe for realtime updates. The actor uses the same Components V2 reply
lifecycle as Ourochest and never calls an external solver.

## Persistence and deployment

`internal/database.Open()` privately reads the small `DB_*` bootstrap contract
and returns `*gorm.DB`; credentials are never retained in a general application
configuration object. `internal/config` maps `config(name, value)`, receives
GORM directly, and resolves the database config keys listed in
[README.md](README.md#database-configuration). Neither layer passes contexts
through synchronous startup queries.

The generic `users_preferences` table uses the Discord user snowflake as its
primary key. Module columns such as nullable `auto_mudae_oc` and
`auto_mudae_oq` retain three
states: explicit enabled, explicit disabled, and `NULL` for the module-owned
default. `created_at` and `updated_at` are Unix UTC milliseconds, matching the
other application tables. The MySQL repository performs the toggle atomically;
the application layer, rather than persistence or a SQL default, decides what
`NULL` means.

`quote_authors` contains one canonical author name and an optional, unique
Discord snowflake. It is not a foreign key to a Discord-user table because none
exists in this application. `quotes` references its author, stores original
text, an optional Portuguese translation, an optional source URL, and an
enabled flag. Imports must map alternate raw spellings to the chosen canonical
author ID before inserting; no author-alias table exists.

The root `database` Go module exclusively owns Goose, the migration executable,
and SQL history. Its SQL files are embedded in a self-contained executable that
is built locally, uploaded, and run manually on the Padinho VM. Padinho and
Compose never create schemas or run migrations.

The Salada production image uses a digest-pinned Go 1.26.5/Alpine 3.24 builder
for its CGO-disabled binaries. Its runtime is static, distroless, non-root,
read-only, and limited to 384 MB. Watchtower is an explicitly accepted archived
dependency: it is digest-pinned, scoped by labels, polls every 300 seconds, and
has no HTTP API. Its Docker client is pinned to API `1.44`, the minimum accepted
by the deployed Docker Engine, because the archived client does not negotiate
correctly with recent daemons.

Terraform provisions an Always Free `VM.Standard.E2.1.Micro` with only SSH from
one administrator `/32`. `MySQL.Free` and `HeatWave.Free` remain private and
accept 3306 only from the bot NSG. An ignored Ansible Vault file protects the
restricted runtime database password at rest; Ansible renders the root-only
environment file and runs Compose. The MySQL administrator remains exclusive to
Terraform provisioning and manual migrations. GitHub Actions publishes
Padinho's `latest` image as a public GHCR package, so the VM pulls it
anonymously and no registry token enters Terraform state.

## Movie tracking

`application/letterboxd` owns metadata freshness, discovery, and activity rules.
Discord handlers orchestrate requests; `discord/letterboxd/presentation.go` builds
result pages and title cards without querying repositories or the provider.
`integration/tmdb` isolates the pinned TMDB wrapper and bounded CDN downloads;
`persistence/mysql` stores global title snapshots, personal wishlists, one rated
watch per member/title, and immutable review revisions. `cmd/padinho` loads
`tmdb.token` and wires separate search/detail clients through the shared routes.

Search never persists partial metadata. Selected titles use a 24-hour full-detail
TTL and stale fallback on provider failure. Names retain original/localized forms
and exact-date precision. Search has a 1.5-second HTTP timeout and a two-second
context budget; details use five seconds after deferral. Automatic retries are
disabled. Artwork has a three-second timeout, 2 MiB download and two-million-pixel
limits. A center-weighted RGB histogram supplies the persisted accent; unchanged
artwork reuses it and failures leave the theme fallback. Images are not stored.

Watch mutations serialize by member/title and reject repeated or older numeric
interaction IDs. Rating and review changes commit atomically without changing
the original watch date. Review withdrawal appends NULL; reads select the latest
revision before interpreting that marker. Ratings belong to watches so revisions
cannot multiply community votes. Wishlist names come from cached title joins.

Modals use only a bounded local lookup for prefill and write only on submission,
with ownership derived from the actor. History uses whitelisted date/rating order,
a watch-ID tie-breaker, a total watch count, and up to three rows per page. Random
selection joins the caller's wishlist and excludes only that caller's watches
using NOT EXISTS and ORDER BY RAND() LIMIT 1; it never broadens an empty pool.

Pagination reconstructs queries from bot-owned embed footers and retains
member/order/page in validated custom IDs. Component routes survive restarts;
no in-memory interaction session is required. Shared cards act on the clicking
member, and personal replies suppress mentions. See [README.md](README.md) for
commands, configuration, verification, and deployment.

Search results carry items, `total_results`, and `total_pages` separately.
TMDB totals survive supported-media filtering and page truncation; the header
uses that absolute provider total. Local-only wishlist results count filtered
matches. On provider failure, cached search counts all matching title rows and
reads a bounded 20-title page with stable ID ordering, rather than limiting the
entire fallback search to 25 titles. Its warning appears only in the
`⚠️ Resultados (x)` heading. Cached pages are not paginated a second time in the
handler, and a one-item fallback still shows the outage warning.

Footers use plain `Consulta: <query>`. Pagination strips that prefix only when
the message has the new result heading; legacy unlabelled footers remain valid,
including queries that themselves begin with `Consulta:`. Navigation follows
the source's page count and retains the existing 500-page ceiling.

### Discord payload constraints

Title cards place `lb.watch:<id>` first and use it for both initial ratings and
later edits. The old `lb.edit:<id>` route remains as an alias for existing messages; `lb.watch-save:<id>` handles
submission. Custom IDs must be unique across the entire message, including
different rows. Result selects deduplicate identity values and cap at 20 options.
Text clipping budgets UTF-16 units without splitting Unicode characters.
Reviews and history retain three entries per page, keeping even fully escaped
500-character reviews below per-embed and aggregate text limits. Navigation
disables next at the supported page ceiling.

`constraints_test.go` checks serialized component placement, unique IDs and
select values, option/row counts, label lengths, modal fields, autocomplete,
embed text budgets, Unicode boundaries, and pagination ceilings. Existing flow
tests cover modal prefilling, ownership, submission and empty/random paths.
These are local contract checks; a Discord smoke test after deployment should
exercise show, all four card buttons, wishlist selection, reviews, history,
random selection, autocomplete, and modal submit/cancel.
References: [components](https://docs.discord.com/developers/components/reference)
and [embed limits](https://docs.discord.com/developers/resources/message#embed-object-embed-limits).

### Rich title metadata

Migration `00010_letterboxd_metadata.sql` adds NOT NULL JSON-array creators and
genres, an official homepage, and a nonnegative season count to the existing title
row. GORM's JSON serializer maps arrays to project-owned string slices. Writes
trim names, remove empty/duplicate entries, and preserve punctuation and order;
empty lists persist as `[]`. No additional tables or JOINs are needed.
TV details supply `created_by` and `number_of_seasons`; movies request
`append_to_response=credits` in the same detail call and retain Director crew.
Genre names use the existing pt-BR request. Refresh/upsert replaces all fields.
This unreleased migration assumes an empty title catalog and adds required
columns directly, without defaults, backfill, or cache invalidation. The
application supplies explicit values on every insert. Clear disposable test
records in foreign-key order before migrating; see the database README.

Cards use ordinary embed fields for release, creators/directors and genres.
The release field combines TV first-air year and a positive season count
(`2018 - 8 temporadas`, or `2018 - 1 temporada`); movies and TV with zero/unknown
seasons show only the year. There is no separate season field. Season counts
refresh with normal TMDB metadata. Exact release dates remain cached but are not
rendered. Synopsis descriptions are capped at 2,000 UTF-16 units including the
`...` suffix; `Ellipsis` preserves whole Unicode characters. List fields
are escaped and clipped to 600 UTF-16 units each; the payload checker includes
field and author text in Discord's aggregate embed budget. Author URLs accept
absolute HTTP(S) homepages without credentials and otherwise use the TMDB title
page; the title itself continues linking to TMDB. Ratings average the current
database opinions, one vote per watch/member, with no TMDB rating substitution.
Footer averages retain one decimal when needed and omit a trailing zero for
whole-number displays (`5/5`, `4,5/5`). One vote uses `avaliado 1 vez`;
multiple votes use `avaliado N vezes`, and no votes keep `Sem avaliações.`.
The existing center-weighted dominant-color algorithm and artwork selection
(backdrop, then poster) remain unchanged.

Brazilian classification codes (`BR`, `L`, `6`, `10`, `12`, `14`, `16`, `18`)
are named constants in `application/letterboxd/classification.go`. Application
emoji markup is kept separately in `discord/letterboxd/classification.go`;
the label “Faixa etária” and unknown text “Não informada” live in the pt-BR locale.
TMDB movie details append `release_dates` alongside credits; TV details append
`content_ratings`. Country, certification, and movie release type are stored as
project-owned `ContentRating` entries in the existing nullable JSON
`letterboxd_titles.content_ratings` column from migration 00011. No new migration
or record deletion is needed for classification storage.

Cards do not display age classifications or their emojis. The Discord formatting
helper has been removed; emoji constants remain reserved for potential future
display. Classification metadata, provider mapping, selection rules and database
storage remain unchanged. When Brazilian release variants disagree, the
application selector still chooses the highest recognized age regardless of
provider order, retaining original entries and excluding foreign substitutions.

SQL NULL (nil slice) marks metadata fetched before classification support and
forces on-demand refresh even within the normal TTL. A successful detail fetch
stores an empty JSON array when no classifications are available, preventing
repeated requests for known-empty results. Provider failure retains the stale
card without inventing a classification. Metadata upserts replace content ratings
including empty lists; local title readers retain the JSON data.

### Review presentation and whitespace

The reviews handler reads the cached title through `Catalog.TitleByID` for its
bounded heading and derives the footer total from all current member opinions,
not just the current three-entry page. Each rating is bold, followed by one
quoted review line or the localized italic “Sem comentário” placeholder.
Entries are separated by one blank line; this fast view omits dates.
Show title selections and initial review buttons defer publicly.
Pagination uses deferred message updates and completes by editing the source
message, preserving its visibility. Review pages use `lb.reviews-page`; legacy
review-page buttons are recognized by their existing heading/footer.
The responder supports both deferred reply and deferred-update acknowledgements;
pagination errors retain the original embeds and controls so navigation can retry.
The component helper defaults to private acknowledgements for personal actions;
wishlist changes and other personal flows retain that default.

`NormalizeReview` collapses whitespace, including CR/LF runs, and trims edges.
`ReviewText` validates the normalized result; `Activity.Save` forwards that
canonical text and the repository also validates/normalizes direct callers.
Whitespace-only text remains a review withdrawal. Rendering and modal prefilling
normalize legacy text without rewriting stored revisions. The shared opinion
renderer escapes Markdown after normalization, preventing input line breaks from
breaking quote layout in shared reviews or expanding personal history.
No schema migration is required.

### Public member history

History commands defer publicly; pagination updates that message in place.
History renders one embed with at most
three entries. Release years are omitted; titles are escaped and capped at
300 UTF-16 units, ratings are bold, and dates use `<t:seconds:d>`. Quoted reviews
use the existing normalization and placeholder. Three worst-case 500-character
escaped reviews plus bounded titles and date markup remain below the
4,096-unit description limit. The footer uses the member's absolute watch count,
not page length or review-revision count; SQL returns that count alongside a
bounded page. Existing order validation and watch-ID tie-breaking remain.

The Discord mapper passes caller/resolved user profiles as project-owned
`CommandRequest.Users` data. History always renders an author: global name,
username, or member snowflake, with the provided avatar and a profile URL.
Pagination copies the previous bot embed author only when its URL matches the
requested member. No separate profile cache or REST user lookup is introduced.
Unknown profiles always use the snowflake rather than a mention.
