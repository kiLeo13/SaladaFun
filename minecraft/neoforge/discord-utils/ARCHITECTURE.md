# Discord Utils architecture

`minecraft/neoforge/discord-utils` is an independent Java 21/NeoForge 21.1.249
server-side mod. It owns one Discord gateway lifecycle for all of its features.
The initial release implements only a bidirectional chat bridge; Discord account
linking remains a future feature.

## Boundaries

- `DiscordUtilsMod` is the composition root. It registers the server-only mod,
  configures NeoForge's common TOML, owns server lifecycle wiring, and applies
  config reloads.
- `config` converts NeoForge values to a validated immutable
  `DiscordChatSettings`. It never permits token or webhook diagnostics.
- `discord` contains the JDA transport and the staged `DiscordChatBridge`
  lifecycle. It contains no Minecraft or NeoForge types.
- `platform.neoforge` translates server chat events into transport calls and
  schedules Discord-originated messages back onto the Minecraft server thread.

## Lifecycle

On server start, the mod creates the bridge and attaches a chat listener.
Configuration creates a candidate JDA session. The candidate becomes active only
after Discord reports `READY` and the configured guild text channel is visible.
Until then, the previous session continues carrying traffic. A failed candidate
is closed and cannot interrupt the previous session. Disabling the setting or
stopping the server closes the active session.

JDA and its runtime dependencies are packaged through NeoForge Jar-in-Jar.
NeoForge supplies the SLF4J API and logging implementation, so JDA's duplicate
SLF4J API is excluded from the nested dependency.
