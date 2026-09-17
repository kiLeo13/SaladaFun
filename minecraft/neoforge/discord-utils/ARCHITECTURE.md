# Discord Utils architecture

`minecraft/neoforge/discord-utils` is an independent Java 21/NeoForge 21.1.249
server-side mod. It owns one Discord gateway lifecycle for all of its features.
The initial release implements a bidirectional chat bridge, player-presence
notifications, and registered message commands; Discord account linking remains
a future feature.

## Boundaries

- `DiscordUtilsMod` is the composition root. It registers the server-only mod,
  configures NeoForge's common TOML, owns server lifecycle wiring, and applies
  config reloads.
- `config` converts NeoForge values to a validated immutable
  `DiscordChatSettings`. It never permits token or webhook diagnostics.
- `discord` contains the JDA transport and the staged `DiscordChatBridge`
  lifecycle. Its frozen message-command registry gives the single inbound
  message listener one dispatch decision: execute a registered first-token
  match or fall through to Minecraft chat. It contains no Minecraft or NeoForge
  types.
- `platform.neoforge` translates server chat and player connection events into
  transport calls, schedules Discord-originated messages back onto the Minecraft
  server thread, and snapshots online-player names there for Discord commands.

## Lifecycle

On server start, the mod creates the bridge and attaches chat and player
connection listeners. Configuration creates a candidate JDA session. The
candidate becomes active only after Discord reports `READY` and the configured
guild text channel is visible. Until then, the previous session continues
carrying traffic. A failed candidate is closed and cannot interrupt the previous
session. Disabling the setting or stopping the server closes the active session.

Minecraft chat uses the configured webhook so each message can retain the
player's name and avatar. Player join and leave notifications use Padinho's bot
identity in the validated guild channel. Each notification is a Components V2
container with one text display and a green or red accent.

NeoForge's `PlayerLoggedOutEvent` does not carry a disconnect reason directly.
The platform listener therefore reads the `DisconnectionDetails` retained by
the logged-out `ServerPlayer` connection while `PlayerList.remove` is firing.
Vanilla's `disconnect.endOfStream` reason identifies the routine client-close
path and remains the ordinary `saiu do servidor` notification. Other nonblank
reasons are normalized and passed through the platform-free presence model for
the reasoned `desconectou: <reason>` notification. Discord rendering escapes
both player-controlled names and reason formatting, disables mentions, and
limits the final text to the Components V2 text-display boundary.

The same inbound JDA listener handles registered literal message commands before
ordinary Discord-to-Minecraft chat. Triggers include their prefix and match the
first whitespace-delimited token case-insensitively. `!players` requests an
immutable name snapshot through the platform boundary; NeoForge fulfills it on
the server thread, and the originating JDA session replies only if it remains
active. Its response uses Padinho's identity and one dark-green Components V2
container with one text display. Large lists paginate only when required by
Discord's component-text limit, preserving every online player.

For ordinary inbound chat, the listener also snapshots any resolved Discord
reply into a JDA-free `DiscordReplyReference`. The NeoForge broadcaster renders
that snapshot as a dark-gray, single-line quote above the normal bridge message,
including supported-media counts when the reference has no text. Missing,
deleted, or unresolved references are omitted without delaying or discarding the
new message. Reply context is one-way presentation metadata and does not create
a Minecraft reply mechanism.

JDA and its runtime dependencies are packaged through NeoForge Jar-in-Jar.
JDA and Commons Collections are first combined into a private nested JAR that
relocates Commons Collections below `sld.saladafun.discordutils.shaded`; this
avoids Java module package conflicts with other NeoForge mods. NeoForge supplies
the SLF4J API and logging implementation, so JDA's duplicate SLF4J API is not
packaged.

OkHttp and Okio bring a Kotlin standard-library runtime requirement. The mod
bundles Kotlin 2.2.21 as a fallback and declares compatibility with 2.2.21 or
newer through Jar-in-Jar metadata. NeoForge can therefore select a newer shared
copy, such as the one packaged by Kotlin for Forge, without making Kotlin for
Forge a mandatory dependency. Kotlin is not relocated because Kotlin-compiled
libraries rely on Kotlin metadata and runtime package names.

The Gradle `check` lifecycle verifies the deployable JAR's mod and Jar-in-Jar
metadata. Every mod property must be expanded, the expected mod ID must be
present, and the Kotlin fallback must exist with the negotiable `[2.2.21,)`
range. These checks catch invalid metadata before deployment.
