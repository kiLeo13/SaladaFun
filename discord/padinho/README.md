# Padinho

Padinho is SaladaFun's Discord bot, written in Go 1.26 for a small private
guild. Its `/birthdays [month:<January...December>]` command displays a
Components V2 calendar with one page per month, arrow-only navigation, a
self-inspection button beside the title, and an add modal opened through the
`Adicionar` button. When `month` is omitted, the command starts at the bot
process's current calendar month. Its
`/move-all destination:<voice channel> [origin:<voice channel>]` command moves
every currently connected member from the chosen origin; if origin is omitted,
it uses the caller's current voice channel. Destination capacity is not checked
before moves are requested from Discord.

Padinho also follows Mudae `$oc` (Ourochest) boards in real time. It replies
below an identified board with distinct balanced, information-first,
reward-first, and direct-red suggestions, then edits that reply after each
reveal. Duplicate objectives are collapsed, so only genuinely different useful
buttons are shown. The helper reply is deleted after five reveals, when Mudae
disables the whole board, when the source message is deleted, or after three
minutes without an update. Solving is local and does not call or scrape the
Mudae Helper website.

Automatic help defaults to enabled per Discord user. `!toggleochelper` toggles
only that automatic trigger and persists the choice; it does not disable the
solver. A user can reply to an active Mudae `$oc` board with `!ochelper` at any
time to start manual assistance from the board's current state.

Both slash-command channel options are restricted to guild voice channels.
Birthday announcements are evaluated every minute against each user's IANA
timezone and delivered once per local calendar date as plain Discord text;
Discord applies its normal mention behavior and server/channel mention settings.
The add-birthday modal
lets a server manager select the member whose birthday is being registered and
offers localized timezone choices for Brasília, Amazonas, and UTC instead of
requiring an IANA timezone string. Its date field accepts `DD/MM/AAAA`.
Saving a birthday for another member returns an ephemeral confirmation
mentioning that member; saving for yourself retains the personal confirmation.
The title's magnifier opens an ephemeral view of the caller's stored full date,
name, raw IANA timezone, and custom-message state. This private legacy embed
uses the cached guild name and guild icon in its footer; when cache data is
unavailable, it renders a localized fallback instead. The public `Editar` button
opens an ephemeral dashboard only for administrators. Its User Select loads one
existing registration into the same dashboard, where name, full date, timezone,
and message each have a pencil that opens a prefilled one-field modal. The
Components V2 dashboard ends with an `Usuário` label and the User Select.
Submitting
the modal atomically updates that column in `birthdays`, reloads the row, and
replaces the same dashboard message. The Discord user ID is never editable.

Each birthday page separates its capitalized month heading, mention-based list,
and footer with Discord dividers. The footer identifies the next upcoming
birthday across the server and renders its local-date start as a Discord
relative timestamp without notifying the member.

## Command composition

All commands and their related component/modal routes are registered once in
`internal/commands.Register`:

```go
routes := discord.NewRoutes()
commands.Register(routes, birthdayService, gateway, ouroChestListener)
if err := routes.Freeze(); err != nil {
    return err
}
```

`HandlerFunc` is an ordinary function signature, so the final argument can be a
free function or a method such as `memberService.HandleAdd`. Files and packages
follow feature cohesion instead of one module per command. The birthday package
therefore owns its slash handler, page buttons, modal, Components V2 rendering,
and announcement sender together.

Slash metadata remains project-owned and is translated when global commands are
synchronized. Fixed string choices are declared in the command framework and
compiled into Discord dropdown options; birthday month labels are kept in
`internal/locale/enus`. Response bodies are native `discordgo.InteractionResponse`
values passed through a small responder that binds the originating interaction
and rejects a second initial response. Component IDs encode validated page,
direction, edit field, and target user; the source message remains attached to
component-opened modal submissions, so no in-memory dashboard session is lost
during Watchtower restarts. Brazilian
Portuguese response text and English command metadata are centralized as typed
constants in `internal/locale` without a runtime translation dependency.

## Database configuration

`internal/database.Open()` reads `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`,
and `DB_NAME` directly and returns `*gorm.DB`. Optional pool limits use
`DB_MAX_OPEN`, `DB_MAX_IDLE`, and `DB_MAX_LIFETIME`. There is no environment
configuration struct and callers never provide a DSN.

`cmd/padinho` passes GORM directly to `internal/config`, loads `app.token`,
`birthday.channel_id`, and the Mudae settings below, and then constructs Discord. Padinho derives its
application ID after connecting and always synchronizes global commands; there
are no Discord environment switches. When an announced birthday has no custom
message, Padinho reads `birthday.defaultMessage` from `config` and applies
`{age}`, `{name}`, and `{mention}` before sending it.

Schema creation and migration belong exclusively to the root
[`database`](../../database/README.md) project. Build its self-contained Linux
executable locally, upload it to the Padinho VM, and run it there before
deploying code that requires a new schema. Compose never applies migrations.
Apply `00002_user_preferences.sql` before deploying this version; automatic
preference reads and `!toggleochelper` require `users_preferences`.
Insert the following values through a trusted private database session. Mudae's
six values are custom emoji IDs only, without names or Discord `<:...:...>`
markup:

```text
app.token
birthday.channel_id
birthday.defaultMessage
bots.mudae.id
bots.mudae.oc.emoji.blue
bots.mudae.oc.emoji.teal
bots.mudae.oc.emoji.green
bots.mudae.oc.emoji.yellow
bots.mudae.oc.emoji.orange
bots.mudae.oc.emoji.red
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
and currently expose `!toggleochelper` and `!ochelper`. Arguments use Go's
`strings.Fields` behavior, so repeated spaces and tabs do not create empty
values.

## Verification and container

```sh
go test -race ./...
go test -cover ./internal/command ./internal/application/birthday ./internal/application/ourochest ./internal/discord/birthday ./internal/discord/ourochest ./internal/job/...
docker build -f discord/padinho/Dockerfile -t salada:local .
```

Run Docker builds from the repository root. The image contains only Padinho;
migrations remain an independently built manual tool. See
[ARCHITECTURE.md](ARCHITECTURE.md) for the detailed boundaries.

The `Padinho` GitHub Actions workflow tests this Go module and builds the image
for pull requests. Pushes to `master` publish only
`ghcr.io/kileo13/salada:latest`. The workflow authenticates with its built-in
`GITHUB_TOKEN`, so it requires no repository secret. GHCR creates the package as
private on its first publication; make it public in the package settings before
deploying, because the OCI host intentionally pulls it without registry
credentials.
