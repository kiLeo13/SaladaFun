# Padinho

Padinho is SaladaFun's Discord bot foundation, written in Go 1.26 for a small
private guild. It currently declares no user-facing commands. The repository
contains the typed registry, middleware, Discord adapter, MySQL/GORM lifecycle,
migration runner, and production container shape needed to add them later.

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

## Verification and container

```sh
go test -race ./...
go test -cover ./internal/command
docker build -f padinho/Dockerfile -t padinho:local .
```

Run Docker builds from the repository root because migrations are shared at
`database/migrations`. The image includes `/app/migrate` for manual migration
runs. See [ARCHITECTURE.md](ARCHITECTURE.md) for the detailed boundaries.
