# Padinho architecture

## Startup and shutdown

`cmd/padinho` is the visible composition root. It opens a bounded `*gorm.DB`,
retrieves `app.token`, `birthday.channel_id`, the Mudae bot ID, and six Mudae
custom emoji IDs through the `config` repository, constructs the application
services and Discord listeners, registers and freezes all Discord routes, and
starts the gateway plus recurring-job scheduler. Missing, empty, non-numeric,
or duplicate Mudae identifiers fail before Discord connects.

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
modal route maps. It freezes them as one composition and dispatches application
commands, message components, and modal submissions from the same gateway.
Component `custom_id` values use `route:param...`; handlers validate every
parameter because client input is untrusted.

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
labels from January through December and lowercase values internally. When
omitted, it uses the bot process's current calendar month. Each of the twelve pages
queries and renders one month in day/name order with Components V2. Arrow-only
buttons carry only direction and current month in a stateless custom ID, so any
member can browse the list. A magnifier accessory in the title section looks up
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
mentions are explicitly restricted, and display names are Markdown-escaped.

The public `Editar` button opens one ephemeral Components V2 dashboard, but the
button, target selector, field buttons, and modal submissions each independently
require Discord's exact `Administrator` permission. A User Select updates the
same dashboard with either a not-found state or the chosen registration. User ID
is rendered without an accessory; name, full date, raw IANA timezone, and custom
message use Sections with pencil-button accessories. Each pencil opens one
prefilled text-input modal whose stateless custom ID carries the validated field
and target user. The final dashboard Text Display labels its attached User Select
as `Usuário`. Successful submission performs a validated atomic update of
only that column in the existing `birthdays` table, reloads the row, and updates
the source dashboard message. No edit table, schema migration, or expiring
server-side interaction state is used.

Birthday pages use Components V2 separators between the capitalized month
heading, the dated member-mention list, and a compact footer. The application
service calculates the next strictly future birthday across all stored members
in each member's timezone, applying the February 29 rule, and the footer renders
that member plus the local-date start with Discord's relative timestamp syntax.
Allowed mentions remain disabled for page browsing, so these are visible member
references rather than notifications.

The standard-library scheduler runs the birthday job every minute. The service
converts the current instant into each stored IANA timezone, so DST and
quarter-hour offsets are handled by Go's embedded timezone database. A due
announcement is sent as plain Discord message content, using Discord's default
mention behavior, and recorded in
`birthday_announcements` only after successful Discord delivery. Sequential
execution prevents a job from overlapping itself; the ledger prevents later
checks from sending the same local-date birthday again.

Birthday messages are trimmed before storage. An empty stored `message` remains
empty rather than receiving a copied default. When the announcement selector
encounters that empty value, it fetches `birthday.defaultMessage` from `config`
once for that scheduler run and supplies its `{age}`, `{name}`, and `{mention}`
template to the Discord sender. A birthday with a non-empty stored message never
reads that configuration value.

The add-birthday modal uses Discord's current modal component contract: text
inputs, the required User Select, and the required timezone string select are
wrapped in `Label` components. The User Select supplies the target member's
snowflake, while the timezone select offers localized Brazilian Portuguese
labels for Brasília (`America/Sao_Paulo`), Amazonas (`America/Manaus`), and
UTC. The DiscordGo dependency is replaced with the `sajfer/discordgo` v0.30.0
fork until the upstream dependency exposes these modal component and
submitted-select value types.

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

Game classification is conservative. An explicit `$oc`/Ourochest marker in
Mudae's content, embeds, or component IDs is sufficient. Otherwise, the board
must follow an exact `$oc` or `$ourochest` user command in the same channel
within 15 seconds. `$oh`/`$ouroharvest` correlations are consumed but never
solved. Optional play counts from 1 through 10 reserve that many sequential
responses. Padinho does not inspect an arbitrary history window, guess from the
grid shape alone, or use reactions as a human-in-the-loop mode switch.

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

Each Mudae message owns a small actor that serializes solving and REST edits,
so updates for separate boards can proceed independently without racing the
same helper message. Repeated, stale, and out-of-order snapshots are ignored.
The actor deletes its reply after five reveals, an all-disabled board, source
deletion, context shutdown, or three idle minutes. All replies suppress allowed
mentions and use a soft Discord message reference. No external solver endpoint
or website scraping participates in this flow.

## Persistence and deployment

`internal/database.Open()` privately reads the small `DB_*` bootstrap contract
and returns `*gorm.DB`; credentials are never retained in a general application
configuration object. `internal/config` maps `config(name, value)`, receives
GORM directly, and owns the `app.token`, `birthday.channel_id`,
`birthday.defaultMessage`, `bots.mudae.id`, and six
`bots.mudae.oc.emoji.*` keys. Neither layer passes contexts through
synchronous startup queries.

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
