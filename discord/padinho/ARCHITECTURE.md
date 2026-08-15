# Padinho architecture

## Startup and shutdown

`cmd/padinho` is the visible composition root. It opens a bounded `*gorm.DB`,
retrieves `app.token` and `birthday.channel_id` through the `config` repository,
constructs the birthday repository/service, registers and freezes all Discord
routes, and starts the gateway plus recurring-job scheduler. Missing or empty
runtime configuration fails before Discord connects.

## Command boundary

The unique `internal/command.Registry` exposes typed options, top-level slash
commands, direct subcommands, and Discord's one supported subcommand-group
level. It intentionally has no recursive groups that Discord cannot represent.

Metadata belongs in feature registration functions. Handlers receive a typed
`CommandRequest`; `context.Context` is reserved for cancellation and deadlines.
`Freeze` validates declarations, snapshots definitions, and composes dispatch:

```text
registry -> command group -> subcommand group -> route -> handler
```

`internal/discord.Routes` owns that command registry plus stable component and
modal route maps. It freezes them as one composition and dispatches application
commands, message components, and modal submissions from the same gateway.
Component `custom_id` values use `route:param...`; handlers validate every
parameter because client input is untrusted.

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

The `/birthdays` command always starts at January. Each of the twelve pages
queries and renders one month in day/name order with Components V2. Arrow-only
buttons carry direction, current month, and invoking user in a stateless custom
ID; the ➕ button opens the registration modal. All visible copy except command
names lives in the typed `internal/locale/ptbr` package. Allowed mentions are
explicitly restricted, and display names are Markdown-escaped.

The standard-library scheduler runs the birthday job every minute. The service
converts the current instant into each stored IANA timezone, so DST and
quarter-hour offsets are handled by Go's embedded timezone database. A due
announcement is sent through Components V2 and recorded in
`birthday_announcements` only after successful Discord delivery. Sequential
execution prevents a job from overlapping itself; the ledger prevents later
checks from sending the same local-date birthday again.

## Persistence and deployment

`internal/database.Open()` privately reads the small `DB_*` bootstrap contract
and returns `*gorm.DB`; credentials are never retained in a general application
configuration object. `internal/config` maps `config(name, value)`, receives
GORM directly, and owns the `app.token` and `birthday.channel_id` keys. Neither
layer passes contexts through synchronous startup queries.

The root `database` Go module exclusively owns Goose, the migration executable,
and SQL history. Compose runs that executable as a one-shot dependency before
starting Padinho. Padinho never creates its schema or runs migrations.

The Salada production image uses a digest-pinned Go 1.26.5/Alpine 3.24 builder
for its CGO-disabled binaries. Its runtime is static, distroless, non-root,
read-only, and limited to 384 MB. Watchtower is an explicitly accepted archived
dependency: it is digest-pinned, scoped by labels, polls every 300 seconds, and
has no HTTP API.

Terraform provisions an Always Free `VM.Standard.E2.1.Micro` with only SSH from
one administrator `/32`. `MySQL.Free` and `HeatWave.Free` remain private and
accept 3306 only from the bot NSG. Runtime and read-only GHCR credentials live in
OCI Vault; Ansible retrieves them with the instance principal and runs Compose.
