# Discord chat bridge

## Scope

The optional Discord bridge connects accepted global Minecraft player chat with
one Discord guild message channel.

- Minecraft to Discord uses an incoming webhook. The apparent webhook username
  is the Minecraft username and the avatar is the player's current head rendered
  by `https://api.mcheads.org/head/{uuid}/128`.
- Discord to Minecraft uses JDA gateway events. Messages render as
  `[Discord] effective name message`, using the Discord user's effective name
  rather than a server-specific member nickname. The prefix uses Discord blurple.
- Uploaded image attachments and stickers add a magenta counter on the next line,
  for example `[+2 images] [+1 sticker]`. Non-image files, link-preview embeds,
  reactions, edits, and deletions are not mirrored.
- Bot and webhook-authored Discord messages are ignored. This prevents the
  Minecraft webhook from echoing back into Minecraft.

Minecraft messages are sent only after Paper's final, uncancelled
`AsyncChatEvent`. Adventure formatting is reduced to plain text for Discord.
Discord messages use JDA's `Message.getContentDisplay()` so resolved display text
is used instead of raw mention syntax.

## Discord setup

1. Create a Discord application and bot in the
   [Discord Developer Portal](https://discord.com/developers/applications).
2. On the bot page, enable the privileged **Message Content Intent**. The bridge
   cannot read ordinary guild message content without it.
3. Invite the bot to the guild and grant it **View Channel** for the channel being
   bridged. The bot does not send chat messages itself, so the webhook owns the
   outbound Discord permission.
4. Create an incoming webhook in the Discord destination channel and copy its URL.
5. In Discord developer mode, copy the guild message channel ID.
6. Add or update the following section in the runtime `config.yml`:

```yaml
discord-chat:
  enabled: true
  token: "BOT_TOKEN"
  webhook-url: "https://discord.com/api/webhooks/WEBHOOK_ID/WEBHOOK_TOKEN"
  channel-id: "DISCORD_CHANNEL_ID"
```

Existing installations do not automatically receive new default keys because
Bukkit preserves their current configuration file. A missing `discord-chat`
section is equivalent to `enabled: false`.

Both the bot token and complete webhook URL are credentials. Restrict read access
to the server configuration, never paste either value into logs or support
messages, and rotate them immediately if exposed. SaladaFun redacts both values
from its settings diagnostics and does not include them in connection errors.

## Runtime behavior

JDA uses `JDABuilder.createLight` with only `GUILD_MESSAGES` and
`MESSAGE_CONTENT`. This disables JDA's optional entity cache flags, member
chunking, and member cache. JDA still maintains the minimal guild/channel and
gateway session state needed to receive events.

Discord networking never blocks the Minecraft server thread:

- webhook sends use JDA's asynchronous rate-limited request queue;
- gateway callbacks copy message data and schedule the Minecraft broadcast on the
  server thread;
- Discord connection startup occurs asynchronously.

`/saladafun reloadconfig` validates all Discord values before applying them. A
changed enabled configuration starts an inactive candidate JDA session. The
candidate becomes active only after Discord reports READY and the configured
guild message channel is visible. Until then, the previous session remains
active. A failed candidate is closed without interrupting the previous bridge.
Disabling the feature closes both candidate and active sessions.

Minecraft-originated webhook messages have all Discord mentions disabled. Text
such as `@everyone` remains visible but cannot ping Discord users or roles.
Minecraft messages exceeding Discord's 2,000-code-point limit are truncated with
an ellipsis.

## Verification

After `mvn clean package`, deploy only `target/saladafun-1.0.jar` and perform
these checks on a non-production Purpur server:

1. Start with `enabled: false` and verify no Discord connection is attempted.
2. Enable valid credentials, run `/saladafun reloadconfig`, and verify the log
   reports that the bridge connected.
3. Send Minecraft chat containing `@everyone`; verify the webhook uses the
   player's username and head while producing no Discord ping.
4. Send Discord text from an account with a server nickname and include
   user/channel mentions; verify Minecraft shows the user's effective name
   without angle brackets and JDA's display form for the mentions.
5. Upload two images and one sticker; verify the next Minecraft line is magenta
   and reads `[+2 images] [+1 sticker]`.
6. Verify bot and webhook messages do not return to Minecraft.
7. Cancel a Minecraft chat event with another plugin and verify it is not sent.
8. Reload with an invalid/unavailable Discord channel and verify the previous
   working session remains active.
9. Disable or stop SaladaFun and verify the JDA session closes.
