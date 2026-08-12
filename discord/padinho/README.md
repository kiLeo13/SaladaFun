# Padinho

Padinho is SaladaFun's Discord bot foundation, written in Go 1.26 for a small
private guild. It currently declares no user-facing commands. The repository
contains the typed registry, middleware, Discord adapter, MySQL/GORM lifecycle,
and production container shape needed to add them later.

## Command composition

All commands are registered once in `internal/commands.Register`:

```go
r := command.NewRegistry()
r.Use(globalMiddleware)

groups := r.Group("groups", "Commands to manage groups")
members := groups.Group("members", "Commands to manage group members")
members.Sub(
    "add",
    "Add a member",
    HandleAddMemberCommand,
    command.UserOption("user", "Member to add").Required(),
).Use(commandCooldown)

if err := r.Freeze(); err != nil {
    return err
}
```

`HandlerFunc` is an ordinary function signature, so the final argument can be a
free function or a method such as `memberService.HandleAdd`. Files and packages
follow feature cohesion instead of one module per command. `Freeze` only
validates and compiles immutable metadata and dispatch; the Discord adapter
later translates metadata to DiscordGo and bulk-overwrites commands.

## Database configuration

The runtime accepts `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_USERNAME`,
`DATABASE_PASSWORD`, and `DATABASE_NAME` as separate values. It never accepts a
DSN from configuration. `cmd/padinho` opens `*gorm.DB`, passes it directly to
the configuration repository, and loads the required `app.token` value before
constructing the Discord gateway.

Schema creation and migration belong exclusively to the root
[`database`](../../database/README.md) project. Compose runs its one-shot
`migrate` service successfully before starting Padinho. Insert `app.token`
through a trusted private database session before expecting the bot to start.

## Verification and container

```sh
go test -race ./...
go test -cover ./internal/command
docker build -f discord/padinho/Dockerfile -t salada:local .
```

Run Docker builds from the repository root because the image packages the
independently built database migration executable and shared SQL history. See
[ARCHITECTURE.md](ARCHITECTURE.md) for the detailed boundaries.
