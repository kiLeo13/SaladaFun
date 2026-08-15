# Padinho

Padinho is SaladaFun's Discord bot, written in Go 1.26 for a small private
guild. Its `/birthdays` command displays a Components V2 calendar with one page
per month, arrow-only navigation, and a modal opened through the ➕ button.
Birthday announcements are evaluated every minute against each user's IANA
timezone and delivered once per local calendar date.

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
synchronized. Response bodies are native `discordgo.InteractionResponse`
values passed through a small responder that binds the originating interaction
and rejects a second initial response. Component IDs encode validated page,
direction, and owner state; no in-memory state is lost during Watchtower
restarts. Brazilian Portuguese text is centralized as typed constants in
`internal/locale/ptbr` without a runtime translation dependency.

## Database configuration

`internal/database.Open()` reads `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`,
and `DB_NAME` directly and returns `*gorm.DB`. Optional pool limits use
`DB_MAX_OPEN`, `DB_MAX_IDLE`, and `DB_MAX_LIFETIME`. There is no environment
configuration struct and callers never provide a DSN.

`cmd/padinho` passes GORM directly to `internal/config`, loads `app.token` and
`birthday.channel_id`, and then constructs Discord. Padinho derives its
application ID after connecting and always synchronizes global commands; there
are no Discord environment switches.

Schema creation and migration belong exclusively to the root
[`database`](../../database/README.md) project. Compose runs its one-shot
`migrate` service successfully before starting Padinho. Insert both required
configuration values through a trusted private database session before
expecting the bot to start.

## Verification and container

```sh
go test -race ./...
go test -cover ./internal/command ./internal/application/birthday ./internal/discord/birthday ./internal/job/...
docker build -f discord/padinho/Dockerfile -t salada:local .
```

Run Docker builds from the repository root because the image packages the
independently built database migration executable and shared SQL history. See
[ARCHITECTURE.md](ARCHITECTURE.md) for the detailed boundaries.
