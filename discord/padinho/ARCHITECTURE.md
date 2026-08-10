# Padinho architecture

## Startup and shutdown

`cmd/padinho` loads explicit environment configuration and creates a
signal-cancelled context. `internal/app` creates the configured MySQL schema
when absent, opens a bounded GORM pool, pings it, applies pending Goose
migrations, registers and freezes commands, and only then opens Discord. A
migration failure exits before the bot serves against an incompatible schema.
Shutdown closes the Discord session and database pool.

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

`internal/database` exposes the same bounded connection through GORM and
`database/sql`; Goose consumes the latter. Callers provide typed host, port,
username, password, and database fields rather than a DSN. Migrations live at
`../../database/migrations` and are copied into `/app/migrations`.

The Salada production image is static, distroless, non-root, read-only, and limited to
384 MB. Watchtower is an explicitly accepted archived dependency: it is
digest-pinned, scoped by labels, polls every 300 seconds, and has no HTTP API.

Terraform provisions an Always Free `VM.Standard.E2.1.Micro` with only SSH from
one administrator `/32`. `MySQL.Free` and `HeatWave.Free` remain private and
accept 3306 only from the bot NSG. Runtime and read-only GHCR credentials live in
OCI Vault; Ansible retrieves them with the instance principal and runs Compose.
