# Discord chat bridge

The optional bridge mirrors accepted Minecraft chat to one Discord channel and
mirrors supported Discord messages back to every online player. The mod owns one
JDA gateway session for all current and future Discord features.

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
target channel. The bot reads inbound messages; the webhook sends outbound
Minecraft messages, preserving the Minecraft player's name and MC Heads avatar.

## Message behavior

- Minecraft to Discord: forwards uncancelled final chat only, strips messages
  that are blank after Minecraft formatting, limits content to Discord's 2,000
  code-point maximum, and disables all allowed mentions.
- Discord to Minecraft: accepts ordinary user messages only from the configured
  channel. Bot and webhook messages are ignored to prevent loops. Text, image
  attachment counts, and sticker counts are rendered for every online player.
- Discord callbacks never access Minecraft state directly. Delivery is scheduled
  on the dedicated server thread.

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

On a test server, verify both directions, a bot message is ignored, a webhook
message is ignored, and a broken replacement configuration does not interrupt a
working bridge.

Configuration reloads stage a replacement connection until Discord READY proves
the channel is visible; an invalid or failed replacement leaves the existing
bridge active. Tokens and webhook URLs are secrets and must never be committed.
