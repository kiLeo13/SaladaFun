# Padinho architecture

## Startup and shutdown

`cmd/padinho` is the visible composition root. It loads explicit environment
configuration, creates a signal-cancelled context, opens a bounded `*gorm.DB`,
constructs the configuration repository, retrieves `app.token`, registers and
freezes commands, and only then opens Discord. A missing or empty token fails
startup before any Discord connection. Shutdown closes the Discord session and
underlying database pool.

## Command boundary

`internal/command` is framework-neutral. The unique `Registry` exposes typed
options, top-level slash commands, direct subcommands, and Discord's one
supported subcommand-group level. It intentionally has no recursive groups that
Discord cannot represent.

Metadata belongs in feature registration functions. Handlers receive a typed
`CommandRequest`; `context.Context` is reserved for cancellation and deadlines.
`Freeze` validates declarations, snapshots definitions, and composes dispatch:

```text
registry -> command group -> subcommand group -> route -> handler
```

`internal/discord` is the only DiscordGo adapter. It compiles definitions,
bulk-overwrites commands at startup, maps interactions into typed requests, and
turns expected middleware rejections into ephemeral responses.

## Persistence and deployment

`internal/database` returns `*gorm.DB`; repositories receive that value directly
instead of a project-specific connection wrapper. `internal/configuration`
maps the `config(name, value)` table and owns the `app.token` key. Callers
provide typed host, port, username, password, and database fields rather than a
DSN.

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
