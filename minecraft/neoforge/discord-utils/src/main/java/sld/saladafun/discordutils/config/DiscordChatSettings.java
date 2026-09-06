package sld.saladafun.discordutils.config;

import java.net.URI;
import java.util.Locale;
import java.util.Objects;
import java.util.Set;

/** Immutable, validated Discord chat bridge settings with redacted diagnostics. */
public record DiscordChatSettings(boolean enabled, String token, String webhookUrl, String channelId) {
    private static final Set<String> WEBHOOK_HOSTS = Set.of(
        "discord.com",
        "ptb.discord.com",
        "canary.discord.com",
        "discordapp.com",
        "ptb.discordapp.com",
        "canary.discordapp.com"
    );
    private static final DiscordChatSettings DISABLED = new DiscordChatSettings(false, "", "", "");

    public DiscordChatSettings {
        Objects.requireNonNull(token, "token");
        Objects.requireNonNull(webhookUrl, "webhookUrl");
        Objects.requireNonNull(channelId, "channelId");
    }

    /** Returns disabled settings without credentials. */
    public static DiscordChatSettings disabled() {
        return DISABLED;
    }

    /** Reads and strictly validates the current NeoForge configuration. */
    public static DiscordChatSettings fromConfig() {
        if (!DiscordConfig.ENABLED.get()) {
            return disabled();
        }

        String token = required("discord-chat.token", DiscordConfig.TOKEN.get());
        String webhook = required("discord-chat.webhook-url", DiscordConfig.WEBHOOK_URL.get());
        String channel = required("discord-chat.channel-id", DiscordConfig.CHANNEL_ID.get());
        validateWebhook(webhook);

        try {
            if (Long.parseUnsignedLong(channel) == 0L) {
                throw new NumberFormatException("zero");
            }
        } catch (NumberFormatException exception) {
            throw new IllegalArgumentException(
                "Invalid discord-chat.channel-id: expected a non-zero Discord snowflake",
                exception
            );
        }

        return new DiscordChatSettings(true, token, webhook, channel);
    }

    private static String required(String name, String value) {
        if (value == null || value.isBlank()) {
            throw new IllegalArgumentException("Missing required " + name);
        }

        return value.strip();
    }

    private static void validateWebhook(String value) {
        try {
            URI uri = URI.create(value);
            boolean valid = "https".equalsIgnoreCase(uri.getScheme())
                && uri.getHost() != null
                && WEBHOOK_HOSTS.contains(uri.getHost().toLowerCase(Locale.ROOT))
                && uri.getPath() != null
                && uri.getPath().matches("/api(?:/v\\d+)?/webhooks/\\d+/[^/]+/?");
            if (!valid) {
                throw new IllegalArgumentException();
            }
        } catch (RuntimeException exception) {
            throw new IllegalArgumentException(
                "Invalid discord-chat.webhook-url: expected an HTTPS Discord incoming webhook URL",
                exception
            );
        }
    }

    @Override
    public String toString() {
        return "DiscordChatSettings[enabled=%s, token=<redacted>, webhookUrl=<redacted>, channelId=%s]"
            .formatted(enabled, channelId);
    }
}
