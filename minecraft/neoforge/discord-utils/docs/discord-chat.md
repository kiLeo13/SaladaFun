# Discord chat bridge

The optional bridge mirrors accepted Minecraft chat to one Discord channel,
mirrors supported Discord messages back to every online player, and reports
player joins and leaves in that channel. It also provides registered message
commands scoped to that channel. The mod owns one JDA gateway session for all
current and future Discord features.

## Configuration

Start the server once to create `config/discord-utils.toml`, then configure it:

```toml
[discord-chat]
enabled = true
token = "BOT_TOKEN"
webhook-url = "https://discord.com/api/webhooks/WEBHOOK_ID/WEBHOOK_TOKEN"
channel-id = "DISCORD_CHANNEL_ID"
```

`enabled = false` is the safe default. When enabled, all three credential fields
are required. The webhook URL must be an HTTPS Discord incoming-webhook URL, and
the channel ID must be a non-zero Discord snowflake. The mod rejects malformed
configuration without logging either secret.

## Discord setup

Create one bot application and one incoming webhook in the same target channel.
Enable the bot's **Message Content Intent** in the Discord Developer Portal.
Give the bot View Channel, Read Message History, and Send Messages access to the
target channel. The bot reads inbound messages and sends player-presence
notifications. The webhook sends outbound Minecraft chat, preserving the
Minecraft player's name and MC Heads avatar.

## Message behavior

- Minecraft to Discord: forwards uncancelled final chat only, strips messages
  that are blank after Minecraft formatting, limits content to Discord's 2,000
  code-point maximum, and disables all allowed mentions. The webhook avatar uses
  `https://api.mcheads.org/head/{username}/128/hat`, allowing MC Heads to resolve
  the player's current head and outer/hat layer even when an offline server gives
  the player a non-Mojang UUID.
- Discord to Minecraft: accepts ordinary user messages only from the configured
  channel. The single inbound gateway first dispatches a registered command; an
  unknown first token falls through to Minecraft chat. Bot and webhook messages
  are ignored to prevent loops. Text, image attachment counts, and sticker
  counts are rendered for every online player. A Discord reply adds a compact
  dark-gray reference line above the ordinary bridge message:

  ```text
  ┃ respondendo a Lucas: Hello, guys
  [Discord] Leo13: Oiee, Lucas
  ```

  Reference whitespace is collapsed to keep the preview on one logical line.
  A media-only reference shows its image and sticker counts. If Discord cannot
  resolve a deleted or unavailable referenced message, the new message is still
  delivered normally without a reference line. This is display-only context;
  the mod does not add Minecraft reply support.
- Player presence: Padinho sends `**Player** entrou no servidor` when a player
  completes login. A routine client disconnect sends `**Player** saiu do
  servidor`; timeouts, kicks, network failures, and other reasoned disconnects
  instead send `**Player** desconectou: <reason>` using the reason retained by
  the server connection. Each bot message enables Components V2 and contains
  one text display inside one container, using green (`#57F287`) for joins and
  red (`#ED4245`) for leaves. Player names and disconnect reasons are
  Markdown-escaped, reason whitespace is normalized, oversized text is safely
  truncated, and all allowed mentions are disabled.
- Discord callbacks never access Minecraft state directly. Delivery is scheduled
  on the dedicated server thread.

## Message commands

Message-command triggers include their prefix and match the first
whitespace-delimited token case-insensitively. Commands work only in the
configured Minecraft Discord channel. A handled command is not broadcast into
Minecraft; unknown commands remain ordinary chat.

`!players` snapshots the online player names on the Minecraft server thread and
lists them alphabetically through Padinho's bot identity:

```text
## <:mc_grass:1547842476193357885> Players Online
- PlayerOne
- PlayerTwo
```

The response enables Components V2 and uses one text display inside a container
with the dark Minecraft-green `#3C8527` accent. It has no footer, separator, or
player count, and all allowed mentions are disabled. An empty server displays
`- Nenhum jogador online`. A normal response uses one message; if the complete
list would exceed Discord's 4,000-character component-text limit, the mod emits
additional messages with the same minimal shape rather than dropping players.

## Reloading and failure handling

NeoForge config reloads stage a replacement gateway connection. The new
connection must reach `READY` and prove the configured channel is visible before
it replaces the existing one. Invalid credentials, unavailable channels, and
connection failures leave the active bridge untouched. Set `enabled = false` and
reload the configuration to disconnect deliberately.

## Security and verification

Treat `token` and `webhook-url` as passwords: do not commit them, paste them in
issue reports, or share the generated TOML. The bridge redacts both from its
diagnostics.

Before deployment, run:

```text
.\gradlew.bat clean build
```

On a test server, verify both chat directions; join and use the client's
Disconnect button to confirm Padinho emits the ordinary green and red Components
V2 notifications; force a connection timeout or kick to confirm the red message
uses `desconectou: <reason>`; confirm an offline-mode player's username resolves
to their head and outer/hat layer in the webhook avatar; invoke `!players` with
zero, one, and multiple online players; confirm `!players` in another channel is
ignored and `!unknown` still reaches Minecraft chat; verify a bot message and a
webhook message are ignored; reply to text and media-only Discord messages and
confirm the dark-gray reference appears only above the Discord-to-Minecraft
message; and confirm a broken replacement configuration does not interrupt a
working bridge.

Configuration reloads stage a replacement connection until Discord READY proves
the channel is visible; an invalid or failed replacement leaves the existing
bridge active. Tokens and webhook URLs are secrets and must never be committed.
