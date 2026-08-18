# Padinho

Padinho is SaladaFun's Discord bot, written in Go 1.26 for a small private
guild. Its `/birthdays [month:<January...December>]` command displays a
Components V2 calendar with one page per month, arrow-only navigation, and a
modal opened through the ➕ button. When `month` is omitted, the command starts
at the bot process's current calendar month. Its
`/move-all destination:<voice channel> [origin:<voice channel>]` command moves
every currently connected member from the chosen origin; if origin is omitted,
it uses the caller's current voice channel. Destination capacity is not checked
before moves are requested from Discord.
Birthday announcements are evaluated every minute against each user's IANA
timezone and delivered once per local calendar date. The add-birthday modal
lets a server manager select the member whose birthday is being registered and
offers localized timezone choices for Brasília, Amazonas, and UTC instead of
requiring an IANA timezone string. Saving a birthday for another member returns
an ephemeral confirmation mentioning that member; saving for yourself retains
the personal confirmation.

## Command composition

All commands and their related component/modal routes are registered once in
`internal/commands.Register`:

```go
routes := discord.NewRoutes()
commands.Register(routes, birthdayService)
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
and rejects a second initial response. Component IDs encode validated page and
direction; no in-memory state is lost during Watchtower restarts. Brazilian
Portuguese response text and English command metadata are centralized as typed
constants in `internal/locale` without a runtime translation dependency.

## Database configuration

`internal/database.Open()` reads `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`,
and `DB_NAME` directly and returns `*gorm.DB`. Optional pool limits use
`DB_MAX_OPEN`, `DB_MAX_IDLE`, and `DB_MAX_LIFETIME`. There is no environment
configuration struct and callers never provide a DSN.

`cmd/padinho` passes GORM directly to `internal/config`, loads `app.token` and
`birthday.channel_id`, and then constructs Discord. Padinho derives its
application ID after connecting and always synchronizes global commands; there
are no Discord environment switches. When an announced birthday has no custom
message, Padinho reads `birthday.defaultMessage` from `config` and applies
`{age}`, `{name}`, and `{mention}` before sending it.

Schema creation and migration belong exclusively to the root
[`database`](../../database/README.md) project. Build its self-contained Linux
executable locally, upload it to the Padinho VM, and run it there before
deploying code that requires a new schema. Compose never applies migrations.
Insert `app.token`, `birthday.channel_id`, and `birthday.defaultMessage`
through a trusted private database session before expecting the first birthday
without a custom message to be announced.

## Verification and container

```sh
go test -race ./...
go test -cover ./internal/command ./internal/application/birthday ./internal/discord/birthday ./internal/job/...
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
